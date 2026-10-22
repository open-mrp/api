#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
versions="${root}/tools/tool-versions"

gosec_version="$(awk '$1 == "github.com/securego/gosec/v2" {print $2}' "$versions")"
xtools_version="$(awk '$1 == "golang.org/x/tools" {print $2}' "$versions")"

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
cd "$work"

go mod init gosec-install >/dev/null 2>&1
go get "github.com/securego/gosec/v2/cmd/gosec@${gosec_version}" "golang.org/x/tools@${xtools_version}"
go install github.com/securego/gosec/v2/cmd/gosec
