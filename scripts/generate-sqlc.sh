#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEFAULT_SERVICES=("core-service" "auth-service" "notification-service" "platform-service" "billing-service" "api-gateway")

usage() {
  echo "Usage: $0 [target1] [target2] ..."
  echo "Available services: ${DEFAULT_SERVICES[*]}"
  exit 1
}

generate_sqlc() {
  local dir="$1"
  local name="$2"
  
  if [[ ! -d "${dir}" ]]; then
    echo "Skipping ${name}: directory not found at ${dir}" >&2
    return
  fi
  
  if [[ ! -f "${dir}/sqlc.yaml" ]]; then
    echo "Skipping ${name}: sqlc.yaml not found in ${dir}" >&2
    return
  fi

  echo "Generating sqlc code for ${name}..."
  (cd "${dir}" && sqlc generate)
}

main() {
  if ! command -v sqlc >/dev/null 2>&1; then
    echo "sqlc not found in PATH. Please run 'make install-tools' first." >&2
    exit 1
  fi

  local targets=()

  if [[ "$#" -eq 0 || "${1}" == "all" ]]; then
    targets=("${DEFAULT_SERVICES[@]}")
  else
    for arg in "$@"; do
      local clean_arg="${arg#-}"

      case "${clean_arg}" in
        core) targets+=("core-service") ;;
        auth) targets+=("auth-service") ;;
        notification) targets+=("notification-service") ;;
        logging) targets+=("platform-service") ;;
        payment) targets+=("billing-service") ;;
        api) targets+=("api-gateway") ;;
        *)
          local found=false
          for s in "${DEFAULT_SERVICES[@]}"; do
            if [[ "${s}" == "${clean_arg}" ]]; then
              targets+=("${s}")
              found=true
              break
            fi
          done
          if [[ "${found}" == "false" ]]; then
            echo "Unknown target: ${arg}"
            usage
          fi
          ;;
      esac
    done
  fi

  for service in "${targets[@]}"; do
    generate_sqlc "${ROOT_DIR}/services/${service}" "${service}"
  done
}

main "$@"
