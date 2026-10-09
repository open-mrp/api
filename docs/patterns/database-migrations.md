# Database Migration Patterns

Both databases are migrated with goose. The migration files are the source of truth for the schema.

## Where things go

|                            | Directory                                   | Create with                                        |
| -------------------------- | ------------------------------------------- | -------------------------------------------------- |
| Core schema (MySQL)        | `shared/db/migrations`                      | `make migrate-create name=add_foo`                 |
| Core backfills (MySQL)     | `shared/db/data-migrations`                 | `make migrate-create-data name=backfill_foo`       |
| Agent schema (Postgres)    | `services/agent-service/db/migrations`      | `make migrate-agent-create name=add_foo`           |
| Agent backfills (Postgres) | `services/agent-service/db/data-migrations` | `make migrate-agent-create-data name=backfill_foo` |

Backfills — copy a column into another, seed a row, reshape existing rows — are DML and belong in the `data-migrations` directories, tracked in their own `goose_db_version_data` table. They are separate because a PlanetScale deploy request diffs _schema only_: DML written into a schema migration runs against the dev branch and silently never reaches prod.

## Writing one

The `migrate-create` targets scaffold the file, numbered sequentially. Fill in both halves:

```sql
-- +goose NO TRANSACTION
-- +goose Up

ALTER TABLE `sales_order` ADD COLUMN `note` varchar(255) NULL;

-- +goose Down

ALTER TABLE `sales_order` DROP COLUMN `note`;
```

`NO TRANSACTION` is in the schema template because Vitess rejects DDL inside an explicit transaction. Backfills keep their transaction and omit it.

Then locally:

```bash
make migrate-up     # apply to the local Docker MySQL
make sqlc           # regenerate typed queries for affected services
```

sqlc reads the whole migrations directory, so a new file reaches every service's generated code on its own.

`shared/db/migrations/00001_initial.sql` is a frozen baseline. Never edit or regenerate it — it opens by dropping every table.

## How it ships

Automated off the release PR. You never cut branches or open deploy requests by hand.

While the release PR is open, `prepare-migrations` applies the pending core schema migrations to a fresh PlanetScale branch, opens a deploy request, and comments the full plan on the PR. **Review that deploy request's diff before merging** — merging is what deploys it. A deploy request may change at most 10 tables; a release over that is split automatically (`tools/schemasplit`) into several deploy requests of at most 10 tables each, listed in order in the PR comment and deployed one after another on merge.

Merging runs `deploy-migrations`, which applies everything in order: core schema, agent schema, core backfills, agent backfills. The EKS rollout requires it to succeed, so a failed migration stops the release before any image ships.

That order is the point: new code never meets a missing column, then never meets an empty one. The matching contract step — dropping the old column — belongs in a _later_ release, once nothing running reads it.

## Keeping data changes off the database's back

Every statement against production stays under **50 ms**, migrations included. Deploy-time backfills run on the primary, unthrottled, while live traffic is on it. One heavy statement there can hold locks, flood the binlog, and evict the buffer pool pages that requests are reading.

**A deploy-time backfill (`data-migrations/*.sql`) must be small.** Each statement must be measured on production-sized data (`EXPLAIN ANALYZE` on a branch) and stay under 50 ms. In practice that means seeding or fixing a bounded set of rows by primary key or a selective index. Repeating a `LIMIT 5000` statement does not qualify: each copy is still one long statement.

**Anything bigger is a background backfill on `shared/db/backfill`, not a migration.** For a worked example, see the `request_log_payloads` backfill in commit `68d752de` (`platform-service/cmd/backfills.go` and `repository/request_log_payload_backfill.go`). It wires a small replica pool, a `backfill.Progress` on `backfill_progress`, and `Runner.Keep` under the service's lease, behind a `BACKFILLS_PAUSED` kill switch. Delete it once it has completed and the change it served has shipped. The runner gives every backfill:

- **Keyset batches from a saved cursor** (`backfill_progress`), so it resumes after a restart or deploy, and never runs again once complete.
- **Adaptive batch size.** Each batch wraps its statements in `Meter.Time`. The size shrinks when the slowest statement passes 25 ms and grows when well under, so statements stay inside the budget whatever the row sizes.
- **Duty-cycle pacing.** After each batch it sleeps four times the batch's database time, so a backfill never takes more than about 20% of one connection.
- **One pod at a time** (`Runner.Keep`, under a lease), with a smaller retry after a failed batch.

Write each batch to be repeatable. Walk a narrow index for the next page instead of the clustered rows. Read bulk data from the replica when the rows are immutable. Keep primary writes to short primary-key updates, guarded so a re-run changes nothing (`… WHERE id IN (…) AND new_col IS NULL`). Work outside the database, such as an S3 upload, happens before the row update that depends on it, never inside a transaction.

Expand-contract still applies: ship the code that reads both shapes first, let the backfill finish, then drop the old column in a later release.

Postgres has no deploy request to review, since PlanetScale applies Postgres DDL directly. The PR comment is the review surface for it and for both sets of backfills.