# STLC codegen for OpenMRP SDKs

This reflects your **account migration guide** at repo root [`migration-guide.md`](../../migration-guide.md) and the official **STLC docs** (`stlc-main/packages/stlc/docs/`, especially [`migration-plan.md`](../../stlc-main/packages/stlc/docs/migration-plan.md) and [`codegen.md`](../../stlc-main/packages/stlc/docs/codegen.md)).

## What the migration guide assumes

| Concept | Intended shape |
| --- | --- |
| **Config repo** | The git repo where the OpenAPI spec and `stainless/` workspace live—the **API repo** (`api/`), not each SDK checkout. |
| **Workspace** | A directory tree that contains **`workspace.json`**, **`stainless.yml`**, the spec snapshot, **`custom-code/`**, build manifests (`builds/`), etc. |
| **Discovery** | Run `stlc` **from somewhere under the config repo**; it walks up until it finds **`stainless/workspace.json`**, unless you pass `--workspace`. |
| **SDK repos** | `targets.<lang>.staging_repo` / `production_repo` describe **GitHub repos** (`open-mrp/internal-sdk`, `open-mrp/typescript-sdk`). Builds write **into clones** rooted under the workspace’s **`output_path`** (`output_path` + repo name ⇒ e.g. `core/internal-sdk` in our monorepo). |
| **Day-to-day loop** | `make openapi-stainless` (refresh specs **and** `stainless.yml` configs) → `stlc build [--push]`; CI uses one workflow against the config repo with `STLC_READ_TOKEN` / `SDK_WRITE_TOKEN`. |

**Anti-patterns** for this model:

- Copying **`stainless/`** or **OpenAPI** into **`internal-sdk`** and running codegen there—you duplicate truth, fight `git clean`, and drift from **`stlc init --from-cloud`** bundle layout.
- Vendoring **`stlc`** inside an SDK repo—install **`stlc` + `stlc-typescript`** from **`sdk-gen/*`** (see **Install stlc** below).

We have **two workspaces**. The **internal** workspace builds one TypeScript target; the **public**
workspace is **multi-target** (TypeScript + Python + Go), each target building to its own repo:

