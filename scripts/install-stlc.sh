#!/usr/bin/env bash
# Install stlc + the TypeScript, Python, Go, and MCP workers from OpenMRP's
# sdk-gen forks (not stainless/*). The MCP worker (stlc-mcp) generates the
# packages/mcp-server sub-package when a target enables options.mcp_server.
set -euo pipefail

STLC_GITHUB_ORG="${STLC_GITHUB_ORG:-sdk-gen}"

if [[ -z "${STLC_READ_TOKEN:-}" ]]; then
  if command -v gh >/dev/null 2>&1 && gh auth token >/dev/null 2>&1; then
    STLC_READ_TOKEN="$(gh auth token)"
  else
    echo "Set STLC_READ_TOKEN (PAT with read access to ${STLC_GITHUB_ORG}/stlc*) or run: gh auth login" >&2
    exit 1
  fi
fi

git_config_key="url.https://x-access-token:${STLC_READ_TOKEN}@github.com/${STLC_GITHUB_ORG}/.insteadOf"
cleanup() { git config --global --unset-all "$git_config_key" 2>/dev/null || true; }
trap cleanup EXIT
git config --global --add "$git_config_key" "https://github.com/${STLC_GITHUB_ORG}/"

npm install -g \
  "git+https://x-access-token:${STLC_READ_TOKEN}@github.com/${STLC_GITHUB_ORG}/stlc.git" \
  "git+https://x-access-token:${STLC_READ_TOKEN}@github.com/${STLC_GITHUB_ORG}/stlc-typescript.git" \
  "git+https://x-access-token:${STLC_READ_TOKEN}@github.com/${STLC_GITHUB_ORG}/stlc-python.git" \
  "git+https://x-access-token:${STLC_READ_TOKEN}@github.com/${STLC_GITHUB_ORG}/stlc-go.git" \
  "git+https://x-access-token:${STLC_READ_TOKEN}@github.com/${STLC_GITHUB_ORG}/stlc-mcp.git"

echo "Installed stlc from github.com/${STLC_GITHUB_ORG}."
echo "Ensure npm global bin is on PATH, e.g.:"
echo '  export PATH="$(npm config get prefix)/bin:$PATH"'
