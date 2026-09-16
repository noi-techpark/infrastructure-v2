// SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>
//
// SPDX-License-Identifier: CC0-1.0

package rdt

import (
	"context"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// Run with a throwaway mongo:
//
//	docker run -d --rm -p 27099:27017 mongo:7
//	RDB_TEST_MONGO_URI=mongodb://localhost:27099 go test ./rdt/ -run Integration -v
func mongoURIOrSkip(t *testing.T) string {
	t.Helper()
	uri := os.Getenv("RDB_TEST_MONGO_URI")
	if uri == "" {
		t.Skip("RDB_TEST_MONGO_URI not set")
	}
	return uri
}

// initOrSkip connects, or fails this test rather than the binary.
func initOrSkip(t *testing.T) {
	t.Helper()
	if err := InitRawDataConnection(mongoURIOrSkip(t), ""); err != nil {
		t.Fatalf("connect: %v", err)
	}
}

func seed(t *testing.T, db, coll string) {
	t.Helper()
	ctx := context.Background()
	c := mongoClient.Database(db).Collection(coll)
	if err := c.Drop(ctx); err != nil {
		t.Fatalf("drop: %v", err)
	}

	now := time.Now().UTC()
	old := now.AddDate(-4, 0, 0)

	// Documents carry the key under `meta`, which is where the raw writer puts
	// publisher-set headers and therefore the only field the bridge can group on.
	docs := []any{
		// A: enriched once, four years ago, never touched since. The case that
		// makes insertion-order tailing unusable as a bootstrap mechanism.
		bson.M{"bsontimestamp": old, "meta": bson.M{"key": "A"}, "rawdata": bson.M{"rev": 1}},

		// B: three revisions; only the newest must survive compaction.
		bson.M{"bsontimestamp": old, "meta": bson.M{"key": "B"}, "rawdata": bson.M{"rev": 1}},
		bson.M{"bsontimestamp": now.Add(-48 * time.Hour), "meta": bson.M{"key": "B"}, "rawdata": bson.M{"rev": 2}},
		bson.M{"bsontimestamp": now.Add(-1 * time.Hour), "meta": bson.M{"key": "B"}, "rawdata": bson.M{"rev": 3}},

		// C: single recent revision.
		bson.M{"bsontimestamp": now.Add(-30 * time.Minute), "meta": bson.M{"key": "C"}, "rawdata": bson.M{"rev": 1}},

		// no key at all — must be ignored by the compacted view entirely
		bson.M{"bsontimestamp": now, "rawdata": bson.M{"rev": 99}},

		// meta present but carrying a different field — also ignored, since the
		// grouped field is missing rather than the whole sub-document.
		bson.M{"bsontimestamp": now, "meta": bson.M{"other": "z"}, "rawdata": bson.M{"rev": 98}},
	}
	// Insert one at a time so _id ordering matches the intended revision order.
	for _, d := range docs {
		if _, err := c.InsertOne(ctx, d); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
}

func keyRev(t *testing.T, d Document) (string, int32) {
	t.Helper()
	// The driver decodes nested documents as Document, not map[string]any —
	// asserting the wrong one fails silently and takes the paging tests with it.
	meta, ok := d["meta"].(Document)
	if !ok {
		t.Fatalf("document has no meta sub-document: %T %v", d["meta"], d)
	}
	k, ok := meta["key"]
	if !ok {
		t.Fatalf("meta carries no key: %v", meta)
	}
	raw, ok := d["rawdata"].(Document)
	if !ok {
		t.Fatalf("rawdata is not a document: %T", d["rawdata"])
	}
	return k.(string), raw["rev"].(int32)
}

func TestIntegrationGetCompacted(t *testing.T) {
	initOrSkip(t)
	const db, coll = "enrichment_test", "parking"
	seed(t, db, coll)
	ctx := context.Background()

	t.Run("bootstrap returns current value of every key", func(t *testing.T) {
		page, err := GetCompacted(ctx, CompactedQuery{DB: db, Collection: coll, Field: "key"})
		if err != nil {
			t.Fatalf("GetCompacted: %v", err)
		}
		if page.Next != "" {
			t.Errorf("Next = %q, want empty on a complete page", page.Next)
		}
		want := map[string]int32{"A": 1, "B": 3, "C": 1}
		if len(page.Docs) != len(want) {
			t.Fatalf("got %d documents, want %d", len(page.Docs), len(want))
		}
		for _, d := range page.Docs {
			k, rev := keyRev(t, d)
			if want[k] != rev {
				t.Errorf("key %s: rev %d, want %d", k, rev, want[k])
			}
			delete(want, k)
		}
		if len(want) != 0 {
			t.Errorf("keys missing from result: %v", want)
		}
	})

	// The regression that shipped: the cursor used to be derived by the caller,
	// which read the wrong field and returned "". Bootstrap then stopped after
	// one page and a table silently loaded a fraction of its keys.
	t.Run("paging reports a usable cursor and covers every key", func(t *testing.T) {
		seen := map[string]int32{}
		cursor := ""
		for pages := 0; ; pages++ {
			if pages > 10 {
				t.Fatal("paging did not terminate")
			}
			page, err := GetCompacted(ctx, CompactedQuery{
				DB: db, Collection: coll, Field: "key", Limit: 1, Cursor: cursor,
			})
			if err != nil {
				t.Fatalf("GetCompacted: %v", err)
			}
			for _, d := range page.Docs {
				k, rev := keyRev(t, d)
				seen[k] = rev
			}
			if page.Next == "" {
				break
			}
			if page.Next == cursor {
				t.Fatalf("cursor did not advance past %q", cursor)
			}
			cursor = page.Next
		}
		want := map[string]int32{"A": 1, "B": 3, "C": 1}
		if len(seen) != len(want) {
			t.Fatalf("paged bootstrap saw %v, want %v", seen, want)
		}
		for k, rev := range want {
			if seen[k] != rev {
				t.Errorf("key %s: rev %d, want %d", k, seen[k], rev)
			}
		}
	})

	t.Run("rejects unsafe identifiers", func(t *testing.T) {
		for _, q := range []CompactedQuery{
			{DB: "$where", Collection: coll, Field: "key"},
			{DB: db, Collection: "a..b", Field: "key"},
			{DB: db, Collection: coll, Field: "meta.key"},
			{DB: db, Collection: coll, Field: ""},
		} {
			if _, err := GetCompacted(ctx, q); err != ErrBadIdentifier {
				t.Errorf("%+v: err = %v, want ErrBadIdentifier", q, err)
			}
		}
	})
}

func TestIntegrationGetDocuments(t *testing.T) {
	initOrSkip(t)
	const db, coll = "enrichment_test", "parking"
	seed(t, db, coll)
	ctx := context.Background()

	t.Run("returns the log newest first, nothing collapsed", func(t *testing.T) {
		page, err := GetDocuments(ctx, DocumentsQuery{DB: db, Collection: coll})
		if err != nil {
			t.Fatalf("GetDocuments: %v", err)
		}
		// Every revision is present, unlike the compacted view.
		if len(page.Docs) != 7 {
			t.Fatalf("got %d documents, want all 7 revisions", len(page.Docs))
		}
		// Newest first: the seed inserts in order, so the last inserted leads.
		first, ok := page.Docs[0].ID()
		if !ok {
			t.Fatal("document has no id")
		}
		last, _ := page.Docs[len(page.Docs)-1].ID()
		if first.Hex() <= last.Hex() {
			t.Errorf("documents are not newest first: %s then %s", first.Hex(), last.Hex())
		}
	})

	t.Run("pages through every document without repeating one", func(t *testing.T) {
		seen := map[string]bool{}
		cursor := ""
		for pages := 0; ; pages++ {
			if pages > 20 {
				t.Fatal("paging did not terminate")
			}
			page, err := GetDocuments(ctx, DocumentsQuery{
				DB: db, Collection: coll, Limit: 2, Cursor: cursor,
			})
			if err != nil {
				t.Fatalf("GetDocuments: %v", err)
			}
			for _, d := range page.Docs {
				id, _ := d.ID()
				if seen[id.Hex()] {
					t.Errorf("document %s returned twice", id.Hex())
				}
				seen[id.Hex()] = true
			}
			if page.Next == "" {
				break
			}
			cursor = page.Next
		}
		if len(seen) != 7 {
			t.Errorf("paged listing saw %d documents, want 7", len(seen))
		}
	})

	t.Run("a cursor it did not issue is refused", func(t *testing.T) {
		_, err := GetDocuments(ctx, DocumentsQuery{DB: db, Collection: coll, Cursor: "not-an-objectid"})
		if err != ErrBadCursor {
			t.Errorf("err = %v, want ErrBadCursor", err)
		}
	})
}

// MarshalJSON must not destroy the document it is asked to serialize.
func TestMarshalJSONDoesNotMutate(t *testing.T) {
	d := Document{"_id": "x", "bsontimestamp": "y", "rawdata": "keep"}
	if _, err := d.MarshalJSON(); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"_id", "bsontimestamp", "rawdata"} {
		if _, ok := d[k]; !ok {
			t.Errorf("%q was removed from the document by marshalling it", k)
		}
	}
}

// A deployment that sets DB_PREFIX has raw-writer-2 writing to <prefix>db. The
// bridge must read from the same place, or every lookup 404s against a database
// nothing writes to — and callers keep using the logical name throughout.
func TestIntegrationDatabasePrefixIsApplied(t *testing.T) {
	uri := mongoURIOrSkip(t)
	if err := InitRawDataConnection(uri, "pfx_"); err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = InitRawDataConnection(uri, "") })

	// Seed the *physical* database, exactly as a prefixed writer would.
	ctx := context.Background()
	c := mongoClient.Database("pfx_prefix_test").Collection("parking")
	if err := c.Drop(ctx); err != nil {
		t.Fatalf("drop: %v", err)
	}
	if _, err := c.InsertOne(ctx, bson.M{
		"bsontimestamp": time.Now().UTC(),
		"meta":          bson.M{"key": "A"},
		"rawdata":       bson.M{"rev": 1},
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}

	// Ask with the logical name.
	page, err := GetCompacted(ctx, CompactedQuery{DB: "prefix_test", Collection: "parking", Field: "key"})
	if err != nil {
		t.Fatalf("GetCompacted: %v", err)
	}
	if len(page.Docs) != 1 {
		t.Fatalf("got %d documents, want 1 — the prefix was not applied", len(page.Docs))
	}

	if _, err := GetDocuments(ctx, DocumentsQuery{DB: "prefix_test", Collection: "parking"}); err != nil {
		t.Errorf("GetDocuments: %v", err)
	}
	// And the unprefixed database must stay empty, or the prefix is being
	// applied somewhere it should not be.
	n, err := mongoClient.Database("prefix_test").Collection("parking").CountDocuments(ctx, bson.M{})
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("the unprefixed database holds %d documents; something bypassed the prefix", n)
	}
}

func TestInitRejectsAnUnusablePrefix(t *testing.T) {
	uri := mongoURIOrSkip(t)
	t.Cleanup(func() { _ = InitRawDataConnection(uri, "") })
	if err := InitRawDataConnection(uri, "bad prefix/"); err == nil {
		t.Error("a prefix that cannot be a database name was accepted")
	}
}
