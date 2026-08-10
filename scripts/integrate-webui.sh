#!/usr/bin/env bash
set -Eeuo pipefail

# Bonghos Web UI Integration
#
# Safely validates and optionally integrates the webui branch into main.
#
# Safe default:
#   ./scripts/integrate-webui.sh
#
# This creates a temporary Git worktree, merges origin/webui into origin/main,
# runs validation, then exits without changing main, pushing, or restarting
# Bonghos.

REMOTE="${REMOTE:-origin}"
MAIN_BRANCH="${MAIN_BRANCH:-main}"
WEBUI_BRANCH="${WEBUI_BRANCH:-webui}"
BONGHOS_HOME="${BONGHOS_HOME:-$HOME/bonghos}"

APPLY=0
PUSH=0
INSTALL=0
RUN_RACE=0
KEEP_WORKTREE=0

ROOT=""
WORKTREE=""
INTEGRATION_BRANCH=""
BASE_SHA=""
WEBUI_SHA=""
MERGED_SHA=""

say() { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
ok() { printf '\033[1;32mOK:\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33mWARN:\033[0m %s\n' "$*" >&2; }
err() { printf '\033[1;31mERROR:\033[0m %s\n' "$*" >&2; }

usage() {
  cat <<'EOF'
Usage:
  ./scripts/integrate-webui.sh [options]

Options:
  --apply          Fast-forward local main after validation
  --push           Push validated main to the configured remote
                   Implies --apply
  --install        Install the validated Bonghos build and restart its service
  --race           Run Go race tests
  --keep-worktree  Keep the temporary integration worktree after success
  -h, --help       Show this help

Safe default:
  With no options, the script only performs a temporary merge and validation.
  It does not modify main, push anything, or restart Bonghos.

Examples:
  ./scripts/integrate-webui.sh
  ./scripts/integrate-webui.sh --apply
  ./scripts/integrate-webui.sh --apply --push
  ./scripts/integrate-webui.sh --apply --push --install
  ./scripts/integrate-webui.sh --race
EOF
}

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    err "Required command not found: $1"
    exit 127
  fi
}

cleanup() {
  local exit_code=$?

  if [[ -z "${WORKTREE:-}" ]]; then
    return "$exit_code"
  fi

  if (( KEEP_WORKTREE )); then
    warn "Keeping integration worktree:"
    warn "  $WORKTREE"
    if [[ -n "${INTEGRATION_BRANCH:-}" ]]; then
      warn "Integration branch:"
      warn "  $INTEGRATION_BRANCH"
    fi
    return "$exit_code"
  fi

  git -C "$ROOT" worktree remove --force "$WORKTREE" >/dev/null 2>&1 || true
  if [[ -n "${INTEGRATION_BRANCH:-}" ]]; then
    git -C "$ROOT" branch -D "$INTEGRATION_BRANCH" >/dev/null 2>&1 || true
  fi

  return "$exit_code"
}

trap cleanup EXIT

print_preserved_worktree_help() {
  echo
  echo "Inspect the failed integration in:"
  echo "  $WORKTREE"
  echo
  echo "Clean it up when done with:"
  echo "  git -C '$ROOT' worktree remove '$WORKTREE'"
  echo "  git -C '$ROOT' branch -D '$INTEGRATION_BRANCH'"
}

validation_failed() {
  local step="$1"
  KEEP_WORKTREE=1
  err "Validation failed during: $step"
  err "main has NOT been modified."
  print_preserved_worktree_help
  exit 12
}

run_source_step() {
  local label="$1"
  shift
  say "$label"
  (
    cd "$WORKTREE/source"
    "$@"
  ) || validation_failed "$label"
}

while (($#)); do
  case "$1" in
    --apply)
      APPLY=1
      ;;
    --push)
      APPLY=1
      PUSH=1
      ;;
    --install)
      INSTALL=1
      ;;
    --race)
      RUN_RACE=1
      ;;
    --keep-worktree)
      KEEP_WORKTREE=1
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      err "Unknown argument: $1"
      echo >&2
      usage >&2
      exit 2
      ;;
  esac
  shift
done

need_cmd git
need_cmd make
need_cmd go
need_cmd mktemp

ROOT="$(git rev-parse --show-toplevel 2>/dev/null)" || {
  err "This script must be run inside the Bonghos Git repository."
  exit 1
}

cd "$ROOT"

if [[ ! -d "$ROOT/source" ]]; then
  err "Expected Bonghos source directory not found:"
  err "  $ROOT/source"
  exit 1
fi

say "Repository: $ROOT"
say "Remote:     $REMOTE"
say "Main:       $MAIN_BRANCH"
say "WebUI:      $WEBUI_BRANCH"

say "Fetching latest refs from $REMOTE..."
git fetch --prune "$REMOTE"

