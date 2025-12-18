#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODULE="github.com/augno/api"
DEFAULT_SERVICES=("auth-service" "api-gateway" "notification-service" "logging-service")
COMPONENTS=("factories" "mediators" "publishers" "repositories" "services" "utils")

to_snake_case() {
  local input="${1}"
  echo "${input}" \
    | sed 's/APIKey/api_key/g' \
    | sed -E 's/([A-Z]+)([A-Z][a-z])/\1_\2/g' \
    | sed -E 's/([a-z0-9])([A-Z])/\1_\2/g' \
    | tr '[:upper:]' '[:lower:]' \
    | sed 's/^_//'
}

component_dir() {
  case "${1}" in
    factories) echo "factory" ;;
    mediators) echo "mediator" ;;
    publishers) echo "publisher" ;;
    repositories) echo "repository" ;;
    services) echo "service" ;;
    utils) echo "utils" ;;
    *) return 1 ;;
  esac
}

component_package() {
  case "${1}" in
    factories) echo "factorymock" ;;
    mediators) echo "mediatormock" ;;
    publishers) echo "publishermock" ;;
    repositories) echo "repositorymock" ;;
    services) echo "servicemock" ;;
    utils) echo "utilsmock" ;;
    *) return 1 ;;
  esac
}

generate_for_component() {
  local service="${1}"
  local component="${2}"
  local domain_dir="${3}"
  local module_import="${4}"
  local domain_file="${domain_dir}/${component}.go"

  if [[ ! -f "${domain_file}" ]]; then
    return
  fi

  local interfaces
  interfaces=$(grep -h "^type .*interface" "${domain_file}" | awk '{print $2}')

  if [[ -z "${interfaces}" ]]; then
    return
  fi

  local dest_dir="${domain_dir}/mock/$(component_dir "${component}")"
  local package_name="$(component_package "${component}")"
  mkdir -p "${dest_dir}"

  for interface in ${interfaces}; do
    local filename
    filename="$(to_snake_case "${interface}")"
    echo "Generating ${service} ${component} mock for ${interface}..."
    mockgen -destination "${dest_dir}/${filename}_mock.go" -package "${package_name}" "${module_import}" "${interface}"
  done
}

generate_for_service() {
  local service="${1}"
  local domain_dir="${ROOT_DIR}/services/${service}/internal/domain"
  local module_import="${MODULE}/services/${service}/internal/domain"

  if [[ ! -d "${domain_dir}" ]]; then
    echo "Skipping ${service}: domain directory not found at ${domain_dir}" >&2
    return
  fi

  for component in "${COMPONENTS[@]}"; do
    generate_for_component "${service}" "${component}" "${domain_dir}" "${module_import}"
  done
}

main() {
  if ! command -v mockgen >/dev/null 2>&1; then
    echo "mockgen not found in PATH. Please run 'make install-tools' first." >&2
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
        api) services+=("api-gateway") ;;
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
             # If not found in defaults, check if directory exists anyway
             if [[ -d "${ROOT_DIR}/services/${clean_arg}" ]]; then
               services+=("${clean_arg}")
             else
               echo "Warning: Unknown service '${arg}'. Skipping." >&2
             fi
          fi
          ;;
      esac
    done
  fi

  for service in "${services[@]}"; do
    generate_for_service "${service}"
  done
}

main "$@"
