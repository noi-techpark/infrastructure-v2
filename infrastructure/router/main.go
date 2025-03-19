package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"reflect"
	"strings"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v2/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/kelseyhightower/envconfig"
	"github.com/noi-techpark/go-opendatahub-ingest/urn"
	"go.opentelemetry.io/otel/codes"
	"opendatahub.com/x/gotel"
	"opendatahub.com/x/logger"
	"opendatahub.com/x/qmill"
)

var cfg struct {
	MQ_URI             string
	MQ_READY_EXCHANGE  string
	MQ_READY_QUEUE     string
	MQ_ROUTED_EXCHANGE string
	MQ_ROUTED_QUEUE    string
	LogLevel           string `default:"INFO"`
}

type mqErr struct {
	err string
	ctx []any
}

func (e *mqErr) Error() string { return e.err }
func NewMqErr(msg string, args ...any) *mqErr {
	return &mqErr{err: msg, ctx: args}
}

func initConfig() {
	err := envconfig.Process("APP", &cfg)
	if err != nil {
		log.Panic("Unable to initialize config", err)
	}
}

func handleMq(messages <-chan *message.Message) {
	for msg := range messages {
		go func() {
			ctx := msg.Context()
			defer gotel.GetSpan(ctx).End()
			log := logger.Get(ctx)

			err := handleMqMsg(msg)

			if err != nil {
				log.Error(err.err, err.ctx...)
				span := gotel.GetSpan(ctx)
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

	routed_msg := message.NewMessage(watermill.NewUUID(), msg.Payload)
	err = routed.Publish(ctx, publisher, routed_msg, routing_key)

	if err != nil {
		return NewMqErr("cannot publish message", err)
	}
	return nil
}

var publisher *amqp.Publisher
var routed *qmill.QMill

func main() {
	initConfig()
	logger.InitLogging()

	telc, err := gotel.NewConfigFromEnv()
	if err != nil {
		log.Panic("Unable to initialize telemetry config", err)
	}
	tel, err := gotel.NewTelemetry(context.Background(), telc)
	if err != nil {
		log.Panic("Unable to initialize config", err)
	}

	subs, err := qmill.NewQmill(cfg.MQ_URI,
		qmill.WithQueue(cfg.MQ_READY_QUEUE, true),
		qmill.WithBind(cfg.MQ_READY_EXCHANGE, ""),
		qmill.WithDeadLetter(fmt.Sprintf("%s-dl", cfg.MQ_READY_QUEUE), "fanout"),
		qmill.WithNoRequeueOnNack(true),
		qmill.WithLogger(watermill.NewSlogLogger(slog.Default())),
		qmill.WithTracing(tel),
	)
	if err != nil {
		panic(err)
	}

	routed, err = qmill.NewQmill(cfg.MQ_URI,
		qmill.WithExchange(cfg.MQ_ROUTED_EXCHANGE, "topic", true),
		qmill.WithAlternateExchange(fmt.Sprintf("%s-dl", cfg.MQ_ROUTED_EXCHANGE), "topic"),
		qmill.WithNoRequeueOnNack(true),
		qmill.WithLogger(watermill.NewSlogLogger(slog.Default())),
		qmill.WithTracing(tel),
	)
	if err != nil {
		panic(err)
	}

	publisher, err = routed.GetPublisher()
	if err != nil {
		log.Panic("Unable to initialize publisher", err)
	}

	sub, err := subs.GetSubscriber(context.Background())
	if err != nil {
		log.Panic("Unable to initialize sub", err)
	}

	handleMq(sub)
}
