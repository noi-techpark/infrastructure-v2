<!--
SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>

SPDX-License-Identifier: CC0-1.0
-->

# Raw data bridge

Read access to the Open Data Hub raw data lake. Serves documents out of MongoDB and
transparently inlines payloads that the writer spilled to object storage.

| route | returns |
|---|---|
| `GET /urns/:urn` | one document, by canonical address |
| `GET /:db/:collection/documents` | the log — every revision, newest written first |
| `GET /:db/:collection/compacted/:field` | the newest document per distinct value of a field |
| `GET /health` | liveness |

Both collection routes take `limit` (1–10000, default 1000) and `cursor`, and return the
same envelope:

```jsonc
{
  "count": 3,
  "next": "GwAAAAJmAAQAAABrZXkAAnYABAAAAEIwMQAA", // pass back as `cursor`; empty when done
  "data": [ … ],
  "field": "key"                                  // compacted only: the field it grouped on
}
```

`next` is opaque. The two routes order by different things — `documents` by `_id` descending,
`compacted` by field value ascending — and each cursor carries the ordering it belongs to, so
one issued by the other route, or for another field, is refused with 400. Hand it back
unchanged; do not construct one, and do not expect it to be an ObjectId.

There is no incremental catch-up parameter, deliberately. See "Reconciling" below.

## `GET /urns/:urn`

```
GET /urns/urn:raw:parking-offstreet-famas:bolzano:683ff7249237c6dd345cbf29
```

The URN maps to storage as `urn:raw:{db}:{collection…}:{objectId}`, where the collection is
the remaining namespace tokens joined by `.`. It addresses a document globally — a permalink
rather than a query within a collection, which is why it sits outside the hierarchy below.
Every listed document carries its own `id`, so a permalink can be built for anything you read.

## `GET /:db/:collection/documents`

The log as written: every revision of every key, newest written first.

```
# the last document written
GET /skidata/parking-stations/documents?limit=1

# the recent tail
GET /skidata/parking-stations/documents?limit=20

# the next page
GET /skidata/parking-stations/documents?limit=20&cursor=<next>
```

"Newest" is **write order**, not the publisher's timestamp. `bsontimestamp` comes from a path
segment on the collector's own request, so a backfill is written now and dated whenever the
publisher chose: it leads this list while carrying an old timestamp.

This does not describe current state. A key written once, years ago, still has a current value
and would sit arbitrarily far down the list. That is what the compacted view is for.

## `GET /:db/:collection/compacted/:field`

The newest document per distinct value of `meta.<field>` — a compacted view. Intended for
reference data: configuration, lookup tables, enrichment, anything a consumer needs the
*current* value of for every key.

```
# the current value of every key
GET /enrichment/parking/compacted/key

# paging through it
GET /enrichment/parking/compacted/key?limit=500&cursor=<next>
```

"Newest" here is the publisher's timestamp, not write order, and that is the one place the
distinction matters: a backfill inserted today carrying last year's date must not displace the
revision that is actually current. `_id` breaks ties.

**The field lives under `meta`, and that is the only place it can.** A payload posted as
`application/json` is stored as a *string* in `rawdata`, so nothing inside it can be grouped
on, and the rest of the document root is written by the raw writer. `meta` is the one part a
publisher controls that is not the payload: the writer harvests `X-OpenDataHub-*` request
headers, strips the prefix, lowercases the remainder and stores the result there.

So a collector that wants its documents grouped by facility sends `X-OpenDataHub-facility:
0607242` and the consumer asks for `/compacted/facility`. The path segment is the field name
alone — `meta.` is implied, and passing it explicitly is rejected, as is anything with a dot.

Those values arrive as HTTP headers, so **every one of them is a string**. Nothing the writer
can produce puts another type there, which is why paging compares values directly instead of
carrying a type ladder around.

**Why grouping happens server-side.** Reading the collection in insertion order does not let a
consumer build a complete picture: a key written once, years ago, still has a current value, so
a cold consumer would have to replay the entire history to be sure it had seen every key. That
history grows without bound while the answer stays small. Kafka avoids this by compacting in
the broker; MongoDB does not compact, so the bridge groups instead.

## Reconciling

**Walk every page and you have the whole current state. Walk it again to reconcile.**

That is the entire contract, and there is no incremental alternative on purpose. A re-read
cannot miss a change: not one whose notification was dropped, not one written with an old
timestamp, not one written while the previous walk was running. Nothing to resume from, no
second clock to reason about, no ordering assumption to violate.

It is affordable because a reference table is operator configuration — tens to hundreds of
rows, one page of one response. The measurements below put a 25,000-key collection at around
eleven seconds for a complete walk; a real enrichment table is two orders of magnitude smaller
and answers in milliseconds. Anything cleverer here would be optimising a cost nobody pays,
and paying for it in failure modes that are hard to see.

If a collection ever does grow enough to matter, a lower-bound parameter is an additive change
to a read API. Deferring it costs nothing; shipping it early costs a permanent concept.

A collection that does not exist answers `404`, which an empty collection does not. Only the
first is unambiguously a mistake, and a consumer that reads an empty `200` as "nothing written
yet" cannot otherwise see a typo in its own configuration. The check costs one round trip and
runs only on a first page that came back empty.

## Known limits

