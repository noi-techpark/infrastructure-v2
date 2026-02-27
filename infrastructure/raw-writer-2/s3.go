// SPDX-FileCopyrightText: 2024 NOI Techpark <digital@noi.bz.it>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"bytes"
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/noi-techpark/opendatahub-go-sdk/tel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type S3Client struct {
	svc    *s3.Client
	bucket string
}

func newS3Client(ctx context.Context) *S3Client {
	awsConfig, err := awscfg.LoadDefaultConfig(ctx,
		awscfg.WithRegion(cfg.S3_REGION),
	)
	if err != nil {
		log.Panic("s3 config load failed: ", err)
	}

	svc := s3.NewFromConfig(awsConfig, func(o *s3.Options) {
		if cfg.S3_ENDPOINT != "" {
			o.BaseEndpoint = aws.String(cfg.S3_ENDPOINT)
			o.UsePathStyle = true // required for MinIO and other path-style S3 endpoints
		}
	})

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

	_, err := c.svc.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("s3 PutObject failed: %w", err)
	}
	return nil
}
