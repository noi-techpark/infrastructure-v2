// SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>
//
// SPDX-License-Identifier: CC0-1.0

// HTTP-level tests.
//
// These exist because the layer below them was already covered and the layer
// here was not, which is exactly where a defect lived: the cursor was derived
// in the handler, from a field the aggregation had stopped using, so paging
// returned an empty continuation token and a consumer silently stopped after
// one page. The query tests passed throughout — they never went through a
// handler.
//
//	docker run -d --rm -p 27099:27017 mongo:7
//	RDB_TEST_MONGO_URI=mongodb://localhost:27099 go test -run Integration -v
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"opendatahub.com/infrav2/raw-data-bridge/rdt"
)

const testDB, testColl = "bridge_http_test", "parking"

type pageResponse struct {
	Count int               `json:"count"`
	Next  string            `json:"next"`
	Field string            `json:"field"`
	Data  []json.RawMessage `json:"data"`
}

func serverOrSkip(t *testing.T, keys int) *Server {
	t.Helper()
	uri := os.Getenv("RDB_TEST_MONGO_URI")
	if uri == "" {
		t.Skip("RDB_TEST_MONGO_URI not set")
	}
	if err := rdt.InitRawDataConnection(uri, ""); err != nil {
		t.Fatalf("connect: %v", err)
	}
	seedKeys(t, keys)
	return NewServer()
}

// mongoCollection opens its own connection: rdt keeps its client unexported,
// and a test that seeds through the same handle it reads through would be
// asserting less than it looks like it is.
func mongoCollection(t *testing.T) *mongo.Collection {
	t.Helper()
	client, err := mongo.Connect(context.Background(),
		options.Client().ApplyURI(os.Getenv("RDB_TEST_MONGO_URI")))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })
	return client.Database(testDB).Collection(testColl)
}

// seedKeys writes `keys` distinct keys, each with two revisions.
func seedKeys(t *testing.T, keys int) {
	t.Helper()
	ctx := context.Background()
	c := mongoCollection(t)
	if err := c.Drop(ctx); err != nil {
		t.Fatalf("drop: %v", err)
	}
	now := time.Now().UTC()
	for i := 0; i < keys; i++ {
		key := fmt.Sprintf("K%03d", i)
		for rev := 1; rev <= 2; rev++ {
			_, err := c.InsertOne(ctx, bson.M{
				"bsontimestamp": now.Add(time.Duration(rev) * time.Minute),
				"meta":          bson.M{"key": key},
				"content_type":  "application/json",
				"rawdata":       fmt.Sprintf(`{"rev":%d}`, rev),
			})
			if err != nil {
				t.Fatalf("insert: %v", err)
			}
		}
	}
}

func get(t *testing.T, s *Server, path string) (int, pageResponse) {
	t.Helper()
	w := httptest.NewRecorder()
	s.e.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	var body pageResponse
	if w.Code == http.StatusOK {
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("response is not a page: %v (%s)", err, w.Body.String())
		}
	}
	return w.Code, body
}

// The regression this file exists for. With a page limit below the number of
// keys, a caller must be able to walk to the end — and must actually see every
// key, not just the first page.
func TestIntegrationCompactedPagingOverHTTP(t *testing.T) {
	const keys = 7
	s := serverOrSkip(t, keys)

	seen := map[string]bool{}
	path := fmt.Sprintf("/%s/%s/compacted/key?limit=2", testDB, testColl)
	for pages := 0; ; pages++ {
		if pages > 20 {
			t.Fatal("paging did not terminate")
		}
		code, body := get(t, s, path)
		if code != http.StatusOK {
			t.Fatalf("GET %s: status %d", path, code)
		}
		if body.Field != "key" {
			t.Errorf("field = %q, want the grouped field echoed back", body.Field)
		}
		for _, raw := range body.Data {
			var d struct {
				Meta map[string]string `json:"meta"`
			}
			if err := json.Unmarshal(raw, &d); err != nil {
				t.Fatalf("document is not json: %v", err)
			}
			seen[d.Meta["key"]] = true
		}
		if body.Next == "" {
			break
		}
		path = fmt.Sprintf("/%s/%s/compacted/key?limit=2&cursor=%s", testDB, testColl, body.Next)
	}

	if len(seen) != keys {
		t.Errorf("paged over HTTP saw %d keys, want %d — a consumer would bootstrap incomplete",
			len(seen), keys)
	}
}

