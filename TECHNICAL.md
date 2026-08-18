# Bonghos Technical Reference

This document contains the implementation, architecture, security, storage, development, and validation details intentionally kept out of the user-focused [README](README.md).

## Architecture

Bonghos runs natively on Linux. It does not use Docker, Compose, Kubernetes, or another container runtime.

```text
systemd user manager
|
|-- bonghos.service
|   `-- Bonghos control plane
|       |-- Embedded Web UI
|       |-- REST API and WebSocket API
|       |-- Authentication and authorization
|       `-- Scheduler, imports, backups, bots, and monitoring
|
|-- bonghos-minecraft.service
|   `-- Bonghos supervisor
|       `-- Selected server startup script
|           `-- Java / Minecraft
|
`-- bonghos-playit.service (optional, started on demand)
    `-- Bonghos credential bridge
        `-- official playitd
```

The control panel and Minecraft run as a normal Linux user. Bonghos does not require root.

### Process ownership

systemd and the Bonghos supervisor own Minecraft's lifecycle. The optional tmux session is only a console client:

```text
bonghos console
    `-- tmux session named "bonghos"
        `-- console client
            `-- framed Unix-domain socket
                `-- Bonghos supervisor
                    `-- Minecraft standard input and output
```

Killing the tmux session does not stop, restart, or signal Minecraft. Minecraft can run without tmux installed.

### Technology stack

| Layer | Choice |
|---|---|
| Backend, CLI, and supervisor | Go 1.26.6 |
| Database | SQLite with WAL, foreign keys, and versioned migrations |
| Frontend | Dependency-free HTML, CSS, and JavaScript embedded with Go `embed` |
| Process management | systemd user services |
| Console transport | Framed Unix-domain socket |
| Backups | `tar.zst`, with `tar.gz` fallback |

Production ships as one executable with the Web UI compiled into it. There is no Node.js, npm, or separate frontend process at runtime.

Archive imports use built-in readers for `.zip`, `.tar`, `.tar.gz`, and `.tar.zst`. Bonghos does not delegate extraction to an external archive utility.

The frontend deliberately differs from the earlier React/TypeScript/Vite/Tailwind proposal. The dependency-free implementation builds reproducibly without a JavaScript supply chain, but currently has no Playwright or equivalent browser automation. The rationale and frontend workflow are documented in [source/web/README.md](source/web/README.md).

## Runtime data layout

The source checkout and installed runtime are separate:

```text
~/bonghos-source/       cloned or extracted source
~/bonghos/              installed runtime and user data
```

The runtime root has three top-level areas:

```text
<BONGHOS_HOME>/
|-- servers/            Minecraft worlds, mods, configs, and scripts
|-- backups/            portable backup archives
`-- system/             executable, config, database, logs, runtime, and temp data
```

`backups/` is the default archive location. An operator can configure an
absolute backup directory outside `BONGHOS_HOME`; those archives remain
available to Bonghos but are intentionally excluded from the Bonghos
disk-usage total, which measures only files physically inside the runtime root.

Minecraft files remain normal files. They can be edited over SSH, SFTP, or directly on disk. Bonghos watches for outside changes and does not use SQLite as a second source of truth for server files.

SQLite stores Bonghos metadata such as users, projects, schedules, audit records, and operational state. Backups are normal archives that can be extracted without Bonghos, for example:

```bash
tar --zstd -xf 2026-08-02_04-00-00_full.tar.zst
```

### Runtime state reconciliation

The supervisor process and its verified Java child determine whether Minecraft
is online. A persisted PID or phase alone is not treated as proof that the
server is still running. When Minecraft stops, crashes, restarts, or leaves
stale supervisor state, Bonghos closes active player sessions, clears Java PID
and uptime from monitoring samples, records one terminal lifecycle event, and
broadcasts refreshed server and player state to the Overview and Players pages.

### Runtime location resolution

Bonghos resolves its runtime directory in this order:

1. The global `--home DIR` CLI flag
2. `BONGHOS_HOME`
3. `$HOME/bonghos`

Examples:

```bash
bonghos --home /mnt/storage/bonghos doctor
BONGHOS_HOME=/mnt/storage/bonghos bonghos doctor
```

