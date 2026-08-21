#!/usr/bin/env bash
# Download a release artifact from the release S3 buckets.
#
# Usage: fetch-openapi-spec-s3.sh <internal|public> <dest-path> [current|previous] [openapi|stainless]
#   current  — latest artifact (default)
#   previous — the object version immediately before the latest (requires bucket versioning)
#   openapi  — openapi.json (default)
#   stainless — stainless.yml
set -euo pipefail

sdk="${1:?usage: fetch-openapi-spec-s3.sh <internal|public> <dest-path> [current|previous] [openapi|stainless]}"
dest="${2:?usage: fetch-openapi-spec-s3.sh <internal|public> <dest-path> [current|previous] [openapi|stainless]}"
which="${3:-current}"
artifact="${4:-openapi}"

# Bucket names are deployment configuration, supplied by the caller (CI passes the repo's Actions
# variables). Keeping them out of source keeps this repo from naming the account's buckets.
case "$sdk" in
  internal)
    bucket="${INTERNAL_SPEC_BUCKET:?INTERNAL_SPEC_BUCKET is required}"
    ;;
  public)
    bucket="${PUBLIC_SPEC_BUCKET:?PUBLIC_SPEC_BUCKET is required}"
    ;;
  *)
    echo "unknown sdk: $sdk" >&2
    exit 1
    ;;
esac

case "$artifact" in
  openapi)
    key="openapi.json"
    ;;
  stainless)
    key="stainless.yml"
    ;;
  *)
    echo "unknown artifact: $artifact (expected openapi or stainless)" >&2
    exit 1
    ;;
esac
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
