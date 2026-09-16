// SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>
//
// SPDX-License-Identifier: CC0-1.0

package rdt

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/noi-techpark/opendatahub-go-sdk/ingest/urn"
	"github.com/noi-techpark/opendatahub-go-sdk/tel"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var (
	ErrDocumentNotFound = errors.New("document not found")
	ErrBadURN           = errors.New("bad urn format")
	ErrBadIdentifier    = errors.New("bad identifier")
	ErrBadCursor        = errors.New("bad cursor")

	// ErrCollectionNotFound separates "nothing has been written here" from
	// "this is not a place anything writes to". Both used to be an empty 200,
	// and a consumer cannot tell a typo'd collection from a young one.
	ErrCollectionNotFound = errors.New("collection not found")
)

// nameRe constrains one component of a db name, collection name or field name.
// Excluding '$' keeps a caller-supplied string from being read as an operator.
var nameRe = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_\-]*$`)

const maxIdentifierLen = 128

// DefaultLimit is the page size a query with no limit of its own gets. It lives
// here, beside the queries it bounds, and the HTTP layer passes 0 rather than
// keeping a second copy that could drift from it.
const defaultLimit = 1000

// documentsCursorField names the ordering a documents cursor belongs to, so a
// cursor is refused by the same check that refuses a compacted one issued for
// the wrong field. The '$' makes the collision impossible rather than unlikely:
// nameRe excludes it, so no meta field can ever be called this.
const documentsCursorField = "$documents"

// ValidCollectionName reports whether s is safe as a collection name.
// Collections may be dotted, because a URN's namespace tokens are joined with
// '.' — requiring every segment to be non-empty rejects leading, trailing and
// doubled dots.
func ValidCollectionName(s string) bool {
	if s == "" || len(s) > maxIdentifierLen {
		return false
	}
	for _, seg := range strings.Split(s, ".") {
		if !nameRe.MatchString(seg) {
			return false
		}
	}
	return true
}

// ValidDatabaseName reports whether s is safe as a database name.
//
// Stricter than ValidCollectionName by exactly one character: MongoDB forbids
// '.' in a database name. Sharing one validator between the two let a dotted db
// through to the driver, which failed the operation with InvalidNamespace — a
// 500 for what is plainly a malformed request.
func ValidDatabaseName(s string) bool {
	return s != "" && len(s) <= maxIdentifierLen && nameRe.MatchString(s)
}

// MetaField is the sub-document the raw writer harvests publisher-set headers
// into. It is the only part of a stored document a collector controls that is
// not the payload, so it is the only place a group field can live.
const MetaField = "meta"

// ValidFieldName reports whether s names a field inside the meta sub-document.
//
// No dots and no traversal: s is a single field one level under `meta`, which
// is exactly the shape the writer produces (prefix stripped, remainder
// lowercased, stored flat).
func ValidFieldName(s string) bool {
	return s != "" && len(s) <= maxIdentifierLen && nameRe.MatchString(s)
}

// metaPath returns the dotted path the database groups on.
func metaPath(field string) string { return MetaField + "." + field }

var (
	mongoClient *mongo.Client

	// dbPrefix is prepended to every database name at the point the driver is
	// asked for one, and nowhere else.
	//
	// raw-writer-2 applies the same prefix when it decides where to write, so a
	// deployment that sets one and a bridge that ignored it would read from a
	// database nothing writes to — every URN 404s and every reference table
	// bootstraps empty, silently. Callers keep using logical names throughout;
	// the prefix exists only at the boundary.
	dbPrefix string
)

// databaseName maps a logical database name to the physical one.
func databaseName(db string) string { return dbPrefix + db }

// NOTE: using a map does not preserve field order.
type Document map[string]any

// MarshalJSON hides the fields that exist only for raw-data storage.
//
// It copies rather than deleting in place. Document is a map, so stripping the
// receiver would mutate the caller's document — a marshal that destroys its
// input, forcing every caller to read what it needs *before* serializing and
// leaving a trap for the one that forgets.
func (r Document) MarshalJSON() ([]byte, error) {
	out := make(map[string]any, len(r))
	for k, v := range r {
		switch k {
		case "_id", "bsontimestamp":
			continue
		}
		out[k] = v
	}
	// The ObjectId comes back as `id`. A document the caller just listed has to
	// be addressable or it cannot build the `/urns/` permalink for it, and the
	// only id a page used to expose was the last one, disguised as `next`. The
	// storage spelling stays hidden.
	if id, ok := r.ID(); ok {
		out["id"] = id.Hex()
	}
	return json.Marshal(out)
}

// ID returns the document's ObjectId, if it has one.
func (r Document) ID() (primitive.ObjectID, bool) {
	id, ok := r["_id"].(primitive.ObjectID)
	return id, ok
}

// InitRawDataConnection connects to the lake. prefix is prepended to every
// database name and must match what the writer is configured with; empty is the
// normal case.
//
// It returns an error rather than panicking. Failing to reach Mongo is fatal to
// the process and main treats it that way, but a panic here also took down the
// whole test binary — one unreachable database and the suite reported a stack
// trace instead of a failed test.
func InitRawDataConnection(uri, prefix string) error {
	if prefix != "" && !nameRe.MatchString(prefix) {
		return fmt.Errorf("db prefix %q may contain only letters, digits, '_' and '-'", prefix)
	}
	dbPrefix = prefix

	mclient, err := mongoConnect(uri)
	if err != nil {
		return fmt.Errorf("connect to raw data lake: %w", err)
	}
	mongoClient = mclient
	if prefix != "" {
		slog.Info("database names are prefixed", "prefix", prefix)
	}
	return nil
}

func constructTableTarget(urn *urn.URN) (string, string, error) {
	provider_tokens := urn.GetNSSWithoutID()
	if len(provider_tokens) < 2 {
		return "", "", fmt.Errorf("urn format invalid: %s", urn.String())
	}
	return provider_tokens[0], strings.Join(provider_tokens[1:], "."), nil
}

func mongoConnect(uri string) (*mongo.Client, error) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	// Send a ping to confirm a successful connection
	var result bson.M
	if err := client.Database("admin").RunCommand(context.TODO(), bson.D{{Key: "ping", Value: 1}}).Decode(&result); err != nil {
		return nil, err
	}
	return client, nil
}

// Page is a slice of documents together with what a consumer needs to continue.
//
// Next is opaque: it is produced beside the ordering that defines it and must
// be handed back unchanged. It exists on this type, rather than being derived
// by whoever called, because the caller cannot know what the ordering was —
// which is exactly how a cursor once ended up being read off the wrong field
// and paging stopped after one page, silently.
type Page struct {
	Docs []Document
	Next string
}

// maxPageBytes caps what one response may materialise.
//
// A page was bounded by document count alone, and the writer stores payloads
// inline up to 5 MB, so a default limit against a collection of large documents
// projects to gigabytes inside a pod capped at a few hundred. Measured on a live
// collection: 2.24 MB per document, which at the default limit of 1000 is 2.1 GB.
// A var, not a const, only so a test can lower it: no single stored document can
// exceed it, because MongoDB caps a document at 16 MB and the writer offloads at
// 5 MB, so the always-return-one guard below is otherwise unreachable.
var maxPageBytes = 32 << 20

// finish trims the over-read row, records the high-water mark, and asks cursorOf
// for the continuation token. truncated means the byte budget ended the read, so
// a short page still has a continuation.
func finish(docs []Document, limit int, truncated bool, cursorOf func(Document) string) Page {
	p := Page{}
	more := truncated || len(docs) > limit
	if len(docs) > limit {
		docs = docs[:limit]
	}
	if more && len(docs) > 0 {
		p.Next = cursorOf(docs[len(docs)-1])
	}
	p.Docs = docs
	return p
}

// pageCursor is the opaque continuation token.
//
// Value keeps its BSON type, and Field is checked on the way back in so a cursor
// issued for one ordering is refused rather than compared against another it was
// never part of.
type pageCursor struct {
	Field string `bson:"f"`
	Value any    `bson:"v"`
}

func encodeCursor(c pageCursor) string {
	b, err := bson.Marshal(c)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func decodeCursor(field, s string) (pageCursor, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return pageCursor{}, ErrBadCursor
	}
	var c pageCursor
	if err := bson.Unmarshal(b, &c); err != nil {
		return pageCursor{}, ErrBadCursor
	}
	if c.Field != field {
		return pageCursor{}, ErrBadCursor
	}
	return c, nil
}

// metaOf reads the meta sub-document.
//
// The driver decodes it as the enclosing named type today, but only by way of a
// deprecated decoder field, so the other shapes it could legitimately produce
// are handled rather than assumed away.
func metaOf(d Document) (map[string]any, bool) {
	switch m := d[MetaField].(type) {
	case Document:
		return m, true
	case primitive.M:
		return m, true
	case map[string]any:
		return m, true
	}
	return nil, false
}

// collectionExists distinguishes an empty collection from an absent one.
func collectionExists(ctx context.Context, db, coll string) (bool, error) {
	names, err := mongoClient.Database(databaseName(db)).
		ListCollectionNames(ctx, bson.M{"name": coll})
	if err != nil {
		return false, err
	}
	return len(names) > 0, nil
}

// CompactedQuery selects the newest document per distinct value of a field.
type CompactedQuery struct {
	DB         string
	Collection string
	// Field names a field inside the document's `meta` sub-document — not a
	// path, and not a field at the document root. That is the one place a
	// publisher can put something groupable: a payload posted as
	// application/json is stored as a *string*, so nothing inside it can be
	// grouped on. A collector sets it by sending an X-OpenDataHub-<Field>
	// header.
	Field string
	// Cursor continues a previous page. Opaque; pass back Page.Next unchanged.
	Cursor string
	Limit  int
}

// GetCompacted returns, for every distinct value of q.Field, the document
// carrying that value with the newest event time — the equivalent of reading a
// compacted log. Grouping happens in the database on purpose: a client cannot
// obtain the current value of every key by tailing insertion order without
// replaying the entire history of the collection.
//
// Results are ordered by field value so paging is stable, and a walk of every
// page is the whole current state of the collection.
//
// Cost: the match, sort and group all run before $limit, so every page repeats
// them over the whole collection. A walk costs pages × that, and no index
// removes it — $first over $$ROOT fetches every document however it is reached.
// See README, "On indexing".
func GetCompacted(ctx context.Context, q CompactedQuery) (Page, error) {
	if !ValidDatabaseName(q.DB) || !ValidCollectionName(q.Collection) || !ValidFieldName(q.Field) {
		return Page{}, ErrBadIdentifier
	}
	if q.Limit <= 0 {
		q.Limit = defaultLimit
	}

	fieldPath := metaPath(q.Field)

	fieldFilter := bson.M{"$exists": true, "$ne": nil}
	if q.Cursor != "" {
		c, err := decodeCursor(q.Field, q.Cursor)
		if err != nil {
			return Page{}, err
		}
		// Every value under `meta` is a string: raw-writer-2 harvests these from
		// request headers, so its extras are a map[string]string and nothing
		// else can reach this field. $gt therefore compares within one BSON type
		// bracket, which is the only case it handles.
		fieldFilter["$gt"] = c.Value
	}
	match := bson.M{fieldPath: fieldFilter}

	ctx, span := startSpan(ctx, "compacted", q.DB, q.Collection, fieldPath)
	defer span.End()

	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: match}},
		// bsontimestamp descending inside each group gives the newest document
		// first, so $first below picks the current value. It must be the event
		// time and not _id: a backfill is written later but dated earlier, so
		// ordering on _id would serve that stale revision as the current one.
		// The publisher's timestamp is the authority on which revision is
		// current; _id remains only as a deterministic tiebreak when two writes
		// share a timestamp.
		bson.D{{Key: "$sort", Value: bson.D{
			{Key: fieldPath, Value: 1},
			{Key: "bsontimestamp", Value: -1},
			{Key: "_id", Value: -1},
		}}},
		bson.D{{Key: "$group", Value: bson.M{
			"_id": "$" + fieldPath,
			"doc": bson.M{"$first": "$$ROOT"},
		}}},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},
		// one extra row tells us whether a further page exists
		bson.D{{Key: "$limit", Value: int64(q.Limit + 1)}},
		bson.D{{Key: "$replaceWith", Value: "$doc"}},
	}

	cur, err := mongoClient.Database(databaseName(q.DB)).Collection(q.Collection).
		Aggregate(ctx, pipeline, options.Aggregate().SetAllowDiskUse(true))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return Page{}, err
	}
	docs, truncated, err := drain(ctx, cur, span)
	if err != nil {
		return Page{}, err
	}

	if len(docs) == 0 {
		if err := absentCollection(ctx, q.DB, q.Collection, q.Cursor); err != nil {
			return Page{}, err
		}
	}

	// The cursor is the grouped value, read from where the grouping actually
	// happened rather than guessed at by the caller.
	return finish(docs, q.Limit, truncated, func(d Document) string {
		meta, ok := metaOf(d)
		if !ok {
			return ""
		}
		v, ok := meta[q.Field]
		if !ok {
			return ""
		}
		return encodeCursor(pageCursor{Field: q.Field, Value: v})
	}), nil
}

// DocumentsQuery selects documents in write order, newest first.
type DocumentsQuery struct {
	DB         string
	Collection string
	// Cursor continues a previous page. Opaque; pass back Page.Next unchanged.
	Cursor string
	Limit  int
}

// GetDocuments returns a collection's documents in write order, newest first.
//
// This is the log as written, with nothing collapsed: every revision of every
// key, in the order it arrived. Limit 1 is the last document written. It is not
// a way to learn current state — a key written once years ago still has a
// current value and would sit arbitrarily far down this list, which is what
// GetCompacted exists for.
//
// "Newest" here means most recently *written*. A backfill is written now and
// dated whenever the publisher chose, so it leads this list while carrying an
// old timestamp.
func GetDocuments(ctx context.Context, q DocumentsQuery) (Page, error) {
	if !ValidDatabaseName(q.DB) || !ValidCollectionName(q.Collection) {
		return Page{}, ErrBadIdentifier
	}
	if q.Limit <= 0 {
		q.Limit = defaultLimit
	}

	filter := bson.M{}
	if q.Cursor != "" {
		// Ordering is _id descending, so continuing means "older than this".
		c, err := decodeCursor(documentsCursorField, q.Cursor)
		if err != nil {
			return Page{}, err
		}
		id, ok := c.Value.(primitive.ObjectID)
		if !ok {
			return Page{}, ErrBadCursor
		}
		filter["_id"] = bson.M{"$lt": id}
	}

	ctx, span := startSpan(ctx, "documents", q.DB, q.Collection, "")
	defer span.End()

	cur, err := mongoClient.Database(databaseName(q.DB)).Collection(q.Collection).Find(ctx, filter,
		options.Find().
			SetSort(bson.D{{Key: "_id", Value: -1}}).
			SetLimit(int64(q.Limit)+1))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return Page{}, err
	}
	docs, truncated, err := drain(ctx, cur, span)
	if err != nil {
		return Page{}, err
	}

	if len(docs) == 0 {
		if err := absentCollection(ctx, q.DB, q.Collection, q.Cursor); err != nil {
			return Page{}, err
		}
	}

	return finish(docs, q.Limit, truncated, func(d Document) string {
		if id, ok := d.ID(); ok {
			return encodeCursor(pageCursor{Field: documentsCursorField, Value: id})
		}
		return ""
	}), nil
}

// absentCollection reports ErrCollectionNotFound when an empty result came from
// a query that should have seen everything.
//
// Only a first page qualifies. A continuation page legitimately returns nothing,
// and paying a round trip to learn that would be a cost for no information.
func absentCollection(ctx context.Context, db, coll, cursor string) error {
	if cursor != "" {
		return nil
	}
	exists, err := collectionExists(ctx, db, coll)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%w: %s.%s", ErrCollectionNotFound, db, coll)
	}
	return nil
}

func startSpan(ctx context.Context, op, db, coll, field string) (context.Context, trace.Span) {
	tracer := otel.Tracer(tel.GetServiceName())
	ctx, span := tracer.Start(ctx, op, trace.WithSpanKind(trace.SpanKindClient))
	attrs := []attribute.KeyValue{
		attribute.String("db.operation", op),
		attribute.String("db.mongodb.db", db),
		attribute.String("db.mongodb.collection", coll),
	}
	if field != "" {
		attrs = append(attrs, attribute.String("rdb.field", field))
	}
	span.SetAttributes(attrs...)
	return ctx, span
}

// drain reads a cursor into memory, stopping once the page has grown past
// maxPageBytes. The second return reports that stop, so the caller offers a
// continuation rather than serving a short page as though it were the end.
func drain(ctx context.Context, cur *mongo.Cursor, span trace.Span) ([]Document, bool, error) {
	defer cur.Close(ctx)
	var docs []Document
	total := 0
	for cur.Next(ctx) {
		// One document is always returned, however large. A page that returns
		// nothing stalls the walk instead of advancing it.
		if len(docs) > 0 && total+len(cur.Current) > maxPageBytes {
			return docs, true, nil
		}
		total += len(cur.Current)
		d := Document{}
		if err := cur.Decode(&d); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, false, err
		}
		docs = append(docs, d)
	}
	if err := cur.Err(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, false, err
	}
	return docs, false, nil
}

func GetDocument(ctx context.Context, urn *urn.URN) (*Document, error) {
	db, coll, err := constructTableTarget(urn)
	if err != nil {
		return nil, ErrBadURN
	}
	id, err := primitive.ObjectIDFromHex(urn.GetResourceID())
	if err != nil {
		return nil, ErrBadURN
	}

	// Start a new client span for the MongoDB FindOne operation.
	tracer := otel.Tracer(tel.GetServiceName())
	ctx, span := tracer.Start(ctx, "find-raw", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	// Set attributes for the MongoDB operation.
	span.SetAttributes(
		attribute.String("db.name", "mongo-raw-data-table"),
		attribute.String("db.operation", "FindOne"),
		attribute.String("db.mongodb.db", db),
		attribute.String("db.mongodb.collection", coll),
		attribute.String("peer.host", "mongo-raw-data-table"),
	)

	r := &Document{}
	if err := mongoClient.Database(databaseName(db)).Collection(coll).FindOne(ctx, bson.M{"_id": id}).Decode(r); err != nil {
		// Record the error on the span.
		span.RecordError(err)
		span.SetStatus(codes.Error, fmt.Sprintf("findOne error: %s", err.Error()))
		if err == mongo.ErrNoDocuments {
			return nil, ErrDocumentNotFound
		}
		return nil, err
	}
	return r, nil
}
