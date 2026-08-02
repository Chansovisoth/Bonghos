# Contributing to Bonghos

Thanks for wanting to help. Bonghos is free-forever open-source software and
contributions of all sizes are welcome.

## Before you start

- Search [existing issues](https://github.com/Chansovisoth/Bonghos/issues) first.
- For anything substantial, open an issue to discuss it before writing code.
- For security vulnerabilities, **do not open an issue** — see [SECURITY.md](SECURITY.md).

## Development setup

```bash
git clone https://github.com/Chansovisoth/Bonghos.git ~/bonghos-source
cd ~/bonghos-source
./setup.sh --dev
```

Then from `source/`:

```bash
make build     # build ./bin/bonghos
make test      # run the test suite
make vet       # go vet
make fmt       # gofmt the tree
make run       # run with BONGHOS_HOME=./devhome
```

Dependencies are vendored under `source/third_party/`, so the project builds
offline with `GOPROXY=direct`.

**cgo is required.** The SQLite driver is a cgo package, so a `CGO_ENABLED=0`
build links and starts but fails at the first database access. Keep cgo enabled;
cross-compiling for ARM64 also needs `gcc-aarch64-linux-gnu`. `make cross`
handles this and `make verify-binaries` proves each binary can open a database.

## Project rules

These are not style preferences — they are architectural commitments. Pull
requests that break them will not be merged.

1. **No Docker or containers.** No Dockerfile, Compose file, Kubernetes
   manifest, or container-based deployment path. Bonghos runs natively.
2. **No root.** Bonghos and Minecraft run as a normal Linux user.
3. **systemd owns Minecraft's lifecycle, not tmux.** tmux is an optional
   console client created lazily by `bonghos console`. Never treat tmux session
   existence as authoritative process state, and never use `send-keys` or pane
   scraping for lifecycle control.
4. **Never expose a shell.** Not through the Web UI, not through the console
   protocol, not through schedules. Use argument arrays, never concatenated
   shell strings.
5. **The filesystem is the source of truth** for Minecraft files. SQLite holds
   only Bonghos metadata. Never hide worlds, mods, configs or scripts in the
   database.
6. **Backups stay portable** — plain archives extractable without Bonghos.
7. **Store paths relative to `BONGHOS_HOME`.** Never hardcode the root.
8. **Validate containment canonically.** Never trust a string prefix.
9. **Enforce permissions in the backend.** Hiding a button is not security.
10. **Never log secrets** — passwords, TOTP codes or secrets, recovery codes,
    session cookies, encryption keys, export passphrases, sensitive URL
    parameters.
11. **No required telemetry.** Any future telemetry is opt-in and off by default.

## Code standards

**Go**
- `gofmt` everything; `go vet` must pass.
- Use contexts with explicit timeouts. No unbounded operations.
- Use structured logging. No `fmt.Println` debugging left in.
- Wrap errors with context (`fmt.Errorf("...: %w", err)`).
- Filesystem writes must be atomic (temp file in the same directory + rename).
- Keep packages focused; avoid circular dependencies.

**Frontend**
- Dependency-free HTML/CSS/JavaScript by design. Please do not introduce a
  JavaScript build pipeline or npm dependencies — see `source/web/README.md`
  for the reasoning.
- Keep it accessible: keyboard navigation, visible focus states, sufficient
  contrast, and respect `prefers-reduced-motion`.

**Shell**
- `setup.sh` uses `set -Eeuo pipefail`, resolves its own directory, and never
  assumes the caller's working directory.
- Never destroy the user's Git work.

## Tests

New code needs tests, and security-relevant code needs them especially:

```bash
cd source && make test
```

Anything touching path handling, archive extraction, URL fetching,
authentication, authorization, or process lifecycle must come with tests for
the failure cases, not just the happy path.

## Pull requests

1. Fork and branch from `main`.
2. Keep the change focused — one concern per PR.
3. Run `make fmt vet test` before pushing.
4. Update `Tutorial.txt`, `README.md` and `source/docs/CHANGELOG.md` if
   behaviour, commands or options change. These must stay synchronized with
   what the code actually does.
5. Fill in the pull-request template.

Write commit messages in the imperative mood:

```
Add crash-loop backoff to the supervisor

The supervisor now applies exponential backoff between restart attempts
and gives up after the configured limit.
```

## Reporting bugs

Use the issue templates. Include your distribution, `bonghos version` and
relevant output from `bonghos doctor`.

**Never paste `secret.key`, session cookies or recovery codes into an issue.**

## License

Contributions are licensed under AGPL-3.0-only, the same as the project.