| Workspace | Target | Package | Spec | Repo (`production_repo`) | Registry |
| --- | --- | --- | --- | --- | --- |
| [`stainless/internal`](../stainless/internal/) | typescript | `@openmrp/internal-sdk` | `internal_openapi_spec.json` | [`open-mrp/internal-sdk`](https://github.com/open-mrp/internal-sdk) | GitHub Packages |
| [`stainless/public`](../stainless/public/) | typescript | `@openmrp/sdk` | `public_openapi_spec.json` | [`open-mrp/typescript-sdk`](https://github.com/open-mrp/typescript-sdk) | npmjs |
| [`stainless/public`](../stainless/public/) | python | `openmrp` | `public_openapi_spec.json` | [`open-mrp/python-sdk`](https://github.com/open-mrp/python-sdk) | PyPI (OIDC) |
| [`stainless/public`](../stainless/public/) | go | `github.com/open-mrp/openmrp-go` | `public_openapi_spec.json` | [`open-mrp/openmrp-go`](https://github.com/open-mrp/openmrp-go) | git tag → pkg.go.dev |

A single `stainless/public/stainless.yml` declares all three public targets under `targets:`; `stlc build
--targets <lang>` selects one. The Go module path is **derived** as `github.com/<go.production_repo>`.
`workspace.json` in each workspace sets `output_path` to the monorepo root (`../../../` from
`stainless/*/`), so `stlc` targets the SDK sibling directories—not `sdks/` under `api/`—without extra flags.

> **Editing `stainless.yml`:** `make stainless` regenerates only the `resources:` node
> (`tools/apidocs/stainless.go` `rewriteStainlessResources`); the `targets:`, `client_settings:`, and
> `environments:` blocks are hand-maintained and preserved. Add/adjust targets there directly.

## Install stlc

OpenMRP uses **forks under [`sdk-gen`](https://github.com/sdk-gen)** (`stlc` plus the `stlc-typescript`,
`stlc-python`, `stlc-go`, and `stlc-mcp` workers), not the upstream `stainless/*` repos. `stlc-mcp` is
the worker that generates the MCP server (`packages/mcp-server`) when a target enables
`options.mcp_server` — see [MCP server](#mcp-server) below.

From **`api/`**:

```bash
make install-stlc
# or: ./scripts/install-stlc.sh
```

The script uses `STLC_READ_TOKEN` if set, otherwise `gh auth token`. Scope the PAT to **Contents: Read**
on `sdk-gen/stlc`, `sdk-gen/stlc-typescript`, `sdk-gen/stlc-python`, `sdk-gen/stlc-go`, and
`sdk-gen/stlc-mcp`.

Manual install:

```bash
export STLC_GITHUB_ORG=sdk-gen   # default in scripts/CI
npm install -g \
  git+https://github.com/sdk-gen/stlc.git \
  git+https://github.com/sdk-gen/stlc-typescript.git \
  git+https://github.com/sdk-gen/stlc-python.git \
  git+https://github.com/sdk-gen/stlc-go.git \
  git+https://github.com/sdk-gen/stlc-mcp.git
```

> **Worker install location gotcha:** `stlc` resolves workers as siblings of itself (the directory the
> `stlc` bin's package lives in). If a Node upgrade has moved `npm prefix` away from where `stlc` is
> installed, a plain `npm install -g stlc-mcp` lands in the wrong global root and `stlc` still reports
> the plugin missing. Install into the prefix `stlc` actually uses, e.g.
> `npm install -g --prefix "$(dirname "$(dirname "$(readlink -f "$(command -v stlc)")")")" git+https://github.com/sdk-gen/stlc-mcp.git`,
> or just reinstall the whole set with `make install-stlc`.

> **Worker resolution gotcha:** `stlc` finds language workers via Node module resolution as siblings of
> itself. If you have a `stlc` from another source ahead on `PATH` (e.g. Homebrew) it may not see the
> npm-global workers, so `stlc build --targets python` fails with *"the `stlc-python` plugin is
> missing"*. Put the npm global bin first: `export PATH="$(npm config get prefix)/bin:$PATH"`.

If the forks are private, use authenticated URLs (as in `scripts/install-stlc.sh`).

Ensure the npm global bin is on your `PATH`:

```bash
export PATH="$(npm config get prefix)/bin:$PATH"
stlc version
```

## Local workflow (recommended)

1. **Install** `stlc` (above).
2. From **`api/`**:
   - Regenerate OpenAPI specs and Stainless configs when protos/services change:

     ```bash
     make openapi-stainless
     ```

   - Regenerate SDKs:

     ```bash
     make stlc-internal-sdk           # open-mrp/internal-sdk  → @openmrp/internal-sdk (TS)
     make stlc-public-typescript-sdk  # open-mrp/typescript-sdk → @openmrp/sdk          (TS)
     make stlc-public-python-sdk      # open-mrp/python-sdk     → openmrp               (PyPI)
     make stlc-public-go-sdk          # open-mrp/openmrp-go       → github.com/open-mrp/openmrp-go
     make stlc-public-sdks            # all three public targets
     make stlc-sdks                   # internal + all public targets
     ```

     `make stlc-public-sdks` is equivalent to one multi-target build:
     `stlc build --workspace stainless/public --targets typescript,python,go`.

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

## MCP server

The **public** TypeScript target also generates an **MCP server** as a sub-package
(`typescript-sdk/packages/mcp-server`). There is **no standalone `mcp` target** in stlc — it is enabled
via `targets.typescript.options.mcp_server` in [`stainless/public/stainless.yml`](../stainless/public/stainless.yml)
and built by the `stlc-mcp` worker. The server wraps `@openmrp/sdk` and is published as `@openmrp/sdk-mcp`.

Key facts about our configuration:

- **One code tool, no separate per-endpoint tools.** This stlc edition exposes a single code-execution
  tool; agents call the API by writing TypeScript against the SDK (all endpoints are reachable via
  `packages/mcp-server/src/methods.ts`). `enable_docs_tool` is **off** (it needs an SDK docs-search API
  we don't serve). The `MCP/NoToolsEnabled` diagnostic is a hard error if you disable **both** tools.
- **Code execution is local, not the Stainless sandbox.** Because we generate self-hosted (not
  `stainlessManaged`), `useLocalCodeMode` is forced on: the code tool runs in an **in-container Deno
  worker** (`@valtown/deno-http-worker`), so there is no external runtime dependency. The generated
  runtime image is `denoland/deno:alpine` with Node added.
- **`--preserve-symlinks` is required.** The public TS build loads two plugins (`stlc-typescript` +
  `stlc-mcp`). They must share one `codegen.lib.mjs`; without `NODE_OPTIONS=--preserve-symlinks` each
  symlinked plugin resolves its own copy and stlc fails with *"the `stlc-mcp` plugin is missing"*. This
  flag is wired into `make stlc-public-typescript-sdk` and the public leg of
  [`stlc-generate-reusable.yml`](../.github/workflows/stlc-generate-reusable.yml). Internal/Python/Go
  builds do not set it.

### Hosting (EKS)

stlc generates a runnable HTTP server (`mcp-server --transport=http --port=<n>`, routes `GET /health`
and `GET`/`POST /`, Streamable HTTP, stateless) **and** its own Dockerfile at
`packages/mcp-server/Dockerfile`. We host it on the existing EKS cluster:

- **ECR:** `open-mrp/mcp-server` (private [open-mrp/infra](https://github.com/open-mrp/infra): `production/terraform/ecr.tf`) —
  a dedicated repo kept **out** of `var.service_names` so the Go build matrix never tries to build it
  with the shared Go Dockerfile.
- **k8s:** private [open-mrp/infra](https://github.com/open-mrp/infra): `production/kubernetes/apps/mcp-server.yaml`
  — Deployment + NodePort Service + Ingress on **`mcp.augno.com`**, sharing the api-gateway ALB
  (`group.name: api-gateway`, so no second ALB). Each caller passes their own OpenMRP API key as a Bearer
  token (`parseClientAuthHeaders`), so no shared credential is mounted.
- **CI:** the `build-deploy-mcp` job in [`release.yml`](../.github/workflows/release.yml) runs after
  `generate-sdks` (gated on the public spec changing), checks out `open-mrp/typescript-sdk@main`, builds
  the image from the generated Dockerfile, pushes `open-mrp/mcp-server:<tag>`/`:latest`, and rolls it out
  with `kubectl`.

**Prerequisites before the first deploy:**

- An **ACM certificate covering `mcp.augno.com`** (e.g. a `*.openmrp.ai` wildcard) so the shared ALB has
  a listener cert for the host (the ingress relies on cert auto-discovery, like api-gateway).
- `make install-stlc` has been re-run so CI/agents have the `stlc-mcp` worker.

## CI

### Release (canonical)

SDK generation runs **only** from [`.github/workflows/release.yml`](../.github/workflows/release.yml) **`generate-sdks`** after **`publish-openapi-specs`** succeeds:

1. **`publish-openapi-specs`** downloads **`openapi.json`** from each bucket into **`specs/sdk-baseline/`** (pre-upload baseline), runs **`make openapi-stainless`** (specs + Stainless configs, since both are uploaded to S3), compares with [`scripts/sdk-openapi-spec-changed.sh`](../scripts/sdk-openapi-spec-changed.sh) for internal and public, then uploads **`openapi.json`** and **`stainless.yml`** (plus versioned copies) to the buckets named by the **`PUBLIC_SPEC_BUCKET`** and **`INTERNAL_SPEC_BUCKET`** Actions variables. Job outputs **`internal_spec_changed`** and **`public_spec_changed`** gate SDK generation.

2. **`generate-sdks`** calls [`stlc-generate-reusable.yml`](../.github/workflows/stlc-generate-reusable.yml) with **`openapi_specs_source: s3`** and inputs **`openapi_internal_gate`** / **`openapi_public_gate`** (from **`internal_spec_changed`** / **`public_spec_changed`** on **`publish-openapi-specs`**). The job runs a **matrix of four SDK targets** — internal TS, public TS, public Python, public Go — each gated on its spec's change flag. When a flag is **`false`**, that target's **`stlc build --push`** is **skipped**. When **`true`**, it downloads the published specs from S3 and runs **`stlc build --push --targets <lang>`** to the target's **`main`**:
   - **All targets** (`internal`, `public`, `public-python`, `public-go` — all `release: release-please`): **no changeset is added.** The conventional-commit sync message (`feat(sdk):`/`fix(sdk):`/`feat(sdk)!:` `sync with deployed API <tag>`) is what each repo's **release-please** workflow consumes to open a `release: <version>` PR → merge to publish. The public targets publish to npmjs/PyPI/Go; `internal` publishes to **GitHub Packages** (its committed `release.yml` runs release-please, then `pnpm publish`es to `npm.pkg.github.com`).

   All three public targets share the single `stainless/public` workspace and the same `public_spec_changed` gate, so a public-spec change fans out to `typescript-sdk`, `python-sdk`, and `openmrp-go` together.

[`stlc-generate.yml`](../.github/workflows/stlc-generate.yml) is **manual-only** (`workflow_dispatch`); it sets **`openapi_specs_source: generate`** and leaves change flags unset so **`stlc build --push`** always runs without an S3 pre-compare gate.

### Lockfile hygiene (pnpm bootstrap)

TypeScript codegen overwrites `pnpm-lock.yaml` using merged templates from `stlc-typescript`, which may omit fields such as `publishDirectory` even when the generated `package.json` includes `publishConfig.directory`. On GitHub Actions, `pnpm` enables `--frozen-lockfile` whenever `CI` is set, so bootstrap would fail on that mismatch. The stlc workflows run **`env -u CI -u GITHUB_ACTIONS stlc build ...`** so `pnpm` inside bootstrap does not default to `--frozen-lockfile` (GitHub sets `CI` and `GITHUB_ACTIONS`; codegen rewrites `pnpm-lock.yaml` from templates that may omit `publishDirectory` while `package.json` gains `publishConfig.directory`). After **manual** edits to an SDK’s `package.json`, still run `pnpm install` locally and commit an updated `pnpm-lock.yaml` when you are not going through `stlc build`.

| When | SDK branch | Release merge |
| --- | --- | --- |
| After **release-please** tag, **deploy** succeeds, and **OpenAPI** changed vs S3 baseline | **`main`** on each SDK repo | release-please opens **release: <version>** → merge once to publish |

All four SDK repos release via **release-please** on push to **`main`**: release automation pushes SDK codegen as a single conventional `feat(sdk)/fix(sdk): sync with deployed API <tag>` commit, and each repo's release-please workflow derives the bump from that commit and opens a `release: <version>` PR. The bump mirrors the API release: derived from the tag shape (`X.0.0` → major, `X.Y.0` → minor, otherwise patch — release-please always zeroes lower components on a bump), so a minor/major API release produces a minor/major SDK release. There is no separate `api-release` dispatch to those repos and no bot sync PR to merge first.

> **Note:** `internal-sdk` was previously on Changesets (publishing to GitHub Packages). The v0.23.2 regen stripped its Changesets tooling, so it moved to release-please like the others — but still publishes to **GitHub Packages** via a committed `release.yml`, not npmjs. See [`open-mrp/internal-sdk` `.github/workflows/release.yml`](https://github.com/open-mrp/internal-sdk/blob/main/.github/workflows/release.yml).

Production flow (keeps SDKs aligned with what is deployed):

1. Release-please creates an API version tag on `main`.
2. Terraform → build/push images → deploy to EKS.
3. **`publish-openapi-specs`** writes **`openapi.json`** (and versioned keys) to S3 once generation and comparisons finish (see numbered steps under **Release (canonical)** above).

4. **`generate-sdks`** and **`notify-consumers`** run in parallel after step 3:
   - `generate-sdks` downloads specs from S3 when needed and runs **`stlc build --push`** to **`main`** as a conventional `feat(sdk)/fix(sdk): sync ...` commit that release-please consumes (skipped per SDK when that spec matched S3 **`openapi.json`** before upload).
   - `notify-consumers` dispatches `api-release` to public-docs and openapi-spec so those repos sync from S3 (same buckets as [`fetch-openapi-spec-s3.sh`](../scripts/fetch-openapi-spec-s3.sh); pass **`stainless`** as the fourth argument to fetch **`stainless.yml`**).

**Timing:** Consumer repos sync from the deployed API and S3-published OpenAPI specs; they do not wait for SDK publishes. Downstream npm/GitHub Packages SDK versions update after the release-please **release: <version>** PR on each SDK repo is **merged** (only when the OpenAPI spec actually changed for that SDK).

When `stlc build` fails, the release job runs **Print STLC failure report** (`stlc status`, `stlc diagnostics`, `stlc show`, and the latest `builds/*.json` manifest) in the job log and Actions step summary.

Local preview before release: `make openapi-stainless` then `make stlc-internal-sdk` / `make stlc-public-sdks` (TS + Python + Go) from `api/`.

### Repository secrets (required before CI can push)

Add these secrets on **`open-mrp/api`** (Settings → Secrets and variables → Actions):

| Secret | Permissions |
| --- | --- |
| **`STLC_READ_TOKEN`** | Fine-grained PAT, **Contents: Read** on `sdk-gen/stlc`, `sdk-gen/stlc-typescript`, `sdk-gen/stlc-python`, `sdk-gen/stlc-go`, and `sdk-gen/stlc-mcp` |
| **`SDK_WRITE_TOKEN`** | Fine-grained PAT, **Contents: Write** on `open-mrp/internal-sdk`, `open-mrp/typescript-sdk`, `open-mrp/python-sdk`, and `open-mrp/openmrp-go` (push to **`main`**). Add **Pull requests: Write** only if you use manual `stlc-generate` with `open_pr: true`. |

Authorize both tokens for SSO if your org requires it.

The API release workflow regenerates SDKs and dispatches `api-release` to spec consumers in parallel after deploy and OpenAPI publish. SDK repos (`internal-sdk`, `typescript-sdk`, `python-sdk`, `openmrp-go`) receive direct pushes to **`main`** when their OpenAPI spec changed.

### SDK release checklist (`internal-sdk`, `typescript-sdk`)

When the API release changed that SDK’s OpenAPI spec, automation pushes **`fix(sdk)`/`feat(sdk)`/`feat(sdk)!`** **`: sync with deployed API <tag>`** (prefix matches the API bump level) to **`main`**. Then:

1. **`release.yml` on `main`** — release-please opens **release: <version>** (or publishes when a release PR is merged).
2. **Merge the release PR** when opened — that completes the publish: GitHub Packages for `@openmrp/internal-sdk`, npmjs for `@openmrp/sdk`.

If the spec was unchanged for that SDK, no push runs and no new SDK version is cut.

### SDK release model (`internal-sdk`, `typescript-sdk`, `python-sdk`, `openmrp-go`)

All SDK repos use **release-please**, not Changesets. `stlc` generates the per-repo config
(`release-please-config.json`, `.release-please-manifest.json`, the version `extra-files`, and — for
Python — `bin/publish-pypi`, `publish-pypi.yml`, `release-doctor.yml`). It does **not** generate a
workflow that opens the release PR or publishes on release (Stainless's hosted backend normally does
that). In the self-hosted model each repo therefore needs **one committed `release.yml`** (e.g.
`release-please.yml` on `typescript-sdk`, `release.yml` on `internal-sdk`):

- **`open-mrp/python-sdk`** — on push to `main`: run `googleapis/release-please-action@v5` (opens
  `release: <version>` PR). On `release_created`: `uv build` + `uv publish` via **PyPI Trusted
  Publishing** (job needs `permissions: id-token: write`; `bin/publish-pypi` already does this — it
  uses OIDC when `PYPI_TOKEN` is unset). One-time on PyPI: register a **Trusted Publisher** for project
  `openmrp` → repo `open-mrp/python-sdk`, workflow filename = that `release.yml`.
- **`open-mrp/openmrp-go`** — on push to `main`: run `googleapis/release-please-action@v5` only. The git tag
  release-please creates **is** the publish; `pkg.go.dev` indexes the public repo. No registry secret.
- **`open-mrp/internal-sdk`** — on push to `main`: run `googleapis/release-please-action@v4` (opens
  `release: <version>` PR). On `release_created`: `pnpm build` + `pnpm publish` from `dist/` to
  **GitHub Packages** (`npm.pkg.github.com`), not npmjs — so do **not** wire up the generated
  `bin/publish-npm` (it targets `registry.npmjs.org`). Publish runs in the same job as the
  release-please step, so the default `GITHUB_TOKEN` suffices (no PAT / cross-workflow trigger).

Two one-time config notes per repo:
- `stlc` defaults `release-please-config.json` to `"versioning": "prerelease"` / `"prerelease": true`
  (alpha versions). For a stable public SDK, set `"prerelease": false` and remove the prerelease
  versioning once you cut `v1`-style releases.
- `release-doctor.yml` references a **`RELEASE_PLEASE_TOKEN`** secret; provide it (a PAT or the
  app token used by the release workflow) or remove that workflow.

The Go module path must equal the repo URL (`github.com/open-mrp/openmrp-go`), the repo must be **public**,
and `go.production_repo` in `stainless/public/stainless.yml` must match it — this is derived, not a
separate field. The legacy Stainless-cloud stub `open-mrp/go-sdk` (module `github.com/stainless-sdks/...`)
is **not** this repo; archive it to avoid confusion.

### Publishing packages

`stlc build --push` updates SDK **git** repos only; the actual npm/GitHub Packages/PyPI/Go publish runs in each SDK repo via its committed **`release.yml`**/`release-please.yml`. Local **`make stlc-*`** runs push a plain build commit — if you need a version bump outside the API release pipeline, push a conventional `feat(sdk)/fix(sdk):` commit so release-please opens a `release: <version>` PR.

### Recovering missed publishes (before automation landed)

If **`main`** contains API sync commits but no `release: <version>` PR was opened (historical drift, e.g. a missing `release.yml`), restore/repair the repo's release workflow and push any conventional commit to **`main`** — release-please re-scans history and opens a `release: <version>` PR covering the accumulated `feat(sdk)/fix(sdk):` syncs.

## Parity check

Before cutover ([migrate-validate](../../stlc-main/packages/stlc/docs/migrate-validate.md)): diff SaaS-published SDK trees against `stlc build` output until behavior-bearing drift is gone.
