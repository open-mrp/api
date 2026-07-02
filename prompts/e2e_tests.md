Write a workflow for achieving thorough end-to-end test coverage of every public API endpoint and every agent-facing endpoint.

## Goal

Every public endpoint and every agent endpoint must have e2e tests that exercise the real stack — not just happy-path CRUD, but every meaningful request-body field, query parameter, include, validation failure, and authorization edge case. Tests exist to expose production defects; when a test reveals a bug, fix the underlying service/repository/gRPC layer until the suite passes. Do not weaken assertions, skip 5xx responses, or paper over failures.

## Scope

In scope — add or extend tests in:

* `tests/e2e/api/` — all `//go:build e2e` test files
* `tests/e2e/api/seed.go` — seed entity IDs and path-parameter mappings needed by new tests
* Production code in `services/` — **only when a failing e2e test exposes a real bug** (repository SQL, service logic, gRPC handlers, gateway mapping)

Endpoint scope:

* **Public endpoints** — every operation in the public OpenAPI spec (`LoadPublicSpec()` / the generated public spec consumed by `tests/e2e/api/spec_test.go`). These are the customer-facing API surface.
* **Agent endpoints** — every route under `/v1/ai/` (agent definitions, agent runs, agent memories, agent tools, etc.) plus any other endpoint group that lives in `services/api-gateway/endpoints/` under agent-related packages. Treat these with the same coverage bar as public CRUD resources.
* **Auth endpoints** - every auth related endpoint

Out of scope — do not duplicate or replace cross-cutting suite tests that already sweep many endpoints generically:

* `pagination_errors_test.go`, `pagination_edge_cases_test.go`, `sorting_filtering_test.go`
* `idempotency_test.go`, `error_response_test.go`, `input_validation_test.go`, `schema_validation_test.go`
* `response_contract_test.go`, `response_validator_test.go`, `include_errors_test.go`
* `rate_limiting_test.go`, `headers_test.go`, `sql_injection_test.go`, `search_sanitization_test.go`

Resource-specific tests should complement these — not re-test the same generic behavior unless the endpoint has unique semantics.

## Source of truth

Derive what to test from the implementation, not from guesswork:

| What | Where |
|------|-------|
| Endpoint routes, methods, operation IDs, query params | OpenAPI spec (parsed by `tests/e2e/api/spec_test.go`; harness loads list/update/create/put endpoints in `harness_test.go`) |
| Public vs preview surface | Public OpenAPI spec + gateway endpoint `Public: true` flags in `services/api-gateway/endpoints/**/endpoint_*.go` |
| Response field list (every field must be asserted) | Resource struct in `services/api-gateway/pkg/resource/` |
| Request fields (required / optional / clearable) | Create/update request structs in `services/api-gateway/endpoints/` — follow `docs/patterns/nullable-field-patterns.md` |
| Expandable subresources and valid `?include=` keys | `expandable:"true"` tags on resource fields + `IncludeConfig` on endpoint definitions |
| Test conventions and required categories | `docs/patterns/e2e-test-patterns.md` (authoritative checklist) |
| Seed IDs for GET/list/foreign-key references | `tests/e2e/api/seed.go` |
| E2E rules (never skip 5xx, no bandaids) | `AGENTS.md` → "End-to-end (e2e) tests" |

## Step 0 — Split the work into individual endpoint task files

Enumerate every in-scope endpoint (or coherent endpoint group — e.g. one CRUD resource, or one action endpoint family) and create a task file for each. Group related routes together when they share one resource struct and one test file (e.g. all `/v1/inventory/items` operations → one task).

Each task file must contain:

* **Endpoint paths and HTTP methods** (list, get, create, update, delete, actions)
* **Operation IDs** from the OpenAPI spec
* **Resource and request struct paths** in the gateway
* **Existing test files** that already cover this endpoint (grep `tests/e2e/api/` for the path prefix)
* **Coverage gap summary** — which required test categories from `e2e-test-patterns.md` are missing or incomplete
* **Request-body matrix** — for each create/update/action body field:
  * required vs optional vs clearable (`*field.Clearable[T]`)
  * valid value(s) to send (use seed IDs for foreign keys)
  * invalid values to send (missing, empty string, explicit `null` where rejected, out-of-range enum, wrong type)
  * expected status code for each case
* **Query-parameter matrix** — for each supported query param (`include`, `limit`, `cursor`, `q`, filters, sort):
  * valid values and expected response shape
  * invalid values and expected error
* **Failure-mode checklist** — 400/422 validation, 404 not found, 409 conflict, 401/403 auth (where testable with existing harness credentials), idempotency replay
* **Seed data needs** — new constants in `seed.go`, or new seeded entities required
* **Review objective** — what "done" looks like for this endpoint
* **Remediation criteria** — explicit pass/fail conditions (see below)

Pass these task files into the workflow.

## Step 1 — Implement tests for each endpoint task

For each endpoint task:

### 1a. Gap analysis