The installer creates `~/.local/bin/bonghos` as a managed launcher instead of a simple symlink. This lets custom runtime locations remain associated with the short command. Repair recreates the launcher; uninstall removes it only if Bonghos owns it.

## Portability and migration

Paths stored by Bonghos are relative to the runtime root where possible, so the runtime can be moved or copied.

### Move to another directory

```bash
systemctl --user stop bonghos-minecraft.service bonghos.service
mv ~/bonghos /mnt/storage/bonghos
/mnt/storage/bonghos/system/bin/bonghos --home /mnt/storage/bonghos doctor --repair
/mnt/storage/bonghos/system/bin/bonghos --home /mnt/storage/bonghos service repair
```

### Copy to another Linux machine

```bash
rsync -a ~/bonghos/ user@new-host:~/bonghos/
ssh user@new-host
~/bonghos/system/bin/bonghos doctor --repair
~/bonghos/system/bin/bonghos service repair
```

Because the account database and `system/config/secret.key` travel with the runtime, accounts, authenticator secrets, passkey records, notification bot credentials, and a managed Playit credential remain usable after migration when the WebAuthn origin is unchanged. Losing `secret.key` makes those encrypted credentials unrecoverable.

Portable exports provide an alternative:

```bash
bonghos export --output bonghos-export.tar.zst
bonghos import <archive.tar.zst>
```

Exports exclude the account database and `secret.key` unless `--include-secrets` is supplied. An export containing secrets must be stored as carefully as the live runtime.

### Update safety

`./setup.sh --update` builds and tests in a temporary directory before touching the installed executable. It checks the database, preserves a safety copy of critical configuration and data, replaces the executable atomically, repairs paths and services, and rolls back if post-update health checks fail.

`./setup.sh --update --pull` accepts only a clean Git worktree and fast-forward-only history. It does not run `reset --hard`, `clean -fd`, `stash`, an automatic merge, or a rebase. Local changes or diverged history stop the update for manual review.

Updates preserve `servers/`, `backups/`, `bonghos.toml`, `secret.key`, `bonghos.db`, and `logs/`.

## Accounts and authorization

There is no public registration. Setup creates the first Owner. By default,
Owners and Admins invite other users through single-use activation links;
delegated `users.manage` access follows the fixed role hierarchy.

| Capability | Owner | Admin | Member | Viewer |
|---|:---:|:---:|:---:|:---:|
| View status and players | Yes | Yes | Yes | Yes |
| View detailed Performance page | Yes | Yes | No | No |
| Run a manual internet speed test | Yes | Yes | No | No |
| Start, stop, and restart | Yes | Yes | Yes | No |
| View console | Yes | Yes | No | Yes |
| Use console | Yes | Yes | No | No |
| Force stop | Yes | Yes | No | No |
| Manage players | Yes | Yes | No | No |
| Edit files, configuration, and JVM settings | Yes | Yes | No | No |
| Import, upload, and download projects | Yes | Yes | No | No |
| Create backups, restore, and manage schedules | Yes | Yes | No | No |
| Manage notification bots | Yes | Yes | No | No |
| Manage users | Yes | Limited | No | No |
| Manage role permissions | Yes | No | No | No |
| View activity and host/service details | Yes | Yes | No | No |
| Manage personal security and appearance | Yes | Yes | Yes | Yes |

These are defaults. The Users page includes a persisted role-permission manager
for Admin, Member, and Viewer. Owner permissions are fixed and cannot be
changed. An Owner may grant `roles.manage` to Admin; an authorized Admin may
then edit Member and Viewer only and cannot grant a permission the Admin does
not hold. Viewer is structurally read-only: its profile may contain only view
permissions, while all operational and management permissions are disabled in
the editor and rejected by the API. User administration follows the same
hierarchy, so delegated managers act only on lower roles. The final active
Owner cannot be disabled,
demoted, or deleted. All of these rules are enforced in backend authorization
checks, not only hidden in the UI.

