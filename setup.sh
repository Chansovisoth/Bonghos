#!/usr/bin/env bash
#
# Bonghos — guided install, build, update, repair and uninstall entrypoint.
#
#   https://github.com/Chansovisoth/Bonghos
#
# Normal users only ever need this script. Development commands live in
# source/Makefile. See Tutorial.txt for complete copyable instructions.
#
# This script never installs Docker, never runs Bonghos or Minecraft as root,
# never touches firewall/router/tunnel configuration, and never destroys local
# Git work.

set -Eeuo pipefail

# ---------------------------------------------------------------------------
# Paths — resolved from this script's own location, never the caller's cwd.
# ---------------------------------------------------------------------------
PROJECT_ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
SOURCE_DIR="$PROJECT_ROOT/source"
OFFICIAL_REPO="https://github.com/Chansovisoth/Bonghos"

BONGHOS_VERSION="0.1.0"

# Some minimal environments (containers, cron) do not export USER.
USER="${USER:-$(id -un)}"
MIN_GO_MINOR=22          # require go1.22+
MIN_DISK_MB=1024

# ---------------------------------------------------------------------------
# Output helpers
# ---------------------------------------------------------------------------
if [ -t 1 ] && [ -z "${NO_COLOR:-}" ]; then
  C_RESET=$'\033[0m'; C_BOLD=$'\033[1m'; C_DIM=$'\033[2m'
  C_RED=$'\033[31m'; C_GREEN=$'\033[32m'; C_YELLOW=$'\033[33m'; C_BLUE=$'\033[34m'
else
  C_RESET=""; C_BOLD=""; C_DIM=""; C_RED=""; C_GREEN=""; C_YELLOW=""; C_BLUE=""
fi

say()   { printf '%s\n' "$*"; }
info()  { printf '%s\n' "${C_BLUE}::${C_RESET} $*"; }
ok()    { printf '%s\n' "${C_GREEN}[ok]${C_RESET} $*"; }
warn()  { printf '%s\n' "${C_YELLOW}[warn]${C_RESET} $*" >&2; }
err()   { printf '%s\n' "${C_RED}[error]${C_RESET} $*" >&2; }
head1() { printf '\n%s\n' "${C_BOLD}$*${C_RESET}"; }
die()   { err "$*"; exit 1; }

# Temporary workspace, always cleaned.
TMP_ROOT=""
cleanup() {
  local rc=$?
  [ -n "$TMP_ROOT" ] && [ -d "$TMP_ROOT" ] && rm -rf "$TMP_ROOT"
  return $rc
}
trap cleanup EXIT
trap 'err "interrupted"; exit 130' INT TERM

mktemp_root() {
  TMP_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/bonghos-build.XXXXXXXX")"
}

confirm() { # confirm "question" [default:Y|N]
  local q="$1" def="${2:-Y}" ans prompt
  if [ "${ASSUME_YES:-0}" = "1" ]; then return 0; fi
  case "$def" in Y) prompt="[Y/n]";; *) prompt="[y/N]";; esac
  read -r -p "$q $prompt: " ans || true
  ans="${ans:-$def}"
  case "$ans" in [Yy]*) return 0;; *) return 1;; esac
}

ask() { # ask "question" "default" -> echoes answer
  local q="$1" def="$2" ans
  if [ "${ASSUME_YES:-0}" = "1" ]; then printf '%s' "$def"; return; fi
  read -r -p "$q [$def]: " ans || true
  printf '%s' "${ans:-$def}"
}

have() { command -v "$1" >/dev/null 2>&1; }

# ---------------------------------------------------------------------------
# Option parsing (order-flexible)
# ---------------------------------------------------------------------------
MODE="install"
DO_PULL=0
BONGHOS_HOME_ARG=""
ASSUME_YES=0

