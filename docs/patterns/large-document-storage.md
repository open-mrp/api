# Large documents in object storage

Request and response bodies, bulk payloads, and cached responses are large, unbounded, and read
rarely. Kept in MySQL they dominate the table size and the buffer pool, and evict the rows that
queries actually read. They live in the payloads bucket instead, and the owning row keeps a key.
`shared/blobstore` is the only way in and out.

## Rules

1. **The row keeps metadata and a key, not the document.** Filtering, sorting, scoping, and
   authorization stay in SQL. A NULL key means any document is still inline in the row's own column.
2. **Write the object first, then the row that points at it.** A row never names a missing object.
   An object whose row never lands is unreferenced, and the bucket's lifecycle rule expires it. No
   sweeper is needed.
3. **Key by the owning record's id.** For example `request-logs/<id>.json.gz` or
   `jobs/<job_id>/items.json.gz`. A retry, redelivery, or replay rewrites the same bytes under the
   same key, so every write is idempotent with no coordination.
4. **No object-store call inside a database transaction.** Either place the document before the
   transaction (`JobSvc.StageJobItems`), or write it inline and move it after commit
   (`db.AfterCommit`). Idempotent responses do the second: the move uploads, then
   `UPDATE … SET body = NULL, key = ? WHERE key IS NULL`. If a move fails part way, the body stays
   inline and is still read.
5. **Objects are immutable.** Never update one in place. Bundle a record's large fields into one
   gzipped object, so it costs one PUT and one GET.
6. **Read lazily.** Only a single-record read that asked for the document (an `include`) touches
   the bucket. List endpoints never do.
7. **Small documents stay inline.** `blobstore.Store.Spill` keeps anything up to
   `blobstore.InlineLimit` (16 KiB) in the row, so the common case pays no round trip. Request logs
   are the exception and always go to the bucket: their volume, not their size, is the problem.
   If the gateway's upload fails, platform-service uploads the bodies when it consumes the log.
   Without a bucket (local stacks), request-log bodies are not kept.
8. **Retention is a bucket lifecycle rule per prefix, not a DELETE loop.** Each prefix's expiry is
   at least as long as its row's.
9. **Redact before upload.** The bucket holds what the row would have held, no more. It is private
   and encrypted, and clients reach a document only through the API, after the normal access check.

## Prefixes

| Prefix | Written by | Inline limit | Expiry |
|---|---|---|---|
| `request-logs/` | api-gateway, after the response | none (always stored) | 7 years |
| `jobs/` | core-service, before the job's transaction | 16 KiB | 30 days |
| `idempotency/` | core-service and platform-service, after commit | 16 KiB | 35 days (rows expire at 30) |

## Moving existing rows

Rows written before a document moved to the bucket are moved by a background backfill on
`shared/db/backfill`, never by a deploy-time migration (see `database-migrations.md`). Once it
reports complete, a later release drops the old columns, and that rebuild copies only the slim rows.
`request_log` went this way: its bodies now exist only in the bucket. A log with no
`payload_key` had no bodies, and its body includes read as `{}`.

## Local stacks

`PAYLOADS_BUCKET` is optional. When it is unset, `blobstore.Open` returns a nil `*Store`, and a nil
store keeps every document inline, except request-log bodies, which are dropped. The e2e stack sets it against MinIO, so the object-storage path
is what e2e exercises.