The backend owns the authoritative permission catalog with labels, assignment limits,
and prerequisites. For example, using the console requires console visibility,
and managing players requires player visibility. A customized role is stored as
an explicit allow/deny snapshot in SQLite migration
`0017_role_permission_profiles.sql`; newly introduced permissions therefore
remain denied until deliberately granted. Each role profile has a revision, so
simultaneous edits fail with HTTP `409` instead of silently overwriting another
administrator's changes. Successful edits record exact grants and revocations
in the audit log and disconnect affected live WebSocket sessions.

Overview is permission-filtered rather than treated as a shortcut around the
catalog. Basic `server.view` access returns project and lifecycle status;
performance samples and history, backup metadata, player details, and schedule
metadata require their corresponding permissions. Live Overview telemetry uses
its own performance-authorized WebSocket topic.

Every account enrolls in TOTP. Recovery codes are one-use fallback credentials, stored as hashes and shown in plaintext only when generated. Passkeys use WebAuthn and are scoped to the site origin on which they were registered.

## Networking model

Bonghos binds to `127.0.0.1` by default. It does not modify firewall rules,
router settings, reverse proxies, Cloudflare Tunnel, Tailscale, or VPN
configuration. Player networking may remain direct/manual or use the optional
Playit.gg integration. Existing databases default to direct/manual; new guided
setups offer Playit first but require explicit browser approval.

The Playit integration stores one global configuration because Bonghos runs one
active project at a time. The agent credential is AES-256-GCM encrypted with
`secret.key` and is never returned to the Web UI. When the managed agent runs,
the credential is materialized only in a mode-0600 runtime file and removed
when the daemon exits. Bonghos uses the official `playitd` executable and a
separate `bonghos-playit.service`; it does not download or replace agent
software. The Web UI also performs read-only detection of the official system
service, user service, Docker/Podman images, and Playit processes without
reading container environments or process secrets. An externally managed
agent remains outside Bonghos lifecycle control.

Playit claim and tunnel requests use fixed HTTPS API origins, bounded response
sizes and timeouts, and `Agent-Key` authentication only after a credential has
been encrypted. The `playit.manage` permission is Owner-only by default and
can be delegated to Admin by an Owner. Tunnel configuration always targets
`127.0.0.1` and the active project's Minecraft port.

For remote administration, an operator can use an SSH tunnel:

```bash
ssh -L 8080:127.0.0.1:8080 user@your-server
```

The panel can report whether Minecraft appears to be listening locally, but local listening does not prove that a game port is reachable from the public internet.

While an authorized operator has the Performance page open, its selected
update interval drives lightweight TCP connection checks against Cloudflare
and Google plus DNS and HTTPS diagnostics. A single failed round is reported
as degraded; Bonghos reports offline only after three consecutive rounds where
neither target is reachable. The backend refreshes one shared snapshot only
when it is stale, so additional dashboards reuse the result instead of
multiplying outbound probes. Leaving Performance stops automatic Internet
checks. The separate manual speed test uses Cloudflare's public speed-test endpoints and
transfers up to about 53 MB across its download and upload rounds. It never
runs automatically, requires `server.performance.test`, permits only one test
at a time, and may briefly reduce bandwidth available to a running server.

When Bonghos is placed behind a reverse proxy or tunnel, the operator is responsible for TLS, stable origin configuration, trusted proxy settings, access policy, and upload limits. WebAuthn passkeys are bound by browser standards to their relying-party ID and origin; changing the panel's hostname or IP can require registering another passkey at the new origin.

## Security design

- Passwords are hashed with Argon2id.
- TOTP secrets, notification bot tokens, and the managed Playit agent credential are encrypted with authenticated encryption.
- Recovery codes are stored as one-way hashes and can be used once.
- Passkeys use WebAuthn challenge-response verification.
- Login performs dummy verification work and returns non-enumerating errors.
- Authentication endpoints and other sensitive operations are rate limited.
- Optional Cloudflare Turnstile protects password and passkey sign-in without
  requiring member email addresses. The public endpoint returns only the site
  key; the secret is AES-256-GCM encrypted in SQLite and every token is checked
  server-side against Siteverify, the `login` action, and the request hostname.
  Only the immutable Owner role can change this setting. The local
  `bonghos security turnstile disable` command provides lockout recovery.
