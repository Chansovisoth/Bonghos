# Changelog

All notable changes to Bonghos are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- Live updates never worked. The frontend sent `{"type": "subscribe"}` while the
  WebSocket hub read the `action` key, so no client subscribed to anything and
  console output, performance charts, import and backup progress, player updates
  and activity silently never arrived. The hub now accepts either key, the
  frontend sends the canonical one, and CI checks that the two sides agree.
- Restoring a backup through the Web UI replaced live server files without
  taking an emergency pre-restore copy first. The CLI did this correctly; the
  HTTP handler called `Restore` directly. Restore now creates a verified full
  backup first and refuses to continue if that fails.
- Restore scope was silently ignored. The UI sent `world` while the backend
  compared against `world_only`, so a world-only restore became a full restore
  that overwrote mods and startup scripts. Scope names are normalized and
  unknown values are rejected instead of defaulting to a full restore.
- Backup archives could overwrite each other. Names use one-second resolution,
  so the emergency copy taken immediately before a restore could collide with an
  existing archive and invalidate its stored checksum. Names now disambiguate
  with the backup ID and the manager refuses to overwrite.
- Listing schedules required only `server.view`, and the audit trail and host
  page were readable by every role. None of these belong to a Member's five
  permitted actions or a Viewer's view list. Backups were also removed from the
  Viewer role.
- Player actions were built by an inline switch that had drifted from
  `minecraft.PlayerCommand`, so `ban_ip`, `pardon_ip` and `send_message` were
  rejected by the API despite being implemented. All callers now share one set
  of validated templates.
- The file manager never enforced a text-editing size limit, and new projects
  did not default to recovering after an unclean shutdown.

### Added

- API tests driving the real HTTP handler through `httptest`, covering login,
  resistance to account enumeration, CSRF rejection, session revocation,
  disabled accounts, the exact Member and Viewer restrictions, Owner
  protections, and restore safety and scope handling. The previous release
  claimed an end-to-end suite that existed only as an uncommitted script.
- Importing an existing directory now refuses when a Java process is already
  running in it, on both the API and CLI paths. Detection is best-effort: it
  reads `/proc`, so it only sees the current user's processes.
- Drag-and-drop archive upload with real progress, speed, estimated time
  remaining and a working Cancel button, uploading through `XMLHttpRequest`
  so the transfer can be observed and aborted.
- The two-step login flow: credentials, then the authenticator code. Step one
  contacts nothing, so the interface reveals no more than the API does.
- Per-page WebSocket subscriptions that unsubscribe on navigation and
  re-subscribe after a reconnect.

## [0.1.0] — 2026-08-02

First release. Bonghos is a free-forever, open-source, self-hosted web control
panel for modded Minecraft Java Edition servers on Linux, running natively with
no Docker or containers.

### Added

**Installation and updates**
- Root `setup.sh` providing guided install, `--dev`, `--build`, `--update`,
  `--update --pull`, `--repair`, `--uninstall`, `--home` and `--help`
- Fast-forward-only Git updates that never reset, clean, stash, merge or rebase
  local work, and stop safely on diverged history or uncommitted changes
- Updates that build and test in a temporary area before touching the installed
  runtime, install the executable atomically, and roll back on failed health
  verification
- Safety copies of the database and configuration before every update
- Root `Tutorial.txt` with complete copyable instructions

**Portable runtime**
- Runtime root resolved from `--home`, then `BONGHOS_HOME`, then `$HOME/bonghos`
- Runtime containing only `servers/`, `backups/` and `system/`
- Internal paths stored relative to the root, so the directory can be moved to
  another location or copied to another computer
- `bonghos doctor`, `doctor --repair` and `doctor --fix-permissions`
- Portable `bonghos export` and `bonghos import`, with optional secret
  inclusion behind an explicit typed confirmation
- Encryption key at `system/config/secret.key` (mode 0600) used for
  authenticated encryption of TOTP secrets, never logged or exposed by the API

**Accounts and authorization**
- Multiple accounts with mandatory RFC 6238 TOTP two-factor authentication
- Owner, Admin, Member and Viewer roles enforced in the backend
- Member limited to exactly: view status, start, stop, restart, view players
- Viewer read-only
- Final-Owner protection: the last Owner cannot be deleted or demoted
- Admin-created single-use, expiring invitations; no public registration
- Argon2id password hashing, encrypted TOTP secrets, hashed one-use recovery codes
- Anti-enumeration login with dummy verification work, generic errors and rate
  limiting
- Server-side sessions with HttpOnly and SameSite cookies, CSRF tokens and
  session revocation
- Audit logging of authentication, lifecycle, player, file, configuration,
  backup, schedule, user and portability events

**Server projects**
- Display names with generated, validated, permanent directory slugs
- Import by archive upload (drag-and-drop or file picker), server-side URL
  download, local archive path, or existing directory (copy, move or link)
- Streaming uploads and downloads with progress, speed, ETA and cancellation
- Server-side downloads that continue across browser disconnection
- Persistent operations in SQLite so the UI can reconnect and show progress,
  interruptions and failures
- Safe extraction: traversal, absolute path, Windows path, symlink and
  hard-link protection, decompression-bomb, size and file-count limits, and
  free-space checks
