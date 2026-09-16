// SPDX-FileCopyrightText: 2026 NOI Techpark <digital@noi.bz.it>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// The measurements the README's indexing section quotes.
//
// They live here because a performance claim nobody can re-run is a rumour with
// a number in it. The first version of that section timed a single bootstrap
// query and concluded that an index was not worth it — but a bootstrap is not a
// query, it is a walk, and every page of it repeats the same scan, sort and
// group before $limit ever applies. Measuring one page and generalising to the
// walk understated the unindexed cost by the number of pages.
//
//	RDB_TEST_MONGO_URI=mongodb://127.0.0.1:27099 RDB_BENCH=1 \
//	  go test ./rdt/ -run TestWalkCost -v -timeout 30m

package rdt

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	benchDocs = 250_000 // ~a year of an hourly-refreshed collection
	benchKeys = 25_000  // 10 revisions per key
	benchPad  = 500     // bytes of payload, to land near the ~672 B/doc measured live
)

func seedBench(t *testing.T, db, coll string) {
	t.Helper()
	ctx := context.Background()
	c := mongoClient.Database(db).Collection(coll)
	if err := c.Drop(ctx); err != nil {
		t.Fatalf("drop: %v", err)
	}

	pad := strings.Repeat("x", benchPad)
	now := time.Now().UTC()
	start := time.Now()
	batch := make([]any, 0, 1000)
	for i := 0; i < benchDocs; i++ {
		// _id is backdated onto the same timeline as bsontimestamp, so the
		// seeded collection looks like one that filled up over a year rather
		// than one written in two seconds.
		ts := now.Add(-time.Duration(benchDocs-i) * time.Minute)
		batch = append(batch, bson.M{
			"_id":           primitive.NewObjectIDFromTimestamp(ts),
			"provider":      "bench/coll",
			"bsontimestamp": ts,
			"content_type":  "application/json",
			"meta":          bson.M{"key": fmt.Sprintf("K%06d", i%benchKeys)},
			"rawdata":       pad,
		})
		if len(batch) == cap(batch) {
			if _, err := c.InsertMany(ctx, batch); err != nil {
				t.Fatalf("insert: %v", err)
			}
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		if _, err := c.InsertMany(ctx, batch); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	t.Logf("seeded %d documents in %s", benchDocs, time.Since(start).Round(time.Millisecond))
}

// walk pages the whole compacted view and returns how long that took and how
// many round trips it cost.
func walk(t *testing.T, db, coll string, limit int) (time.Duration, int, int) {
	t.Helper()
	ctx := context.Background()
	start := time.Now()
	cursor, pages, seen := "", 0, 0
	for {
		page, err := GetCompacted(ctx, CompactedQuery{
			DB: db, Collection: coll, Field: "key", Limit: limit, Cursor: cursor})
		if err != nil {
			t.Fatalf("walk: %v", err)
		}
		pages++
		seen += len(page.Docs)
		if page.Next == "" {
			break
		}
		cursor = page.Next
	}
	return time.Since(start), pages, seen
}

func TestWalkCost(t *testing.T) {
	if os.Getenv("RDB_BENCH") == "" {
		t.Skip("RDB_BENCH not set")
	}
	initOrSkip(t)
	const db, coll = "bench_test", "walk"
	seedBench(t, db, coll)
	ctx := context.Background()
	c := mongoClient.Database(db).Collection(coll)

	single := func(q CompactedQuery) time.Duration {
		start := time.Now()
		if _, err := GetCompacted(ctx, q); err != nil {
			t.Fatal(err)
		}
		return time.Since(start).Round(time.Millisecond)
	}

	// Every index the README has ever recommended, measured on the same data.
	variants := []struct {
		label string
		keys  bson.D
	}{
		{"none", nil},
		{"key+_id", bson.D{{Key: "meta.key", Value: 1}, {Key: "_id", Value: -1}}},
		{"key+ts+_id", bson.D{{Key: "meta.key", Value: 1}, {Key: "bsontimestamp", Value: -1}, {Key: "_id", Value: -1}}},
		{"ts", bson.D{{Key: "bsontimestamp", Value: -1}}},
	}
	t.Log("index        walk(1000)  pages  page1")
	for _, v := range variants {
		if _, err := c.Indexes().DropAll(ctx); err != nil {
			t.Fatalf("drop indexes: %v", err)
		}
		if v.keys != nil {
			if _, err := c.Indexes().CreateOne(ctx, mongo.IndexModel{
				Keys: v.keys, Options: options.Index().SetName("bench"),
			}); err != nil {
				t.Fatalf("create index: %v", err)
			}
		}
		d, pages, _ := walk(t, db, coll, 1000)
		t.Logf("%-12s %9s  %5d  %6s", v.label,
			d.Round(time.Millisecond), pages,
			single(CompactedQuery{DB: db, Collection: coll, Field: "key", Limit: 1000}))
	}

	// The walk's cost is per *page*, not per bootstrap: the scan, sort and group
	// all run before $limit. Halving the page size doubles the work, which is
	// what makes the byte budget expensive on collections of large documents —
	// there, a 32 MB page holds a dozen or so documents, not a thousand.
	if _, err := c.Indexes().DropAll(ctx); err != nil {
		t.Fatalf("drop indexes: %v", err)
	}
	t.Log("unindexed walk cost against page size:")
	for _, limit := range []int{1000, 250, 50} {
		d, pages, _ := walk(t, db, coll, limit)
		t.Logf("  limit=%-5d %4d pages  %9s total  %7s/page",
			limit, pages, d.Round(time.Millisecond),
			(d / time.Duration(pages)).Round(time.Millisecond))
	}
}
