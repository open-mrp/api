#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEFAULT_SERVICES=("core-service" "auth-service" "notification-service" "platform-service" "billing-service" "agent-service" "api-gateway")
TMP_DIR=""

usage() {
  echo "Usage: $0 [target1] [target2] ..."
  echo "Available services: ${DEFAULT_SERVICES[*]}"
  exit 1
}

generate_sqlc() {
  local dir="$1"
  local name="$2"
  local log_file="$3"

  if [[ ! -d "${dir}" ]]; then
    echo "Skipping ${name}: directory not found" >> "${log_file}"
    return
  fi

  if [[ ! -f "${dir}/sqlc.yaml" ]]; then
    echo "Skipping ${name}: sqlc.yaml not found" >> "${log_file}"
    return
  fi

  (cd "${dir}" && sqlc generate) >> "${log_file}" 2>&1
  echo "  OK: ${name}" >> "${log_file}"
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
        agent) targets+=("agent-service") ;;
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

  echo "Generating sqlc..."

  TMP_DIR="$(mktemp -d)"
  trap 'rm -rf "${TMP_DIR}"' EXIT

  # Run each service in parallel, buffering output per service
  local pids=()
  local failed=0

  for service in "${targets[@]}"; do
    generate_sqlc "${ROOT_DIR}/services/${service}" "${service}" "${TMP_DIR}/${service}.log" &
    pids+=($!)
  done

  for pid in "${pids[@]}"; do
    if ! wait "${pid}"; then
      failed=1
    fi
  done

  # Print buffered output in order
  for service in "${targets[@]}"; do
    local log_file="${TMP_DIR}/${service}.log"
    if [[ -f "${log_file}" ]]; then
      cat "${log_file}"
    fi
  done

  if [[ "${failed}" -ne 0 ]]; then
    echo "ERROR: one or more services failed sqlc generation" >&2
    exit 1
  fi

  echo "Done."
}

main "$@"