if ! git rev-parse --verify --quiet "refs/remotes/$REMOTE/$MAIN_BRANCH" >/dev/null; then
  err "Remote branch does not exist: $REMOTE/$MAIN_BRANCH"
  exit 1
fi

if ! git rev-parse --verify --quiet "refs/remotes/$REMOTE/$WEBUI_BRANCH" >/dev/null; then
  err "Remote branch does not exist: $REMOTE/$WEBUI_BRANCH"
  exit 1
fi

BASE_SHA="$(git rev-parse "$REMOTE/$MAIN_BRANCH")"
WEBUI_SHA="$(git rev-parse "$REMOTE/$WEBUI_BRANCH")"

say "Remote main:"
printf '    %s\n' "$BASE_SHA"
say "Remote WebUI:"
printf '    %s\n' "$WEBUI_SHA"

if git merge-base --is-ancestor "$REMOTE/$WEBUI_BRANCH" "$REMOTE/$MAIN_BRANCH"; then
  ok "$WEBUI_BRANCH is already fully contained in $MAIN_BRANCH."
  exit 0
fi

echo
say "WebUI commits not currently in main:"
git log --oneline --decorate "$REMOTE/$MAIN_BRANCH..$REMOTE/$WEBUI_BRANCH"
echo

timestamp="$(date +%Y%m%d-%H%M%S)"
INTEGRATION_BRANCH="integrate-webui-$timestamp"
WORKTREE="$(mktemp -d "${TMPDIR:-/tmp}/bonghos-webui-integration.XXXXXX")"
rmdir "$WORKTREE"

say "Creating temporary integration worktree:"
printf '    %s\n' "$WORKTREE"
git worktree add -b "$INTEGRATION_BRANCH" "$WORKTREE" "$REMOTE/$MAIN_BRANCH"

say "Merging $REMOTE/$WEBUI_BRANCH..."
if ! git -C "$WORKTREE" merge --no-ff -m "Merge Web UI" "$REMOTE/$WEBUI_BRANCH"; then
  KEEP_WORKTREE=1

  echo
  err "Merge conflicts detected."
  err "main has NOT been modified."

  echo
  git -C "$WORKTREE" status --short || true

  echo
  echo "Conflicted files:"
  git -C "$WORKTREE" diff --name-only --diff-filter=U || true

  echo
  echo "Resolve conflicts in:"
  echo "  $WORKTREE"

  echo
  echo "After resolving:"
  echo "  git -C '$WORKTREE' add -A"
  echo "  git -C '$WORKTREE' commit"

  echo
  echo "Or abort and remove it:"
  echo "  git -C '$WORKTREE' merge --abort"
  echo "  git -C '$ROOT' worktree remove '$WORKTREE'"
  echo "  git -C '$ROOT' branch -D '$INTEGRATION_BRANCH'"

  exit 10
fi

ok "Merge completed in temporary worktree."

say "Checking merged diff..."
git -C "$WORKTREE" diff --check "$REMOTE/$MAIN_BRANCH..HEAD" || validation_failed "git diff --check"

run_source_step "Running Bonghos Web UI build..." make web
run_source_step "Checking formatting..." make fmt-check
run_source_step "Running go vet..." go vet ./...
run_source_step "Running Go tests..." go test ./...

if (( RUN_RACE )); then
  run_source_step "Running Go race tests..." go test -race ./...
else
  warn "Race tests skipped. Use --race to enable them."
fi

run_source_step "Building all Go packages..." go build ./...

if command -v node >/dev/null 2>&1; then
  if [[ -f "$WORKTREE/source/web/src/app.js" ]]; then
    say "Checking web/src/app.js syntax..."
    node --check "$WORKTREE/source/web/src/app.js" || validation_failed "node --check web/src/app.js"
  fi
else
  warn "Node.js is unavailable; standalone JS syntax check skipped."
fi

if [[ -n "$(git -C "$WORKTREE" status --porcelain --untracked-files=no)" ]]; then
  KEEP_WORKTREE=1

  err "Validation modified tracked files."
  err "Generated or formatted changes must be committed explicitly."

  echo
  git -C "$WORKTREE" status --short
  print_preserved_worktree_help

  exit 11
fi

MERGED_SHA="$(git -C "$WORKTREE" rev-parse HEAD)"

echo
ok "Integration validation passed."
say "Validated merge commit:"
printf '    %s\n' "$MERGED_SHA"

if (( ! APPLY )); then
  echo
  say "Check-only mode complete."
  say "main was NOT changed."

  echo
  echo "To apply this integration:"
  echo "  ./scripts/integrate-webui.sh --apply"
  echo
  echo "To apply and push:"
  echo "  ./scripts/integrate-webui.sh --apply --push"

  exit 0
fi

say "Preparing to update local $MAIN_BRANCH..."

