// SPDX-FileCopyrightText: 2024 NOI Techpark <digital@noi.bz.it>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"context"
	"log"
	"log/slog"

	"github.com/noi-techpark/opendatahub-go-sdk/ingest/ms"
	"github.com/noi-techpark/opendatahub-go-sdk/tel"
	"golang.org/x/sync/errgroup"
	"opendatahub.com/infrav2/raw-data-bridge/rdt"
	s3client "opendatahub.com/infrav2/raw-data-bridge/s3"
)

var cfg struct {
	MONGO_URI            string
	DB_PREFIX            string
	S3_ENDPOINT          string
	S3_ACCESS_KEY_ID     string
	S3_SECRET_ACCESS_KEY string
	S3_REGION            string
}

func main() {
	ms.InitWithEnv(context.Background(), "APP", &cfg)
	defer tel.FlushOnPanic()
	rdt.InitRawDataConnection(cfg.MONGO_URI)

	if err := InitRetrievers(s3client.Config{
		Endpoint:        cfg.S3_ENDPOINT,
		AccessKeyID:     cfg.S3_ACCESS_KEY_ID,
		SecretAccessKey: cfg.S3_SECRET_ACCESS_KEY,
		Region:          cfg.S3_REGION,
	}); err != nil {
		log.Panic("Unable to initialize retrievers", err)
	}

	s := NewServer(context.Background())

	g, ctx := errgroup.WithContext(context.Background())
	g.Go(func() error { return s.Run(ctx, ":2000") })

	if err := g.Wait(); err != nil {
		slog.Error("failed to run")
	}
}
