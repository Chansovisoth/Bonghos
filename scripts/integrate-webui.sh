#!/usr/bin/env bash
set -euo pipefail

# Safely integrate the webui branch into main.
#
# The merge is first attempted in a temporary worktree on a disposable
# integration branch. main is only updated after the merge and validation pass
# and after the operator confirms.

REMOTE="${REMOTE:-origin}"
MAIN_BRANCH="${MAIN_BRANCH:-main}"
WEBUI_BRANCH="${WEBUI_BRANCH:-webui}"
INTEGRATION_PREFIX="${INTEGRATION_PREFIX:-integrate-webui}"
RUN_RACE="${RUN_RACE:-0}"
INSTALL_AFTER="${INSTALL_AFTER:-ask}"
BONGHOS_HOME="${BONGHOS_HOME:-$HOME/bonghos}"

ROOT="$(git rev-parse --show-toplevel)"
TS="$(date +%Y%m%d-%H%M%S)"
INTEGRATION_BRANCH="${INTEGRATION_PREFIX}-${TS}"
WORKTREE=""
KEEP_WORKTREE=0

say() { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33mWARN:\033[0m %s\n' "$*" >&2; }
err() { printf '\033[1;31mERROR:\033[0m %s\n' "$*" >&2; }

cleanup() {
  if [ "$KEEP_WORKTREE" = "1" ] || [ -z "$WORKTREE" ]; then
    return
  fi
  git -C "$ROOT" worktree remove --force "$WORKTREE" >/dev/null 2>&1 || true
  git -C "$ROOT" branch -D "$INTEGRATION_BRANCH" >/dev/null 2>&1 || true
}
trap cleanup EXIT

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    err "Required command not found: $1"
    exit 1
  fi
}

confirm() {
  local prompt="$1"
  local answer
  read -r -p "$prompt [y/N] " answer
  case "$answer" in
    y|Y|yes|YES) return 0 ;;
    *) return 1 ;;
  esac
}

ensure_current_worktree_clean() {
  if [ -n "$(git -C "$ROOT" status --porcelain)" ]; then
    err "Current worktree has uncommitted changes. Commit or stash them first."
    git -C "$ROOT" status --short
    exit 1
  fi
}

ensure_ref_exists() {
  local ref="$1"
  if ! git -C "$ROOT" rev-parse --verify --quiet "$ref" >/dev/null; then
    err "Missing required ref: $ref"
    exit 1
  fi
}

ensure_local_main_can_fast_forward() {
  if ! git -C "$ROOT" rev-parse --verify --quiet "$MAIN_BRANCH" >/dev/null; then
    return
  fi
  if ! git -C "$ROOT" merge-base --is-ancestor "$MAIN_BRANCH" "$INTEGRATION_BRANCH"; then
    err "Local $MAIN_BRANCH has commits that are not in $INTEGRATION_BRANCH."
    err "Refusing to move it automatically. Inspect with:"
    echo "  git log --oneline --left-right $MAIN_BRANCH...$INTEGRATION_BRANCH"
    exit 1
  fi
}

run_validation() {
  say "Running validation"
  git -C "$WORKTREE" diff --check "$REMOTE/$MAIN_BRANCH..HEAD" || return
  (
    cd "$WORKTREE/source"
    make web || return
    make fmt-check || return
    go vet ./cmd/... ./internal/... || return
    go test ./internal/... || return
    if [ "$RUN_RACE" = "1" ]; then
      go test -race ./internal/... || return
    else
      warn "Skipping go test -race ./internal/... (set RUN_RACE=1 to include it)."
    fi
    go build ./... || return
    if command -v node >/dev/null 2>&1; then
      node --check web/src/app.js || return
    else
      warn "Skipping node --check web/src/app.js because node is not installed."
    fi
  )
}

install_running_service() {
  need_cmd systemctl
  say "Installing validated build into $BONGHOS_HOME"
  (
    cd "$WORKTREE/source"
    make build
  )
  if [ ! -x "$BONGHOS_HOME/system/bin/bonghos" ]; then
    err "Installed Bonghos binary not found at $BONGHOS_HOME/system/bin/bonghos"
    exit 1
  fi
  local stamp
  stamp="$(date +%Y%m%d-%H%M)"
  cp "$BONGHOS_HOME/system/bin/bonghos" "$BONGHOS_HOME/system/bin/bonghos.backup-$stamp"
  install -m 755 "$WORKTREE/source/bin/bonghos" "$BONGHOS_HOME/system/bin/bonghos"
  systemctl --user restart bonghos.service
  systemctl --user is-active bonghos.service
  say "Backup: $BONGHOS_HOME/system/bin/bonghos.backup-$stamp"
}

