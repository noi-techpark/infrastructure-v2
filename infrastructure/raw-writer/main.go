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
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v2/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/kelseyhightower/envconfig"
	"github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var cfg struct {
	MQ_URI      string
	MQ_Exchange string
	MQ_QUEUE    string
	LogLevel    string `default:"INFO"`
	MONGO_URI   string
	DB_PREFIX   string
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

func setupDeadletter() {
	amqpConfig := amqp.NewDurablePubSubConfig(cfg.MQ_URI, amqp.GenerateQueueNameTopicName)

	subscriber, err := amqp.NewSubscriber(
		amqpConfig,
		watermill.NewSlogLogger(slog.Default()),
	)
	if err != nil {
		panic(err)
	}

	if err := subscriber.SubscribeInitialize(cfg.MQ_Exchange + "-dl"); err != nil {
		panic(err)
	}
}

func setupMQ() <-chan *message.Message {
	setupDeadletter()

	amqpConfig := amqp.NewDurablePubSubConfig(cfg.MQ_URI, amqp.GenerateQueueNameConstant(cfg.MQ_QUEUE))
	amqpConfig.Queue.Arguments = amqp091.Table{"x-dead-letter-exchange": cfg.MQ_Exchange + "-dl"}
	amqpConfig.Consume.NoRequeueOnNack = true

	subscriber, err := amqp.NewSubscriber(
		amqpConfig,
		watermill.NewSlogLogger(slog.Default()),
	)
	if err != nil {
		panic(err)
	}

	messages, err := subscriber.Subscribe(context.Background(), cfg.MQ_Exchange)
	if err != nil {
		panic(err)
	}
	return messages
}

func setupOutMQ() *amqp.Publisher {
	amqpConfig := amqp.NewDurablePubSubConfig(cfg.MQ_URI, amqp.GenerateQueueNameConstant(""))
	amqpConfig.Exchange.Durable = true
	amqpConfig.Exchange.AutoDeleted = false
	amqpConfig.Exchange.Type = "direct"

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

func handleMqMsg(msg *message.Message) *mqErr {
	slog.Debug("received message", "id", msg.UUID, "payload", string(msg.Payload))
	var body map[string]any
	if err := json.Unmarshal(msg.Payload, &body); err != nil {
		return NewMqErr("cannot unmarshal json", "payload", string(msg.Payload))
	}
	slog.Debug("unmarshalled json", "json", body)

	// parse timestamp field, put into bsontimestamp
	timestamp, err := parseTimestamp(&body)
	if err != nil {
		return NewMqErr(err.Error(), "json", body)
	}
	body["bsontimestamp"] = timestamp

	// parse provier string to retrieve mongo db and collection names
	db, coll, err := parseProvider(body)
	if err != nil {
		return NewMqErr(err.Error(), "json", body)
	}

	inserted_id := primitive.ObjectID{}
	atom := NewAtom(
		NewQuark(func() (interface{}, error) {
			inserted_id, err = mongoWrite(db, coll, body, mongoClient)
			return inserted_id, err
		},
			func(toRollback interface{}) {
				id := toRollback.(primitive.ObjectID)
				mongoDelete(db, coll, id, mongoClient)
			}),
		NewQuark(func() (interface{}, error) {
			messagePayload, err := json.Marshal(map[string]any{
				"id":         inserted_id.String(),
				"db":         db,
				"collection": coll,
			})
			if err != nil {
				return nil, err
			}
			msg := message.NewMessage(watermill.NewUUID(), messagePayload)
			return nil, readyQueue.Publish("ready", msg)
		}, nil),
	)

	err = atom.Run()
	if err != nil {
		return NewMqErr(err.Error(), err)
	}

	return nil
}

func parseTimestamp(body *map[string]any) (time.Time, error) {
	tstr, ok := (*body)["timestamp"]
	if !ok {
		return time.Time{}, errors.New("missing timestamp field")
	}
	t, err := time.Parse(time.RFC3339, tstr.(string))
	if err != nil {
		return time.Time{}, fmt.Errorf("unparseable timestamp: %s", tstr)
	}

	return t, nil
}

func parseProvider(body map[string]any) (string, string, error) {
	// provider string is in format "db/collection"
	providerstr, ok := body["provider"]
	if !ok || reflect.TypeOf(providerstr).Kind() != reflect.String {
		return "", "", errors.New("provider missing or wrong type")
	}
	provider := strings.SplitN(providerstr.(string), "/", 2)
	if provider == nil || len(provider) != 2 {
		return "", "", fmt.Errorf("provider format invalid: %s", providerstr)
	}
	return provider[0], provider[1], nil
}

func initMongo() *mongo.Client {
	mclient, err := mongoConnect()
	if err != nil {
		slog.Error("could not initialize mongo. aborting", "err", err)
		panic(err)
	}
	return mclient
}

func mongoConnect() (*mongo.Client, error) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(cfg.MONGO_URI))
	if err != nil {
		return nil, err
	}
	// Send a ping to confirm a successful connection
	var result bson.M
	if err := client.Database("admin").RunCommand(context.TODO(), bson.D{{"ping", 1}}).Decode(&result); err != nil {
		return nil, err
	}
	return client, nil
}

func mongoWrite(db string, coll string, obj map[string]any, client *mongo.Client) (primitive.ObjectID, error) {
	collection := client.Database(cfg.DB_PREFIX + db).Collection(coll)
	result, err := collection.InsertOne(context.TODO(), obj)
	if err != nil {
		return primitive.ObjectID{}, NewMqErr("error inserting msg to mongo", err)
	}
	return result.InsertedID.(primitive.ObjectID), nil
}

func mongoDelete(db string, coll string, id primitive.ObjectID, client *mongo.Client) error {
	collection := client.Database(cfg.DB_PREFIX + db).Collection(coll)
	_, err := collection.DeleteOne(context.TODO(), bson.M{"_id": id})
	if err != nil {
		return NewMqErr("error deleting msg from mongo", err)
	}
	return nil
}

var mongoClient *mongo.Client
var readyQueue *amqp.Publisher

func main() {
	initConfig()
	initLog()
	mongoClient = initMongo()
	readyQueue = setupOutMQ()

	messages := setupMQ()

	handleMq(messages)
}
