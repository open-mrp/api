Write a workflow for reviewing and updating the public documentation so it stays current with how the application actually works.

## Goal

The hand-authored guides in `public-docs/src/docs/` drift out of sync with the real product over time: field names change, statuses/enums change, features get added/removed/renamed, and described workflows no longer match the implementation. The job is to review every non-API-reference doc against the actual codebase and correct anything that is inaccurate or stale.

## Scope

In scope — edit these:

* `public-docs/src/docs/**/*.mdx` — the hand-authored guides. The main groupings are:
  * `get-started/` — onboarding, account activation, first API request, release phases
  * `workflows/` — how the product works: `build-products` (items, inventory, BOM, production, purchasing, scanning), `generate-sales` (customers, sales orders, pricing, customer portal), `collect-payments` (invoicing, collecting, AR), `ship-products` (picking, packing, shipping), `dashboards` (analytics), `manage-account` (team, roles, billing, integrations, security, sandboxes)
  * `developer-resources/api/**` — conceptual API guides that are **hand-authored MDX** (authentication, error handling, idempotency, includes, pagination, rate limiting, request IDs, versioning, MCP server, request logs, SDKs). These ARE in scope even though they live under an `api/` folder — they are prose, not the generated reference.

Out of scope — never touch these:

* The generated OpenAPI **API reference**. This is produced from the spec, not hand-written. Do not edit:
  * `public-docs/src/static/*.generated.ts` (apiEndpoints, apiNav, apiSnippets, apiVersion*, docPaths, navData, pagePreview, routeMap, etc.)
  * `public-docs/src/app/(docs)/api-reference/**`
  * `public-docs/specs/**`, the OpenAPI spec, or any Stainless config
* Any other generated file. If a value is wrong in the API reference, the fix belongs in the API codebase / spec, not here — note it but do not change it.

## Source of truth

Verify every factual claim against the real implementation, not against other docs:

* **API behavior** → the Go services in `api/services/`: `api-gateway`, `core-service`, `auth-service`, `billing-service`, `agent-service`, `notification-service`, `platform-service`. Look at gateway handlers, request/response shapes, enums/statuses, validation, and proto definitions in `api/proto/` and `api/services/*/internal/`.
* **Product UX and workflows** → the dashboard frontend in `dashboard/apps/frontend/src/app/(user)/dashboard/**`. Each feature has a route directory (e.g. `customers/`, `sales-orders/`, `inventory/`, `batch-flow/`, `purchase-orders/`, `deliveries/`, `agents/`). The components, forms, filters, and detail pages there show what users actually see and can do.
* When the API and the frontend disagree with a doc, the running behavior wins — but call out the discrepancy in the task notes.

## Step 0 — Split the work into individual doc-review tasks

Enumerate the in-scope MDX files and group them into review tasks by feature subsection (one task per coherent group of related pages, e.g. "build-products / inventory", "generate-sales / sales-order", "developer-resources / make-requests"). For each task, create a task file containing:

* The source doc file path(s)
* A concise summary of the factual/behavioral claims the doc makes (field names, statuses, enums, available actions, described flows, prerequisites)
* The specific source-of-truth code areas to check it against (which API service/handlers/protos and which frontend route directory)
* The review objective
* The expected remediation criteria

Pass these task files into the workflow.

## Step 1 — Review each doc against the codebase

For each doc-review task:

* Read the doc and extract every verifiable claim.
* Open the corresponding source-of-truth code and confirm each claim. Specifically check for:
  * Renamed/removed/added fields, statuses, and enum values
  * Workflows or steps that no longer match the implementation (order of operations, prerequisites, what triggers what)
  * Features described that no longer exist, or new behavior that exists but isn't documented
  * Stale terminology (e.g. old resource names — cross-check against current conventions like the `owner` model)
  * Incorrect API details in the conceptual guides (auth scheme, error shapes, pagination params, versioning header, idempotency semantics, rate-limit behavior, include syntax)
  * Broken internal links / invalid `<InternalLink pathKey="..." />` pathKeys and dead route references
* Fix what is wrong. Update the prose to match actual behavior.

Editing rules:

* Preserve the existing voice, structure, frontmatter, and formatting. Change facts, not style. Keep marketing/positioning language intact unless it is factually false.
* Do not invent features or document aspirational behavior. If the code doesn't do it, the docs shouldn't claim it.
* Match the surrounding MDX conventions: YAML frontmatter (`title`, `subtitle`, `route`, `nav.section/subsection/order`), `<InternalLink pathKey="..." />` for internal links, and the existing custom MDX components.
* If a page documents a feature that no longer exists, prefer flagging it in the task notes over deleting the page outright — removing a page affects navigation and routing and needs a judgment call at reconcile time.
* Do not use any git commands.
* Do not run the dev server (`bun run dev`) — it is started manually by a human only.
* Do not run broad build or test commands at this stage.
* Avoid actions that could interfere with another agent working in the same branch.

## Step 2 — Adversarial review (2 agents per task)

Use 2 adversarial review agents to refute each doc fix independently. Each should inspect the changes against the codebase and try to find flaws, including:

* Claims that were "corrected" but are actually still wrong (the new prose also doesn't match the code)
* Missed inaccuracies elsewhere in the same doc
* Over-correction — rewriting accurate prose, or changing intended product positioning into something narrower/incorrect
* Under-correction — leaving stale field names, statuses, or flows in place
* Invented behavior not backed by the code
* Broken or invalid `pathKey` links and route references introduced or left behind
* Inconsistencies with neighboring docs or with current cross-package conventions

The adversarial agents must not use any git, build, or broad-test commands.

## Step 3 — Reconcile, validate, and open a PR

After all doc-review tasks are complete:

* Combine the fixes across all tasks.
* Resolve conflicts: prefer the description that matches the running code; when two docs describe the same behavior, make them consistent with each other and with the implementation.
* Make the page-deletion / nav-restructure judgment calls flagged during review.
* Re-check internal links and pathKeys across the changed set.
* Regenerate navigation/slugs and validate the build from `public-docs/`:
  * `bun run format` (or `format:check`) and `bun run lint`
  * `bun run build:docs` (regenerates nav/slugs; in the monorepo it sources specs from `../api`)
  * `bun run test`
  * Do **not** run `bun run dev`.
* Fix any failures (build errors, broken slugs/links, lint).
* Once the build and tests pass, commit the changes.
* Create a PR with a summary that includes:
  * Which doc sections were reviewed
  * What categories of drift were found (renamed fields, changed statuses, removed/added features, broken links, stale workflows)
  * What changes were made
  * What build/test commands were run
  * Any pages or behaviors that were ambiguous, flagged for a human decision (e.g. proposed deletions), or where the code and existing docs disagreed