- Sessions are server-side; cookies use HttpOnly and SameSite protections.
- State-changing browser requests require CSRF tokens.
- Responses include a Content Security Policy and other security headers. The
  CSP permits scripts and frames from `https://challenges.cloudflare.com` only
  for the optional Turnstile integration; other external scripts remain blocked.
- File access uses canonical path containment rather than string-prefix checks.
- File browsing is jailed to `<BONGHOS_HOME>/servers`. The servers root and
  non-project directories are browse-only; mutations require a recognized
  managed project. Cross-project copy and move operations validate both source
  and destination containment on the backend.
- Archive extraction rejects traversal, absolute paths, symlink and hard-link escapes, decompression bombs, and configured size/file-count excesses.
- Server-side URL downloads enforce SSRF protections, including address validation, redirect revalidation, HTTPS defaults, and size/disk-space limits.
- Process launch uses argument arrays rather than concatenated shell commands.
- The Web UI console talks only to the selected Minecraft process; it is not a Linux shell.
- Audit records omit passwords, TOTP codes, recovery codes, session cookies, encryption keys, bot tokens, and sensitive URL parameters.

Private vulnerability reports should follow [.github/SECURITY.md](.github/SECURITY.md).

## Repository layout

```text
Bonghos/
|-- setup.sh                 install, build, update, repair, and uninstall
|-- README.md                user introduction and command reference
|-- TECHNICAL.md             this technical reference
|-- Tutorial.txt             complete operator walkthrough
|-- LICENSE                  AGPL-3.0-only
|-- scripts/                 repository maintenance and integration helpers
|-- .github/                 CI, release, security, and contribution files
`-- source/
    |-- cmd/bonghos/         executable entry point and CLI
    |-- internal/            implementation packages
    |-- migrations/          versioned SQLite migrations
    |-- web/src/             frontend source
    |-- deploy/              systemd unit templates
    |-- docs/                changelog
    |-- third_party/         maintained local dependency replacements
    `-- Makefile             development commands
```

Normal operators should not need to work inside `source/`.

## Development

From `source/`, use the existing Makefile targets:

```bash
make web          # rebuild the embedded frontend assets
make fmt-check    # check Go formatting
make vet          # run go vet
make test         # run the test suite
make build        # build ./bin/bonghos
make run          # build and run with BONGHOS_HOME=./devhome
make fmt          # format the Go tree
make clean        # remove build artifacts
```

Or prepare and verify the development environment from the repository root:

```bash
./setup.sh --dev
```

### Release preparation

Before creating a stable tag, update the version consistently in `setup.sh`,
`source/cmd/bonghos/main.go`, `source/Makefile`, `source/internal/app/app.go`,
the demo values in `source/web/src/app.js`, `README.md`, `Tutorial.txt`, and the
validation-status heading in this file. Move the completed changelog entries
from **Unreleased** into a dated version section and add a fresh **Unreleased**
section.

Run every development check above, then validate a clean installation and an
in-place update on Linux. Because browser automation is not yet present, the
stable-release pass should manually cover login and TOTP, the Web service and
lingering status, server start/stop/crash state, player reconciliation,
console input, file containment and cross-project operations, backup creation
and restore, external backup storage, schedules, and both bot providers. Push
the tested commit before creating and pushing its annotated `vX.Y.Z` tag; the
tag-triggered release workflow builds, tests, scans, and publishes the AMD64,
ARM64, source, and checksum assets.

The frontend has no JavaScript build tool. Edit `source/web/src/`, then run `make web` before Go builds so the generated embedded assets are current.

Notification bots use the host's system DNS by default. Each Telegram or
Discord bot can instead be given an explicit resolver IP in its advanced
settings. The resolver is used only for that bot's provider connections, and
Bonghos never enables it automatically. It must provide conventional DNS on
port 53. Restricted networks can also override Discord endpoints with
`BONGHOS_DISCORD_BASE_URL`, `BONGHOS_DISCORD_FALLBACK_BASE_URL`, and
`BONGHOS_DISCORD_GATEWAY_URL`. The fallback endpoint is opt-in; idempotent REST
calls can retry safe transport failures, while ambiguous message or callback
POST failures are not retried unless DNS failed before connecting.

