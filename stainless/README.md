# Stainless workspaces (generated SDK configs)

Augno publishes **multiple** Stainless workspaces from this directory:

| Subdirectory | OpenAPI source | NPM package output | Checked-out repo folder (under augno/core) |
|--------------|----------------|--------------------|--------------------------------------------|
| `internal/` | `api/specs/internal_openapi_spec.json` | `@augno/internal-sdk` | `internal-sdk/` |
| `public/` | `api/specs/public_openapi_spec.json` | `@augno/sdk` | `typescript-sdk/` |

Paths in each `workspace.json` are relative to that JSON file (`openapi_spec`, `stainless_config`) and climb to the monorepo root via `output_path`.

## Regenerate specs and configs

From **`api/`** (regenerates both the OpenAPI specs that Stainless reads and the `stainless.yml` configs in this directory):

```bash
make openapi-stainless
```

To regenerate only the `stainless.yml` configs (without touching the specs), use `make stainless`. To regenerate only the specs, use `make openapi`.

## Run STLC (`stlc` CLI)

Examples (run from **`api/`**, or anywhere with `--workspace`):

```bash
# Internal SDK
stlc build --workspace stainless/internal

# Public TypeScript SDK
stlc build --workspace stainless/public
```

CI / automation should pass an explicit `--workspace` so nested layouts do not rely on cwd.
