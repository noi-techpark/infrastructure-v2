// SPDX-FileCopyrightText: 2024 NOI Techpark <digital@noi.bz.it>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/noi-techpark/opendatahub-go-sdk/tel"
	"github.com/noi-techpark/opendatahub-go-sdk/tel/logger"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const largeSizeThreshold = 5 * 1024 * 1024 // 5 MB

type meta struct {
	Provider   string
	Timestamp  time.Time
	Provenance string
}

func parseMeta(r *http.Request) (meta, error) {
	provider1 := r.PathValue("provider1")
	if provider1 == "" {
		return meta{}, fmt.Errorf("missing path segment: provider1")
	}

	provider2 := r.PathValue("provider2")
	if provider2 == "" {
		return meta{}, fmt.Errorf("missing path segment: provider2")
	}

	tsStr := r.PathValue("timestamp")
	if tsStr == "" {
		return meta{}, fmt.Errorf("missing path segment: timestamp")
	}
	ts, err := time.Parse(time.RFC3339, tsStr)
	if err != nil {
		return meta{}, fmt.Errorf("invalid timestamp (RFC3339 required): %w", err)
	}

	provenance := r.Header.Get("User-Agent")
	if provenance == "" {
		return meta{}, fmt.Errorf("missing header User-Agent")
	}

	return meta{
		Provider:   provider1 + "/" + provider2,
		Timestamp:  ts,
		Provenance: provenance,
	}, nil
}

// detectMediaType returns a clean media type string (without parameters).
// Falls back to content sniffing when the Content-Type header is absent.
func detectMediaType(r *http.Request, body []byte) string {
	ct := r.Header.Get("Content-Type")
	if ct == "" {
		ct = http.DetectContentType(body)
	}
	mediaType, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return ct
	}
	return mediaType
}

// isBinary returns true for formats whose bytes are not meaningful as UTF-8 text.
// Everything else (including unknown types) is treated as text/string in storage.
func isBinary(mediaType string) bool {
	if strings.HasPrefix(mediaType, "image/") ||
		strings.HasPrefix(mediaType, "audio/") ||
		strings.HasPrefix(mediaType, "video/") {
		return true
	}
	switch mediaType {
	case "application/octet-stream",
		"application/bson",
		"application/zip",
		"application/gzip", "application/x-gzip",
		"application/zstd",
		"application/x-tar",
		"application/pdf",
		"application/parquet",
		"application/protobuf", "application/x-protobuf",
		"application/msgpack", "application/x-msgpack",
		"application/cbor":
		return true
	}
	return false
}

// isCompressible returns true for text-based formats that benefit from compression.
// Binary and already-compressed formats are excluded.
func isCompressible(mediaType string) bool {
	if strings.HasPrefix(mediaType, "text/") {
		return true
	}
	switch mediaType {
	case "application/json",
		"application/xml",
		"application/javascript",
		"application/x-ndjson",
		"application/yaml",
		"application/x-yaml",
		"application/csv",
		"application/x-www-form-urlencoded":
		return true
	}
	return false
}

func compressZstd(data []byte) ([]byte, error) {
	enc, err := zstd.NewWriter(nil)
	if err != nil {
		return nil, fmt.Errorf("zstd encoder init failed: %w", err)
	}
	return enc.EncodeAll(data, make([]byte, 0, len(data)/2)), nil
}

func handleIngest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx, span := tel.TraceStart(ctx, "handle-ingest", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()
	log := logger.Get(ctx)

	// validate necessary metadata like provider, timestamp
	m, err := parseMeta(r)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Based on payload size put message into mongo or S3
	r.Body = http.MaxBytesReader(w, r.Body, cfg.MAX_SIZE)
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		http.Error(w, "failed reading body", http.StatusRequestEntityTooLarge)
		return
	}

	mediaType := detectMediaType(r, raw)

	span.SetAttributes(
		attribute.String("provider", m.Provider),
		attribute.Int("body.bytes", len(raw)),
		attribute.String("content_type", mediaType),
	)

	if len(raw) >= largeSizeThreshold {
		// For large files, compress payloads before uploading to S3.
		data := raw
		s3KeySuffix := ""
		if isCompressible(mediaType) {
			compressed, cerr := compressZstd(raw)
			if cerr != nil {
				log.Warn("zstd compression failed, storing uncompressed", "err", cerr)
			} else {
				data = compressed
				s3KeySuffix = ".zst"
			}
		}

		hash := sha256.Sum256(raw)

		// S3 path is : {provider1}/{provider2}/{timestamp}_{hash}
		// If the file is compressed, indicate in the filename suffix
		s3Key := fmt.Sprintf("%s/%s_%x%s",
			m.Provider,
			m.Timestamp.UTC().Format("20060102T150405Z"),
			hash[:8],
			s3KeySuffix,
		)

		if err := s3Client.Upload(ctx, s3Key, data); err != nil {
			log.Error("s3 upload failed", "err", err, "key", s3Key)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			http.Error(w, "s3 upload error", http.StatusInternalServerError)
			return
		}

		s3URN := fmt.Sprintf("urn:s3:%s:%s", cfg.S3_BUCKET, s3Key)
		span.SetAttributes(attribute.String("s3.urn", s3URN))

		wr, err := mongoWriteLarge(ctx, m, s3URN, mediaType)
		if err != nil {
			log.Error("mongo write failed after s3 upload", "err", err, "s3_key", s3Key)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}

		if err := publishReady(ctx, m, wr); err != nil {
			log.Error("mq publish failed", "err", err)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			http.Error(w, "failed publishing to MQ", http.StatusInternalServerError)
			return
		}

		log.Info("stored large message", "id", wr.ID, "s3_ref", s3URN)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"id":%q,"s3_ref":%q}`, wr.ID, s3URN)
	} else {
		wr, err := mongoWriteSmall(ctx, m, raw, mediaType)
		if err != nil {
			log.Error("mongo write failed", "err", err)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}

		if err := publishReady(ctx, m, wr); err != nil {
			log.Error("mq publish failed", "err", err)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			http.Error(w, "failed publishing to MQ", http.StatusInternalServerError)
			return
		}

		log.Info("stored small message", "id", wr.ID, "provider", m.Provider)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"id":%q}`, wr.ID)
	}
}
