#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
GMT_DIR="${REPO_ROOT}/green-metrics-tool"
KWA_ENV_EXAMPLE="${REPO_ROOT}/kwa/.env.example"
KWA_GO_MOD="${REPO_ROOT}/kwa/go.mod"

GMT_REPO_URL="https://github.com/green-coding-solutions/green-metrics-tool.git"
API_URL_DEFAULT="http://api.green-coding.internal:9142"
METRICS_URL_DEFAULT="http://metrics.green-coding.internal:9142"

NEEDS_RELOGIN=false
INSTALL_WARNINGS=()

log() {
  printf '[setup] %s\n' "$*"
}

warn() {
  printf '[setup][warn] %s\n' "$*" >&2
}

die() {
  printf '[setup][error] %s\n' "$*" >&2
  exit 1
}

require_file() {
  local path="$1"
  [ -f "$path" ] || die "Required file not found: $path"
}

require_linux_ubuntu() {
  [ "$(uname -s)" = "Linux" ] || die "This setup script only supports Linux."
  [ -f /etc/os-release ] || die "Missing /etc/os-release; cannot validate distribution."

  # shellcheck disable=SC1091
  source /etc/os-release

  [ "${ID:-}" = "ubuntu" ] || die "Unsupported distro '${ID:-unknown}'. Supported: Ubuntu 22.04 and 24.04."
  case "${VERSION_ID:-}" in
    22.04|24.04) ;;
    *)
      die "Unsupported Ubuntu version '${VERSION_ID:-unknown}'. Supported: 22.04 and 24.04."
      ;;
  esac

  log "Detected Ubuntu ${VERSION_ID}."
}

ensure_sudo() {
  if ! command -v sudo >/dev/null 2>&1; then
    die "sudo is required for setup."
  fi

  log "Validating sudo access..."
  sudo -v
}

apt_update() {
  log "Running apt-get update..."
  sudo apt-get update -y
}

apt_install_missing() {
  local -a packages=("$@")
  local -a missing=()
  local pkg

  for pkg in "${packages[@]}"; do
    if ! dpkg -s "$pkg" >/dev/null 2>&1; then
      missing+=("$pkg")
    fi
  done

  if [ "${#missing[@]}" -gt 0 ]; then
    log "Installing missing apt packages: ${missing[*]}"
    sudo DEBIAN_FRONTEND=noninteractive apt-get install -y "${missing[@]}"
  else
    log "All required apt packages already installed."
  fi
}

ensure_base_tools() {
  apt_update
  apt_install_missing ca-certificates curl git make gcc gnupg lsb-release software-properties-common
}

# ensure_python312 installs and validates Python 3.12 plus venv/dev support needed by GMT.
# It exits on unsupported Python binaries or failed package installation.
ensure_python312() {
  local py312_bin=""

  if command -v python3.12 >/dev/null 2>&1; then
    py312_bin="$(command -v python3.12)"
  else
    log "Python 3.12 not found. Installing python3.12 packages..."
    if ! sudo DEBIAN_FRONTEND=noninteractive apt-get install -y python3.12 python3.12-venv python3.12-dev; then
      warn "Direct Python 3.12 install failed. Trying deadsnakes PPA..."
      sudo add-apt-repository -y ppa:deadsnakes/ppa
      apt_update
      sudo DEBIAN_FRONTEND=noninteractive apt-get install -y python3.12 python3.12-venv python3.12-dev
    fi
    py312_bin="$(command -v python3.12 || true)"
  fi

  apt_install_missing python3.12-venv python3.12-dev

  [ -n "$py312_bin" ] || die "Python 3.12 installation failed."

  "$py312_bin" -c 'import sys; raise SystemExit(0 if sys.version_info[:2] == (3, 12) else 1)' \
    || die "Python 3.12 is required; found incompatible python3.12 binary."

  log "Using Python 3.12 at: $py312_bin"
}

ensure_docker_apt_repo() {
  apt_install_missing ca-certificates curl gnupg lsb-release
  sudo install -m 0755 -d /etc/apt/keyrings

  if [ ! -f /etc/apt/keyrings/docker.gpg ]; then
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
      | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    sudo chmod a+r /etc/apt/keyrings/docker.gpg
  fi

  local codename
  codename="$(. /etc/os-release && echo "$VERSION_CODENAME")"
  local arch
  arch="$(dpkg --print-architecture)"

  echo "deb [arch=${arch} signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu ${codename} stable" \
    | sudo tee /etc/apt/sources.list.d/docker.list >/dev/null
}

