package main

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	sloggin "github.com/samber/slog-gin"
	"opendatahub.com/infrav2/raw-data-bridge/rdt"

	"github.com/noi-techpark/opendatahub-go-sdk/ingest/urn"
	"github.com/noi-techpark/opendatahub-go-sdk/tel"
	httptel "github.com/noi-techpark/opendatahub-go-sdk/tel/http"
	"github.com/noi-techpark/opendatahub-go-sdk/tel/logger"
)

type Server struct {
	e *gin.Engine
}

func NewServer(ctx context.Context) *Server {
	gin.SetMode(gin.ReleaseMode)
	e := gin.New()
	e.Use(
		sloggin.NewWithFilters(
			slog.Default(),
			sloggin.IgnorePath("/health", "/favicon.ico")),
		gin.Recovery(),
	)

	e.Use(httptel.TracingMiddleware([]string{"/health", "/favicon.ico"}))

	e.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, ResponseType, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	server := &Server{
		e: e,
	}

	if err := server.buildRouter(ctx); err != nil {
		slog.Error("error building the server's routes")
	}

	return server

}

func (s *Server) buildRouter(ctx context.Context) error {
	s.e.GET("health", s.HealthCheck)
	s.e.GET("urns/:urn", s.GetDocument)
	// s.e.GET("urns/:urn/latest", s.GetDocument)
	return nil
}

func (s *Server) Run(ctx context.Context, port string) error {
	return s.e.Run(port)
}

func (s *Server) HealthCheck(c *gin.Context) {
	c.Status(http.StatusOK)
}

func (s *Server) GetDocument(c *gin.Context) {
	ctx := c.Request.Context()
	log := logger.Get(c.Request.Context())

	requested_urn := c.Param("urn")
	u, ok := urn.Parse(requested_urn)
	if !ok {
		log.Error("requested payload for malformed urn", "urn", requested_urn)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid urn",
		})
		return
	}

	doc, err := rdt.GetDocument(ctx, u)
	if err != nil {
		if err == rdt.ErrDocumentNotFound {
			c.Status(http.StatusNotFound)
			return
		} else if err == rdt.ErrBadURN {
			log.Error("requested payload for malformed urn", "urn", requested_urn)
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid urn",
			})
			return
		}
		tel.OnError(ctx, "error getting raw data document", err)
		log.Error("error getting raw data document", "urn", requested_urn, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get document",
		})
		return
	}
	c.JSON(http.StatusOK, doc)
}
