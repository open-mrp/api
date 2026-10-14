# API Versioning Patterns

This is the canonical reference for how we version the public API: what a version is, what counts as a breaking change, how the gateway serves multiple versions from a single latest-shaped backend, and how versions are introduced, maintained, and fully retired.

**The contract: a pinned consumer never breaks.** A client that pins `OpenMRP-Version` keeps getting byte-compatible request and response shapes for as long as that version is supported — no matter how the latest API evolves underneath it. The only way a pinned client ever sees a behavior change is when its version is deliberately deprecated and removed, which is a planned, communicated event — never a side effect of shipping a feature.

> **Current pair.** `1.0.forge` is `Latest`; `1.0.forge-preview.9` is the only other supported version. Its callers are served by the `1.0.forge` → `1.0.forge-preview.9` transformers in `services/api-gateway/internal/versiontransforms/`.

---

## 1. The version scheme

The mechanism is `github.com/open-mrp/apikit/version`. OpenMRP's versions live in `shared/apiversion/apiversion.go`, which registers them with `version.RegisterVersions` at init.

| Format | Example | Meaning |
|--------|---------|---------|
| `<minor>.<patch>.<codename>` | `1.0.forge` | Stable release |
| `<minor>.<patch>.<codename>-preview.<n>` | `1.0.forge-preview.9` | Preview iteration (sorts before its stable release) |

Ordering is `minor → patch → codename → preview`, with previews before stables (`APIVersion.Before` implements this). The registered versions define the universe:

```go
V1_0_Forge          = version.MustNew("1.0.forge")
V1_0_Forge_Preview9 = version.MustNew("1.0.forge-preview.9") // DeprecatedAt set

func init() {
	version.SetHeader("OpenMRP-Version")
	version.RegisterVersions(V1_0_Forge, V1_0_Forge_Preview9)
}
```

`version.Latest()` returns the newest registered version, the shape the backend natively produces and consumes. `version.Supported()` returns every registered version, newest first. Removal from the registered set IS deprecation enforcement: requests for an unregistered version are rejected with a 400 before reaching any endpoint.

Every request must carry a valid `OpenMRP-Version` header. `middleware.Version` (`apikit/middleware/version.go`) rejects missing or unsupported versions with a 400, parses the header against `version.Supported()`, stores the result in the request context (`appctx.WithAPIVersion`), and echoes the version back in the response header.

---

## 2. One backend shape, many wire shapes

The backend — services, repositories, protos, `apiresource` structs, the OpenAPI spec — only ever speaks **`Latest`**. We do not fork resources, endpoints, or handlers per version. Older versions exist purely as **transformations at the gateway edge**:

```
            request (old shape)                       response (old shape)
   client ─────────────────────▶ ┌─────────────┐ ◀──────────────────────── client
                                 │   Execute    │
        TransformRequest chain   │ (api_endpoint│   Transform chain
        old → ... → Latest       │     .go)     │   Latest → ... → old
                                 └──────┬───────┘
                                        │  Latest shape only
                                        ▼
                            handlers / services / protos
```

Both directions run inside `APIEndpoint.Execute` (`apikit/endpoint/api_endpoint.go`) for any endpoint that declares an `ObjectType`, and for every endpoint when a transformer lists `version.AnyObjectType`:

- **Request upgrade.** Before the JSON body is decoded, `version.TransformRequest(requestVersion, Latest, objectType, body)` rewrites the old-shape body into the latest shape. Validation, decoding, and the handler only ever see latest-shaped input.
- **Response downgrade.** After the handler returns and includes are resolved, the response is marshalled to a `map[string]any` and `version.Transform(Latest, requestVersion, objectType, payload)` rewrites it into the requested version's shape before `RespondWithJSON`.

There is intentionally **no** response-transform middleware. An earlier `VersionTransformMiddleware` existed but was never wired and has been deleted; transformation lives in `Execute` so it runs after include resolution (the data it needs is present) and inside the same code path that owns request logging and error handling. Do not reintroduce a middleware variant.

**Endpoints must declare `ObjectType`.** An endpoint without `ObjectType` is invisible to resource-specific transformers — acceptable only for endpoints whose payloads have never changed across supported versions. When you change a resource's shape, audit that every endpoint returning it (including action endpoints that return the resource) declares the `ObjectType`.

---

## 3. Transformers

A transformer bridges exactly **two adjacent versions** and implements `version.Transformer` (`apikit/version/transformer.go`):

