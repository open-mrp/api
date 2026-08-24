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

While the release PR is open, `prepare-migrations` applies the pending core schema migrations to a fresh PlanetScale branch, opens a deploy request, and comments the full plan on the PR. **Review that deploy request's diff before merging** — merging is what deploys it.

Merging runs `deploy-migrations`, which applies everything in order: core schema, agent schema, core backfills, agent backfills. The EKS rollout requires it to succeed, so a failed migration stops the release before any image ships.

That order is the point: new code never meets a missing column, then never meets an empty one. The matching contract step — dropping the old column — belongs in a _later_ release, once nothing running reads it. A long backfill holds the release open while it runs, so write big ones batched and resumable.

Postgres has no deploy request to review, since PlanetScale applies Postgres DDL directly. The PR comment is the review surface for it and for both sets of backfills.