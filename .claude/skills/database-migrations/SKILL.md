---
name: database-migrations
description: >-
  Goose schema vs data migrations for MySQL (shared/db) and agent Postgres, sqlc regen,
  the frozen 00001_initial dump, and how PlanetScale deploy requests ship. Use when
  adding or changing a table, column, index, goose migration, sqlc query, or backfill.
---

# Database migrations

Goose files are the schema source of truth. Human spec: `docs/patterns/database-migrations.md`. Full workflow also in `AGENTS.md` (Database).

| | Directory | Create with |
|---|---|---|
| Core schema (MySQL) | `shared/db/migrations` | `make migrate-create name=add_foo` |
| Core backfills (MySQL) | `shared/db/data-migrations` | `make migrate-create-data name=backfill_foo` |
| Agent schema (Postgres) | `services/agent-service/db/migrations` | `make migrate-agent-create name=add_foo` |
| Agent backfills (Postgres) | `services/agent-service/db/data-migrations` | `make migrate-agent-create-data name=backfill_foo` |

Backfills are DML. A PlanetScale deploy request diffs **schema only** — DML in a schema migration runs on the branch and silently never reaches prod.

## Writing one

Keep `-- +goose NO TRANSACTION` on schema migrations (Vitess rejects DDL in an explicit tx). Backfills keep the transaction and omit it. Fill both Up and Down.

`shared/db/migrations/00001_initial.sql` is a frozen baseline that **drops every table**. Never edit or regenerate it.

Locally: `make migrate-up` then `make sqlc [service]`. sqlc reads the whole migrations directory.

Every query stays under 50 ms worst case. List filters: `performant-lists` skill.

## Shipping

You never cut PlanetScale branches or open deploy requests by hand. The release PR's `prepare-migrations` opens a deploy request — **review that diff before merging**. Merge runs `deploy-migrations` in order: core schema → agent schema → core backfills → agent backfills. Failed migration stops the image rollout.

Expand-contract: new code never meets a missing column; dropping the old column is a **later** release. Postgres has no deploy request — the PR comment is the review surface.

## Backfills must not bog down the database

Every statement stays under **50 ms**, migrations included: deploy-time backfills run unthrottled on the primary under live traffic.

- `data-migrations/*.sql` is only for **small** changes. Measure each statement on production-sized data (`EXPLAIN ANALYZE` on a branch); it must stay under 50 ms. Repeated `LIMIT 5000` statements do not qualify.
- Anything bigger is a **background backfill on `shared/db/backfill`**, registered in its service (see `platform-service/cmd/backfills.go`). You get a saved cursor (`backfill_progress`), adaptive batches steered to 25 ms statements (wrap each statement in `Meter.Time`), about 20% duty-cycle pacing, one pod under a lease, and a `BACKFILLS_PAUSED` kill switch.
- Make each batch repeatable. Page through a narrow index, read immutable rows from the replica, and write with short guarded primary-key updates (`AND new_col IS NULL`). Work outside the database, such as S3, goes before the row update and never inside a transaction.

Full rationale: `docs/patterns/database-migrations.md` § Keeping data changes off the database's back.
