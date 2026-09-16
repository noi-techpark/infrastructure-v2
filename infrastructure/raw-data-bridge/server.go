// SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>
//
// SPDX-License-Identifier: CC0-1.0

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	sloggin "github.com/samber/slog-gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/sync/errgroup"
	"opendatahub.com/infrav2/raw-data-bridge/rdt"

	"github.com/noi-techpark/opendatahub-go-sdk/ingest/urn"
	"github.com/noi-techpark/opendatahub-go-sdk/tel"
	httptel "github.com/noi-techpark/opendatahub-go-sdk/tel/http"
	"github.com/noi-techpark/opendatahub-go-sdk/tel/logger"
)

const (
	// maxLimit caps what a caller may ask for. The *default* deliberately does
	// not live here: rdt owns it, and a second copy on this side is a number
	// that can drift from the one actually applied.
	maxLimit = 10000

	// maxInlinePerPage bounds how many externally stored payloads one page will
	// fetch.
	//
	// This is a count, and a count is not a size. raw-writer-2 records no length
	// for a spilled payload, so neither this nor rdt's byte budget can see how
	// large these actually are — 32 objects of 500 MB pass this check. It is a
	// latency and blast-radius guard, not a memory bound, and it stops being a
	// bodge the day the writer stores a size. See README, "Known limits".
	maxInlinePerPage = 32

	// inlineConcurrency bounds the parallel fetches for one page. Sequentially,
	// a full page of references was maxInlinePerPage round trips end to end.
	inlineConcurrency = 8
)

// isText returns true for content types that are meaningful as UTF-8 text.
// Everything else is treated as binary and base64-encoded in the response.
func isText(mediaType string) bool {
	if strings.HasPrefix(mediaType, "text/") {
		return true
	}
	switch mediaType {
	case "application/json",
		"application/xml",
		"application/yaml",
		"application/csv",
		"application/x-www-form-urlencoded":
		return true
	}
	return false
}

type Server struct {
	e *gin.Engine
}

func NewServer() *Server {
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

	server.buildRouter()

	return server

}

func (s *Server) buildRouter() {
	s.e.GET("health", s.HealthCheck)

	// A URN already encodes db, collection and id, so it addresses a document
	// globally — a permalink rather than a query within a collection. That is
	// why it sits outside the hierarchy below.
	s.e.GET("urns/:urn", s.GetDocument)

	// Collection views, named for what they return rather than for a parameter
	// that changes the resource underneath the caller.
	s.e.GET(":db/:collection/documents", s.GetDocuments)
	s.e.GET(":db/:collection/compacted/:field", s.GetCompacted)
}

// inlineRawData normalises the storage-level representation of a document's
// payload: BSON binary becomes raw bytes, and an external reference is fetched
// and inlined. Shared by the single-document and latest-by-key handlers.
func inlineRawData(ctx context.Context, doc *rdt.Document) error {
	log := logger.Get(ctx)

	// Inline rawdata stored as BSON binary: expose the raw bytes; json.Marshal base64-encodes []byte automatically.
	if bin, ok := (*doc)["rawdata"].(primitive.Binary); ok {
		(*doc)["rawdata"] = bin.Data
	}

	// If the document references external raw data, fetch and inline it.
	rawRef, ok := (*doc)["raw_ref"].(string)
	if !ok || rawRef == "" {
		return nil
	}
	contentType, _ := (*doc)["content_type"].(string)
	retriever, ok := GetRetriever(rawRef)
	if !ok {
		log.Error("no retriever for raw_ref scheme", "raw_ref", rawRef)
		return errors.New("no retriever for raw data reference")
	}
	data, err := retriever.Retrieve(ctx, rawRef)
	if err != nil {
		log.Error("failed to retrieve raw_ref data", "raw_ref", rawRef, "err", err)
		return err
	}
	if isText(contentType) {
		(*doc)["rawdata"] = string(data)
	} else {
		(*doc)["rawdata"] = data
	}
	return nil
}

// GetDocuments serves a collection's documents in write order, newest first.
//
//	the last one written: GET /{db}/{coll}/documents?limit=1
//	the recent tail:      GET /{db}/{coll}/documents?limit=20
//	the next page:        GET /{db}/{coll}/documents?cursor=<next from the previous page>
//
// This is the log as written, nothing collapsed. It does not describe current
// state — see GetCompacted for that.
func (s *Server) GetDocuments(c *gin.Context) {
	ctx := c.Request.Context()

	q := rdt.DocumentsQuery{
		DB:         c.Param("db"),
		Collection: c.Param("collection"),
		Cursor:     c.Query("cursor"),
	}
	var ok bool
	if q.Limit, ok = parseLimit(c); !ok {
		return
	}

	page, err := rdt.GetDocuments(ctx, q)
	if err != nil {
		respondQueryError(c, ctx, err, q.DB, q.Collection, "")
		return
	}
	respondPage(c, ctx, page, nil)
}