For Windows-only notification debugging, copy `.env.development.example` to the ignored
`.env.development`, configure at most one Telegram bot token and one Discord bot
token, then run
`node scripts/dev-web.js`. The loopback-only relay keeps provider
tokens server-side while `http://127.0.0.1:8000/?demo&debug-bots` uses the normal
mock UI for everything else. Telegram and Discord destinations are connected
with `/bonghos here` inside the target topic or channel. Discord interactions
arrive through an outbound Gateway WebSocket, so development does not require a
public callback URL. See [source/web/README.md](source/web/README.md).

The SQLite driver uses cgo. A `CGO_ENABLED=0` binary may link successfully but cannot open the database, so production builds must keep cgo enabled. Cross-building ARM64 from x86 additionally requires an ARM64 C compiler such as `gcc-aarch64-linux-gnu`.

### Web UI branch integration

Use the guarded helper instead of hand-merging the `webui` branch:

```bash
./scripts/integrate-webui.sh
```

With no flags, the helper fetches refs, tests the merge in a temporary worktree, validates it, and exits without updating `main`, pushing, or installing anything. Additional modes are:

```bash
./scripts/integrate-webui.sh --apply
./scripts/integrate-webui.sh --apply --push
./scripts/integrate-webui.sh --apply --push --install
./scripts/integrate-webui.sh --race
```

If conflicts or validation failures occur, the helper leaves `main` untouched and preserves the temporary worktree for inspection.

See [.github/CONTRIBUTING.md](.github/CONTRIBUTING.md) before contributing.

## Validation status and limitations

This section describes the v0.3.0-rc.1 prerelease.

### Automated coverage

- Unit tests cover canonical path containment, archive safety, authenticated encryption, TOTP against the RFC 6238 vector, role permissions, scheduler next-run calculation, SSRF validation, slug generation, running-process detection, external backup storage, notification destinations, and shutdown-state reconciliation.
- API tests drive the real HTTP handler through `httptest`, including login anti-enumeration behavior, CSRF rejection, session revocation, disabled accounts, Member/Viewer restrictions, Owner protections, restore safety/scope handling, bot management, and cross-project file operations without access outside managed projects.
- Linux validation launches OpenJDK to check Java-process discovery and cleanup, but uses synthetic server-pack fixtures rather than a complete modded server.
- Generated service units are checked with `systemd-analyze verify`, including custom runtime paths containing spaces.
- The cgo-enabled ARM64 release binary is executed under ARM64 emulation and opens its SQLite database.

### Remaining limitations

- There are no automated browser-level tests. Web UI behavior is still verified manually.
- Supervisor crash/restart/backoff behavior, boot autostart with live systemd user managers, unclean-shutdown recovery, tmux lifecycle, archive import, scheduled execution, and retention pruning need more real-system integration coverage.
- A complete modded server has not yet been validated across long-running real hardware scenarios.
- The ARM64 build has not yet been tested on physical ARM hardware.
- Interrupted URL downloads restart from zero; HTTP range resume is not implemented.

Treat a new deployment as unproven for irreplaceable worlds until you have independent backups and have tested a restore yourself.

## Roadmap boundaries

Deferred features that the data model is intended to accommodate include:

- CurseForge browsing, search, API keys, and project-link resolution
- Modrinth browsing
- Vanilla Java and Bedrock server types
- Multiple simultaneously running servers and multiple physical nodes
- In-game TPS, MSPT, JVM heap, chunk, and entity metrics through an optional mod
- HTTP range-based download resumption

The source-type and server-type models are already generalized for `curseforge`, `modrinth`, `manual-upload`, `direct-url`, and `existing-directory` sources, and for modded Java, vanilla Java, and vanilla Bedrock servers. Supporting those deferred types should not require replacing the core schema.

Not planned: Docker or containers, billing, public registration, browser shell access, automatic network configuration, or required telemetry.

## Acknowledgements and license

Bonghos learns from the usability and reliability of BisectHosting, Hostinger, Apex Hosting, Shockbyte, Crafty Controller, and Pterodactyl. It copies none of their branding, proprietary assets, or layouts.

Bonghos is licensed under [AGPL-3.0-only](LICENSE). If you run a modified version as a network service, the AGPL requires you to offer its users the corresponding source code.
