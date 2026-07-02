Write a workflow for reviewing and improving the Go doc comments ("docstrings") on our API resource structs, request structs, and endpoint definitions so they read clearly to a human consuming our public API docs.

## Goal

Every doc comment on a public API resource field, request field, endpoint struct, and endpoint title is rendered verbatim into our public documentation (the generated OpenAPI reference). These comments are the end user's description of what each field and endpoint does. Over time they drift toward terse, developer-shorthand restatements of the field name ("Display name." over a field called `Name`) that add nothing, or they fail to explain non-obvious business logic, enum semantics, defaults, and interactions that a consumer cannot infer from the type alone.

The job is to review these comments and improve them so they genuinely explain the resource/endpoint to an external developer who does not know our internals.

## Key facts

* **Doc comments are Markdown.** They are rendered as Markdown in the public docs. You may use inline code (`` `value` ``), bold, and Markdown bullet lists. Multi-line comments are valid: continue the comment with `//` lines, using a blank `//` line to separate paragraphs/lists from the lead sentence (Go doc convention).
* **Required comment structure — lead sentence, blank `//`, then elaboration.** Every comment that has more than a one-liner MUST be structured as:
  1. **Lead line:** a single, succinct description sentence — and nothing else. This is the whole comment for simple fields, and the summary line for complex ones. Describe the field in plain terms (e.g. `// The account group this access record belongs to.`), not a bare restatement of the field name (`// Account group.`).
  2. **A blank `//` separator line.**
  3. **The elaboration:** the extra info — additional prose sentences, a Markdown bullet list of enum values, default/required notes, side effects, etc.

  Do **not** run the elaboration onto the lead line or let it spill into the lead paragraph as a soft-wrapped continuation. The first sentence stands alone on its own line as the summary; everything beyond that first sentence goes below a blank `//` line.
* **Never soft-wrap a comment paragraph across multiple `//` lines.** Each paragraph — the lead sentence and each elaboration paragraph — is a single `//` line, no matter how long. Do not break it at ~80 columns. The only things that occupy their own `//` lines are: the lead line, the blank `//` separator(s), and each individual bullet-list item (one bullet per line). Example of what NOT to do:

  ```
  // BAD — one sentence soft-wrapped onto a second // line:
  // System-owned types are platform-provided defaults shared across all
  // accounts; account-owned types are custom to one account.

  // GOOD — the whole paragraph on one line:
  // System-owned types are platform-provided defaults shared across all accounts; account-owned types are custom to one account.
  ```

  ```
  // BAD — elaboration wrapped into the lead paragraph, no blank // separator:
  // The account group this access record belongs to. There is at most one
  // access record per account group, so this also identifies the record.

  // GOOD — succinct lead sentence, blank // separator, then elaboration:
  // The account group this access record belongs to.
  //
  // There is at most one access record per account group, so this also identifies the record.
  ```

  (A long single sentence that simply soft-wraps across two `//` lines with no second sentence is fine and needs no separator — the rule is about separating the *lead summary sentence* from any *additional* explanation.)
* **The pattern to emulate** is the "secondary comment" block, e.g. `services/api-gateway/pkg/resource/account_group_resource.go` — `CommissionPolicy`, `FreightPolicy`, and `Type` each have a one-line lead summary, then a blank `//` line, then a Markdown bullet list explaining each enum value in plain business terms. That is the gold standard for an enum whose values carry non-obvious business meaning.
* **Never restate schema facts the docs already render.** The public docs render type, required/optional status, nullability, allowed enum values, default values, and expandability directly from the OpenAPI spec next to each field. Do not add annotations like "Optional.", "Required.", "`null` when not set.", "Nullable.", statements of the field's type, "Expandable via `include=...`", "Defaults to `x`." as a bare schema fact, or an exhaustive list of the enum values for its own sake. Remove such annotations where they already exist. Enum bullet lists are only for values whose business meaning is not obvious from the value name (self-explanatory values like `active`/`inactive` get none). Mention a default only when its behavioral consequence is non-obvious, phrased as behavior, not as a schema fact.

## Scope

In scope — edit doc comments only:

* `services/api-gateway/pkg/resource/**/*.go` — resource struct field comments and the struct-level comment on each resource.
* `services/api-gateway/endpoints/**/endpoint_*.go` — request struct field comments, the request struct-level comment, the endpoint struct-level comment (the comment directly above `type XEndpoint struct{}`), and the `Title:` string on the endpoint.
* `services/api-gateway/pkg/request/**/*.go` — shared request input struct field comments (e.g. `AddressInput`, `QuantityInput`).

Out of scope — do not change:

* Any code, struct tags, `json:`/`query:`/`validate:`/`default:` tags, field names, types, sample values, `SchemaExample` methods, or any non-comment line.
* Endpoint wiring (`Method`, `Route`, `IncludeConfig`, `ServiceHandler`, etc.) — only the `Title` string and the struct doc comment are in scope there.
* Generated files and the OpenAPI spec. If a comment is wrong because the underlying behavior changed, fix the comment to match real behavior; do not touch generated artifacts.
* Files covered by the analytics/bulk-operations pending refactor (`analytics_*`, `bulk_operation_*`) — skip these.