**A page of externally stored payloads is bounded by count, not by size.** `rdt` caps a page at
32 MB of BSON, but a document that spilled its payload to object storage is small in Mongo and
arbitrarily large once fetched, and `raw-writer-2` records no length for it. The bridge refuses
a page carrying more than 32 such references (413) and fetches the rest 8 at a time — a latency
and blast-radius guard, not a memory bound. Thirty-two half-gigabyte objects would pass it. The
fix is a size on the document at write time; until then, ask for smaller pages on collections
that spill.

**A wrong `APP_DB_PREFIX` is not detectable.** See below.

## Configuration

| variable | meaning |
|---|---|
| `APP_MONGO_URI` | raw data lake connection string |
| `APP_DB_PREFIX` | prepended to every database name; **must match raw-writer-2's** |
| `APP_S3_ENDPOINT` | object storage endpoint; empty for AWS S3 |
| `APP_S3_ACCESS_KEY_ID` / `APP_S3_SECRET_ACCESS_KEY` | object storage credentials |
| `APP_S3_REGION` | object storage region |
| `APP_LOG_LEVEL` | `DEBUG`, `INFO`, … |
| `SERVICE_NAME`, `TELEMETRY_TRACE_GRPC_ENDPOINT` | telemetry |

The HTTP port is `:2000` and is not configurable.

**On `APP_DB_PREFIX`.** `raw-writer-2` prepends it when choosing which database to write to,
so the bridge has to apply the same one when reading or it would look in a database nothing
writes to. Callers keep using logical names everywhere — in URNs, in paths, in reference table
configuration — and the prefix is applied only at the point the driver is handed a database
name.

The two services do not read the same variable name: the bridge takes `APP_DB_PREFIX`, and
`raw-writer-2` takes bare `DB_PREFIX`. Only the characters are validated at startup, so a
well-formed but *wrong* prefix is not refused. It reads a database nothing writes to, where
every collection is missing, so every read answers `404 no such collection`. A consumer that
fails closed on that — `reftable` does — refuses to start rather than starting empty. Check the
two deployments agree.

## Tests

```sh
go test ./...                        # unit tests, no dependencies

docker run -d --rm -p 27099:27017 mongo:7
RDB_TEST_MONGO_URI=mongodb://127.0.0.1:27099 go test ./... -v
```

`127.0.0.1`, not `localhost`: the container publishes on IPv4, and a host that resolves
`localhost` to `::1` spends the driver's entire server-selection timeout failing before a
single test runs. CI passes the same address for the same reason.

The integration tests skip themselves when `RDB_TEST_MONGO_URI` is unset. CI always sets it —
a suite that silently skips its integration tests is a suite that does not have any.

They live at three levels. `rdt/` covers the queries; the root package drives them through the
real router over HTTP; `rdt/regression_integration_test.go` pins the specific defects this
service has shipped, each with the failure it caused written beside it. The query/HTTP split
exists because a defect once sat in exactly the gap between them — the paging cursor was
derived in the handler, from a field the aggregation had stopped using, so `next` came back
empty and a consumer silently stopped after one page while every query-level test stayed green.

The client and the consumer are tested against a running instance of this service from the SDK
side: `ingest/reftable/e2e_test.go` in `opendatahub-go-sdk` drives a real `raw-writer-2`, a real
MongoDB and this bridge through the same client a transformer uses.

### Benchmarks

The numbers below are produced by a test, not typed in from a terminal session:

```sh
RDB_TEST_MONGO_URI=mongodb://127.0.0.1:27099 RDB_BENCH=1 \
  go test ./rdt/ -run TestWalkCost -v -timeout 30m
```

Re-run it before changing anything in the next section.

## On indexing

There is deliberately no index management here, and both views run against the `_id` index
every collection already has.

Measured by `TestWalkCost` on 250k documents (~168 MB, about a year of an hourly-refreshed
collection, 25k distinct keys), against the exact pipeline this service issues:

| index | full walk (limit=1000, 25 pages) | first page |
|---|---|---|
| none | **11.1 s** | 736 ms |
| `{meta.key: 1, _id: -1}` | 21.9 s | 965 ms |
| `{meta.key: 1, bsontimestamp: -1, _id: -1}` | 13.2 s | 571 ms |
| `{bsontimestamp: -1}` | 12.4 s | 899 ms |

**No index makes the walk faster, and one makes it twice as slow.** `$group` with
`$first: "$$ROOT"` has to fetch every document whichever way it reaches them, so an index can
only reorder the scan, never shorten it — and walking an index on top of that costs between
12% and 97%.

**The cost is per page, not per walk.** The match, sort and group all run before `$limit`, so
every page repeats the whole scan. Halving the page size doubles the work:

| page size | pages | total | per page |
|---|---|---|---|
| 1000 | 25 | 13.8 s | 553 ms |
| 250 | 100 | 51.7 s | 517 ms |
| 50 | 500 | 4 m 23 s | 526 ms |

Per-page cost is flat at roughly half a second on this collection — a walk costs
`pages × 520 ms`, and `limit` does not change the denominator.

That matters most where nobody chooses the page size. `rdt` caps a page at 32 MB, and on a
collection whose documents average 2.24 MB — one measured in production — a page holds about
fourteen documents whatever `limit` says. Extrapolating the per-page cost, 25k keys would be
roughly 1,800 pages and something like a **quarter of an hour** for one walk. That is an
extrapolation, not a measurement: the benchmark collection has 672-byte documents, and
reproducing the large-document case needs half a terabyte. Reference tables are nowhere near
this; a payload collection would be, and `documents` is the route for those.

`allowDiskUse` is set, so the unindexed sort spills rather than failing.
