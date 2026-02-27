// SPDX-FileCopyrightText: 2024 NOI Techpark <digital@noi.bz.it>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/kelseyhightower/envconfig"
	"github.com/noi-techpark/opendatahub-go-sdk/qmill"
	"github.com/noi-techpark/opendatahub-go-sdk/tel"
	"github.com/noi-techpark/opendatahub-go-sdk/tel/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var cfg struct {
	HTTP_PORT   string `default:"8080"`
	MONGO_URI   string
	DB_PREFIX   string
	S3_BUCKET   string
	S3_ENDPOINT string // optional, for MinIO or custom S3-compatible storage
	S3_REGION   string `default:"eu-west-1"`
	MAX_SIZE    int64  `default:"104857600"` // 100 MB

	MQ_URI          string
	MQ_CLIENT_NAME  string
	MQ_READY_EXCHANGE   string
	MQ_READY_QUEUE      string
	MQ_READY_DL_QUEUE   string
}

var (
	mongoClient   *mongo.Client
	s3Client      *S3Client
	readyPublisher *qmill.QMill
)

func main() {
	if err := envconfig.Process("", &cfg); err != nil {
		log.Panic("config error: ", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	telProvider := tel.NewTelemetryFromEnv(ctx)
	defer tel.Shutdown(ctx)
	defer tel.FlushOnPanic()

	_ = telProvider

	mongoClient = initMongo(ctx)
	defer mongoClient.Disconnect(context.Background())

	s3Client = newS3Client(ctx)

	var err error
	readyPublisher, err = qmill.NewPublisherQmill(ctx, cfg.MQ_URI, cfg.MQ_CLIENT_NAME,
		qmill.WithExchange(cfg.MQ_READY_EXCHANGE, "direct", true),
		qmill.WithQueue(cfg.MQ_READY_QUEUE, true),
		qmill.WithBind(cfg.MQ_READY_EXCHANGE, ""),
		qmill.WithDeadLetter(cfg.MQ_READY_DL_QUEUE, "fanout"),
		qmill.WithNoRequeueOnNack(true),
		qmill.WithLogger(watermill.NewSlogLogger(slog.Default())),
	)
	if err != nil {
		log.Panic("failed to initialize MQ publisher: ", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /live", handleLive)
	mux.HandleFunc("GET /ready", handleReady)
	mux.Handle("POST /{source}/{timestamp}", otelhttp.NewHandler(http.HandlerFunc(handleIngest), "handle-ingest"))

	srv := &http.Server{
		Addr:    ":" + cfg.HTTP_PORT,
		Handler: mux,
	}

	go func() {
		slog.Info("raw-writer-2 listening", "port", cfg.HTTP_PORT)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error: ", err)
		}
	}()

	<-ctx.Done()
	logger.Get(ctx).Info("shutting down")
	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
}

func initMongo(ctx context.Context) *mongo.Client {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MONGO_URI))
	if err != nil {
		log.Panic("mongo connect failed: ", err)
	}
	if err := client.Database("admin").RunCommand(ctx, bson.D{{Key: "ping", Value: 1}}).Err(); err != nil {
		log.Panic("mongo ping failed: ", err)
	}
	slog.Info("connected to mongodb")
	return client
}
