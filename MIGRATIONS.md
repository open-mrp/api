# Database Migrations with PlanetScale

This guide covers best practices for managing database schema changes using PlanetScale's safe migrations workflow.

## Overview

We use [PlanetScale](https://planetscale.com) as our production database with **safe migrations** enabled. This provides:

- **Zero-downtime schema migrations** - Changes are applied online without blocking queries
- **Schema revert** - 30-minute window to undo any deployed schema change
- **Protection against accidental changes** - DDL statements are rejected on protected branches

All schema changes must go through PlanetScale's **deploy request** workflow before being merged to the codebase.

## Workflow

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  Create PR with │────▶│  CI detects      │────▶│  Create PS dev  │
│  schema changes │     │  schema diff     │     │  branch         │
└─────────────────┘     └──────────────────┘     └─────────────────┘
                                                          │
                                                          ▼
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  Merge PR after │◀────│  Deploy request  │◀────│  Open deploy    │
│  schema deploys │     │  approved        │     │  request        │
└─────────────────┘     └──────────────────┘     └─────────────────┘
```

## Making Schema Changes

### 1. Create a Feature Branch

Create a new Git branch for your changes:

```bash
git checkout -b feature/add-user-preferences
```

### 2. Update the Schema File

Edit the schema in `shared/db/migrations/`. This file is used by sqlc to generate Go code.

```sql
-- Add new table
CREATE TABLE user_preferences (
    id VARCHAR(26) NOT NULL PRIMARY KEY,
    user_id VARCHAR(26) NOT NULL,
    theme VARCHAR(20) NOT NULL DEFAULT 'light',
    notifications_enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
```

### 3. Create a PlanetScale Development Branch

```bash
# Using PlanetScale CLI
pscale branch create <database> feature/add-user-preferences --org <org>

# Or via the PlanetScale dashboard
# https://app.planetscale.com/<org>/<database>/branches
```

### 4. Apply Changes to Development Branch

Connect to your development branch and apply the schema changes:

```bash
# Connect to the dev branch
pscale shell <database> feature/add-user-preferences --org <org>

# Then run your DDL statements
CREATE TABLE user_preferences (...);
```

Or use the PlanetScale console in the dashboard.

### 5. Open a Deploy Request

```bash
# Using CLI
pscale deploy-request create <database> feature/add-user-preferences --org <org>

# Or via dashboard
# https://app.planetscale.com/<org>/<database>/deploy-requests
```

The deploy request will show:
- Line-by-line schema diff
- Conflict detection against current production
- Estimated deployment time

### 6. Get Review and Approval

Share the deploy request link with your team. Reviewers can:
- View the exact DDL changes
- Comment on the changes
- Approve or request changes

### 7. Deploy the Schema Changes

Once approved:

1. Click "Deploy changes" in PlanetScale dashboard
2. Wait for deployment to complete (zero-downtime)
3. Verify the changes in production

### 8. Merge Your PR

After the schema is deployed to production, merge your GitHub PR. The CI will confirm the schema is in sync.

## Best Practices

### Keep Schema and Code Changes Together

Always include schema changes in the same PR as the code that uses them. This ensures:
- Reviewers see the full context
- CI can validate sqlc generation
- Changes are documented together

### Test on Development Branch First

Before opening a deploy request:

```bash
# Connect to dev branch
pscale connect <database> feature/add-user-preferences --port 3307 --org <org>

# Run your application against it
DATABASE_URL="root@tcp(127.0.0.1:3307)/<database>" go test ./...
```

### Use Descriptive Branch Names

Match your PlanetScale branch name to your Git branch for traceability:
- Git: `feature/add-user-preferences`
- PlanetScale: `feature/add-user-preferences`

### Avoid Breaking Changes

PlanetScale's safe migrations help, but avoid:
- Dropping columns that are still referenced
- Renaming columns (add new, migrate data, then drop old)
- Changing column types without migration

### Clean Up Development Branches

After merging, delete your PlanetScale development branch:

```bash
pscale branch delete <database> feature/add-user-preferences --org <org>
```

## Reverting Schema Changes

If you need to undo a deployed schema change, you have a 30-minute window:

1. Go to the deploy request in PlanetScale dashboard
2. Click "Revert changes"
3. The schema instantly reverts to the previous state
4. Data added during this time is preserved

After 30 minutes, you'll need to create a new deploy request with the reverse changes.

## CI Integration

Our CI automatically:

1. Detects when PRs modify `shared/db/migrations/`
2. Compares local schema against PlanetScale production
3. Posts a comment with the schema diff
4. Reminds you to create a deploy request

The workflow runs in `.github/workflows/schema-check.yml`.

## Troubleshooting

### "DDL statements are not allowed"

This means you're trying to run DDL on a branch with safe migrations enabled (like `main`). Create a development branch instead.

### Deploy Request Shows Conflicts

Your changes conflict with recent production changes. Sync your development branch:

```bash
# Delete and recreate from latest main
pscale branch delete <database> feature/my-branch --org <org>
pscale branch create <database> feature/my-branch --org <org>
# Re-apply your changes
```

### Schema Diff in CI Doesn't Match

The CI compares your local `shared/db/migrations/` files against production. If they don't match:

1. Ensure your deploy request was actually deployed
2. Pull latest `main` and rebase
3. Check for formatting differences

### sqlc Generation Fails

After schema changes, regenerate sqlc code:

```bash
make gen-sqlc
```

Ensure your queries in `internal/infrastructure/queries/` are compatible with the new schema.

## Resources

- [PlanetScale Safe Migrations Docs](https://planetscale.com/docs/concepts/safe-migrations)
- [PlanetScale Branching Docs](https://planetscale.com/docs/concepts/branching)
- [PlanetScale CLI Reference](https://planetscale.com/docs/reference/planetscale-cli)
- [Deploy Requests Documentation](https://planetscale.com/docs/concepts/deploy-requests)

