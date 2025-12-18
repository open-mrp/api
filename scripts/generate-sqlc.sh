#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEFAULT_SERVICES=("auth-service" "notification-service" "logging-service")

usage() {
  echo "Usage: $0 [service1] [service2] ..."
  echo "Available services: ${DEFAULT_SERVICES[*]}"
  exit 1
}

main() {
  if ! command -v sqlc >/dev/null 2>&1; then
    echo "sqlc not found in PATH. Please run 'make install-tools' first." >&2
    exit 1
  fi

  local services=()
  
  if [[ "$#" -eq 0 || "${1}" == "all" ]]; then
    services=("${DEFAULT_SERVICES[@]}")
  else
    for arg in "$@"; do
      # Remove leading dash if present (e.g., -auth -> auth)
      local clean_arg="${arg#-}"
      
      # Handle shorthand names
      case "${clean_arg}" in
        auth) services+=("auth-service") ;;
        notification) services+=("notification-service") ;;
        logging) services+=("logging-service") ;;
        *)
          # Check if the argument is one of the default services
          local found=false
          for s in "${DEFAULT_SERVICES[@]}"; do
            if [[ "${s}" == "${clean_arg}" ]]; then
              services+=("${s}")
              found=true
              break
            fi
          done
          if [[ "${found}" == "false" ]]; then
            echo "Unknown service: ${arg}"
            usage
          fi
          ;;
      esac
    done
  fi

  for service in "${services[@]}"; do
    local service_dir="${ROOT_DIR}/services/${service}"
    if [[ ! -d "${service_dir}" ]]; then
      echo "Skipping ${service}: directory not found at ${service_dir}" >&2
      continue
    fi
    
    if [[ ! -f "${service_dir}/sqlc.yaml" ]]; then
      echo "Skipping ${service}: sqlc.yaml not found in ${service_dir}" >&2
      continue
    fi

    echo "Generating sqlc code for ${service}..."
    (cd "${service_dir}" && sqlc generate)
  done
}

main "$@"
