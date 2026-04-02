// SPDX-FileCopyrightText: 2024 NOI Techpark <digital@noi.bz.it>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/noi-techpark/opendatahub-go-sdk/tel"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)


// mongo db / collection = provider1 / provider2
func providerParts(m meta) (db, coll string, err error) {
	parts := strings.SplitN(m.Provider, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("provider must be 'part1/part2', got %q", m.Provider)
	}
	return cfg.DB_PREFIX + parts[0], parts[1], nil
}

type writeResult struct {
	ID   string
	DB   string
	Coll string
}

// mongoWriteRaw inserts the raw payload into db=<prefix+part1>, collection=<part2>.
func mongoWriteRaw(ctx context.Context, m meta, raw []byte, contentType string) (writeResult, error) {
	ctx, span := tel.TraceStart(ctx, "mongo.write.small", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	db, coll, err := providerParts(m)
	if err != nil {
		return writeResult{}, err
	}
	span.SetAttributes(
		attribute.String("db.system", "mongodb"),
		attribute.String("db.operation", "InsertOne"),
		attribute.String("db.name", db),
		attribute.String("db.mongodb.collection", coll),
	)

	var rawData any
	switch {
	case contentType == "application/bson":
		var doc bson.M
		if err := json.Unmarshal(raw, &doc); err == nil {
			rawData = doc
		} else if err := bson.Unmarshal(raw, &doc); err == nil {
			rawData = doc
		} else {
			rawData = raw
		}
	case isBinary(contentType):
		rawData = raw
	default:
		rawData = string(raw)
	}
	doc := bson.M{
		"provider":      m.Provider,
		"bsontimestamp": m.Timestamp,
		"provenance":    m.Provenance,
		"content_type":  contentType,
		"rawdata":       rawData,
	}
	if len(m.ExtraHeaders) > 0 {
		doc["meta"] = m.ExtraHeaders
	}
	res, err := mongoClient.Database(db).Collection(coll).InsertOne(ctx, doc)
	if err != nil {
		return writeResult{}, fmt.Errorf("mongo insert failed: %w", err)
	}
	return writeResult{ID: res.InsertedID.(primitive.ObjectID).Hex(), DB: db, Coll: coll}, nil
}

// The actual payload is on S3 at s3URN.
func mongoWriteS3Ref(ctx context.Context, m meta, s3URN, contentType string) (writeResult, error) {
	ctx, span := tel.TraceStart(ctx, "mongo.write.large", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	db, coll, err := providerParts(m)
	if err != nil {
		return writeResult{}, err
	}
	span.SetAttributes(
		attribute.String("db.system", "mongodb"),
		attribute.String("db.operation", "InsertOne"),
		attribute.String("db.name", db),
		attribute.String("db.mongodb.collection", coll),
	)

	doc := bson.M{
		"provider":      m.Provider,
		"bsontimestamp": m.Timestamp,
		"provenance":    m.Provenance,
		"content_type":  contentType,
		"raw_ref":       s3URN,
	}
	if len(m.ExtraHeaders) > 0 {
		doc["meta"] = m.ExtraHeaders
	}
	res, err := mongoClient.Database(db).Collection(coll).InsertOne(ctx, doc)
	if err != nil {
		return writeResult{}, fmt.Errorf("mongo insert failed: %w", err)
	}
	return writeResult{ID: res.InsertedID.(primitive.ObjectID).Hex(), DB: db, Coll: coll}, nil
}
