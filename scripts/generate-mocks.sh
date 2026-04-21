#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEFAULT_SERVICES=("auth-service" "api-gateway" "notification-service" "platform-service" "core-service" "billing-service" "agent-service")
COMPONENTS=("factories" "mediators" "publishers" "repositories" "services" "utils" "clients")
MAX_JOBS="${MOCK_JOBS:-8}"
declare -a services=()

component_dir() {
  case "${1}" in
    factories) echo "factory" ;;
    mediators) echo "mediator" ;;
    publishers) echo "publisher" ;;
    repositories) echo "repository" ;;
    services) echo "service" ;;
    utils) echo "utils" ;;
    clients) echo "client" ;;
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
    clients) echo "clientmock" ;;
    *) return 1 ;;
  esac
}

add_service() {
  local service="${1}"
  for existing in "${services[@]:-}"; do
    if [[ "${existing}" == "${service}" ]]; then
      return
    fi
  done
  services+=("${service}")
}

# Collect all mockgen jobs as tab-separated lines:
#   source_file \t dest_file \t package_name \t service_label
collect_jobs() {
  local jobs=()
  for service in "${services[@]}"; do
    local domain_dir="${ROOT_DIR}/services/${service}/internal/domain"
    if [[ ! -d "${domain_dir}" ]]; then
      echo "Skipping ${service}: domain directory not found" >&2
      continue
    fi

    # Remove stale mocks before regenerating
    local mock_dir="${domain_dir}/mock"
    if [[ -d "${mock_dir}" ]]; then
      rm -rf "${mock_dir}"
    fi

    for component in "${COMPONENTS[@]}"; do
      local source_file="${domain_dir}/${component}.go"
      if [[ ! -f "${source_file}" ]]; then
        continue
      fi

      # Check that the file actually contains interfaces
      if ! grep -q "^type .*interface" "${source_file}"; then
        continue
      fi

      local dest_dir="${domain_dir}/mock/$(component_dir "${component}")"
      local package_name
      package_name="$(component_package "${component}")"
      local dest_file="${dest_dir}/${component}_mock.go"

      mkdir -p "${dest_dir}"
      jobs+=("${source_file}	${dest_file}	${package_name}	${service}/${component}")
    done
  done

  printf '%s\n' "${jobs[@]}"
}

run_jobs() {
  local job_file
  job_file="$(mktemp)"
  collect_jobs > "${job_file}"

  local total
  total="$(wc -l < "${job_file}" | tr -d ' ')"
  echo "Running ${total} mockgen jobs with up to ${MAX_JOBS} parallel workers..."

  local failed=0
  local completed=0

  # Process jobs in parallel using a background job pool
  local pids=()
  while IFS=$'\t' read -r source_file dest_file package_name label; do
    # Wait if we've hit the max concurrent jobs
    while (( ${#pids[@]} >= MAX_JOBS )); do
      local new_pids=()
      for pid in "${pids[@]}"; do
        if kill -0 "${pid}" 2>/dev/null; then
          new_pids+=("${pid}")
        else
          if ! wait "${pid}"; then
            ((failed++))
          else
            ((completed++))
          fi
        fi
      done
      pids=("${new_pids[@]}")
      if (( ${#pids[@]} >= MAX_JOBS )); then
        sleep 0.05
      fi
    done

    (
      mockgen -source="${source_file}" -destination="${dest_file}" -package="${package_name}" 2>&1 && \
        echo "  OK: ${label}" || \
        { echo "  FAIL: ${label}" >&2; exit 1; }
    ) &
    pids+=("$!")
  done < "${job_file}"

  # Wait for remaining jobs
  for pid in "${pids[@]}"; do
    if ! wait "${pid}"; then
      ((failed++))
    else
      ((completed++))
    fi
  done

  rm -f "${job_file}"

  if (( failed > 0 )); then
    echo "ERROR: ${failed} mock generation job(s) failed." >&2
    return 1
  fi

  echo "Done. Generated ${completed} mock file(s)."
}

main() {
  if ! command -v mockgen >/dev/null 2>&1; then
    echo "mockgen not found in PATH. Please run 'make install-tools' first." >&2
    exit 1
  fi

  services=()

  if [[ "$#" -eq 0 ]]; then
    for service in "${DEFAULT_SERVICES[@]}"; do
      add_service "${service}"
    done
  else
    for arg in "$@"; do
      local clean_arg="${arg#-}"

      case "${clean_arg}" in
        all)
          for service in "${DEFAULT_SERVICES[@]}"; do
            add_service "${service}"
          done
          ;;
        auth) add_service "auth-service" ;;
        api) add_service "api-gateway" ;;
        notification) add_service "notification-service" ;;
        logging) add_service "platform-service" ;;
        *)
          local matched_default=false
          for s in "${DEFAULT_SERVICES[@]}"; do
            if [[ "${s}" == "${clean_arg}" ]]; then
              add_service "${s}"
              matched_default=true
              break
            fi
          done

          if [[ "${matched_default}" == "false" ]]; then
            if [[ -d "${ROOT_DIR}/services/${clean_arg}" ]]; then
              add_service "${clean_arg}"
            else
              echo "Warning: Unknown service '${arg}'. Skipping." >&2
            fi
          fi
          ;;
      esac
    done
  fi

  if [[ "${#services[@]}" -eq 0 ]]; then
    for service in "${DEFAULT_SERVICES[@]}"; do
      add_service "${service}"
    done
  fi

  echo "Generating mocks..."
  run_jobs

  # Print per-service summary
  for service in "${services[@]}"; do
    local mock_dir="${ROOT_DIR}/services/${service}/internal/domain/mock"
    if [[ -d "${mock_dir}" ]]; then
      echo "  OK: ${service}"
    fi
  done
}

main "$@"