need_cmd git
need_cmd make
need_cmd go
need_cmd mktemp

say "Repository: $ROOT"
ensure_current_worktree_clean

say "Fetching $REMOTE"
git -C "$ROOT" fetch --prune "$REMOTE"

ensure_ref_exists "$REMOTE/$MAIN_BRANCH"
ensure_ref_exists "$REMOTE/$WEBUI_BRANCH"

BASE_SHA="$(git -C "$ROOT" rev-parse "$REMOTE/$MAIN_BRANCH")"
WEBUI_SHA="$(git -C "$ROOT" rev-parse "$REMOTE/$WEBUI_BRANCH")"

say "Main:  $REMOTE/$MAIN_BRANCH  $BASE_SHA"
say "WebUI: $REMOTE/$WEBUI_BRANCH  $WEBUI_SHA"

if [ "$BASE_SHA" = "$WEBUI_SHA" ]; then
  say "$MAIN_BRANCH already matches $WEBUI_BRANCH; nothing to integrate."
  exit 0
fi

say "Commits to merge from $WEBUI_BRANCH:"
git -C "$ROOT" log --oneline --decorate "$REMOTE/$MAIN_BRANCH..$REMOTE/$WEBUI_BRANCH" || true

WORKTREE="$(mktemp -d "${TMPDIR:-/tmp}/bonghos-webui-merge.XXXXXX")"
rmdir "$WORKTREE"

say "Creating temporary integration worktree: $WORKTREE"
git -C "$ROOT" worktree add -b "$INTEGRATION_BRANCH" "$WORKTREE" "$REMOTE/$MAIN_BRANCH"

say "Merging $REMOTE/$WEBUI_BRANCH into $INTEGRATION_BRANCH"
if ! git -C "$WORKTREE" merge --no-ff "$REMOTE/$WEBUI_BRANCH" -m "Merge Web UI"; then
  KEEP_WORKTREE=1
  err "Merge conflicts detected. main was not changed."
  echo
  git -C "$WORKTREE" status --short
  echo
  echo "Conflicted files:"
  git -C "$WORKTREE" diff --name-only --diff-filter=U || true
  echo
  echo "Resolve conflicts in:"
  echo "  $WORKTREE"
  echo
  echo "Then validate and finish manually, or abort with:"
  echo "  git -C '$WORKTREE' merge --abort"
  echo "  git -C '$ROOT' worktree remove '$WORKTREE'"
  echo "  git -C '$ROOT' branch -D '$INTEGRATION_BRANCH'"
  exit 1
fi

if ! run_validation; then
  KEEP_WORKTREE=1
  err "Validation failed. main was not changed."
  echo
  echo "Inspect the failed integration in:"
  echo "  $WORKTREE"
  echo
  echo "Clean it up when done with:"
  echo "  git -C '$ROOT' worktree remove '$WORKTREE'"
  echo "  git -C '$ROOT' branch -D '$INTEGRATION_BRANCH'"
  exit 1
fi

say "Integration branch is ready: $INTEGRATION_BRANCH"
git -C "$WORKTREE" log --graph --oneline --decorate -12
echo

if ! confirm "Fast-forward local $MAIN_BRANCH to the validated integration branch?"; then
  say "Stopped. $MAIN_BRANCH was not changed."
  exit 0
fi

ensure_local_main_can_fast_forward
if git -C "$ROOT" branch --show-current | grep -qx "$MAIN_BRANCH"; then
  git -C "$ROOT" switch "$MAIN_BRANCH"
  git -C "$ROOT" merge --ff-only "$INTEGRATION_BRANCH"
else
  git -C "$ROOT" branch -f "$MAIN_BRANCH" "$INTEGRATION_BRANCH"
fi

say "Local $MAIN_BRANCH now points to $(git -C "$ROOT" rev-parse "$MAIN_BRANCH")"

if confirm "Push $MAIN_BRANCH to $REMOTE?"; then
  git -C "$ROOT" push "$REMOTE" "$MAIN_BRANCH"
else
  warn "Push skipped. Run this when ready: git push $REMOTE $MAIN_BRANCH"
fi

case "$INSTALL_AFTER" in
  1|yes|true)
    install_running_service
    ;;
  0|no|false)
    warn "Install/restart skipped by INSTALL_AFTER=$INSTALL_AFTER."
    ;;
  ask)
    if confirm "Install validated build and restart local Bonghos service?"; then
      install_running_service
    else
      warn "Install/restart skipped."
    fi
    ;;
  *)
    err "Invalid INSTALL_AFTER value: $INSTALL_AFTER"
    exit 1
    ;;
esac

say "Done."