if git show-ref --verify --quiet "refs/heads/$MAIN_BRANCH"; then
  if ! git merge-base --is-ancestor "$MAIN_BRANCH" "$INTEGRATION_BRANCH"; then
    KEEP_WORKTREE=1

    err "Local $MAIN_BRANCH contains commits not present in the validated integration branch."
    err "Refusing to overwrite local work."

    echo
    echo "Inspect with:"
    echo "  git log --oneline --left-right '$MAIN_BRANCH...$INTEGRATION_BRANCH'"

    exit 20
  fi
fi

CURRENT_BRANCH="$(git branch --show-current)"

if [[ "$CURRENT_BRANCH" == "$MAIN_BRANCH" ]]; then
  if [[ -n "$(git status --porcelain)" ]]; then
    KEEP_WORKTREE=1

    err "The current $MAIN_BRANCH worktree has uncommitted changes."
    err "Commit or stash them before using --apply."

    exit 21
  fi

  say "Fast-forwarding checked-out $MAIN_BRANCH..."
  git merge --ff-only "$INTEGRATION_BRANCH"
else
  if git show-ref --verify --quiet "refs/heads/$MAIN_BRANCH"; then
    say "Updating local $MAIN_BRANCH ref..."
    git branch -f "$MAIN_BRANCH" "$INTEGRATION_BRANCH"
  else
    say "Creating local $MAIN_BRANCH..."
    git branch "$MAIN_BRANCH" "$INTEGRATION_BRANCH"
  fi
fi

LOCAL_MAIN_SHA="$(git rev-parse "$MAIN_BRANCH")"

if [[ "$LOCAL_MAIN_SHA" != "$MERGED_SHA" ]]; then
  KEEP_WORKTREE=1
  err "Local $MAIN_BRANCH does not match validated merge."
  exit 22
fi

ok "Local $MAIN_BRANCH updated to validated integration."

if (( PUSH )); then
  say "Checking that remote $MAIN_BRANCH has not changed..."
  git fetch "$REMOTE" "$MAIN_BRANCH:refs/remotes/$REMOTE/$MAIN_BRANCH"

  CURRENT_REMOTE_MAIN="$(git rev-parse "$REMOTE/$MAIN_BRANCH")"
  if [[ "$CURRENT_REMOTE_MAIN" != "$BASE_SHA" ]]; then
    KEEP_WORKTREE=1

    err "Remote $MAIN_BRANCH changed while integration was running."
    err "Refusing to push a potentially stale integration."

    echo
    echo "Original remote main:"
    echo "  $BASE_SHA"

    echo
    echo "Current remote main:"
    echo "  $CURRENT_REMOTE_MAIN"

    echo
    echo "Run the integration script again against the new main."

    exit 30
  fi

  say "Pushing $MAIN_BRANCH to $REMOTE..."
  git push "$REMOTE" "$MAIN_BRANCH:$MAIN_BRANCH"
  ok "Remote $MAIN_BRANCH updated."
fi

if (( INSTALL )); then
  need_cmd install
  need_cmd systemctl

  say "Building production Bonghos binary..."
  (
    cd "$WORKTREE/source"
    make build
  )

  BUILT_BINARY="$WORKTREE/source/bin/bonghos"
  INSTALLED_BINARY="$BONGHOS_HOME/system/bin/bonghos"

  if [[ ! -x "$BUILT_BINARY" ]]; then
    KEEP_WORKTREE=1
    err "Expected built Bonghos binary was not produced:"
    err "  $BUILT_BINARY"
    exit 40
  fi

  mkdir -p "$(dirname "$INSTALLED_BINARY")"

  if [[ -f "$INSTALLED_BINARY" ]]; then
    backup_stamp="$(date +%Y%m%d-%H%M%S)"
    backup_path="${INSTALLED_BINARY}.backup-${backup_stamp}"

    say "Backing up currently installed binary..."
    cp --preserve=mode,timestamps "$INSTALLED_BINARY" "$backup_path"

    say "Backup:"
    printf '    %s\n' "$backup_path"
  fi

  say "Installing validated Bonghos binary..."
  install -m 0755 "$BUILT_BINARY" "$INSTALLED_BINARY"

  say "Restarting Bonghos service..."
  systemctl --user restart bonghos.service

  if ! systemctl --user is-active --quiet bonghos.service; then
    KEEP_WORKTREE=1

    err "bonghos.service failed to become active."

    echo
    systemctl --user status bonghos.service --no-pager || true

    exit 41
  fi

  ok "Bonghos service restarted successfully."
fi

echo
ok "Web UI integration completed successfully."

say "main:"
printf '    %s\n' "$(git rev-parse "$MAIN_BRANCH")"

if (( ! PUSH )); then
  warn "Changes were not pushed."
  warn "Use --push next time, or run:"
  warn "  git push $REMOTE $MAIN_BRANCH"
fi