usage() {
  cat <<EOF
${C_BOLD}Bonghos setup${C_RESET} — $OFFICIAL_REPO

Usage: ./setup.sh [options]

  (no option)         Guided production build and installation.
  --dev               Prepare and verify development dependencies.
  --build             Build and test without installing.
  --update            Build and install the source currently present here.
  --update --pull     Fast-forward a clean Git checkout first, then update.
  --repair            Run Bonghos repair, service repair and portability checks.
  --uninstall         Remove services and executable, preserving data.
  --home DIR          Use a custom BONGHOS_HOME (default: \$HOME/bonghos).
  --yes               Assume yes for prompts (non-interactive).
  --help              Show this message.

Valid combinations:
  ./setup.sh
  ./setup.sh --dev
  ./setup.sh --build
  ./setup.sh --update
  ./setup.sh --update --pull        (same as --pull --update)
  ./setup.sh --repair
  ./setup.sh --uninstall
  ./setup.sh --home /mnt/storage/bonghos
  ./setup.sh --update --pull --home /mnt/storage/bonghos

Notes:
  --pull is only valid together with --update.
  --pull requires a clean Git worktree and fast-forward-only history.
  An extracted ZIP or tar archive has no Git metadata; use --update instead.
EOF
}

set_mode() {
  if [ "$MODE" != "install" ] && [ "$MODE" != "$1" ]; then
    die "conflicting options: --$MODE and --$1 cannot be combined"
  fi
  MODE="$1"
}

while [ $# -gt 0 ]; do
  case "$1" in
    --dev)       set_mode dev ;;
    --build)     set_mode build ;;
    --update)    set_mode update ;;
    --repair)    set_mode repair ;;
    --uninstall) set_mode uninstall ;;
    --pull)      DO_PULL=1 ;;
    --yes|-y)    ASSUME_YES=1 ;;
    --home)      shift; [ $# -gt 0 ] || die "--home requires a directory"; BONGHOS_HOME_ARG="$1" ;;
    --home=*)    BONGHOS_HOME_ARG="${1#--home=}" ;;
    --help|-h)   usage; exit 0 ;;
    *)           err "unknown option: $1"; say ""; usage; exit 2 ;;
  esac
  shift
done

if [ "$DO_PULL" = "1" ] && [ "$MODE" != "update" ]; then
  die "--pull is only valid together with --update (try: ./setup.sh --update --pull)"
fi

# Resolve BONGHOS_HOME: --home, then env, then default.
if [ -n "$BONGHOS_HOME_ARG" ]; then
  BONGHOS_HOME="$BONGHOS_HOME_ARG"
elif [ -n "${BONGHOS_HOME:-}" ]; then
  BONGHOS_HOME="$BONGHOS_HOME"
else
  BONGHOS_HOME="$HOME/bonghos"
