// SPDX-FileCopyrightText: 2024 NOI Techpark <digital@noi.bz.it>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"context"
	"log"
	"log/slog"

	"github.com/kelseyhightower/envconfig"
	"github.com/noi-techpark/opendatahub-go-sdk/ingest/ms"
	"github.com/noi-techpark/opendatahub-go-sdk/tel"
	"golang.org/x/sync/errgroup"
	"opendatahub.com/infrav2/raw-data-bridge/rdt"
)

var cfg struct {
	MONGO_URI string
	DB_PREFIX string
}

func initConfig() {
	err := envconfig.Process("APP", &cfg)
	if err != nil {
		log.Panic("Unable to initialize config", err)
	}
}

func main() {
	ms.InitWithEnv(context.Background(), "APP", &cfg)
	defer tel.FlushOnPanic()
	rdt.InitRawDataConnection(cfg.MONGO_URI)

	s := NewServer(context.Background())

	g, ctx := errgroup.WithContext(context.Background())
	g.Go(func() error { return s.Run(ctx, ":2000") })

	if err := g.Wait(); err != nil {
		slog.Error("failed to run")
	}
}
