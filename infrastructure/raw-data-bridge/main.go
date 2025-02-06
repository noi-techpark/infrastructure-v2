package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/kelseyhightower/envconfig"
	"golang.org/x/sync/errgroup"
	"opendatahub.com/infrav2/raw-data-bridge/rdt"
)

var cfg struct {
	LogLevel  string `default:"INFO"`
	MONGO_URI string
	DB_PREFIX string
}

func initConfig() {
	err := envconfig.Process("APP", &cfg)
	if err != nil {
		log.Panic("Unable to initialize config", err)
	}
}

func initLog() {
	level := &slog.LevelVar{}
	level.UnmarshalText([]byte(cfg.LogLevel))
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})))
}

func main() {
	initConfig()
	initLog()
	rdt.InitRawDataConnection(cfg.MONGO_URI)

	s := NewServer(context.Background())

	g, ctx := errgroup.WithContext(context.Background())
	g.Go(func() error { return s.Run(ctx, ":2000") })

	if err := g.Wait(); err != nil {
		slog.Error("failed to run")
	}
}
