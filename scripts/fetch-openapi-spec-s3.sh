#!/usr/bin/env bash
# Download an OpenAPI spec from the release S3 buckets.
#
# Usage: fetch-openapi-spec-s3.sh <internal|public> <dest-path> [current|previous]
#   current  — latest openapi.json (default)
#   previous — the object version immediately before the latest (requires bucket versioning)
set -euo pipefail

sdk="${1:?usage: fetch-openapi-spec-s3.sh <internal|public> <dest-path> [current|previous]}"
dest="${2:?usage: fetch-openapi-spec-s3.sh <internal|public> <dest-path> [current|previous]}"
which="${3:-current}"

case "$sdk" in
  internal)
    bucket="augno-private-openapi-specs"
    ;;
  public)
    bucket="augno-public-openapi-specs"
    ;;
  *)
    echo "unknown sdk: $sdk" >&2
    exit 1
    ;;
esac

key="openapi.json"
mkdir -p "$(dirname "$dest")"

case "$which" in
  current)
    aws s3 cp "s3://${bucket}/${key}" "$dest"
    ;;
  previous)
    version_id="$(
      aws s3api list-object-versions \
        --bucket "$bucket" \
        --prefix "$key" \
        --query 'Versions[?IsLatest==`false`] | sort_by(@, &LastModified) | [-1].VersionId' \
        --output text 2>/dev/null || true
    )"
    if [ -z "$version_id" ] || [ "$version_id" = "None" ] || [ "$version_id" = "null" ]; then
      echo "no previous object version for s3://${bucket}/${key}" >&2
      exit 1
    fi
    aws s3api get-object \
      --bucket "$bucket" \
      --key "$key" \
      --version-id "$version_id" \
      "$dest" >/dev/null
    ;;
  *)
    echo "unknown which: $which (expected current or previous)" >&2
    exit 1
    ;;
esac