ensure_docker() {
  local docker_missing=false
  if ! command -v docker >/dev/null 2>&1; then
    docker_missing=true
  fi

  if [ "$docker_missing" = true ]; then
    log "Docker not found. Installing Docker Engine + Compose plugin..."

    ensure_docker_apt_repo
    apt_update

    sudo DEBIAN_FRONTEND=noninteractive apt-get remove -y \
      docker docker.io docker-doc docker-compose docker-compose-v2 podman-docker containerd runc || true

    sudo DEBIAN_FRONTEND=noninteractive apt-get install -y \
      docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
  else
    log "Docker command found."
    if ! docker compose version >/dev/null 2>&1; then
      warn "docker compose plugin not available; installing docker-compose-plugin..."
      apt_update
      if ! sudo DEBIAN_FRONTEND=noninteractive apt-get install -y docker-compose-plugin; then
        warn "docker-compose-plugin install failed; configuring Docker apt repository and retrying..."
        ensure_docker_apt_repo
        apt_update
        sudo DEBIAN_FRONTEND=noninteractive apt-get install -y docker-compose-plugin
      fi
    fi
  fi

  if command -v systemctl >/dev/null 2>&1; then
    log "Ensuring Docker daemon is enabled and running..."
    sudo systemctl enable --now docker
  fi

  if ! id -nG "$USER" | tr ' ' '\n' | grep -qx docker; then
    log "Adding user '$USER' to docker group..."
    sudo usermod -aG docker "$USER"
    NEEDS_RELOGIN=true
  fi

  if docker version >/dev/null 2>&1; then
    log "Docker daemon is reachable for current user."
  else
    warn "Docker is installed but current shell cannot access daemon without sudo yet."
    NEEDS_RELOGIN=true
  fi
}

get_go_required_version() {
  require_file "$KWA_GO_MOD"
  awk '/^go[[:space:]]+/ {print $2; exit}' "$KWA_GO_MOD"
}

map_go_arch() {
  case "$(uname -m)" in
    x86_64) echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    *) die "Unsupported architecture for Go install: $(uname -m)" ;;
  esac
}

ensure_go_version() {
  local required
  required="$(get_go_required_version)"
  [ -n "$required" ] || die "Unable to resolve required Go version from $KWA_GO_MOD"

  local required_tag="go${required}"
  local current_tag=""

  if command -v go >/dev/null 2>&1; then
    current_tag="$(go version | awk '{print $3}')"
  fi

  if [ "$current_tag" = "$required_tag" ]; then
    log "Go version already matches requirement: $current_tag"
    return
  fi

  log "Installing Go ${required} via official tarball (current: ${current_tag:-none})..."

  local arch
  arch="$(map_go_arch)"

  local tarball="go${required}.linux-${arch}.tar.gz"
  local download_url="https://go.dev/dl/${tarball}"
  local tmp_dir
  tmp_dir="$(mktemp -d)"
  local tar_path="${tmp_dir}/${tarball}"

  curl -fL "$download_url" -o "$tar_path"

  sudo rm -rf /usr/local/go
  sudo tar -C /usr/local -xzf "$tar_path"
  sudo ln -sf /usr/local/go/bin/go /usr/local/bin/go

  rm -rf "$tmp_dir"

  export PATH="/usr/local/go/bin:/usr/local/bin:${PATH}"
  hash -r

  local installed_tag
  installed_tag="$(/usr/local/go/bin/go version | awk '{print $3}')"
  [ "$installed_tag" = "$required_tag" ] \
    || die "Go install failed. Expected ${required_tag}, found ${installed_tag}."

  log "Go installed successfully: ${installed_tag}"
}

prompt_overwrite_gmt() {
  local reply
  printf "[setup] '%s' already exists. Overwrite it? [y/N]: " "$GMT_DIR"
  read -r reply
  case "$reply" in
    y|Y|yes|YES)
      log "Removing existing GMT directory..."
      rm -rf "$GMT_DIR"
      ;;
    *)
      log "Keeping existing GMT directory."
      ;;
  esac
}

ensure_gmt_checkout() {
  if [ -e "$GMT_DIR" ]; then
    prompt_overwrite_gmt
  fi

  if [ ! -d "$GMT_DIR" ]; then
    log "Cloning Green Metrics Tool into $GMT_DIR"
    git clone "$GMT_REPO_URL" "$GMT_DIR"
  fi

  [ -f "$GMT_DIR/install_linux.sh" ] || die "GMT checkout invalid: install_linux.sh not found in $GMT_DIR"
}

