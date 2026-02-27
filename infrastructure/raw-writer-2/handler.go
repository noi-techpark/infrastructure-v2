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
	// Provider holds the combined "part1/part2" string.
	// part1 comes from the User-Agent header, part2 from the {source} path segment.
	Provider   string
	Timestamp  time.Time
	Provenance string
}

// parseMeta extracts metadata from the request:
//   - provider  : User-Agent header + "/" + {source} path segment → "part1/part2"
//   - timestamp : {timestamp} path segment (RFC3339)
//   - provenance: ?provenance= query param
func parseMeta(r *http.Request) (meta, error) {
	providerPart := r.Header.Get("User-Agent")
	if providerPart == "" {
		return meta{}, fmt.Errorf("missing User-Agent header (used as provider)")
	}

	sourcePart := r.PathValue("source")
	if sourcePart == "" {
		return meta{}, fmt.Errorf("missing path segment: source")
	}

	tsStr := r.PathValue("timestamp")
	if tsStr == "" {
		return meta{}, fmt.Errorf("missing path segment: timestamp")
	}
	ts, err := time.Parse(time.RFC3339, tsStr)
	if err != nil {
		return meta{}, fmt.Errorf("invalid timestamp (RFC3339 required): %w", err)
	}

	provenance := r.URL.Query().Get("provenance")
	if provenance == "" {
		return meta{}, fmt.Errorf("missing query param: provenance")
	}

	return meta{
		Provider:   providerPart + "/" + sourcePart,
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

// POST /{source}/{timestamp} — single ingest endpoint.
// Routing between MongoDB-direct (small) and S3-backed (large) is determined
// by the original payload size against a 5 MB threshold.
// provider from User-Agent header, provenance from ?provenance= query param.
// Text-based payloads (json, xml, csv, …) are zstd-compressed before storage.
func handleIngest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx, span := tel.TraceStart(ctx, "handle-ingest", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()
	log := logger.Get(ctx)

	m, err := parseMeta(r)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, cfg.MAX_SIZE)
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		http.Error(w, "failed reading body", http.StatusRequestEntityTooLarge)
		return
	}

	mediaType := detectMediaType(r, raw)
	hash := sha256.Sum256(raw) // always hash the original

	span.SetAttributes(
		attribute.String("provider", m.Provider),
		attribute.Int("body.bytes", len(raw)),
		attribute.String("content_type", mediaType),
	)

	if len(raw) >= largeSizeThreshold {
		// For large files, compress text-based payloads before uploading to S3.
		data := raw
		s3KeySuffix := ""
		nid := "s3"
		if isCompressible(mediaType) {
			compressed, cerr := compressZstd(raw)
			if cerr != nil {
				log.Warn("zstd compression failed, storing uncompressed", "err", cerr)
			} else {
				data = compressed
				s3KeySuffix = ".zst"
				nid = "s3+zstd"
			}
		}

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

		// Compression algorithm encoded in URN NID: urn:s3+zstd:… or urn:s3:…
		s3URN := fmt.Sprintf("urn:%s:%s:%s", nid, cfg.S3_BUCKET, s3Key)
		span.SetAttributes(attribute.String("s3.urn", s3URN))

		wr, err := mongoWriteLarge(ctx, m, s3URN, mediaType)
		if err != nil {
			log.Error("mongo write failed after s3 upload", "err", err, "s3_key", s3Key)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			// S3 object is already written — caller must retry or reconcile.
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}

		if err := publishReady(ctx, m, wr); err != nil {
			log.Error("mq publish failed", "err", err)
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
		}

		log.Info("stored small message", "id", wr.ID, "provider", m.Provider)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"id":%q}`, wr.ID)
	}
}