```go
type Transformer interface {
	FromVersion() APIVersion       // the newer version
	ToVersion() APIVersion         // the older version
	ObjectTypes() []object.Type    // payload roots this transformer inspects
	// Downgrade a response payload (newer → older shape).
	Transform(objectType object.Type, data map[string]any) map[string]any
	// Upgrade a request payload (older → newer shape).
	TransformRequest(objectType object.Type, data map[string]any) map[string]any
}
```

Listing `version.AnyObjectType` in `ObjectTypes()` applies a transformer to every response and request, for a change made to every endpoint at once (the list envelope, the deleted stub). Implement the optional `version.ErrorDowngrader` (`TransformError(status, body)`) when a version changed error bodies or statuses.

The registry (`version.DefaultRegistry`) **chains** transformers: a client three versions behind gets each adjacent transformer applied in sequence — newest-to-oldest for responses, oldest-to-newest for requests. You never write a "v5 → v2" transformer; you write v5→v4, v4→v3, v3→v2 and the registry composes them. This keeps each transformer small and lets versions be retired from the old end of the chain without touching the rest.

### Where transformers live

`services/api-gateway/internal/versiontransforms/`, one file per transformer, named `transform_<resource>_forge_to_forge_preview_9.go`. Each file registers itself:

```go
func init() {
	version.RegisterTransformer(deletedForgeToPreview9{})
}
```

The package is blank-imported in `services/api-gateway/cmd/run.go` (same pattern as `resourceregistry`), so registration happens at startup. Transformers are gateway concerns — they reshape wire JSON — which is why they do not live in `shared/apiversion` (that package only declares the versions).

### Rules for writing a transformer

1. **Pure data reshaping, never fabrication.** A downgrade may move, rename, restructure, or drop fields that exist in the newer payload. It must never invent values. If the older shape requires data the newer payload doesn't naturally carry, *force* the data into the payload (see Forced includes below) — do not synthesize plausible-looking defaults. This is the same non-negotiable rule as for expandable sub-resources.
2. **Handle every payload position.** A resource appears as a single object, inside the `list` envelope's `data` array, and embedded inside *other* resources (e.g. `account_user` appears as `customer.defaults.sales_rep`, `transaction.responsible_user`, `shipment.shipped_by`). Write the downgrade as a recursive walk keyed on each object's `"object"` discriminator, and list **every parent object type** that can embed the resource in `ObjectTypes()` — otherwise old-version responses for those parents leak the new shape.
3. **`TransformRequest` is identity when request shapes didn't change.** Return `data` untouched. Most resource-shape changes only affect responses.
4. **Transformers are immutable once their `ToVersion` ships.** They encode a historical contract. Fixing a bug in one is allowed; evolving one to mean something new is not — that's a new version.

### Forced includes: when the old shape needs data the new shape gates

When a new version moves data behind an include (say `?include=user`) that the older version carried unconditionally, an older client cannot ask for an include it has never heard of, so the transformer declares the dependency by implementing the optional `version.IncludeForcer` interface:

```go
func (t accountUserForgeToPreview9) ForcedIncludes(objectType object.Type) []string {
	if objectType == constants.ObjectTypeAccountUser {
		return []string{"user"} // resolve user so the downgrade can hoist real data
	}
	return nil
}
```

`Execute` merges forced keys into the include tree for old-version requests (after client-supplied include validation), so the resolver loads the real sub-resource and the downgrade hoists real values. The cost is an extra batched loader call per old-version request — acceptable, and it disappears when the old version is retired.

**Root vs. nested forced keys.** A root-level forced key (no dot) is applied unconditionally — use it when the old version carried the data without any include. A *nested* forced key (contains a dot, e.g. `defaults.sales_rep.user`) is applied **only when the client requested its parent path** — forcing it unconditionally would expand sub-resources the caller never asked for, which is itself a shape change. So `ForcedIncludes(customer) = ["defaults.sales_rep.user"]` means: *if* an older-version caller asked for `defaults.sales_rep`, also resolve its user so the downgrade can hoist the profile fields the older version carried there; if they didn't ask, nothing changes. `Execute` enforces this rule mechanically (it checks `includeTree.Has(parent)` before adding a dotted key).

---

## 4. What is a breaking change?

Anything that could alter what a pinned client observes. If a change is on this list, it requires a **new version plus a transformer** that preserves the old behavior:

