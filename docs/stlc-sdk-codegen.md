# STLC codegen for Augno SDKs

This reflects your **account migration guide** at repo root [`migration-guide.md`](../../migration-guide.md) and the official **STLC docs** (`stlc-main/packages/stlc/docs/`, especially [`migration-plan.md`](../../stlc-main/packages/stlc/docs/migration-plan.md) and [`codegen.md`](../../stlc-main/packages/stlc/docs/codegen.md)).

## What the migration guide assumes

| Concept | Intended shape |
| --- | --- |
| **Config repo** | The git repo where the OpenAPI spec and `stainless/` workspace live—the **API repo** (`api/`), not each SDK checkout. |
| **Workspace** | A directory tree that contains **`workspace.json`**, **`stainless.yml`**, the spec snapshot, **`custom-code/`**, build manifests (`builds/`), etc. |
| **Discovery** | Run `stlc` **from somewhere under the config repo**; it walks up until it finds **`stainless/workspace.json`**, unless you pass `--workspace`. |
| **SDK repos** | `targets.<lang>.staging_repo` / `production_repo` describe **GitHub repos** (`augno/internal-sdk`, `augno/typescript-sdk`). Builds write **into clones** rooted under the workspace’s **`output_path`** (`output_path` + repo name ⇒ e.g. `core/internal-sdk` in our monorepo). |
| **Day-to-day loop** | `make openapi` (refresh specs) → `stlc build [--push]`; CI uses one workflow against the config repo with `STLC_READ_TOKEN` / `SDK_WRITE_TOKEN`. |

**Anti-patterns** for this model:

- Copying **`stainless/`** or **OpenAPI** into **`internal-sdk`** and running codegen there—you duplicate truth, fight `git clean`, and drift from **`stlc init --from-cloud`** bundle layout.
- Vendoring **`stlc`** inside an SDK repo—install **`stlc` + `stlc-typescript`** from **`sdk-gen/*`** (see **Install stlc** below).

We already have **two workspaces** aligned with **two TS packages**:

