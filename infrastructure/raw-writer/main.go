package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v2/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/kelseyhightower/envconfig"
	"github.com/noi-techpark/go-opendatahub-ingest/urn"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"opendatahub.com/x/gotel"
	"opendatahub.com/x/logger"
	"opendatahub.com/x/qmill"
)

var cfg struct {
	MQ_URI            string
	MQ_Exchange       string
	MQ_QUEUE          string
	MQ_READY_EXCHANGE string
	LogLevel          string `default:"INFO"`
	MONGO_URI         string
	DB_PREFIX         string
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

func handleMqMsg(msg *message.Message) *mqErr {
	ctx := msg.Context()
	log := logger.Get(ctx)

	log.Debug("received message", "id", msg.UUID, "payload", string(msg.Payload))
	var body map[string]any
	if err := json.Unmarshal(msg.Payload, &body); err != nil {
		return NewMqErr("cannot unmarshal json", "payload", string(msg.Payload))
	}
	log.Debug("unmarshalled json", "json", body)

	// parse timestamp field, put into bsontimestamp
	timestamp, err := parseTimestamp(&body)
	if err != nil {
		return NewMqErr(err.Error(), "json", body)
	}
	body["bsontimestamp"] = timestamp

	// parse provier string to retrieve mongo db and collection names
	provider, err := getProvider(body)
	if err != nil {
		return NewMqErr(err.Error(), "json", body)
	}
	db, coll, err := constructTableTarget(provider)
	if err != nil {
		return NewMqErr(err.Error(), "json", body)
	}

	u, ok := urn.RawUrnFromProviderURI(provider)
	if !ok {
		return NewMqErr("provider generated invalid urn", "json", body)
	}

	inserted_id := primitive.ObjectID{}
	atom := NewAtom(
		NewQuark(
			func() (interface{}, error) {
				inserted_id, err = mongoWrite(ctx, db, coll, body, mongoClient)
				return inserted_id, err
			},
			func(toRollback interface{}) {
				id := toRollback.(primitive.ObjectID)
				mongoDelete(db, coll, id, mongoClient)
			}),
		NewQuark(
			func() (interface{}, error) {
				// concat document id to urn
				u.AddNSS(inserted_id.Hex())
				messagePayload, err := json.Marshal(map[string]any{
					"id":         inserted_id.Hex(),
					"db":         db,
					"collection": coll,
					"urn":        u.String(),
				})
				if err != nil {
					return nil, err
				}
				msg := message.NewMessage(watermill.NewUUID(), messagePayload)
				return nil, ready.Publish(ctx, readyPublisher, msg, cfg.MQ_READY_EXCHANGE)
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

func getProvider(body map[string]any) (string, error) {
	// provider string is in format "db/collection"
	providerstr, ok := body["provider"]
	if !ok || reflect.TypeOf(providerstr).Kind() != reflect.String {
		return "", errors.New("provider missing or wrong type")
	}
	uri, err := url.ParseRequestURI("opendatahub:" + providerstr.(string))
	if err != nil {
		return "", err
	}

	return uri.Opaque, nil
}

func constructTableTarget(provider string) (string, string, error) {
	provider_tokens := strings.Split(provider, "/")
	if len(provider_tokens) < 2 {
		return "", "", fmt.Errorf("provider format invalid: %s", provider)
	}
	return provider_tokens[0], strings.Join(provider_tokens[1:], "."), nil
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

func mongoWrite(ctx context.Context, db string, coll string, obj map[string]any, client *mongo.Client) (primitive.ObjectID, error) {
	_, span := tel.Tracer.Start(ctx, "MongoDB.FindOne", trace.WithSpanKind(trace.SpanKindClient), trace.WithAttributes(
		// Set attributes for the MongoDB operation.
		attribute.String("db.system", "mongodb"),
		attribute.String("db.name", db),
		attribute.String("db.operation", "InsertOne"),
		attribute.String("db.mongodb.collection", coll),
		attribute.String("db.mongodb.database", cfg.DB_PREFIX+db),
		attribute.String("peer.service", "raw-data-table"),
	))
	defer span.End()

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
var readyPublisher *amqp.Publisher
var ready *qmill.QMill
var tel *gotel.Telemetry

func main() {
	logger.InitLogging()
	initConfig()

	telc, err := gotel.NewConfigFromEnv()
	if err != nil {
		log.Panic("Unable to initialize telemetry config", err)
	}
	tel, err = gotel.NewTelemetry(context.Background(), telc)
	if err != nil {
		log.Panic("Unable to initialize config", err)
	}

	mongoClient = initMongo()

	println(cfg.MQ_URI)
	subs, err := qmill.NewQmill(cfg.MQ_URI,
		qmill.WithQueue(cfg.MQ_QUEUE, true),
		qmill.WithBind(cfg.MQ_QUEUE, ""),
		qmill.WithDeadLetter(fmt.Sprintf("%s-dl", cfg.MQ_QUEUE), "fanout"),
		qmill.WithNoRequeueOnNack(true),
		qmill.WithLogger(watermill.NewSlogLogger(slog.Default())),
		qmill.WithTracing(tel),
	)
	if err != nil {
		panic(err)
	}

	ready, err = qmill.NewQmill(cfg.MQ_URI,
		qmill.WithExchange(cfg.MQ_READY_EXCHANGE, "direct", true),
		qmill.WithAlternateExchange(fmt.Sprintf("%s-dl", cfg.MQ_READY_EXCHANGE), "fanout"),
		qmill.WithNoRequeueOnNack(true),
		qmill.WithLogger(watermill.NewSlogLogger(slog.Default())),
		qmill.WithTracing(tel),
	)
	if err != nil {
		panic(err)
	}

	readyPublisher, err = ready.GetPublisher()
	if err != nil {
		log.Panic("Unable to initialize readyPublisher", err)
	}

	sub, err := subs.GetSubscriber(context.Background())
	if err != nil {
		log.Panic("Unable to initialize sub", err)
	}

	defer tel.FlushOnPanic()
	handleMq(sub)
}