- SSRF protection: HTTPS by default, blocked loopback, private, link-local,
  multicast, unspecified and cloud-metadata addresses, rejected credentials in
  URLs, redirect revalidation and re-resolution, redirect limits, timeouts and
  size caps, with an Owner-managed trusted-domain allowlist
- Startup-script detection that inspects script contents rather than trusting
  filenames, ranks candidates, and detects blocking interactive prompts and
  offers a reviewable patch
- JVM configuration detection across argument files, shell variables and
  pack-specific files, with the controlling source file shown per value
- Safe JVM editing with validation, timestamped backups, atomic writes and
  comment preservation
- Server icons converted to exactly 64×64 PNG, served through an authenticated
  endpoint

**Runtime**
- systemd user services `bonghos.service` and `bonghos-minecraft.service`
- Supervisor owning the Minecraft process, pseudo-terminal, console stream,
  process group and runtime state
- `bonghos-minecraft.service` deliberately not enabled unconditionally at boot;
  the control plane starts it after validating the active project
- Restart policies (`never`, `on-failure`, `always`) with delay, exponential
  backoff and crash-loop protection, where requested shutdown intent always
  overrides the restart policy
- Graceful stop with `save-all flush`, escalation only after timeout, and
  process-group cleanup
- Boot autostart per project with configurable delay, unclean-shutdown recovery
  and duplicate-start prevention
- Optional on-demand tmux console session named exactly `bonghos`, created
  lazily by `bonghos console` only, never at boot or server start
- `bonghos console --direct` for tmux-free console access
- Killing the tmux session or server never stops, restarts or signals Minecraft
- Framed local Unix-domain supervisor socket restricted to the Bonghos user

**Operations**
- Start, graceful stop, restart and force stop, serialized against conflicts
- Live console in the Web UI, CLI and tmux, all reaching the same supervisor
- Process and host monitoring with historical metrics, honestly labelling
  resident memory as RSS rather than Java heap
- Online and historical player lists from log parsing plus `list` polling only
  while subscribers are present
- Player actions (kick, ban, pardon, ban IP, whitelist, op, deop) using fixed
  command templates with validated parameters
- Persistent scheduler running without a browser, supporting once, hourly,
  daily, weekly, monthly, fixed interval and cron, with timezones, multi-step
  warning sequences, offline and missed-run policies, conflict handling,
  duplicate-run protection and full execution history
- Backups: full, world-and-player-data and configuration-only, online or
  offline, with `save-on` always restored after online backups
- Backup verification with checksums, retention policies with safety rules, and
  restore with emergency pre-restore backups and restore-as-new-instance
- Constrained file manager scoped to the server root
- Filesystem watching so manual SSH and SFTP edits are respected rather than
  overwritten

**Interfaces**
- Embedded Web UI served by the single Go executable, bound to `127.0.0.1` by
  default
- Authenticated WebSocket subscriptions per page, with all background work
  continuing when no browser is connected
- CLI covering serve, supervisor, console, doctor, setup, users, servers,
  backups, export, import, service management and version

### Security

- Canonical path containment everywhere, never string-prefix comparison
- Argument arrays instead of shell string concatenation
- No arbitrary shell execution anywhere; the Web UI console is never a shell
- Security headers and a Content Security Policy for the embedded frontend
- Secrets never written to logs, audit records or API responses

### Notes

- The frontend is dependency-free HTML, CSS and JavaScript rather than the
  React, TypeScript, Vite and Tailwind stack originally specified, so the
  project builds reproducibly offline with no JavaScript supply chain. The
  rationale is documented in `source/web/README.md`.
- Go dependencies are vendored under `source/third_party/`.
- The SQLite driver requires cgo. Builds must use `CGO_ENABLED=1`; a
  `CGO_ENABLED=0` build compiles and starts but fails on the first database
  access. Cross-compiling for ARM64 needs `gcc-aarch64-linux-gnu`.

### Known limitations

- Not yet verified against a real modded Minecraft server on real hardware;
  test fixtures are synthetic
- Supervisor crash and restart behaviour, boot autostart, unclean-shutdown
  recovery, tmux console lifecycle, restore and retention pruning are
  implemented and manually exercised, but not yet covered by automated
  integration tests
- systemd integration was validated in an environment without a live user
  manager
- ARM64 builds are produced by the toolchain but untested on ARM hardware
- Interrupted URL downloads restart from zero; range-based resumption is
  designed for but not implemented
- `export --include-secrets` is not separately passphrase-encrypted in v0.1.0;
  protect those archives like password files
- One Minecraft server runs at a time

### Deferred

CurseForge browsing, search, API keys and project-link resolution; Modrinth
browsing; vanilla Java and Bedrock server types; multiple simultaneously
running servers; multiple physical nodes; in-game metrics such as TPS and MSPT.

### Never planned

Docker or any container runtime; billing or subscriptions; public registration;
browser shell access; automatic router, firewall, port-forwarding or tunnel
configuration; required telemetry.

[Unreleased]: https://github.com/Chansovisoth/Bonghos/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/Chansovisoth/Bonghos/releases/tag/v0.1.0
