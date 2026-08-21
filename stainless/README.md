# Stainless workspaces (generated SDK configs)

Augno publishes **multiple** Stainless workspaces from this directory:

| Subdirectory | OpenAPI source | Target | Package output | Checked-out repo folder (under augno/core) |
|--------------|----------------|--------|----------------|--------------------------------------------|
| `internal/` | `api/specs/internal_openapi_spec.json` | typescript | `@augno/internal-sdk` | `internal-sdk/` |
| `public/` | `api/specs/public_openapi_spec.json` | typescript | `@augno/sdk` | `typescript-sdk/` |
| `public/` | `api/specs/public_openapi_spec.json` | python | `augno` (PyPI) | `python-sdk/` |
| `public/` | `api/specs/public_openapi_spec.json` | go | `github.com/augno/augno-go` | `augno-go/` |

The `public/` workspace's TypeScript target also generates an **MCP server** sub-package
(`typescript-sdk/packages/mcp-server`, published as `@augno/sdk-mcp`) via `options.mcp_server`, hosted on
EKS at `mcp.augno.com`. See [`docs/stlc-sdk-codegen.md` → MCP server](../docs/stlc-sdk-codegen.md#mcp-server).

Paths in each `workspace.json` are relative to that JSON file (`openapi_spec`, `stainless_config`) and climb to the monorepo root via `output_path`.

## Regenerate specs and configs

From **`api/`** (regenerates both the OpenAPI specs that Stainless reads and the `stainless.yml` configs in this directory):

```bash
make openapi-stainless
```

To regenerate only the `stainless.yml` configs (without touching the specs), use `make stainless`. To regenerate only the specs, use `make openapi`.

## Run STLC (`stlc` CLI)

Install `stlc` and the typescript/python/go workers with `make install-stlc`.

Prefer the make targets — they check that `stlc` is on PATH and carry the per-target flags
(the public TypeScript build needs `NODE_OPTIONS=--preserve-symlinks` so the `stlc-typescript`
and `stlc-mcp` plugins share one `codegen.lib.mjs`):

```bash
make stlc-internal-sdk          # @augno/internal-sdk
make stlc-public-typescript-sdk # @augno/sdk + MCP server
make stlc-public-python-sdk     # augno (PyPI)
make stlc-public-go-sdk         # github.com/augno/augno-go
make stlc-public-sdks           # all three public targets
make stlc-sdks                  # every workspace/target
```

Pass extra flags through with `STLC_BUILD_EXTRA`. The underlying invocation is:

```bash
stlc build --workspace stainless/internal --targets typescript
```

CI / automation should pass an explicit `--workspace` so nested layouts do not rely on cwd.

For local dashboard testing, `make sdk-yalc` rebuilds `@augno/internal-sdk` from the current api,
publishes it to yalc, and links it into the dashboard. It regenerates only the spec — run
`make openapi-stainless` first when endpoint shapes changed, or stlc silently drops SDK methods.
