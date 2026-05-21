#!/usr/bin/env bash
# Report whether the generated OpenAPI spec differs from a baseline snapshot on disk.
# Release CI: publish-openapi-specs downloads current S3 openapi.json into specs/sdk-baseline/
# before make openapi and runs this script against the freshly generated specs (pre-upload).
#
# Manual / local use: compare specs/*.json against a saved baseline under specs/sdk-baseline/.
#
# Byte comparison is safe because tools/apidocs emits deterministic JSON (see
# TestOpenAPIGenerationDeterministic): encoding/json sorts map keys, and path order
# follows route strings—not the order endpoints are registered in Go.
#
# Usage: sdk-openapi-spec-changed.sh <internal|public>
# Writes changed=true|false to GITHUB_OUTPUT when set.
set -euo pipefail

sdk="${1:?usage: sdk-openapi-spec-changed.sh <internal|public>}"

case "$sdk" in
  internal)
    spec="specs/internal_openapi_spec.json"
    baseline="specs/sdk-baseline/internal_openapi_spec.json"
    ;;
  public)
    spec="specs/public_openapi_spec.json"
    baseline="specs/sdk-baseline/public_openapi_spec.json"
    ;;
  *)
    echo "unknown sdk: $sdk" >&2
    exit 1
    ;;
esac

if [ ! -f "$spec" ]; then
  echo "missing spec file: $spec" >&2
  exit 1
fi

changed=true
if [ -f "$baseline" ]; then
  if cmp -s "$spec" "$baseline"; then
    changed=false
  fi
fi

if [ -n "${GITHUB_OUTPUT:-}" ]; then
  echo "changed=$changed" >> "$GITHUB_OUTPUT"
fi

if [ "$changed" = true ]; then
  if [ -f "$baseline" ]; then
    echo "OpenAPI spec changed for $sdk (differs from baseline file)."
  else
    echo "No baseline file for $sdk; treating spec as changed."
  fi
else
  echo "OpenAPI spec unchanged for $sdk (matches baseline file)."
fi