fi
# Expand ~ and make absolute without requiring existence.
BONGHOS_HOME="${BONGHOS_HOME/#\~/$HOME}"
case "$BONGHOS_HOME" in /*) ;; *) BONGHOS_HOME="$PWD/$BONGHOS_HOME" ;; esac

BIN_DIR="$BONGHOS_HOME/system/bin"
BIN_PATH="$BIN_DIR/bonghos"

# ---------------------------------------------------------------------------
# Environment checks
# ---------------------------------------------------------------------------
refuse_root() {
  if [ "$(id -u)" = "0" ]; then
    # Escape hatch for CI and container images that have no non-root user.
    # Never use this on a real machine: Bonghos and Minecraft must not run as root.
    if [ "${BONGHOS_ALLOW_ROOT:-0}" = "1" ]; then
      warn "Running as root because BONGHOS_ALLOW_ROOT=1 is set."
      warn "This is intended for CI and container builds only."
      warn "Do NOT run Bonghos or Minecraft as root on a real system."
      return 0
    fi
    err "Do not run this installer as root."
    err "Bonghos and Minecraft are designed to run as a normal Linux user."
    err "Run it again as your own user, without sudo."
    err ""
    err "(For CI or container images without a normal user, set BONGHOS_ALLOW_ROOT=1.)"
    exit 1
  fi
}

detect_system() {
  local distro="unknown" version="" arch
  if [ -r /etc/os-release ]; then
    # shellcheck disable=SC1091
    . /etc/os-release
    distro="${ID:-unknown}"; version="${VERSION_ID:-}"
  fi
  arch="$(uname -m)"
  case "$arch" in
    x86_64)  GOARCH_EXPECT="amd64" ;;
    aarch64|arm64) GOARCH_EXPECT="arm64" ;;
    *)       GOARCH_EXPECT="$arch" ;;
  esac
  DISTRO="$distro"; DISTRO_VERSION="$version"; HOST_ARCH="$arch"
  info "System: ${distro} ${version} (${arch})"
  case "$distro" in
    ubuntu|debian|linuxmint|pop|elementary|raspbian) ;;
    *) warn "Primary support is Ubuntu 24.04 LTS. Other distributions may work but are untested." ;;
  esac
}

validate_source_tree() {
  [ -d "$SOURCE_DIR" ] || die "source/ directory not found at $SOURCE_DIR — is this the Bonghos project root?"
  local required=(
    "$SOURCE_DIR/go.mod"
    "$SOURCE_DIR/cmd/bonghos/main.go"
    "$SOURCE_DIR/internal"
    "$SOURCE_DIR/migrations"
    "$SOURCE_DIR/web/src"
  )
  local p
  for p in "${required[@]}"; do
    [ -e "$p" ] || die "source tree looks incomplete: missing $p"
  done
  ok "Source tree validated at $SOURCE_DIR"
}

APT_MISSING=()
need_pkg() { # need_pkg <command> <apt-package> <required|optional> <why>
  local cmd="$1" pkg="$2" level="$3" why="$4"
  if have "$cmd"; then
    ok "$cmd found${why:+ — $why}"
    return 0
  fi
  if [ "$level" = "required" ]; then
    warn "$cmd missing (required) — $why"
    APT_MISSING+=("$pkg")
    return 1
  fi
  warn "$cmd missing (optional) — $why"
  return 0
}

check_go() {
  if ! have go; then
    warn "Go missing (required to build Bonghos)"
    APT_MISSING+=("golang-go")
    return 1
  fi
  local v minor
  v="$(go version 2>/dev/null | awk '{print $3}')"   # go1.22.2
  minor="$(printf '%s' "$v" | sed -n 's/^go1\.\([0-9]\+\).*/\1/p')"
  if [ -z "$minor" ] || [ "$minor" -lt "$MIN_GO_MINOR" ]; then
    die "Go 1.$MIN_GO_MINOR or newer is required (found ${v:-unknown}). Install a newer Go and try again."
  fi
  ok "Go $v"
}