* Read the resource struct, request structs, and OpenAPI operation definitions.
* Read existing tests for this endpoint. Mark each required category from `e2e-test-patterns.md` as covered, partial, or missing:
  1. Basic CRUD lifecycle
  2. Create and update all fields (assert **every** response struct field)
  3. Omitted fields (create defaults, missing required → 400/422, update preservation)
  4. Response shape (ID prefix, timestamps, `object` field)
  5. List (basic, pagination, search, no results)
  6. Expandable fields (null without `?include`, populated with `?include`)
  7. Idempotency (create and update where applicable)
  8. Validation (per-field invalid inputs)
* For non-CRUD endpoints (actions, price-quote, login, etc.), derive an equivalent matrix: required inputs, success response fields, error cases, idempotency if POST/PATCH.

### 1b. Add seed data

* Add IDs to `seed.go` when tests need stable GET targets or foreign-key references.
* Extend e2e seed fixtures (DB seeds / docker seed scripts) when no existing seeded entity satisfies the test — coordinate additions so parallel tests do not mutate shared seed rows destructively.

### 1c. Write tests

* Follow `docs/patterns/e2e-test-patterns.md` exactly: file naming (`crud_{resource}_test.go` or descriptive name for behavioral endpoints), section comments, `t.Parallel()` on top-level tests, `defer apiClient.Delete(...)` cleanup, `newIdempotencyKey()` on every POST/PATCH.
* Assert every response field in `CreateAndUpdateAllFields` — use the field assertion table in the patterns doc (bools via `jsonField` string comparison, expandable fields null unless included).
* Cover every request-body field and query param from the matrices in the task file.
* Use subtests (`t.Run`) for validation cases; do not call `t.Parallel()` inside subtests that share a created resource.
* When the human has the e2e stack running, run targeted tests after each resource:

  ```bash
  go test -tags=e2e -run 'TestMyResource_' ./tests/e2e/api/ -timeout 120s
  ```

### 1d. Fix production bugs exposed by tests

When a test hits a 5xx or wrong behavior:

* **Fix the root cause** in the owning service layer (repository SQL, domain logic, gRPC handler, gateway presenter).
* **Never** add `t.Skip`, `skipIfBackend500`, status-code guards that swallow 5xx, retries that hide errors, or TODO comments leaving the failure latent.
* **Never** relax assertions to match broken behavior.
* Re-run the targeted test until it passes against a healthy stack.

Editing rules during this step:

* Do not use any git commands.
* Do not run the full e2e stack lifecycle (`make e2e`, `make e2e-up`) — the human keeps the stack running; use `make test-e2e` or targeted `go test -tags=e2e` only.
* Avoid broad test commands (`make test`, entire `./tests/e2e/...` without `-run`) that could interfere with another agent on the same branch.
* Prefer small, focused test runs per endpoint task.

## Step 2 — Adversarial review (2 agents per endpoint task)

Use 2 adversarial review agents to refute the test additions independently. Each should inspect the new tests and any production fixes and attempt to find flaws, including:

* Missing response fields — compare assertions against the full resource struct
* Missing request-body or query-param paths from the task matrix
* Missing failure modes (validation, not found, conflict, clearable null on PATCH)
* Weak assertions (`assert.NotNil` without checking values, no `object`/`id` checks on subresources)
* Expandable fields asserted incorrectly (fabricated stubs, non-null without `?include`, null when included)
* Violations of `e2e-test-patterns.md` (naming, parallelism, cleanup, idempotency keys)
* Duplication of cross-cutting generic tests without endpoint-specific value
* **Skipped or masked 5xx errors** — any sign of bandaid fixes in tests or production code
* Production fixes that belong in a different layer (gateway bandaid for a core-service bug)
* Seed data that breaks parallel tests or lacks cleanup
* Tests that pass only because they assert too little

The adversarial agents must not use any git commands, stack lifecycle commands, or broad test commands.

## Step 3 — Reconcile, run the suite, and open a PR

After all endpoint tasks are complete:

* Combine test additions and production fixes across tasks.
* Resolve conflicts: one test file per resource; deduplicate overlapping validation cases; prefer the more specific assertion when two tests cover the same field.
* Re-check that every in-scope endpoint has a corresponding task marked done and no required category left as "missing".
* Run the full e2e suite against the running stack:

  ```bash
  make test-e2e
  ```

* Fix any failures — production bugs first, then test corrections if expectations were genuinely wrong.
* Once the suite passes, commit the changes.
* Create a PR with a summary that includes:
  * Which endpoints/resources received new or expanded coverage
  * Which test categories were added per resource
  * Any seed data additions
  * Production bugs found and fixed (with layer: repository / service / gRPC / gateway)
  * Targeted and full-suite test commands run
  * Endpoints intentionally deferred (with reason) or blocked on missing seed/fixtures

## Remediation criteria (per endpoint task)

A task is **done** only when all of the following hold:

1. Every required test category from `e2e-test-patterns.md` applicable to this endpoint is implemented (or explicitly N/A with justification — e.g. read-only resource skips delete idempotency).
2. Every field on the response resource struct has at least one assertion somewhere across the test file(s).
3. Every create/update/action request field has at least one valid and one invalid test case (where validation exists).
4. Every documented query param (`include`, pagination, search, filters) has valid and invalid coverage.
5. Expandable fields are `null` without include and truthfully populated with include — never fabricated.
6. No 5xx responses in passing tests; no skipped failures; no weakened assertions.
7. Targeted `go test -tags=e2e -run ...` passes for the resource.