// A single page must not advertise a continuation, or a consumer loops forever.
func TestIntegrationCompactedCompletePageHasNoCursor(t *testing.T) {
	s := serverOrSkip(t, 3)
	code, body := get(t, s, fmt.Sprintf("/%s/%s/compacted/key", testDB, testColl))
	if code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if body.Next != "" {
		t.Errorf("next = %q on a complete page", body.Next)
	}
	if body.Count != 3 {
		t.Errorf("count = %d, want 3 (one per key, revisions collapsed)", body.Count)
	}
}

// The log view returns every revision, unlike the compacted one, and pages the
// same way.
func TestIntegrationDocumentsOverHTTP(t *testing.T) {
	const keys = 4
	s := serverOrSkip(t, keys)

	total := 0
	path := fmt.Sprintf("/%s/%s/documents?limit=3", testDB, testColl)
	for pages := 0; ; pages++ {
		if pages > 20 {
			t.Fatal("paging did not terminate")
		}
		code, body := get(t, s, path)
		if code != http.StatusOK {
			t.Fatalf("status %d", code)
		}
		total += body.Count
		if body.Next == "" {
			break
		}
		path = fmt.Sprintf("/%s/%s/documents?limit=3&cursor=%s", testDB, testColl, body.Next)
	}

	if total != keys*2 {
		t.Errorf("documents returned %d rows, want %d — every revision, nothing collapsed",
			total, keys*2)
	}
}

func TestIntegrationBadRequests(t *testing.T) {
	s := serverOrSkip(t, 2)

	cases := map[string]string{
		"invalid db":       fmt.Sprintf("/$where/%s/compacted/key", testColl),
		"dotted field":     fmt.Sprintf("/%s/%s/compacted/meta.key", testDB, testColl),
		"limit zero":       fmt.Sprintf("/%s/%s/compacted/key?limit=0", testDB, testColl),
		"limit too large":  fmt.Sprintf("/%s/%s/compacted/key?limit=999999", testDB, testColl),
		"bad doc cursor":   fmt.Sprintf("/%s/%s/documents?cursor=nonsense", testDB, testColl),
		"bad docs limit":   fmt.Sprintf("/%s/%s/documents?limit=-1", testDB, testColl),
		"documents bad db": fmt.Sprintf("/$where/%s/documents", testColl),
		// A dotted database is not a namespace MongoDB has; it used to reach
		// the driver and come back as a 500.
		"dotted db":    fmt.Sprintf("/a.b/%s/compacted/key", testColl),
		"stale cursor": fmt.Sprintf("/%s/%s/documents?cursor=683ff7249237c6dd345cbf29", testDB, testColl),
	}
	for name, path := range cases {
		t.Run(name, func(t *testing.T) {
			code, _ := get(t, s, path)
			if code != http.StatusBadRequest {
				t.Errorf("GET %s: status %d, want 400", path, code)
			}
		})
	}
}

// A listed document has to be addressable, or the permalink route is unreachable
// from the listing routes.
func TestIntegrationListedDocumentsCarryTheirId(t *testing.T) {
	s := serverOrSkip(t, 2)

	code, body := get(t, s, fmt.Sprintf("/%s/%s/documents?limit=1", testDB, testColl))
	if code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if len(body.Data) != 1 {
		t.Fatalf("got %d documents, want 1", len(body.Data))
	}
	var d struct {
		ID  string `json:"id"`
		Oid string `json:"_id"`
	}
	if err := json.Unmarshal(body.Data[0], &d); err != nil {
		t.Fatalf("document is not json: %v", err)
	}
	if d.Oid != "" {
		t.Error("the storage spelling `_id` leaked into the response")
	}
	if _, err := primitive.ObjectIDFromHex(d.ID); err != nil {
		t.Fatalf("id = %q, want an object id a caller can build a urn from: %v", d.ID, err)
	}
}
