#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
GMT_DIR="${REPO_ROOT}/green-metrics-tool"
KWA_BUILD_DIR="${REPO_ROOT}/kwa/build"

REMOVED_ITEMS=()
KEPT_ITEMS=()
SKIPPED_ITEMS=()
WARNINGS=()

REMOVE_DB=false
REMOVE_PREINSTALL=false
REMOVE_DOCKER_PKGS=false

log() {
  printf '[uninstall] %s\n' "$*"
}

warn() {
  printf '[uninstall][warn] %s\n' "$*" >&2
  WARNINGS+=("$*")
}

record_removed() {
  REMOVED_ITEMS+=("$*")
}

record_kept() {
  KEPT_ITEMS+=("$*")
}

record_skipped() {
  SKIPPED_ITEMS+=("$*")
}

prompt_yes_no() {
  local question="$1"
  local answer=""
  printf "%s [y/N]: " "$question"
  read -r answer
  case "$answer" in
    y|Y|yes|YES) return 0 ;;
    *) return 1 ;;
  esac
}

ensure_repo_shape() {
  [ -d "$REPO_ROOT" ] || {
    warn "Repository root not found: $REPO_ROOT"
    return
  }

  [ -d "$REPO_ROOT/kwa" ] || warn "KWA folder not found at $REPO_ROOT/kwa"
}

safe_remove_path() {
  local target="$1"
  local label="$2"

  if [ ! -e "$target" ]; then
    record_skipped "$label (not found)"
    return
  fi

  case "$target" in
    "$REPO_ROOT"/*) ;;
    *)
      warn "Refusing to remove path outside repository: $target"
      record_skipped "$label (outside repo)"
      return
      ;;
  esac

  rm -rf "$target"
  record_removed "$label"
}

detect_linux() {
  [ "$(uname -s)" = "Linux" ]
}

have_sudo() {
  command -v sudo >/dev/null 2>&1
}

teardown_gmt_containers() {
  if ! command -v docker >/dev/null 2>&1; then
    record_skipped "Docker cleanup (docker command not found)"
    return
  fi

  local compose_file="$GMT_DIR/docker/compose.yml"

  if [ -f "$compose_file" ]; then
    if [ "$REMOVE_DB" = true ]; then
      if docker compose -f "$compose_file" down --remove-orphans -v; then
        record_removed "GMT containers + networks + DB volume"
      else
        warn "Failed to run 'docker compose down -v'."
      fi
    else
      if docker compose -f "$compose_file" down --remove-orphans; then
        record_removed "GMT containers + networks"
      else
        warn "Failed to run 'docker compose down'."
      fi
      record_kept "PostgreSQL volume/data"
    fi
  else
    record_skipped "Compose down (compose file not found)"
  fi

  if [ "$REMOVE_DB" = true ]; then
    if docker volume inspect docker_green-coding-postgres-data >/dev/null 2>&1; then
      if docker volume rm docker_green-coding-postgres-data >/dev/null 2>&1; then
        record_removed "docker_green-coding-postgres-data volume"
      else
        warn "Failed to remove docker_green-coding-postgres-data volume."
      fi
    else
      record_skipped "docker_green-coding-postgres-data volume (not found)"
    fi
  fi

  if docker system prune -f >/dev/null 2>&1; then
    record_removed "Docker system prune"
  else
    warn "Failed to run docker system prune."
  fi
}

remove_local_artifacts() {
  safe_remove_path "$KWA_BUILD_DIR" "KWA build folder"
  safe_remove_path "$REPO_ROOT/.gocache" "Go cache (.gocache)"
  safe_remove_path "$REPO_ROOT/.gocache_local" "Go local cache (.gocache_local)"
  safe_remove_path "$REPO_ROOT/.gomodcache" "Go module cache (.gomodcache)"
  safe_remove_path "$GMT_DIR" "Local green-metrics-tool folder"
}

cleanup_sudoers_entries() {
  if ! have_sudo; then
    record_skipped "sudoers cleanup (sudo not available)"
    return
  fi

  if sudo rm -f /etc/sudoers.d/green_coding* /etc/sudoers.d/green-coding* 2>/dev/null; then
    record_removed "Green Metrics sudoers entries"
  else
    warn "Failed to remove some sudoers entries."
  fi
}

remove_preinstall_packages_linux() {
  if ! detect_linux; then
    record_skipped "Pre-install package removal (non-Linux)"
    return
  fi

  if ! have_sudo; then
    warn "Skipping package removal prompts because sudo is unavailable."
    record_skipped "Linux package removal (sudo unavailable)"
    return
  fi

  if prompt_yes_no "Do you want to remove pre-install requirements (git make gcc python3 python3-pip python3-venv curl)?"; then
    REMOVE_PREINSTALL=true
    if sudo apt-get remove -y git make gcc python3 python3-pip python3-venv curl; then
      record_removed "Pre-install requirements packages"
    else
      warn "Failed to remove one or more pre-install packages."
    fi
  else
    record_kept "Pre-install requirements packages"
  fi

  if prompt_yes_no "Do you want to remove Docker packages (docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin)?"; then
    REMOVE_DOCKER_PKGS=true
    if sudo apt-get remove -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin; then
      record_removed "Docker packages"
    else
      warn "Failed to remove one or more Docker packages."
    fi
  else
    record_kept "Docker packages"
  fi
}

print_summary() {
  echo
  log "Uninstall summary"

  if [ "${#REMOVED_ITEMS[@]}" -gt 0 ]; then
    printf '  Removed:\n'
    for item in "${REMOVED_ITEMS[@]}"; do
      printf '    - %s\n' "$item"
    done
  fi

  if [ "${#KEPT_ITEMS[@]}" -gt 0 ]; then
    printf '  Kept:\n'
    for item in "${KEPT_ITEMS[@]}"; do
      printf '    - %s\n' "$item"
    done
  fi

  if [ "${#SKIPPED_ITEMS[@]}" -gt 0 ]; then
    printf '  Skipped:\n'
    for item in "${SKIPPED_ITEMS[@]}"; do
      printf '    - %s\n' "$item"
    done
  fi

  if [ "${#WARNINGS[@]}" -gt 0 ]; then
    printf '  Warnings:\n'
    for item in "${WARNINGS[@]}"; do
      printf '    - %s\n' "$item"
    done
  fi
}

main() {
  log "Starting safe uninstall workflow."
  log "This operation is destructive for local bootstrap assets and caches."

  ensure_repo_shape

  if prompt_yes_no "Do you want to remove the database and its data volume?"; then
    REMOVE_DB=true
  else
    REMOVE_DB=false
  fi

  teardown_gmt_containers
  cleanup_sudoers_entries
  remove_local_artifacts
  remove_preinstall_packages_linux
  print_summary
}

main "$@"
