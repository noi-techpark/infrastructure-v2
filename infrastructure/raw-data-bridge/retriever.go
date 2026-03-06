// SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>
//
// SPDX-License-Identifier: CC0-1.0

package main

import (
	"context"
	"fmt"
	"strings"

	s3client "opendatahub.com/infrav2/raw-data-bridge/s3"
)

// RawDataRetriever is a generic interface for fetching raw data referenced by a URN.
// Different URN schemes (s3, ...) map to different implementations.
type RawDataRetriever interface {
	Retrieve(ctx context.Context, urn string) ([]byte, error)
}

var retrievers map[string]RawDataRetriever

// InitRetrievers initialises all retrievers from config.
func InitRetrievers(s3cfg s3client.Config) error {
	s3c, err := s3client.NewClient(s3cfg)
	if err != nil {
		return fmt.Errorf("s3 client init: %w", err)
	}
	retrievers = map[string]RawDataRetriever{
		"s3": s3c,
	}
	return nil
}

// GetRetriever returns the retriever for the scheme embedded in a URN string.
// URN format: urn:{scheme}:...
func GetRetriever(urn string) (RawDataRetriever, bool) {
	parts := strings.SplitN(urn, ":", 3)
	if len(parts) < 2 || parts[0] != "urn" {
		return nil, false
	}
	r, ok := retrievers[parts[1]]
	return r, ok
}