check_java() {
  head1 "Java runtimes"
  local found=0 j
  for j in /usr/lib/jvm/*/bin/java /usr/bin/java; do
    [ -x "$j" ] || continue
    local ver
    ver="$("$j" -version 2>&1 | head -1 || true)"
    say "  ${C_DIM}$j${C_RESET} — $ver"
    found=1
  done
  if [ "$found" = "0" ]; then
    warn "No Java installation detected."
    warn "Modded Minecraft usually needs Java 17 or 21:"
    warn "  sudo apt install openjdk-21-jre-headless"
    warn "Bonghos will still install; select Java later in the Web UI."
  fi
}

check_dependencies() {
  head1 "Build dependencies"
  check_go || true
  need_pkg git  git  required "source retrieval and updates" || true
  need_pkg tar  tar  required "archive handling" || true
  need_pkg gzip gzip required "archive handling" || true

  head1 "Optional runtime tools"
  need_pkg tmux    tmux         optional "optional 'bonghos console' session"
  need_pkg unzip   unzip        optional "ZIP server packs"
  need_pkg xz      xz-utils     optional ".tar.xz server packs"
  need_pkg zstd    zstd         optional "tar.zst backups (recommended)"
  need_pkg 7z      p7zip-full   optional ".7z server packs"
  need_pkg unrar   unrar        optional ".rar server packs (not bundled; proprietary)"

  head1 "systemd user services"
  if have systemctl && systemctl --user show-environment >/dev/null 2>&1; then
    ok "systemd user manager available"
    SYSTEMD_OK=1
  else
    warn "systemd user services unavailable in this environment."
    warn "Bonghos will still run with: $BIN_PATH serve"
    SYSTEMD_OK=0
  fi

  if [ "${#APT_MISSING[@]}" -gt 0 ]; then
    head1 "Missing required packages"
    say "  ${APT_MISSING[*]}"
    if have apt-get; then
      if confirm "Install them now with sudo apt install?" Y; then
        sudo apt-get update
        sudo apt-get install -y "${APT_MISSING[@]}"
        APT_MISSING=()
        check_go
      else
        die "cannot continue without the required packages"
      fi
    else
      die "install these packages with your distribution's package manager, then re-run"
    fi
  fi
}

check_disk() {
  local target="$1" avail
  mkdir -p "$target" 2>/dev/null || true
  avail="$(df -Pm "$target" 2>/dev/null | awk 'NR==2 {print $4}')"
  if [ -n "$avail" ] && [ "$avail" -lt "$MIN_DISK_MB" ]; then
    die "not enough free space at $target (${avail} MB available, ${MIN_DISK_MB} MB needed)"
  fi
  ok "Disk space at $target: ${avail:-unknown} MB free"
}

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------
WEBDIST="$SOURCE_DIR/cmd/bonghos/webdist"

build_frontend() {
  head1 "Building the Web UI"
  # The frontend is dependency-free (no npm/Vite/React toolchain), so the
  # "build" is a deterministic copy into the Go embed directory. See
  # source/web/README.md for why this project avoids a JS build pipeline.
  [ -d "$SOURCE_DIR/web/src" ] || die "frontend source missing at $SOURCE_DIR/web/src"
  mkdir -p "$WEBDIST"
  rm -f "$WEBDIST"/*.html "$WEBDIST"/*.css "$WEBDIST"/*.js 2>/dev/null || true
  cp "$SOURCE_DIR/web/src"/* "$WEBDIST"/
  [ -f "$WEBDIST/index.html" ] || die "frontend build produced no index.html"
  ok "Web UI prepared for embedding ($(find "$WEBDIST" -type f | wc -l) files)"
}

run_tests() {
  head1 "Running tests"
  ( cd "$SOURCE_DIR" && GOFLAGS="${GOFLAGS:-}" go test ./internal/... ) \
    || die "tests failed — refusing to build or install a broken version"
  ok "All tests passed"
}

build_binary() { # build_binary <output-path>
  local out="$1"
  head1 "Building the Bonghos executable"
  ( cd "$SOURCE_DIR" && go build \
      -ldflags "-s -w -X github.com/Chansovisoth/Bonghos/internal/app.Version=$BONGHOS_VERSION" \
      -o "$out" ./cmd/bonghos ) || die "build failed"
  chmod 0755 "$out"
  ok "Built $(basename "$out") ($(du -h "$out" | cut -f1))"
  "$out" version >/dev/null || die "built executable does not run on this machine"
}

# Atomically install the executable via a same-filesystem temp file + rename.
install_binary() { # install_binary <built-path>
  local built="$1" tmp
  mkdir -p "$BIN_DIR"
  tmp="$BIN_DIR/.bonghos.new.$$"
  cp "$built" "$tmp"
  chmod 0755 "$tmp"
  mv -f "$tmp" "$BIN_PATH"
  ok "Installed $BIN_PATH"
}

# ---------------------------------------------------------------------------
# Git --pull safety
# ---------------------------------------------------------------------------
git_pull_ff_only() {
  head1 "Retrieving the latest source"

  if ! have git; then
    die "git is not installed; cannot use --pull"
  fi
  if ! git -C "$PROJECT_ROOT" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    err "This project directory is not a Git worktree."
    err "It looks like an extracted source archive, which has no Git metadata."
    err ""
    err "To update from an archive: download the latest source archive, extract it,"
    err "enter the extracted directory and run:"
    err "    ./setup.sh --update"
    exit 1
  fi

  local branch upstream remote remote_url
  branch="$(git -C "$PROJECT_ROOT" rev-parse --abbrev-ref HEAD)"
  if [ "$branch" = "HEAD" ]; then
    die "the checkout is in detached HEAD state; check out a branch before using --pull"
  fi
  if ! upstream="$(git -C "$PROJECT_ROOT" rev-parse --abbrev-ref --symbolic-full-name '@{u}' 2>/dev/null)"; then
    err "Branch '$branch' has no upstream branch configured."
    err "Set one, for example:"
    err "    git -C \"$PROJECT_ROOT\" branch --set-upstream-to=origin/$branch $branch"
    exit 1
  fi
  remote="${upstream%%/*}"
  remote_url="$(git -C "$PROJECT_ROOT" remote get-url "$remote" 2>/dev/null || echo unknown)"

  say "  Branch:   $branch"
  say "  Upstream: $upstream"
  say "  Remote:   $remote_url"

  case "$remote_url" in
    *Chansovisoth/Bonghos*) ;;
    *) warn "Remote does not match the official repository ($OFFICIAL_REPO)."
       confirm "Continue updating from this remote anyway?" N || exit 1 ;;
  esac

  # Local work protection: never reset, clean, stash, merge or rebase.
  local dirty
  dirty="$(git -C "$PROJECT_ROOT" status --porcelain --untracked-files=normal)"
  if [ -n "$dirty" ]; then
    err "Local changes are present in the source checkout."
    err "Bonghos will not reset, clean, stash, merge or rebase your work."
    say ""
    say "$dirty" | sed 's/^/    /'
    say ""
    err "Review them with:"
    err "    git -C \"$PROJECT_ROOT\" status"
    err "    git -C \"$PROJECT_ROOT\" diff"
    err "    git -C \"$PROJECT_ROOT\" diff --staged"
    err ""
    err "Commit, revert or move your changes, then run ./setup.sh --update --pull again."
    err "To build the source exactly as it is right now instead, run:"
    err "    ./setup.sh --update"
    exit 1
  fi

  info "Fetching $remote ..."
  git -C "$PROJECT_ROOT" fetch --prune "$remote" || die "fetch failed; source not modified"

  # Fast-forward only. Stop cleanly on divergence.
  if ! git -C "$PROJECT_ROOT" merge --ff-only "$upstream" >/dev/null 2>&1; then
    err "Local and upstream histories have diverged; refusing a non-fast-forward update."
    err "No automatic merge or rebase will be performed."
    err ""
    err "Inspect the situation with:"
    err "    git -C \"$PROJECT_ROOT\" status"
    err "    git -C \"$PROJECT_ROOT\" log --oneline --graph --decorate --all"
    err ""
    err "The installed runtime was NOT modified."
    exit 1
  fi
  ok "Source fast-forwarded to $(git -C "$PROJECT_ROOT" rev-parse --short HEAD)"
}

# ---------------------------------------------------------------------------
# Modes
# ---------------------------------------------------------------------------
mode_install() {
  head1 "Bonghos $BONGHOS_VERSION — guided installation"
  say "Repository: $OFFICIAL_REPO"
  refuse_root
  detect_system
  validate_source_tree
  check_dependencies
  check_java

  head1 "Installation location"
  say "Bonghos keeps everything portable inside one runtime directory:"
  say "  ${C_DIM}<BONGHOS_HOME>/servers   real Minecraft server files${C_RESET}"
  say "  ${C_DIM}<BONGHOS_HOME>/backups   portable backups${C_RESET}"
  say "  ${C_DIM}<BONGHOS_HOME>/system    Bonghos internals${C_RESET}"
  say ""
  if [ -z "$BONGHOS_HOME_ARG" ]; then
    BONGHOS_HOME="$(ask "Install Bonghos runtime to" "$BONGHOS_HOME")"
    BONGHOS_HOME="${BONGHOS_HOME/#\~/$HOME}"
    case "$BONGHOS_HOME" in /*) ;; *) BONGHOS_HOME="$PWD/$BONGHOS_HOME" ;; esac
    BIN_DIR="$BONGHOS_HOME/system/bin"; BIN_PATH="$BIN_DIR/bonghos"
  fi

  if [ -e "$BIN_PATH" ]; then
    warn "An existing installation was found at $BONGHOS_HOME"
    warn "Use ./setup.sh --update to upgrade it safely (this preserves servers,"
    warn "backups, database, configuration and secret.key)."
    confirm "Overwrite the executable anyway?" N || exit 1
  fi

  check_disk "$BONGHOS_HOME"
  mktemp_root
  build_frontend
  run_tests
  build_binary "$TMP_ROOT/bonghos"

  head1 "Installing"
  install_binary "$TMP_ROOT/bonghos"

  head1 "First-run setup"
  say "You will now create the first Owner account and enrol two-factor"
  say "authentication. Have an authenticator app ready (Aegis, Ente Auth,"
  say "Google Authenticator, 1Password, Bitwarden ...)."
  say ""
  "$BIN_PATH" --home "$BONGHOS_HOME" setup

  print_next_steps
}

mode_dev() {
  head1 "Bonghos development environment"
  refuse_root
  detect_system
  validate_source_tree
  check_dependencies
  check_java
  info "Downloading Go module dependencies ..."
  ( cd "$SOURCE_DIR" && go mod download ) || warn "go mod download failed (dependencies are vendored under source/third_party)"
  run_tests
  cat <<EOF

${C_BOLD}Development commands${C_RESET} (run from $SOURCE_DIR)

  make build          Build the executable into ./bin/bonghos
  make test           Run the Go test suite
  make vet            Run go vet
  make run            Build and run with BONGHOS_HOME=./devhome
  make fmt            gofmt the tree
  make clean          Remove build artifacts

The frontend has no JS build step. Edit files in source/web/src/ and run
'make build' to re-embed them. See source/web/README.md.
EOF
}

mode_build() {
  head1 "Build and test only (no installation)"
  refuse_root
  validate_source_tree
  check_go
  build_frontend
  run_tests
  mkdir -p "$SOURCE_DIR/bin"
  build_binary "$SOURCE_DIR/bin/bonghos"
  ok "Executable ready at $SOURCE_DIR/bin/bonghos (nothing was installed)"
}

mode_update() {
  head1 "Bonghos update"
  refuse_root
  validate_source_tree

  [ -d "$BONGHOS_HOME" ] || die "no Bonghos installation found at $BONGHOS_HOME (use ./setup.sh to install, or --home DIR)"
  [ -x "$BIN_PATH" ]     || die "no Bonghos executable at $BIN_PATH — try ./setup.sh --repair"

  local installed_version
  installed_version="$("$BIN_PATH" version 2>/dev/null | awk '{print $2}')" || installed_version="unknown"
  say "  Installed: ${installed_version:-unknown}"
  say "  Candidate: $BONGHOS_VERSION"
  say "  Home:      $BONGHOS_HOME"

  if [ "$DO_PULL" = "1" ]; then
    git_pull_ff_only
  else
    info "Building the source currently present here (no source retrieval)."
  fi

  check_dependencies
  check_disk "$BONGHOS_HOME"

  # Build and test BEFORE touching the installed runtime.
  mktemp_root
  build_frontend
  run_tests
  build_binary "$TMP_ROOT/bonghos"

  # Record running state so it can be restored.
  local cp_was_active=0 mc_was_active=0
  if [ "${SYSTEMD_OK:-0}" = "1" ]; then
    systemctl --user is-active --quiet bonghos.service           && cp_was_active=1
    systemctl --user is-active --quiet bonghos-minecraft.service && mc_was_active=1
  fi
  if [ "$mc_was_active" = "1" ]; then
    warn "A Minecraft server is currently running."
    warn "Updating replaces the supervisor executable, which requires stopping it."
    warn "Connected players will be disconnected. The server will be restarted afterwards."
    confirm "Continue with the update?" N || { info "Update cancelled; nothing was changed."; exit 0; }
  fi

  # Safety copies of database and configuration.
  head1 "Creating safety copies"
  local stamp safety
  stamp="$(date +%Y-%m-%d_%H-%M-%S)"
  safety="$BONGHOS_HOME/system/temp/update-$stamp"
  mkdir -p "$safety"
  if [ -f "$BONGHOS_HOME/system/data/bonghos.db" ]; then
    # Checkpoint WAL through the running binary where possible, then copy.
    "$BIN_PATH" --home "$BONGHOS_HOME" doctor >/dev/null 2>&1 || true
    cp -p "$BONGHOS_HOME/system/data/bonghos.db" "$safety/bonghos.db"
    [ -f "$BONGHOS_HOME/system/data/bonghos.db-wal" ] && cp -p "$BONGHOS_HOME/system/data/bonghos.db-wal" "$safety/" || true
    ok "Database copied to $safety"
  fi
  [ -f "$BONGHOS_HOME/system/config/bonghos.toml" ] && cp -p "$BONGHOS_HOME/system/config/bonghos.toml" "$safety/" || true
  cp -p "$BIN_PATH" "$safety/bonghos.previous" || true

  # secret.key must exist and is never copied into logs or safety archives.
  if [ -f "$BONGHOS_HOME/system/config/secret.key" ]; then
    ok "secret.key present and preserved"
  else
    warn "secret.key is missing; encrypted data (TOTP secrets) cannot be read without it."
  fi

  # Stop only what must be replaced.
  if [ "$mc_was_active" = "1" ]; then
    info "Stopping Minecraft gracefully ..."
    systemctl --user stop bonghos-minecraft.service || warn "stop returned an error"
  fi
  if [ "$cp_was_active" = "1" ]; then
    info "Stopping the control plane ..."
    systemctl --user stop bonghos.service || warn "stop returned an error"
  fi

  head1 "Installing the new version"
  install_binary "$TMP_ROOT/bonghos"

  # Migrations, service repair and diagnostics.
  info "Applying migrations and checking the installation ..."
  if ! "$BIN_PATH" --home "$BONGHOS_HOME" doctor --repair; then
    err "Post-update verification failed; rolling back the executable."
    if [ -f "$safety/bonghos.previous" ]; then
      install_binary "$safety/bonghos.previous"
      err "Previous executable restored."
    fi
    err "Safety copies kept at: $safety"
    err "Database and configuration were NOT modified by the rollback."
    exit 1
  fi

  if [ "${SYSTEMD_OK:-0}" = "1" ]; then
    "$BIN_PATH" --home "$BONGHOS_HOME" service repair || warn "service repair reported a problem"
  fi

  # Restore prior running state.
  if [ "$cp_was_active" = "1" ]; then
    info "Starting the control plane ..."
    systemctl --user start bonghos.service || warn "could not start bonghos.service"
  fi
  if [ "$mc_was_active" = "1" ]; then
    info "Restarting Minecraft ..."
    systemctl --user start bonghos-minecraft.service || warn "could not start bonghos-minecraft.service"
  fi

  ok "Updated to $("$BIN_PATH" version | awk '{print $2}')"
  say ""
  say "Safety copies (remove when you are satisfied): $safety"
  say ""
  say "Verify with:"
  say "  $BIN_PATH version"
  say "  $BIN_PATH doctor"
  [ "${SYSTEMD_OK:-0}" = "1" ] && say "  systemctl --user status bonghos.service"
  return 0
}

mode_repair() {
  head1 "Bonghos repair"
  refuse_root
  [ -x "$BIN_PATH" ] || die "no Bonghos executable at $BIN_PATH"
  "$BIN_PATH" --home "$BONGHOS_HOME" doctor --repair || warn "doctor reported problems"
  if have systemctl && systemctl --user show-environment >/dev/null 2>&1; then
    if confirm "Regenerate systemd user services for $BONGHOS_HOME?" Y; then
      "$BIN_PATH" --home "$BONGHOS_HOME" service repair || warn "service repair failed"
      systemctl --user daemon-reload || true
      ok "Services regenerated"
    fi
  fi
  "$BIN_PATH" --home "$BONGHOS_HOME" doctor || true
}

mode_uninstall() {
  head1 "Uninstall Bonghos"
  refuse_root
  say "This removes the systemd user services and the Bonghos executable."
  say ""
  say "${C_BOLD}Your data is preserved by default:${C_RESET}"
  say "  $BONGHOS_HOME/servers/   Minecraft server files"
  say "  $BONGHOS_HOME/backups/   backups"
  say "  $BONGHOS_HOME/system/config/secret.key"
  say "  $BONGHOS_HOME/system/data/bonghos.db"
  say ""
  confirm "Continue with uninstall?" N || { info "Cancelled."; exit 0; }

  if have systemctl && systemctl --user show-environment >/dev/null 2>&1; then
    systemctl --user stop bonghos-minecraft.service 2>/dev/null || true
    systemctl --user stop bonghos.service 2>/dev/null || true
    systemctl --user disable bonghos.service 2>/dev/null || true
    if [ -x "$BIN_PATH" ]; then
      "$BIN_PATH" --home "$BONGHOS_HOME" service uninstall 2>/dev/null || true
    fi
    rm -f "$HOME/.config/systemd/user/bonghos.service" \
          "$HOME/.config/systemd/user/bonghos-minecraft.service"
    systemctl --user daemon-reload 2>/dev/null || true
    ok "systemd user services removed"
  fi

  rm -f "$BIN_PATH"
  ok "Executable removed"

  say ""
  say "Bonghos is uninstalled. Your servers, backups and database remain at:"
  say "  $BONGHOS_HOME"
  say ""
  say "To remove everything permanently, delete that directory yourself:"
  say "  ${C_DIM}rm -rf \"$BONGHOS_HOME\"${C_RESET}"
  warn "That also deletes system/config/secret.key. Encrypted data (including"
  warn "TOTP secrets) becomes permanently unrecoverable without it."
}

print_next_steps() {
  cat <<EOF

${C_BOLD}Bonghos is installed.${C_RESET}

  Executable:  $BIN_PATH
  Runtime:     $BONGHOS_HOME
  Web UI:      http://127.0.0.1:8080

${C_BOLD}Start it${C_RESET}
EOF
  if [ "${SYSTEMD_OK:-0}" = "1" ]; then
    cat <<EOF
  systemctl --user enable --now bonghos.service
  systemctl --user status bonghos.service

  To let Bonghos start at boot without an interactive login, enable lingering
  for your user (this needs sudo once, and Bonghos will not do it for you):
      sudo loginctl enable-linger $USER
EOF
  else
    cat <<EOF
  $BIN_PATH --home "$BONGHOS_HOME" serve
EOF
  fi
  cat <<EOF

${C_BOLD}Remote access${C_RESET}
  Bonghos listens on 127.0.0.1 only. It does not configure port forwarding,
  firewalls or tunnels — that stays your responsibility. For remote access,
  tunnel over SSH from your own computer:

      ssh -L 8080:127.0.0.1:8080 $USER@this-host

  then open http://127.0.0.1:8080 locally.

${C_BOLD}Next steps${C_RESET}
  1. Open the Web UI and sign in with your Owner account.
  2. Add a server project (upload an archive, download from a URL, or import
     an existing directory).
  3. Choose the startup script and Java, set RAM, accept the Minecraft EULA.
  4. Start the server.

  Optional console session:  $BIN_PATH console
  Diagnostics:               $BIN_PATH doctor

Full instructions: $PROJECT_ROOT/Tutorial.txt
EOF
}

# ---------------------------------------------------------------------------
case "$MODE" in
  install)   mode_install ;;
  dev)       mode_dev ;;
  build)     mode_build ;;
  update)    mode_update ;;
  repair)    mode_repair ;;
  uninstall) mode_uninstall ;;
  *)         usage; exit 2 ;;
esac
