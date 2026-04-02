// SPDX-FileCopyrightText: 2024 NOI Techpark <digital@noi.bz.it>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"context"
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

func buildDoc(m meta, contentType string, raw []byte, s3URN string) bson.M {
	doc := bson.M{
		"provider":      m.Provider,
		"bsontimestamp": m.Timestamp,
		"provenance":    m.Provenance,
		"content_type":  contentType,
	}
	if s3URN != "" {
		doc["raw_ref"] = s3URN
	} else {
		var rawData any
		if isText(contentType) {
			rawData = string(raw)
		} else {
			rawData = raw
		}
		doc["rawdata"] = rawData
	}
	if len(m.ExtraHeaders) > 0 {
		doc["meta"] = m.ExtraHeaders
	}
	return doc
}

// mongoWrite inserts a document into db=<prefix+part1>, collection=<part2>.
// If s3URN is non-empty, the payload is stored as a reference (raw_ref); otherwise
// raw bytes are stored inline as string for text types or binary for everything else.
func mongoWrite(ctx context.Context, m meta, contentType string, raw []byte, s3URN string) (writeResult, error) {
	ctx, span := tel.TraceStart(ctx, "mongo.write", trace.WithSpanKind(trace.SpanKindClient))
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

	res, err := mongoClient.Database(db).Collection(coll).InsertOne(ctx, buildDoc(m, contentType, raw, s3URN))
	if err != nil {
		return writeResult{}, fmt.Errorf("mongo insert failed: %w", err)
	}
	return writeResult{ID: res.InsertedID.(primitive.ObjectID).Hex(), DB: db, Coll: coll}, nil
}
