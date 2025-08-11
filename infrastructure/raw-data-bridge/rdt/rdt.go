// SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>
// 
// SPDX-License-Identifier: CC0-1.0

package rdt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/noi-techpark/opendatahub-go-sdk/ingest/urn"
	"github.com/noi-techpark/opendatahub-go-sdk/tel"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var (
	ErrDocumentNotFound = errors.New("document not found")
	ErrBadURN           = errors.New("bad urn format")
)

var mongoClient *mongo.Client

// NOTE: using a map does not preserve field order.
type Document map[string]any

func (r Document) MarshalJSON() ([]byte, error) {
	type tmp Document
	// remove fields added for raw data storage
	delete(r, "_id")
	delete(r, "bsontimestamp")
	return json.Marshal(tmp(r))
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
	if err := client.Database("admin").RunCommand(context.TODO(), bson.D{{Key: "ping", Value: 1}}).Decode(&result); err != nil {
		return nil, err
	}
	return client, nil
}

func GetDocument(ctx context.Context, urn *urn.URN) (*Document, error) {
	db, coll, err := constructTableTarget(urn)
	if err != nil {
		return nil, ErrBadURN
	}
	id, err := primitive.ObjectIDFromHex(urn.GetResourceID())
	if err != nil {
		return nil, ErrBadURN
	}

	// Start a new client span for the MongoDB FindOne operation.
	tracer := otel.Tracer(tel.GetServiceName())
	ctx, span := tracer.Start(ctx, "find-raw", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	// Set attributes for the MongoDB operation.
	span.SetAttributes(
		attribute.String("db.name", "mongo-raw-data-table"),
		attribute.String("db.operation", "FindOne"),
		attribute.String("db.mongodb.db", db),
		attribute.String("db.mongodb.collection", coll),
		attribute.String("peer.host", "mongo-raw-data-table"),
	)

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
