// SPDX-FileCopyrightText: 2024 NOI Techpark <digital@noi.bz.it>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/noi-techpark/opendatahub-go-sdk/tel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type S3Client struct {
	svc    *minio.Client
	bucket string
}

func newS3Client() *S3Client {
	endpoint := "s3.amazonaws.com"
	secure := true

	if cfg.S3_ENDPOINT != "" {
		u, err := url.Parse(cfg.S3_ENDPOINT)
		if err != nil {
			log.Panic("s3 endpoint parse failed: ", err)
		}
		endpoint = u.Host
		secure = strings.EqualFold(u.Scheme, "https")
	}

	svc, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewEnvAWS(),
		Secure: secure,
		Region: cfg.S3_REGION,
	})
	if err != nil {
		log.Panic("s3 client init failed: ", err)
	}

	return &S3Client{svc: svc, bucket: cfg.S3_BUCKET}
}

// Upload puts data at key inside the configured bucket.
func (c *S3Client) Upload(ctx context.Context, key string, data []byte) error {
	ctx, span := tel.TraceStart(ctx, "s3.upload", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	span.SetAttributes(
		attribute.String("s3.bucket", c.bucket),
		attribute.String("s3.key", key),
		attribute.Int("s3.bytes", len(data)),
	)

	_, err := c.svc.PutObject(ctx, c.bucket, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("s3 PutObject failed: %w", err)
	}
	return nil
}