read_env_default() {
  local key="$1"
  local file="$2"
  local value

  [ -f "$file" ] || {
    printf '%s' ""
    return
  }

  value="$(grep -E "^${key}=" "$file" | tail -n1 | cut -d'=' -f2- || true)"
  printf '%s' "$value"
}

# build_provider_skip_flags maps a provider-skip code into GMT install flags.
# It updates ATTEMPT_FLAGS and always succeeds, including when no providers are skipped.
build_provider_skip_flags() {
  local code="$1"
  ATTEMPT_FLAGS=()

  [[ "$code" == *I* ]] && ATTEMPT_FLAGS+=("-I")
  [[ "$code" == *S* ]] && ATTEMPT_FLAGS+=("-S")
  [[ "$code" == *R* ]] && ATTEMPT_FLAGS+=("-R")

  return 0
}

run_gmt_install_best_effort() {
  local db_password
  db_password="$(read_env_default "DATABASE_PASSWORD" "$KWA_ENV_EXAMPLE")"
  if [ -z "$db_password" ]; then
    db_password="postgres"
    warn "DATABASE_PASSWORD not found in kwa/.env.example. Falling back to default 'postgres'."
  fi

  local py312
  py312="$(command -v python3.12)"
  [ -n "$py312" ] || die "python3.12 not found before GMT install."

  local pyshim
  pyshim="${GMT_DIR}/.setup-python-shim"
  mkdir -p "$pyshim"
  rm -f "${pyshim}/python3"
  ln -s "$py312" "${pyshim}/python3"

  local -a base_args=(
    -L
    -T
    -z
    -f
    -J
    -a "$API_URL_DEFAULT"
    -m "$METRICS_URL_DEFAULT"
    -p "$db_password"
  )

  local -a attempt_codes=("" "I" "S" "R" "IS" "IR" "SR" "ISR")
  local attempt
  local chosen_code=""
  local install_succeeded=false

  for attempt in "${!attempt_codes[@]}"; do
    local code="${attempt_codes[$attempt]}"
    build_provider_skip_flags "$code"

    if [ -n "$code" ]; then
      warn "GMT install attempt $((attempt + 1))/${#attempt_codes[@]} with provider-skip flags: ${ATTEMPT_FLAGS[*]}"
    else
      log "GMT install attempt $((attempt + 1))/${#attempt_codes[@]} with full provider setup"
    fi

    set +e
    (
      cd "$GMT_DIR"
      PATH="${pyshim}:$PATH" ./install_linux.sh "${base_args[@]}" "${ATTEMPT_FLAGS[@]}"
    )
    local rc=$?
    set -e

    if [ "$rc" -eq 0 ]; then
      chosen_code="$code"
      install_succeeded=true
      break
    fi

    INSTALL_WARNINGS+=("GMT install attempt $((attempt + 1)) failed (flags: ${ATTEMPT_FLAGS[*]:-none}).")
  done

  if [ "$install_succeeded" = false ]; then
    die "GMT install failed after all best-effort attempts."
  fi

  if [ "$install_succeeded" = true ]; then
    local -a skipped=()
    [[ "$chosen_code" == *I* ]] && skipped+=("IPMI tools")
    [[ "$chosen_code" == *S* ]] && skipped+=("lm-sensors")
    [[ "$chosen_code" == *R* ]] && skipped+=("msr-tools")

    if [ "${#skipped[@]}" -gt 0 ]; then
      warn "GMT install completed with skipped provider dependencies: ${skipped[*]}"
    else
      log "GMT install completed with full provider dependencies enabled."
    fi
  fi
}

print_summary() {
  log "Setup complete."
  log "GMT directory: $GMT_DIR"

  if [ "${#INSTALL_WARNINGS[@]}" -gt 0 ]; then
    warn "Warnings captured during setup:"
    local item
    for item in "${INSTALL_WARNINGS[@]}"; do
      warn "- $item"
    done
  fi

  if [ "$NEEDS_RELOGIN" = true ]; then
    warn "Please relogin (or run 'newgrp docker') before running Docker commands without sudo."
  fi
}

main() {
  require_file "$KWA_GO_MOD"
  require_file "$KWA_ENV_EXAMPLE"

  require_linux_ubuntu
  ensure_sudo
  ensure_base_tools
  ensure_python312
  ensure_docker
  ensure_go_version
  ensure_gmt_checkout
  run_gmt_install_best_effort
  print_summary
}

main "$@"
