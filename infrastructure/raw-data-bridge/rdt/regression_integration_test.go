// SPDX-FileCopyrightText: 2026 NOI Techpark <digital@noi.bz.it>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Regressions in the two things the compacted view promises: that it serves the
// current revision, and that one page cannot exhaust the process.
//
// Both were reachable with ordinary data and neither turned a test red.

package rdt

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// The publisher supplies bsontimestamp: raw-writer-2 routes
// POST /{provider1}/{provider2}/{timestamp} and stores that path segment. So
// insert order and write time are independent, and a backfill produces an older
// document with a later _id. Ordering the group on _id served that stale
// document as the current value, with nothing downstream able to tell.
func TestCompactedPrefersWriteTimeOverInsertOrder(t *testing.T) {
	initOrSkip(t)
	const db, coll = "regression_test", "backfill"
	ctx := context.Background()
	c := mongoClient.Database(db).Collection(coll)
	if err := c.Drop(ctx); err != nil {
		t.Fatalf("drop: %v", err)
	}
	t.Cleanup(func() { _ = mongoClient.Database(db).Drop(context.Background()) })

	now := time.Now().UTC()

	// Inserted first, and current.
	if _, err := c.InsertOne(ctx, bson.M{
		"meta": bson.M{"key": "K"}, "bsontimestamp": now, "rawdata": "CURRENT",
	}); err != nil {
		t.Fatal(err)
	}
	// Inserted second, so a later _id, but four years older: a replay.
	if _, err := c.InsertOne(ctx, bson.M{
		"meta": bson.M{"key": "K"}, "bsontimestamp": now.AddDate(-4, 0, 0), "rawdata": "STALE",
	}); err != nil {
		t.Fatal(err)
	}

	page, err := GetCompacted(ctx, CompactedQuery{DB: db, Collection: coll, Field: "key"})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Docs) != 1 {
		t.Fatalf("got %d documents, want 1", len(page.Docs))
	}
	if got := page.Docs[0]["rawdata"]; got != "CURRENT" {
		t.Errorf("compacted view served %q; the newest write is CURRENT", got)
	}
}

// A page was bounded by document count alone. Payloads are stored inline up to
// 5 MB, so a default limit against large documents projects to gigabytes in a
// pod capped at a few hundred megabytes.
func TestPageIsBoundedByBytesNotOnlyByCount(t *testing.T) {
	initOrSkip(t)
	const db, coll = "regression_test", "large"
	ctx := context.Background()
	c := mongoClient.Database(db).Collection(coll)
	if err := c.Drop(ctx); err != nil {
		t.Fatalf("drop: %v", err)
	}
	t.Cleanup(func() { _ = mongoClient.Database(db).Drop(context.Background()) })

	// 40 documents of 1 MB: over the 32 MB budget, well under any count limit.
	const each, count = 1 << 20, 40
	payload := strings.Repeat("x", each)
	now := time.Now().UTC()
	docs := make([]any, 0, count)
	for i := 0; i < count; i++ {
		docs = append(docs, bson.M{
			"meta":          bson.M{"key": string(rune('a'+i/26)) + string(rune('a'+i%26))},
			"bsontimestamp": now.Add(time.Duration(i) * time.Second),
			"rawdata":       payload,
		})
	}
	if _, err := c.InsertMany(ctx, docs); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name string
		get  func() (Page, error)
	}{
		{"compacted", func() (Page, error) {
			return GetCompacted(ctx, CompactedQuery{DB: db, Collection: coll, Field: "key", Limit: count})
		}},
		{"documents", func() (Page, error) {
			return GetDocuments(ctx, DocumentsQuery{DB: db, Collection: coll, Limit: count})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			page, err := tc.get()
			if err != nil {
				t.Fatal(err)
			}
			if len(page.Docs) == count {
				t.Fatalf("returned all %d documents, about %d MB: the byte budget did not apply",
					count, count*each>>20)
			}
			if len(page.Docs) == 0 {
				t.Fatal("returned nothing; a page must always yield at least one document or the walk stalls")
			}
			// A short page is only honest if it offers a continuation.
			if page.Next == "" {
				t.Errorf("stopped at %d documents with next=\"\", which reads as the end of the collection",
					len(page.Docs))
			}
		})
	}
}

