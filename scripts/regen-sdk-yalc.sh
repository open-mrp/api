#!/usr/bin/env bash
# Rebuild @augno/internal-sdk from the CURRENT api code and publish it to the local
# yalc store, then (by default) link it into dashboard for local testing.
#
# This lives in the api repo on purpose: regenerating the SDK with `stlc` overwrites
# the entire internal-sdk working tree, so any orchestration kept inside that repo gets
# wiped on every regen. Owning the pipeline from here is stable across regens.
#
# Usage:
#   make sdk-yalc                          # full: regen spec + SDK, build, yalc publish, link dashboard
#   ./scripts/regen-sdk-yalc.sh --skip-regen   # skip spec+SDK regen; just rebuild, publish, relink
#   ./scripts/regen-sdk-yalc.sh --no-link      # stop after yalc publish (don't touch dashboard)
#
# Teardown (restore dashboard to the published SDK):  (cd ../dashboard && bun run sdk:unlink)
set -euo pipefail

API_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# internal-sdk and dashboard are private sibling checkouts, so this only works from inside the Augno
# monorepo. Say so, rather than dying on an opaque "no such file or directory" two lines down.
for sibling in internal-sdk dashboard; do
  if [ ! -d "$API_DIR/../$sibling" ]; then
    echo "Missing sibling checkout: $API_DIR/../$sibling" >&2
    echo "This script links a locally built SDK into the dashboard and needs the private" >&2
    echo "augno/internal-sdk and augno/dashboard repositories checked out alongside this one." >&2
    echo "To regenerate just the OpenAPI spec and SDK config, use 'make generate' instead." >&2
    exit 1
  fi
done

SDK_DIR="$(cd "$API_DIR/../internal-sdk" && pwd)"
DASH_DIR="$(cd "$API_DIR/../dashboard" && pwd)"

SKIP_REGEN=0
DO_LINK=1
for arg in "$@"; do
  case "$arg" in
    --skip-regen) SKIP_REGEN=1 ;;
    --no-link) DO_LINK=0 ;;
    *) echo "unknown flag: $arg" >&2; exit 2 ;;
  esac
done

log() { printf '\n\033[1;34m==> %s\033[0m\n' "$1"; }

command -v yalc >/dev/null 2>&1 || { echo "yalc not found on PATH (npm i -g yalc)." >&2; exit 1; }

if [ "$SKIP_REGEN" -eq 0 ]; then
  log "Regenerating the OpenAPI spec and Stainless config from the current api code"
  make -C "$API_DIR" openapi-stainless

  # stlc refuses to run on a dirty tree and regenerates the whole repo from the spec +
  # Stainless config, so discarding local working-tree state in the SDK repo is safe and
  # expected (it is overwritten regardless).
  log "Resetting internal-sdk to a clean tree"
  git -C "$SDK_DIR" reset --hard HEAD
  git -C "$SDK_DIR" clean -fd

  log "Regenerating the SDK source (stlc)"
  make -C "$API_DIR" stlc-internal-sdk
fi

# Build the publishable dist/. node_modules survives the regen (it is gitignored, so the
# reset/clean above keeps it), so no install is needed for a spec-only regen.
log "Building dist/"
( cd "$SDK_DIR" && ./scripts/build )

log "Publishing to the yalc store (from dist/)"
( cd "$SDK_DIR/dist" && yalc publish )

if [ "$DO_LINK" -eq 1 ]; then
  log "Linking into dashboard (yalc add + bun install)"
  ( cd "$DASH_DIR" && bun run sdk:link )
  printf '\n\033[1;32mDone.\033[0m Dashboard is using the yalc-linked local @augno/internal-sdk.\n'
  echo "Teardown: (cd $DASH_DIR && bun run sdk:unlink)"
else
  printf '\n\033[1;32mDone.\033[0m @augno/internal-sdk published to yalc.\n'
  echo "Link into dashboard with: (cd $DASH_DIR && bun run sdk:link)"
fi