// GetCompacted serves the newest document per distinct value of a field inside
// the document's `meta` sub-document — a compacted view of the collection.
//
//	everything:  GET /{db}/{coll}/compacted/key
//	next page:   GET /{db}/{coll}/compacted/key?cursor=<next from the previous page>
//
// A consumer walks every page and has the current value of every key. It
// reconciles by walking again: the walk is the whole state, so a re-read cannot
// miss anything, whenever and however a document was written.
func (s *Server) GetCompacted(c *gin.Context) {
	ctx := c.Request.Context()

	q := rdt.CompactedQuery{
		DB:         c.Param("db"),
		Collection: c.Param("collection"),
		Field:      c.Param("field"),
		Cursor:     c.Query("cursor"),
	}
	var ok bool
	if q.Limit, ok = parseLimit(c); !ok {
		return
	}

	page, err := rdt.GetCompacted(ctx, q)
	if err != nil {
		respondQueryError(c, ctx, err, q.DB, q.Collection, q.Field)
		return
	}
	respondPage(c, ctx, page, gin.H{"field": q.Field})
}

func parseLimit(c *gin.Context) (int, bool) {
	raw := c.Query("limit")
	if raw == "" {
		// 0 means "no opinion" — rdt applies its own default, so there is only
		// ever one number in play.
		return 0, true
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 || n > maxLimit {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("limit must be between 1 and %d", maxLimit),
		})
		return 0, false
	}
	return n, true
}

func respondQueryError(c *gin.Context, ctx context.Context, err error, db, coll, field string) {
	switch {
	case errors.Is(err, rdt.ErrBadIdentifier):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid db, collection or field"})
	case errors.Is(err, rdt.ErrBadCursor):
		c.JSON(http.StatusBadRequest, gin.H{"error": "cursor is not one this endpoint issued"})
	case errors.Is(err, rdt.ErrCollectionNotFound):
		// Distinct from an empty collection on purpose: a consumer that reads
		// an empty 200 as "no data yet" cannot see a typo, and a reference
		// table built from it is silently wrong.
		c.JSON(http.StatusNotFound, gin.H{"error": "no such collection"})
	default:
		tel.OnError(ctx, "error listing raw data documents", err)
		logger.Get(ctx).Error("error listing raw data documents",
			"db", db, "collection", coll, "field", field, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list documents"})
	}
}

// respondPage inlines external payloads and writes the shared envelope. The
// envelope is identical across both collection views on purpose: what differs
// between them is which documents they select, not how a page is described.
func respondPage(c *gin.Context, ctx context.Context, page rdt.Page, extra gin.H) {
	// A raw_ref document is small in Mongo and arbitrarily large once fetched,
	// and the writer records no size, so rdt's budget cannot see this path. The
	// page is refused rather than inlined, because returning it half-resolved
	// would hand the consumer a document whose rawdata is silently missing.
	//
	// Counting is a separate pass from fetching on purpose. Doing both at once
	// meant a page one reference over the cap paid for maxInlinePerPage blocking
	// round trips before refusing, and then discarded every byte it had fetched.
	refs := 0
	for i := range page.Docs {
		if ref, ok := page.Docs[i]["raw_ref"].(string); ok && ref != "" {
			refs++
		}
	}
	if refs > maxInlinePerPage {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error": "too many externally stored payloads to inline in one page; lower limit",
		})
		return
	}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(inlineConcurrency)
	for i := range page.Docs {
		g.Go(func() error { return inlineRawData(gctx, &page.Docs[i]) })
	}
	if err := g.Wait(); err != nil {
		// The detail names a bucket, an object key and a request id. It is
		// logged and traced; the caller is told what failed, not where.
		tel.OnError(ctx, "failed to inline raw data", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve raw data"})
		return
	}
	// An empty page serialises as [], never null: the documented shape is a
	// list, and a consumer iterating it should not have to special-case one.
	docs := page.Docs
	if docs == nil {
		docs = []rdt.Document{}
	}
	body := gin.H{
		"count": len(docs),
		"next":  page.Next,
		"data":  docs,
	}
	for k, v := range extra {
		body[k] = v
	}
	c.JSON(http.StatusOK, body)
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
		switch err {
		case rdt.ErrDocumentNotFound:
			c.Status(http.StatusNotFound)
			return
		case rdt.ErrBadURN:
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

	if err := inlineRawData(ctx, doc); err != nil {
		tel.OnError(ctx, "failed to inline raw data", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve raw data"})
		return
	}

	c.JSON(http.StatusOK, doc)
}
