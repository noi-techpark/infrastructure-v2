package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"reflect"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v2/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/kelseyhightower/envconfig"
	"github.com/rabbitmq/amqp091-go"
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

func initLog() {
	level := &slog.LevelVar{}
	level.UnmarshalText([]byte(cfg.LogLevel))
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})))
}

// Ready stack is the ready-q and ready-dl bind to ready exchange.
// Here we find all messages successfully written to raw-data.
// The ready-dl is ment to contain messages sent to ready exchange but which for some reason we couldn't publish to "routed" exchange
//
// setupReadyStack setups both queues and returns the consumer for ready-q
func setupReadyStack() <-chan *message.Message {
	// ready-dl
	amqpConfig := amqp.NewDurablePubSubConfig(cfg.MQ_URI, amqp.GenerateQueueNameTopicName)

	subscriber, err := amqp.NewSubscriber(
		amqpConfig,
		watermill.NewSlogLogger(slog.Default()),
	)
	if err != nil {
		panic(err)
	}

	if err := subscriber.SubscribeInitialize(fmt.Sprintf("%s-dl", cfg.MQ_READY_QUEUE)); err != nil {
		panic(err)
	}

	// consumer
	amqpConfig = amqp.NewDurablePubSubConfig(cfg.MQ_URI,
		amqp.GenerateQueueNameConstant(cfg.MQ_READY_QUEUE))
	amqpConfig.Queue.Arguments = amqp091.Table{"x-dead-letter-exchange": fmt.Sprintf("%s-dl", cfg.MQ_READY_EXCHANGE)}
	amqpConfig.Exchange.Type = "direct"
	amqpConfig.Consume.NoRequeueOnNack = true

	subscriber, err = amqp.NewSubscriber(
		amqpConfig,
		watermill.NewSlogLogger(slog.Default()),
	)
	if err != nil {
		panic(err)
	}

	messages, err := subscriber.Subscribe(context.Background(), cfg.MQ_READY_EXCHANGE)
	if err != nil {
		panic(err)
	}
	return messages
}

// Routed stack is composed by routed exchange and routed-dl queue.
// Messages read from "ready-q" are routed using a derived routing_key to routed exchange.
// routed-dl queue us used as alternate-exchange to handle all those messages published with a routing key not bind to any queue
//
// setupRoutedStack setups routed-dl queue and returns routed exchange publisher
func setupRoutedStack() *amqp.Publisher {
	// routed-dl
	amqpConfig := amqp.NewDurablePubSubConfig(cfg.MQ_URI, amqp.GenerateQueueNameConstant(fmt.Sprintf("%s-dl", cfg.MQ_ROUTED_QUEUE)))
	amqpConfig.Exchange.GenerateName = func(n string) string { return fmt.Sprintf("%s-dl", cfg.MQ_ROUTED_EXCHANGE) }

	subscriber, err := amqp.NewSubscriber(
		amqpConfig,
		watermill.NewSlogLogger(slog.Default()),
	)
	if err != nil {
		panic(err)
	}

	if err := subscriber.SubscribeInitialize(fmt.Sprintf("%s-dl", cfg.MQ_ROUTED_QUEUE)); err != nil {
		panic(err)
	}

	// publisher
	amqpConfig = amqp.NewDurablePubSubConfig(cfg.MQ_URI, amqp.GenerateQueueNameTopicName)
	amqpConfig.Publish.GenerateRoutingKey = func(topic string) string { return topic }
	amqpConfig.Exchange.Durable = true
	amqpConfig.Exchange.AutoDeleted = false
	amqpConfig.Exchange.Type = "topic"
	amqpConfig.Exchange.GenerateName = func(n string) string { return cfg.MQ_ROUTED_QUEUE }
	amqpConfig.Exchange.Arguments = amqp091.Table{"alternate-exchange": fmt.Sprintf("%s-dl", cfg.MQ_ROUTED_EXCHANGE)}

	amqpConfig.Queue.Exclusive = false
	amqpConfig.Queue.NoWait = false

	publisher, err := amqp.NewPublisher(
		amqpConfig,
		watermill.NewSlogLogger(slog.Default()),
	)
	if err != nil {
		panic(err)
	}

	return publisher
}

func handleMq(messages <-chan *message.Message) {
	for msg := range messages {
		go func() {
			err := handleMqMsg(msg)

			if err != nil {
				slog.Error(err.err, err.ctx...)
				msg.Nack()
			} else {
				msg.Ack()
			}
		}()
	}
}

func generateRoutingKey(body map[string]any) (string, error) {
	// body must include "db" and "collection" field and we construct routing key
	// as db.collection
	db, ok := body["db"]
	if !ok || reflect.TypeOf(db).Kind() != reflect.String {
		return "", errors.New("'db' payload field missing or wrong type")
	}
	collection, ok := body["collection"]
	if !ok || reflect.TypeOf(collection).Kind() != reflect.String {
		return "", errors.New("'collection' payload field missing or wrong type")
	}
	return fmt.Sprintf("%s.%s", db, collection), nil
}

func handleMqMsg(msg *message.Message) *mqErr {
	slog.Debug("received message", "id", msg.UUID, "payload", string(msg.Payload))
	var body map[string]any
	if err := json.Unmarshal(msg.Payload, &body); err != nil {
		return NewMqErr("cannot unmarshal json", "payload", string(msg.Payload))
	}
	slog.Debug("unmarshalled json", "json", body)
	routing_key, err := generateRoutingKey(body)
	if err != nil {
		return NewMqErr(err.Error(), "json", body)
	}

	routed_msg := message.NewMessage(watermill.NewUUID(), msg.Payload)
	err = publisher.Publish(routing_key, routed_msg)

	if err != nil {
		return NewMqErr("cannot publish message", err)
	}
	return nil
}

var publisher *amqp.Publisher

func main() {
	initConfig()
	initLog()
	publisher = setupRoutedStack()
	messages := setupReadyStack()

	handleMq(messages)
}