- Removing or renaming a field (request or response).
- Changing a field's type, format, nullability, or enum value set.
- Moving data behind an include that was previously unconditional.
- Changing a default, a status code, an error `type`/`param`, or pagination semantics.
- Tightening request validation in a way that rejects previously accepted bodies.
- Changing the meaning of an existing parameter or field, even with the same shape.

Changes that are **not** breaking and ship directly on `Latest` without a version bump:

- Adding a new endpoint.
- Adding a new **response** field (clients must tolerate unknown fields).
- Adding a new **optional** request field or query parameter.
- Adding a new enum value to a field documented as open/extensible.
- Adding a new include key.
- Bug fixes where the documented contract was always the fixed behavior.

When in doubt, treat it as breaking. The cost of an unnecessary transformer is small; the cost of a silently broken consumer is not.

---

## 5. Shipping a new version: the checklist

1. **Declare the version** in `shared/apiversion/apiversion.go`: add the constant and pass it to `version.RegisterVersions` (the newest registered version is `version.Latest()`).
2. **Change the backend to the new shape only.** Resources (`pkg/resource/`), loaders, registries, endpoints, protos, services — all speak the new shape. No version conditionals anywhere below the gateway edge.
3. **Write the transformer(s)** in `services/api-gateway/internal/versiontransforms/`: one per changed resource, `FromVersion` = new, `ToVersion` = previous. Recursive walk; all embedding parent object types listed; `ForcedIncludes` where the downgrade needs gated data; identity `TransformRequest` if requests are unchanged. Unit-test the downgrade for single, list, nested, and data-missing cases, plus a `version.Transform`/`version.ForcedIncludes` end-to-end test through `version.DefaultRegistry`.
4. **Audit `ObjectType` coverage** on every endpoint that returns or accepts the changed resource.
5. **Update e2e to the new contract**: bump `defaultAPIVersion` in `tests/e2e/api/client_test.go`; rewrite shape assertions for the new version.
6. **Add version-compat e2e tests** (`tests/e2e/api/version_compat_<resource>_test.go`): pin a client to the previous version via `apiClient.WithAPIVersion(...)` and assert the *old* shape end-to-end — list, get, create, patch — plus one test asserting the latest version is unaffected by the transformer. These tests are the executable definition of "no breaking changes"; they stay green until the old version is removed.
7. **Regenerate artifacts**: `make openapi` (the spec documents `Latest` only — old versions are documented by their transformers and compat tests, not by parallel specs), bump the `api-version` seed in `tools/apidocs/httpie_seed_data.go`, regenerate SDKs as needed.
8. **Communicate**: an entry under the version in `docs/api-changelog.md` describing the new shape, what older versions receive instead, the migration path for clients, and the deprecation expectation for the previous version. A fix that applies to every version is recorded there too.

---

## 6. Deprecating and removing a version

Support is intentionally finite — the transformer chain should stay short. Deprecation is a two-phase process:

**Phase 1 — announce.** Declare the sunset date in the changelog and client communications. The version keeps working unchanged during this window (previews get short windows; stables get longer ones).

**Phase 2 — remove.** In one change:

1. Remove the version from `version.RegisterVersions` in `shared/apiversion/apiversion.go` (keep the constant itself only if a remaining transformer references it; otherwise delete it). From this moment the gateway rejects the version with a 400 `unsupported API version` — that is the entire enforcement mechanism.
2. Delete the transformer(s) whose `ToVersion` is the removed version, and their unit tests.
3. Delete the version-compat e2e tests pinned to it.
4. Remove any `MinVersion` guards that referenced it.
5. If the removed version was the oldest, check whether any `ForcedIncludes`-driven loader calls or shape-gap workarounds existed solely for it and remove them.

Nothing below the gateway edge changes — the backend never knew the old version existed.

A removed version is **gone**: requests for it fail loudly rather than being silently served the latest shape. Serving a new shape under an old pinned version string would violate the core contract worse than a clean 400 does.

---

## 7. Invariants (the short list)

- A pinned, supported version's wire contract never changes. Ever.
- The backend speaks `Latest` only; versioning is a gateway-edge transformation.
- Transformers bridge adjacent versions and are composed by the registry; they reshape real data and never fabricate.
- Every shape-changing release ships with: the transformer, its unit tests, and pinned version-compat e2e tests.
- Deprecation is explicit: removal from the registered versions → 400, never a silent shape change.
- The OpenAPI spec and SDKs track `Latest`; old versions live entirely in `versiontransforms/` and their compat tests.
