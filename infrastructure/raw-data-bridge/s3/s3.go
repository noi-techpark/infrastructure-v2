// SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>
//
// SPDX-License-Identifier: CC0-1.0

package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Config struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	Region          string
}

type S3Client struct {
	mc *minio.Client
}

// NewClient creates a long-lived S3 client from config.
// Endpoint may include a scheme (http:// or https://); SSL is inferred from it.
// If no endpoint is given, AWS S3 (s3.amazonaws.com) is used over HTTPS.
func NewClient(cfg Config) (*S3Client, error) {
	endpoint := "s3.amazonaws.com"
	useSSL := true

	if cfg.Endpoint != "" {
		useSSL = strings.HasPrefix(cfg.Endpoint, "https://")
		endpoint = strings.TrimPrefix(cfg.Endpoint, "https://")
		endpoint = strings.TrimPrefix(endpoint, "http://")
	}

	mc, err := minio.New(endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure:       useSSL,
		Region:       cfg.Region,
		BucketLookup: minio.BucketLookupPath,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client init: %w", err)
	}
	return &S3Client{mc: mc}, nil
}

// Retrieve fetches the object at the given S3 URN and returns its contents.
// URN format: urn:s3:{bucket}:{key}
// If the key ends in .zst the object is zstd-decompressed before returning.
// The download (GetObject + ReadAll) is retried up to 3 times with backoff to
// handle transient network errors (e.g. unexpected EOF mid-stream).
func (c *S3Client) Retrieve(ctx context.Context, urn string) ([]byte, error) {
	bucket, key, err := ParseURN(urn)
	if err != nil {
		return nil, err
	}

	const maxAttempts = 3
	var lastErr error
	for attempt := range maxAttempts {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
			}
		}

		obj, err := c.mc.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
		if err != nil {
			lastErr = fmt.Errorf("s3 get object: %w", err)
			slog.WarnContext(ctx, "s3 get object failed, retrying", "attempt", attempt+1, "key", key, "err", err)
			continue
		}

		data, err := io.ReadAll(obj)
		obj.Close()
		if err != nil {
			lastErr = fmt.Errorf("s3 read object: %w", err)
			slog.WarnContext(ctx, "s3 read object failed, retrying", "attempt", attempt+1, "key", key, "err", err)
			continue
		}

		if attempt > 0 {
			slog.WarnContext(ctx, "s3 retrieve succeeded after retry", "attempt", attempt+1, "key", key)
		}

		if strings.HasSuffix(key, ".zst") {
			return DecompressZstd(data)
		}
		return data, nil
	}
	return nil, lastErr
}

// ParseURN splits a urn:s3:{bucket}:{key} URN into its bucket and key parts.
func ParseURN(urn string) (bucket, key string, err error) {
	// Expected format: urn:s3:{bucket}:{key}
	// SplitN with n=4 so the key (which may contain colons) is kept intact.
	parts := strings.SplitN(urn, ":", 4)
	if len(parts) != 4 || parts[0] != "urn" || parts[1] != "s3" || parts[2] == "" || parts[3] == "" {
		return "", "", fmt.Errorf("invalid S3 URN: %q", urn)
	}
	return parts[2], parts[3], nil
}

// DecompressZstd decompresses a zstd-compressed byte slice.
func DecompressZstd(data []byte) ([]byte, error) {
	dec, err := zstd.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("zstd reader init: %w", err)
	}
	defer dec.Close()
	return io.ReadAll(dec)
}
