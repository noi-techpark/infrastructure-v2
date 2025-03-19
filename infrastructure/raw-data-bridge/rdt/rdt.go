package rdt

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/noi-techpark/go-opendatahub-ingest/urn"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"opendatahub.com/x/gotel"
)

var (
	ErrDocumentNotFound = errors.New("document not found")
	ErrBadURN           = errors.New("bad urn format")
)

var mongoClient *mongo.Client

type Document struct {
	Provider  string    `json:"provider"`
	Timestamp time.Time `json:"timestamp"`
	Rawdata   string    `bson:"rawdata" json:"rawdata"`
}

func InitRawDataConnection(uri string) {
	mclient, err := mongoConnect(uri)
	if err != nil {
		slog.Error("could not initialize mongo. aborting", "err", err)
		panic(err)
	}
	mongoClient = mclient
}

func constructTableTarget(urn *urn.URN) (string, string, error) {
	provider_tokens := urn.GetNSSWithoutID()
	if len(provider_tokens) < 2 {
		return "", "", fmt.Errorf("urn format invalid: %s", urn.String())
	}
	return provider_tokens[0], strings.Join(provider_tokens[1:], "."), nil
}

func mongoConnect(uri string) (*mongo.Client, error) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
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

func GetDocument(ctx context.Context, tel *gotel.Telemetry, urn *urn.URN) (*Document, error) {
	db, coll, err := constructTableTarget(urn)
	if err != nil {
		return nil, ErrBadURN
	}
	id, err := primitive.ObjectIDFromHex(urn.GetResourceID())
	if err != nil {
		return nil, ErrBadURN
	}

	ctx, span := tel.Tracer.Start(ctx, "MongoDB.FindOne", trace.WithSpanKind(trace.SpanKindClient), trace.WithAttributes(
		// Set attributes for the MongoDB operation.
		attribute.String("db.system", "mongodb"),
		attribute.String("db.name", "raw-data-table"),
		attribute.String("db.operation", "findOne"),
		attribute.String("db.mongodb.collection", coll),
		attribute.String("db.mongodb.db", db),
		attribute.String("peer.host", "raw-data-table"),
	))
	defer span.End()

	r := &Document{}
	if err := mongoClient.Database(db).Collection(coll).FindOne(ctx, bson.M{"_id": id}).Decode(r); err != nil {
		// Record the error on the span.
		span.RecordError(err)
		span.SetStatus(codes.Error, fmt.Sprintf("findOne error: %s", err.Error()))
		if err == mongo.ErrNoDocuments {
			return nil, ErrDocumentNotFound
		}
		return nil, err
	}
	return r, nil
}