## Source of truth — understand the business logic before rewriting

A good comment requires understanding what the endpoint/field actually does. Do not paraphrase the field name. Verify behavior against the implementation:

* **Enum/constant semantics** → `shared/constants/` for the type's values and any documented meaning.
* **Validation, defaults, required-ness** → the `validate:`, `default:`, and `field.Optional[...]` / pointer / value type on the field itself. Use these to keep behavioral claims accurate — but never restate them as schema annotations (see Key facts).
* **Endpoint behavior** → trace the `ServiceHandler` (e.g. `svc.(CustomerSvc).CreateCustomer`) to the actual service implementation in the relevant service (`core-service`, `auth-service`, `billing-service`, etc.) to understand what the endpoint really does, what it auto-generates, what side effects it has, and what the title/summary should convey.
* **Cross-field interactions** → if a field's effect depends on another field, or can be overridden elsewhere (as the account-group freight/commission comments note), say so.

## Step 0 — Split the work into individual review tasks

Enumerate the in-scope files and group them into review tasks by resource/domain (one task per coherent group, e.g. "customers (resource + all customer endpoints)", "account groups", "shared request inputs"). Keep a resource and its endpoints together in one task where practical so the reviewer sees the full picture. For each task, create a task file containing:

* The source file path(s)
* The resource/endpoint(s) covered
* The source-of-truth code areas to check (which constants, which service handler implementation)
* The review objective and remediation criteria below

Pass these task files into the workflow.

## Step 1 — Review and improve each task's comments

For each field, struct, endpoint, and title in the task:

1. Read the current comment and the field's type, tags, and surrounding context.
2. Determine, from the source of truth, what a consumer actually needs to know.
3. Decide whether the comment is adequate or needs improvement. Improve it when any of these apply:
   * It merely restates the field name and the field's meaning is not fully obvious from name + type (e.g. what does this status actually control? what happens at each enum value?).
   * The field is an enum/constant whose values' business meaning is not obvious from the value names. Add a secondary Markdown bullet list explaining those values (the account-group pattern); add nothing for self-explanatory values.
   * There is a non-obvious auto-generation behavior, a side effect, or a behavioral consequence of a default the consumer should know.
   * The comment contains a schema-fact annotation (optional/required, nullable, type, expandable, bare enum-value list, bare default) that must be removed.
   * There is a cross-field interaction or override behavior worth calling out.
   * The comment is stale or factually wrong relative to the implementation.
   * The endpoint struct comment or `Title` does not clearly and accurately describe what the endpoint does for an end user.
4. Rewrite to be clear, accurate, human-readable Markdown.

Editing rules:

* **Do not add secondary comments everywhere.** Many simple fields (`id`, `created_at`, a plainly-named string) are fine with a short one-liner — leave them. Add the extra explanation only where it genuinely helps a human understand something non-obvious. Restraint is part of the quality bar.
* **Enforce the required comment structure (see Key facts):** a single succinct lead sentence, then a blank `//` line, then any elaboration. Never run extra explanation onto the lead line or wrap it into the lead paragraph without the blank `//` separator.
* Use Markdown bullet lists for enumerating values/options; otherwise prefer prose. Don't force bullets where a sentence reads better.
* Match the existing voice and formatting. Change comments only — never code, tags, names, or sample values.
* End comment sentences with periods, consistent with the existing style.
* Do not use any git commands.
* Do not use build commands.
* Do not use broad test commands.
* Avoid actions that could interfere with another Claude working in the same branch.

## Step 2 — Adversarial review

Use 2 adversarial review agents to independently inspect the changes and try to find flaws, including:

* Comments that are now inaccurate relative to the actual implementation/business logic.
* Secondary detail added where it wasn't needed (noise), or needed elaboration still missing.
* Enum value explanations that are wrong, incomplete, or describe values that don't exist in `shared/constants`.
* Stated defaults / required-ness that don't match the field's tags and type.
* Markdown that would render badly in the public docs (broken lists, missing blank `//` separator, stray backticks).
* Any non-comment line that was changed (must be none).
* Voice/style drift from neighboring comments.

The adversarial agents must not use any git commands, build commands, or broad test commands.

## Step 3 — Reconcile and finish

* Combine the fixes across all tasks and apply adversarial feedback.
* Re-check that only comment lines changed.
* Run the build.
* Run the relevant tests.
* Fix any failures.
* Once the build and relevant tests pass, commit the changes.
* Create a PR with a summary that includes:
  * Which resources/endpoints were reviewed
  * What categories of improvements were made (enum explanations added, stale comments corrected, titles clarified, etc.)
  * Which fields were intentionally left as-is and why (restraint applied)
  * What build/test commands were run
  * Any places where the intended behavior was ambiguous and required a judgment call
