# Agent Notes

These notes are for Codex and other AI coding agents working in this repository.

## Web UI Integration

- The Web UI preview branch is named `webui`.
- Do not hand-merge `webui` into `main` unless the helper script is unavailable or the user explicitly asks for a manual merge.
- Prefer the guarded integration helper:

```bash
./scripts/integrate-webui.sh
```

With no flags, the helper fetches remote refs, tests the merge in a temporary worktree, validates the result, then exits without updating `main`, pushing, or installing/restarting the local Bonghos service. This check-only mode is the safe default for AI agents.

If conflicts or validation failures happen, it leaves `main` untouched and preserves the temporary worktree for inspection. Do not bypass a failed integration check; inspect the preserved worktree and fix the underlying conflict or test failure.

Useful noninteractive options:

```bash
./scripts/integrate-webui.sh --apply
./scripts/integrate-webui.sh --apply --push
./scripts/integrate-webui.sh --apply --push --install
./scripts/integrate-webui.sh --race
```

## Development Checks

From `source/`, use the existing Makefile targets:

```bash
make web
make fmt-check
make vet
make test
make build
```

The frontend is dependency-free vanilla HTML/CSS/JavaScript under `source/web/src/`; there is no npm/Vite/React build step.