// The budget must never produce an empty page, even when a single document
// exceeds it on its own, or a consumer paging to completion never terminates.
func TestOneOversizedDocumentIsStillReturned(t *testing.T) {
	initOrSkip(t)
	const db, coll = "regression_test", "oversized"
	ctx := context.Background()
	c := mongoClient.Database(db).Collection(coll)
	if err := c.Drop(ctx); err != nil {
		t.Fatalf("drop: %v", err)
	}
	t.Cleanup(func() { _ = mongoClient.Database(db).Drop(context.Background()) })

	// MongoDB caps a document at 16 MB, so the only way to reach this state is
	// with a budget below one document. Lowering it is the point: the guard must
	// hold for any budget, not just the shipped one.
	restore := maxPageBytes
	maxPageBytes = 1 << 20
	t.Cleanup(func() { maxPageBytes = restore })

	for i, k := range []string{"a", "b"} {
		if _, err := c.InsertOne(ctx, bson.M{
			"meta":          bson.M{"key": k},
			"bsontimestamp": time.Now().UTC().Add(time.Duration(i) * time.Second),
			"rawdata":       strings.Repeat("y", 2<<20),
		}); err != nil {
			t.Fatal(err)
		}
	}

	page, err := GetDocuments(ctx, DocumentsQuery{DB: db, Collection: coll, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Docs) != 1 {
		t.Fatalf("got %d documents, want exactly 1", len(page.Docs))
	}
	if page.Next == "" {
		t.Error("no continuation offered, so the second document is unreachable")
	}
}

// A cursor from another field, or one nobody issued, must be refused. It used to
// be fed straight to $gt, so it returned a plausible-looking short page and the
// consumer read that as the end of the collection.
func TestCompactedRefusesACursorItDidNotIssue(t *testing.T) {
	initOrSkip(t)
	const db, coll = "regression_test", "cursors"
	ctx := context.Background()
	c := mongoClient.Database(db).Collection(coll)
	if err := c.Drop(ctx); err != nil {
		t.Fatalf("drop: %v", err)
	}
	t.Cleanup(func() { _ = mongoClient.Database(db).Drop(context.Background()) })
	if _, err := c.InsertOne(ctx, bson.M{
		"meta":          bson.M{"key": "K", "other": "O"},
		"bsontimestamp": time.Now().UTC(), "rawdata": "v",
	}); err != nil {
		t.Fatal(err)
	}

	otherCursor := encodeCursor(pageCursor{Field: "other", Value: "O"})
	good := encodeCursor(pageCursor{Field: "key", Value: "K"})

	for _, tc := range []struct{ name, cursor string }{
		{"not a cursor at all", "NOT-A-CURSOR"},
		{"issued for another field", otherCursor},
		{"truncated", good[:len(good)/2]},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := GetCompacted(ctx, CompactedQuery{
				DB: db, Collection: coll, Field: "key", Cursor: tc.cursor})
			if !errors.Is(err, ErrBadCursor) {
				t.Errorf("err = %v, want ErrBadCursor; a bad cursor served a page instead", err)
			}
		})
	}
}

// A collection that does not exist and one that is merely empty both returned an
// empty 200. Only the first is unambiguously wrong, and a consumer reading the
// second as "no data yet" cannot see a typo in its own configuration.
func TestAbsentCollectionIsNotAnEmptyOne(t *testing.T) {
	initOrSkip(t)
	const db = "regression_test"
	ctx := context.Background()
	t.Cleanup(func() { _ = mongoClient.Database(db).Drop(context.Background()) })

	// An existing but empty collection is a legitimate answer.
	if err := mongoClient.Database(db).CreateCollection(ctx, "empty_but_real"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := GetCompacted(ctx, CompactedQuery{
		DB: db, Collection: "empty_but_real", Field: "key"}); err != nil {
		t.Errorf("an empty collection must not be an error, got %v", err)
	}

	// A collection nothing writes to is a misconfiguration.
	for _, name := range []string{"compacted", "documents"} {
		t.Run(name, func(t *testing.T) {
			var err error
			if name == "compacted" {
				_, err = GetCompacted(ctx, CompactedQuery{
					DB: db, Collection: "typo_in_the_values_file", Field: "key"})
			} else {
				_, err = GetDocuments(ctx, DocumentsQuery{
					DB: db, Collection: "typo_in_the_values_file"})
			}
			if !errors.Is(err, ErrCollectionNotFound) {
				t.Errorf("err = %v, want ErrCollectionNotFound", err)
			}
		})
	}

}

// documents accepted any hex ObjectId, including one that belonged to another
// collection's ordering, and paged from wherever it landed. It now goes through
// the same envelope as the compacted cursor.
func TestDocumentsRefusesACursorFromAnotherOrdering(t *testing.T) {
	initOrSkip(t)
	const db, coll = "regression_test", "doc_cursors"
	ctx := context.Background()
	c := mongoClient.Database(db).Collection(coll)
	if err := c.Drop(ctx); err != nil {
		t.Fatalf("drop: %v", err)
	}
	t.Cleanup(func() { _ = mongoClient.Database(db).Drop(context.Background()) })
	if _, err := c.InsertOne(ctx, bson.M{
		"meta": bson.M{"key": "K"}, "bsontimestamp": time.Now().UTC(), "rawdata": "v"}); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct{ name, cursor string }{
		{"a bare object id", "683ff7249237c6dd345cbf29"},
		{"a compacted cursor", encodeCursor(pageCursor{Field: "key", Value: "K"})},
		{"not a cursor at all", "nonsense"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := GetDocuments(ctx, DocumentsQuery{
				DB: db, Collection: coll, Cursor: tc.cursor}); !errors.Is(err, ErrBadCursor) {
				t.Errorf("err = %v, want ErrBadCursor", err)
			}
		})
	}

	// And the one it issued itself is accepted.
	page, err := GetDocuments(ctx, DocumentsQuery{DB: db, Collection: coll, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if page.Next != "" {
		if _, err := GetDocuments(ctx, DocumentsQuery{
			DB: db, Collection: coll, Cursor: page.Next}); err != nil {
			t.Errorf("a cursor this endpoint issued was refused: %v", err)
		}
	}
}

// A dotted database name is not a namespace MongoDB has; it used to reach the
// driver and come back as a 500.
func TestDottedDatabaseNameIsRejected(t *testing.T) {
	initOrSkip(t)
	ctx := context.Background()
	if _, err := GetDocuments(ctx, DocumentsQuery{
		DB: "a.b", Collection: "c"}); !errors.Is(err, ErrBadIdentifier) {
		t.Errorf("err = %v, want ErrBadIdentifier", err)
	}
	if _, err := GetCompacted(ctx, CompactedQuery{
		DB: "a.b", Collection: "c", Field: "key"}); !errors.Is(err, ErrBadIdentifier) {
		t.Errorf("err = %v, want ErrBadIdentifier", err)
	}
}