| SDK | Workspace | Spec | Stainless config | Repo (`staging_repo`) |
| --- | --- | --- | --- | --- |
| **Internal dashboard client** `@augno/internal-sdk` | [`stainless/internal`](../stainless/internal/) | [`specs/internal_openapi_spec.json`](../specs/) | [`stainless/internal/stainless.yml`](../stainless/internal/stainless.yml) | [`augno/internal-sdk`](https://github.com/augno/internal-sdk) |
| **Public npm client** `@augno/sdk` | [`stainless/public`](../stainless/public/) | [`specs/public_openapi_spec.json`](../specs/) | [`stainless/public/stainless.yml`](../stainless/public/stainless.yml) | [`augno/typescript-sdk`](https://github.com/augno/typescript-sdk) |

`workspace.json` in each workspace sets `output_path` to the monorepo root (`../../../` from `stainless/*/`), so `stlc` targets **`internal-sdk`** and **`typescript-sdk`** sibling directories—not `sdks/` under `api/`—without extra flags.

## Install stlc

Augno uses **forks under [`sdk-gen`](https://github.com/sdk-gen)** (`stlc`, `stlc-typescript`), not the upstream `stainless/*` repos.

From **`api/`**:

```bash
make install-stlc
# or: ./scripts/install-stlc.sh
```

The script uses `STLC_READ_TOKEN` if set, otherwise `gh auth token`. Scope the PAT to **Contents: Read** on `sdk-gen/stlc` and `sdk-gen/stlc-typescript`.

Manual install:

```bash
export STLC_GITHUB_ORG=sdk-gen   # default in scripts/CI
npm install -g \
  git+https://github.com/sdk-gen/stlc.git \
  git+https://github.com/sdk-gen/stlc-typescript.git
```

If the forks are private, use authenticated URLs (as in `scripts/install-stlc.sh`).

Ensure the npm global bin is on your `PATH`:

```bash
export PATH="$(npm config get prefix)/bin:$PATH"
stlc version
```

## Local workflow (recommended)

1. **Install** `stlc` (above).
2. From **`api/`**:
   - Regenerate OpenAPI when protos/services change:

     ```bash
     make openapi
     ```

   - Regenerate SDKs:

     ```bash
     make stlc-internal-sdk           # augno/internal-sdk → @augno/internal-sdk
     make stlc-public-typescript-sdk  # augno/typescript-sdk → @augno/sdk
     # or both:
     make stlc-sdks
     ```

3. **Commits**: use a clean tree per target repo, then e.g.

   ```bash
   STLC_COMMIT='feat(api): regenerate internal SDK' make stlc-internal-sdk
   ```

   (`STLC_COMMIT` expands to `--commit "…"`—same idea as **`stlc build --commit`** in migrate-build.md.)

4. **Custom code**: after bundle cutover, new patches live under **`stainless/*/custom-code/`** and integrate via **`stlc build`**—not in the old Stainless SaaS UI.

### Local STLC checkout + changing `@pkg/sdk-codegen`

When you run **`node …/stlc-main/packages/stlc/dist/index.cjs`** (or iterate on a submodule), remember the TypeScript codegen plugin **`codegen.plugin.*.mjs`** **imports shared logic from** **`codegen.lib.mjs`** via `external: ['./codegen.lib.mjs']`.

- Rebuilding **`@pkg/sdk-codegen-typescript`** refreshes **`codegen.plugin.typescript.mjs`** only.
- **`@pkg/sdk-codegen`** (examples, **`makePrintValue`**, **`generateExample`**, etc.) is bundled into **`packages/sdk-worker/dist/codegen.lib.mjs`**, which **`pnpm turbo build:bundle --filter=@pkg/stlc`** (or **`stlc`**’s bundle step) copies next to **`index.cjs`**.

So after edits under **`stlc-main/packages/sdk-codegen/`**, rebuild the worker codegen lib (e.g. from **`stlc-main/packages/sdk-worker`** run **`pnpm exec tsn scripts/build.ts --skip-plugins`**) and ensure **`stlc-main/packages/stlc/dist/codegen.lib.mjs`** / **`codegen.worker.mjs`** pick up **`sdk-worker/dist`**, otherwise **`stlc build`** keeps using stale shared codegen while the plugin shim points at **`stlc/dist/codegen.lib.mjs`**.

## CI

### Release (canonical)

SDK generation runs **only** from [`.github/workflows/release.yml`](../.github/workflows/release.yml) **`generate-sdks`** after **`publish-openapi-specs`** succeeds:

1. **`publish-openapi-specs`** runs `make openapi` (same as S3 upload), then uploads the **`specs/`** tree as workflow artifact **`release-openapi-specs`**.
2. **`generate-sdks`** calls [`stlc-generate-reusable.yml`](../.github/workflows/stlc-generate-reusable.yml), which **downloads that artifact** (so SDKs match the specs just published to S3), runs **`stlc build --push`**, pushes to **`bot/sdk-sync-<tag>`** on each SDK repo, and **opens (or reuses) a PR** into **`main`** on [`Augno/internal-sdk`](https://github.com/Augno/internal-sdk) and [`Augno/typescript-sdk`](https://github.com/Augno/typescript-sdk).

[`stlc-generate.yml`](../.github/workflows/stlc-generate.yml) is **manual-only** (`workflow_dispatch`); it regenerates specs in the job (`make openapi-quiet`) unless you extend it to consume an artifact.

### Lockfile hygiene (pnpm bootstrap)

TypeScript codegen overwrites `pnpm-lock.yaml` using merged templates from `stlc-typescript`, which may omit fields such as `publishDirectory` even when the generated `package.json` includes `publishConfig.directory`. On GitHub Actions, `pnpm` enables `--frozen-lockfile` whenever `CI` is set, so bootstrap would fail on that mismatch. The stlc workflows run **`env -u CI -u GITHUB_ACTIONS stlc build ...`** so `pnpm` inside bootstrap does not default to `--frozen-lockfile` (GitHub sets `CI` and `GITHUB_ACTIONS`; codegen rewrites `pnpm-lock.yaml` from templates that may omit `publishDirectory` while `package.json` gains `publishConfig.directory`). After **manual** edits to an SDK’s `package.json`, still run `pnpm install` locally and commit an updated `pnpm-lock.yaml` when you are not going through `stlc build`.

| When | SDK branch | PR / merge |
| --- | --- | --- |
| After **release-please** tag, **deploy** succeeds, and **OpenAPI** is published to S3 | `bot/sdk-sync-<tag>` on each SDK repo | Bot opens PR → **`main`**; merge then run SDK release (npm) as today |

`internal-sdk` must have [`.github/workflows/release.yml`](https://github.com/augno/internal-sdk/blob/main/.github/workflows/release.yml) (release-please on `main` + npm publish). Required repo secrets on **`augno/internal-sdk`**: `RELEASE_PLEASE_TOKEN`, `AUGNO_NPM_TOKEN` (or `NPM_TOKEN`). The `api-release` dispatch from the API repo re-triggers release-please after deploy; merging the SDK PR updates **`main`** and can trigger the same.

Production flow (keeps SDKs aligned with what is deployed):

1. Release-please creates an API version tag on `main`.
2. Terraform → build/push images → deploy to EKS.
3. `publish-openapi-specs` uploads specs to S3 and uploads **`specs/`** as a workflow artifact for the next job.
4. **`generate-sdks`** and **`notify-consumers`** run in parallel after step 3:
   - `generate-sdks` downloads that artifact, runs **`stlc build --push`** on branch **`bot/sdk-sync-<tag>`**, and opens SDK PRs into **`main`**.
   - `notify-consumers` dispatches `api-release` to each consumer repo (dashboard, public-docs, openapi-spec, internal-sdk).

**Timing:** Consumer repos sync from the deployed API and S3-published OpenAPI specs; they do not wait for SDK PRs. Downstream npm/GitHub Packages SDK versions may not update until those SDK PRs are **merged** and SDK release workflows publish.

When `stlc build` fails, the release job runs **Print STLC failure report** (`stlc status`, `stlc diagnostics`, `stlc show`, and the latest `builds/*.json` manifest) in the job log and Actions step summary.

Local preview before release: `make openapi` then `make stlc-internal-sdk` / `make stlc-public-typescript-sdk` from `api/`.

### Repository secrets (required before CI can push)

Add these secrets on **`augno/api`** (Settings → Secrets and variables → Actions):

| Secret | Permissions |
| --- | --- |
| **`STLC_READ_TOKEN`** | Fine-grained PAT, **Contents: Read** on `sdk-gen/stlc` and `sdk-gen/stlc-typescript` |
| **`SDK_WRITE_TOKEN`** | Fine-grained PAT, **Contents: Write** and **Pull requests: Write** on `augno/internal-sdk` and `augno/typescript-sdk` (push branches + open/update PRs from automation). |

Authorize both tokens for SSO if your org requires it.

The API release workflow regenerates SDKs and dispatches `api-release` to consumers in parallel after deploy and OpenAPI publish. `internal-sdk` receives both the stlc push (SDK PR) and the dispatch (re-triggers its release workflow).

### internal-sdk release checklist

1. **`augno/internal-sdk` secrets:** `RELEASE_PLEASE_TOKEN`, `AUGNO_NPM_TOKEN` (or `NPM_TOKEN`) — verify with **Actions → Release Doctor**.
2. **`release.yml` on `main`** — release-please opens/merges a release PR or tags; `publish-npm` runs when `release_created` is true.
3. **Conventional commit from stlc** — `fix(sdk): sync with deployed API v…` so release-please cuts a patch release (plain `chore` commits do not bump).

### Publishing packages

`stlc build --push` updates SDK **git** repos only. GitHub Packages / npm publish still requires `release-please` + publish workflows on each SDK repo (see `internal-sdk`’s `release-please-config.json`).

## Parity check

Before cutover ([migrate-validate](../../stlc-main/packages/stlc/docs/migrate-validate.md)): diff SaaS-published SDK trees against `stlc build` output until behavior-bearing drift is gone.
