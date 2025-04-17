package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"reflect"
	"strings"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/noi-techpark/opendatahub-go-sdk/ingest/ms"
	"github.com/noi-techpark/opendatahub-go-sdk/ingest/urn"
	"github.com/noi-techpark/opendatahub-go-sdk/qmill"
	"github.com/noi-techpark/opendatahub-go-sdk/tel"
	"github.com/noi-techpark/opendatahub-go-sdk/tel/logger"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var cfg struct {
	MQ_URI                 string
	MQ_READY_QUEUE         string
	MQ_ROUTED_EXCHANGE     string
	MQ_ROUTED_QUEUE        string
	MQ_UNROUTABLE_EXCHANGE string
	MQ_CLIENT_NAME         string
}

type mqErr struct {
	err string
	ctx []any
}

func (e *mqErr) Error() string { return e.err }
func NewMqErr(msg string, args ...any) *mqErr {
	return &mqErr{err: msg, ctx: args}
}

func handleMq(messages <-chan *message.Message) {
	for msg := range messages {
		go func() {
			ctx := msg.Context()
			span := trace.SpanFromContext(ctx)
			defer span.End()

			log := logger.Get(ctx)

			err := handleMqMsg(msg)

			if err != nil {
				log.Error(err.err, err.ctx...)
				if span != nil && span.IsRecording() {
					span.RecordError(err)
					span.SetStatus(codes.Error, err.Error())
				}

				msg.Nack()
			} else {
				msg.Ack()
			}
		}()
	}
}

func generateRoutingKey(body map[string]any) (string, error) {
	// body must include "urn" field and we construct routing key
	// NSS without last part replacing : -> .
	body_urn, ok := body["urn"]
	if !ok || reflect.TypeOf(body_urn).Kind() != reflect.String {
		return "", errors.New("'urn' payload field missing or wrong type")
	}
	u, ok := urn.Parse(body_urn.(string))
	if !ok {
		return "", errors.New("invalid 'urn' format")
	}

	return strings.Join(u.GetNSSWithoutID(), "."), nil
}

func handleMqMsg(msg *message.Message) *mqErr {
	ctx := msg.Context()
	log := logger.Get(ctx)

	log.Debug("received message", "id", msg.UUID, "payload", string(msg.Payload))
	var body map[string]any
	if err := json.Unmarshal(msg.Payload, &body); err != nil {
		return NewMqErr("cannot unmarshal json", "payload", string(msg.Payload))
	}
	log.Debug("unmarshalled json", "json", body)
	routing_key, err := generateRoutingKey(body)
	if err != nil {
		return NewMqErr(err.Error(), "json", body)
	}

	err = publisher.Publish(ctx, msg.Payload, routing_key)
	if err != nil {
		return NewMqErr("cannot publish message", "err", err, "json", body)
	}
	return nil
}

var publisher *qmill.QMill

func main() {
	ms.InitWithEnv(context.Background(), "APP", &cfg)
	defer tel.FlushOnPanic()

	subMill, err := qmill.NewSubscriberQmill(context.Background(), cfg.MQ_URI, cfg.MQ_CLIENT_NAME,
		qmill.WithQueue(cfg.MQ_READY_QUEUE, false),
		qmill.WithNoRequeueOnNack(true),
		qmill.WithLogger(watermill.NewSlogLogger(slog.Default())),
	)
	if err != nil {
		slog.Error("failed to initialize subscriber channel", "err", err)
		os.Exit(1)
	}

	publisher, err = qmill.NewPublisherQmill(context.Background(), cfg.MQ_URI, cfg.MQ_CLIENT_NAME,
		qmill.WithExchange(cfg.MQ_ROUTED_EXCHANGE, "topic", true),
		qmill.WithAlternateExchange(cfg.MQ_UNROUTABLE_EXCHANGE, "topic"),
		qmill.WithNoRequeueOnNack(true),
		qmill.WithLogger(watermill.NewSlogLogger(slog.Default())),
	)
	if err != nil {
		slog.Error("failed to initialize publisher channel", "err", err)
		os.Exit(1)
	}

	handleMq(subMill.Sub())
}
