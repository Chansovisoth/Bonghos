# Bonghos

**A free-forever, open-source, self-hosted web control panel for modded Minecraft servers on Linux.**

*Bonghos* (បង្ហោះ) is the Khmer word for **hosting**.

- Repository: <https://github.com/Chansovisoth/Bonghos>
- License: AGPL-3.0-only
- Status: v0.1.1 — early, actively developed. See [Known limitations](#known-limitations).

Bonghos imports, configures, runs, monitors, schedules, backs up, restores and manages
Minecraft Java Edition modded servers directly on Linux — no Docker, no cloud account,
no subscription, no telemetry.

---

## Quick start

```bash
git clone https://github.com/Chansovisoth/Bonghos.git ~/bonghos-source
cd ~/bonghos-source
./setup.sh
```

That's the whole installation. The guided installer checks dependencies, builds
everything, creates your Owner account with two-factor authentication, and offers to
install the systemd services.

Then open <http://127.0.0.1:8080>.

**New here? Read [`Tutorial.txt`](Tutorial.txt)** — it has complete copyable instructions
for everything from installation to backups to migration.

---

## Free forever

Bonghos is and will remain:

- Free, with no paid tier, license key or subscription
- Open source under AGPL-3.0-only
- Usable with no Bonghos account, no advertisements and no required telemetry
- Usable with no external database

Any future telemetry would be disabled by default and strictly opt-in.

---

## Features

**Server projects**
- Multiple stored projects; one runs at a time (v1)
- Add packs by drag-and-drop upload, file picker, direct server-side URL download,
  local archive, or importing an existing directory
- Streaming uploads with progress, speed, ETA and cancellation
- Server-side downloads that continue after you close the browser
- Safe extraction with traversal, symlink, size and file-count protection
- Automatic startup-script and JVM-configuration detection
- 64×64 PNG server icons

**Running servers**
- Start, graceful stop, restart and force stop
- Live console in the Web UI, in the terminal, or via an optional tmux session
- Java selection and safe RAM/JVM argument editing
- Restart policies with backoff and crash-loop protection
- Autostart after reboot with unclean-shutdown recovery

**Operations**
- Process and host monitoring with historical charts
- Online and historical player lists; kick, ban, whitelist and operator management
- Persistent schedules that run without a browser (start/stop/restart/commands/
  broadcasts/saves/backups), with timezones and multi-step warning sequences
- Full, world-only and configuration-only backups, online or offline
- Backup verification, retention policies and restore (including restore-as-new)
- Constrained file manager scoped to the server directory

**Accounts and security**
- Multiple accounts with **mandatory** TOTP two-factor authentication, enrolled
  by scanning a QR code in the terminal or the Web UI (with the secret and
  `otpauth://` URI always shown as a fallback)
- Owner / Admin / Member / Viewer roles enforced in the backend
- Admin-created invitations; no public registration
- Argon2id passwords, encrypted TOTP secrets, hashed one-use recovery codes
- Anti-enumeration login, rate limiting, CSRF protection, audit logging

**Portability**
- One self-contained runtime directory you can move or copy anywhere
- `doctor --repair` re-resolves everything after a move
- Portable export/import archives

---

## Architecture: no Docker, no containers

Bonghos runs **natively** on Linux. There is no Dockerfile, no Compose file, no
Kubernetes manifest, and no container runtime anywhere in the project.

```
systemd user manager
    │
    ├── bonghos.service
    │       └── Bonghos control plane
    │           ├── Embedded Web UI
    │           ├── REST API + WebSocket API
    │           ├── Authentication and authorization
    │           └── Scheduler, imports, backups, monitoring
    │
    └── bonghos-minecraft.service
            └── Bonghos supervisor
                └── Selected modpack startup script
                    └── Java / Minecraft
```

**systemd and the supervisor own Minecraft's lifecycle — not tmux.**

tmux is an *optional console client*, created lazily only when you run
`bonghos console`. Killing the tmux session, or the whole tmux server, does not stop,
restart or signal Minecraft. Minecraft runs fine with tmux not installed at all.

Optional console access:

```
bonghos console  →  tmux session `bonghos`  →  console client
                                                    ↓
                                        supervisor Unix socket
                                                    ↓
                                        running Minecraft console
```

Bonghos and Minecraft run as a **normal Linux user**. Root is never required.

---

## Technology stack

| Layer | Choice |
|---|---|
| Backend, CLI, supervisor | Go 1.22+ |
| Database | SQLite (WAL, foreign keys, versioned migrations) |
| Frontend | Dependency-free HTML/CSS/JavaScript, embedded via Go `embed` |
| Process management | systemd **user** services |
| Console transport | Framed Unix-domain socket |
| Backups | `tar.zst` (falls back to `tar.gz`) |

Production ships as **one executable** with the Web UI compiled in. No Node.js, no
npm, no separate frontend process at runtime.

> **Known deviation — the frontend stack.** The specification asked for React,
> TypeScript, Vite, Tailwind, shadcn-style components, TanStack Query, Motion, Recharts,
> Vitest, React Testing Library, Playwright and pnpm. **None of that is present.** The
> frontend is dependency-free HTML, CSS and JavaScript, chosen so the project builds
> reproducibly offline with no JavaScript supply chain. The rationale is in
> [`source/web/README.md`](source/web/README.md).
>
> This is a real trade-off, not a free win. The visual design, layout, motion and
> accessibility goals are met, and there is no build step to break — but the project
> also has **no browser-level tests**, because Vitest, RTL and Playwright came with the
> stack that was dropped and nothing replaced them. A frontend/backend protocol
> mismatch that silently disabled every live update shipped in v0.1.0 for exactly this
> reason. If you want the specified stack, this is the piece to revisit first.

---

## Repository layout

```
Bonghos/
├── setup.sh          Install, build, update, repair, uninstall
├── Tutorial.txt      Complete instructions for users
├── README.md         This file
├── LICENSE           AGPL-3.0-only
├── .github/          Community files, issue templates, CI
└── source/           Everything developers need
    ├── cmd/bonghos/  Executable entry point
    ├── internal/     Implementation packages
    ├── migrations/   Versioned SQLite migrations
    ├── web/src/      Frontend source
    ├── deploy/       systemd unit templates
    ├── docs/         CHANGELOG
    └── Makefile      Developer commands
```

Normal users never need to look inside `source/`.

---

## Installation

### From Git (recommended)

```bash
git clone https://github.com/Chansovisoth/Bonghos.git ~/bonghos-source
cd ~/bonghos-source
./setup.sh
```

### From an extracted source archive

```bash
cd ~/Downloads/Bonghos-0.1.1
chmod +x setup.sh
./setup.sh
```

`setup.sh` and `Tutorial.txt` are at the top level of the archive.

### Options

```bash
./setup.sh                  # guided install
./setup.sh --dev            # verify development dependencies
./setup.sh --build          # build and test without installing
./setup.sh --update         # install the source present here
./setup.sh --update --pull  # fast-forward Git, then update
./setup.sh --repair         # repair installation and services
./setup.sh --uninstall      # remove services and executable, keep data
./setup.sh --home DIR       # custom runtime location
./setup.sh --help
```

### Runtime dependencies

Required: a 64-bit Linux system, Java 17 or 21 (for Minecraft), `tar`, `gzip`.
Optional: `tmux` (console), `unzip`, `xz-utils`, `zstd`, `p7zip-full`, `unrar`.

Build-time only: Go 1.22+, Git, and a C compiler (`gcc`, usually already
present). The SQLite driver is a cgo package, so `CGO_ENABLED=0` builds link
successfully but fail at the first database access — keep cgo enabled. Building
for ARM64 from an x86 machine additionally needs `gcc-aarch64-linux-gnu`.

---

## Directory structure

Keep the source checkout and the installed runtime separate:

```
~/bonghos-source/     cloned or extracted source (the code)
~/bonghos/            installed runtime (your data)
```

The runtime root contains exactly three things:

```
~/bonghos/
├── servers/     real Minecraft files — worlds, mods, configs, scripts
├── backups/     portable backup archives
└── system/      Bonghos internals (bin, config, database, logs, runtime, temp)
```

Minecraft files stay **normal files**. Edit them over SSH, SFTP or with any editor —
Bonghos watches for external changes and will not overwrite your manual edits. Backups
are plain archives you can extract without Bonghos:

```bash
tar --zstd -xf 2026-08-02_04-00-00_full.tar.zst
```

SQLite stores only Bonghos metadata: users, schedules, audit records and operational
state. It is never a second source of truth for Minecraft files.

### Custom location

```bash
./setup.sh --home /mnt/storage/bonghos
# or
export BONGHOS_HOME=/mnt/storage/bonghos
```

Resolution order: `--home`, then `BONGHOS_HOME`, then `$HOME/bonghos`.

---

## Moving and migrating

Paths are stored relative to the runtime root, so the whole directory is portable.

**To another directory:**

```bash
systemctl --user stop bonghos-minecraft.service bonghos.service
mv ~/bonghos /mnt/storage/bonghos
/mnt/storage/bonghos/system/bin/bonghos --home /mnt/storage/bonghos doctor --repair
/mnt/storage/bonghos/system/bin/bonghos --home /mnt/storage/bonghos service repair
```

**To another computer:**

```bash
rsync -a ~/bonghos/ user@new-host:~/bonghos/
~/bonghos/system/bin/bonghos doctor --repair
```

Because `system/config/secret.key` travels with it, accounts and authenticator apps
keep working. **Losing `secret.key` makes encrypted data permanently unrecoverable.**

Or use portable exports:

```bash
bonghos export --output bonghos-export.tar.zst
bonghos import <archive>
```

---

## Updating

Bonghos updates from source. There is no Web UI button that silently downloads and
installs unverified code.

```bash
# Retrieve source and update in one command (clean Git checkout)
cd ~/bonghos-source
./setup.sh --update --pull

# Retrieve manually, review, then update
cd ~/bonghos-source
git pull --ff-only
./setup.sh --update

# From a newly extracted source archive (no Git metadata)
cd ~/Downloads/Bonghos-0.2.0
./setup.sh --update
```

Understanding the difference:

| Command | What it does |
|---|---|
| `git pull --ff-only` | Updates the local source checkout only |
| `./setup.sh --update` | Builds and installs the source present here |
| `./setup.sh --update --pull` | Fast-forwards a clean checkout, then updates |

Updates build and **run the tests in a temporary area first** — a failure stops the
update before anything installed is touched. The executable is replaced atomically and
rolled back automatically if health verification fails.

`--pull` is fast-forward-only and **never** runs `reset --hard`, `clean -fd`, `stash`,
an automatic merge or a rebase. If you have local changes or diverged history, it stops
and shows you the commands to inspect the situation yourself.

Always preserved: `servers/`, `backups/`, `bonghos.toml`, `secret.key`, `bonghos.db`,
`logs/`.

---

## Users and roles

No public registration. The first Owner is created during setup; everyone else is
invited by an Owner or Admin.

| | Owner | Admin | Member | Viewer |
|---|:--:|:--:|:--:|:--:|
| View status, players | ✓ | ✓ | ✓ | ✓ |
| Start / stop / restart | ✓ | ✓ | ✓ | |
| Console (view / use) | ✓ | ✓ | | view |
| Force stop | ✓ | ✓ | | |
| Manage players | ✓ | ✓ | | |
| Files, configuration, JVM | ✓ | ✓ | | |
| Import / upload / download | ✓ | ✓ | | |
| Backups, restore, schedules | ✓ | ✓ | | |
| Manage users | ✓ | partial | | |
| Security, host, portability | ✓ | | | |

Admins cannot modify, demote, delete or create Owners, and nobody can raise their own
role. The last active Owner can never be deleted or demoted. **All of this is enforced
in the backend, not merely hidden in the interface.**

TOTP is mandatory for every account, with its own secret and one-use recovery codes.
Login errors are identical whether the username, password or code was wrong.

---

## Networking is your responsibility

Bonghos binds to `127.0.0.1` by default and **never** configures port forwarding,
firewall rules, routers, tunnels, Cloudflare Tunnel, playit.gg, Tailscale, VPNs or
reverse proxies. That stays entirely with you.

For remote access, tunnel over SSH from your own machine:

```bash
ssh -L 8080:127.0.0.1:8080 user@your-server
```

Bonghos shows its listening address and whether Minecraft appears to be listening
locally — but local listening never proves public reachability.

---

## Security

- Argon2id password hashing; TOTP secrets encrypted with authenticated encryption
- Anti-enumeration login with dummy verification work and rate limiting
- Server-side sessions, HttpOnly + SameSite cookies, CSRF tokens, security headers, CSP
- Canonical path containment everywhere (never string-prefix comparison)
- Safe archive extraction: traversal, absolute paths, symlink and hard-link escapes,
  decompression bombs, file-count and size limits
- SSRF protection on URL downloads: HTTPS by default, blocked loopback/private/
  link-local/metadata addresses, redirect revalidation, size and disk-space limits
- Argument arrays instead of shell string concatenation; **no arbitrary shell execution
  anywhere**, and the Web UI console is never a Linux shell
- Audit logging that never records passwords, TOTP codes or secrets, session cookies,
  encryption keys or sensitive URL parameters

Report vulnerabilities privately — see [`.github/SECURITY.md`](.github/SECURITY.md).

---

## Development

```bash
cd source
make build     # build into ./bin/bonghos
make test      # run the test suite
make vet       # go vet
make run       # run with BONGHOS_HOME=./devhome
make fmt       # gofmt
```

Or verify your environment first with `./setup.sh --dev`.

The frontend has no JavaScript build step: edit `source/web/src/` and rebuild to
re-embed. Dependencies are vendored under `source/third_party/`, so the project builds
offline (`GOPROXY=direct`).

To safely merge the `webui` branch into `main`, use the guarded integration helper:

```bash
./scripts/integrate-webui.sh
```

With no flags, it fetches the latest refs, tests the merge in a temporary worktree,
runs validation, and exits without changing `main`, pushing, or installing anything.
Use `--apply` to update local `main`, add `--push` to publish it, and add `--install`
to install/restart the local `~/bonghos` service. If conflicts occur, `main` is
untouched and the script leaves the temporary worktree in place for manual resolution.

Contributions welcome — see [`.github/CONTRIBUTING.md`](.github/CONTRIBUTING.md).

---

## Known limitations

This is v0.1.1. Being honest about what is and is not proven:

- **Covered by unit tests** (`source/internal/*/`): canonical path containment,
  archive-extraction safety, authenticated encryption, TOTP against the RFC 6238
  vector, the role permission matrix, scheduler next-run calculation, SSRF URL
  validation, slug generation, and running-process detection.
- **Covered by API tests** (`source/internal/app/`): these drive the real HTTP handler
  through `httptest` — login, resistance to account enumeration, CSRF rejection,
  session revocation, disabled accounts, the exact Member and Viewer restrictions,
  Owner protections, and restore safety and scope handling.
- **Exercised manually, not yet under automated integration tests:** the supervisor's
  crash/restart/backoff behaviour against real Minecraft, boot autostart and
  unclean-shutdown recovery, tmux console lifecycle, archive import end to end,
  scheduled execution, and retention pruning.
- **No browser-level tests.** There is no Playwright or equivalent suite, so the Web UI
  is verified by hand. A subscription-key mismatch that silently disabled every live
  update survived release precisely because nothing tested the browser side; treat UI
  behaviour as the least-proven part of the project.
- **Not yet verified against a real modded server on real hardware.** The test fixtures
  are synthetic.
- systemd integration is implemented but was validated in an environment without a live
  user manager.
- ARM64 builds are supported by the toolchain but have not been run on ARM hardware.
- URL downloads restart from zero if interrupted (range-resume is designed for, not
  implemented).

Do not treat this as production-ready for a server you care about until you have taken
your own backups and tested a restore.

---

## Roadmap

**Deferred from v1, kept extensible:**

- CurseForge browsing, search, API keys and project-link resolution
- Modrinth browsing
- Vanilla Java and Bedrock server types
- Multiple simultaneously running servers, multiple physical nodes
- In-game metrics (TPS, MSPT, JVM heap, chunk and entity counts) via an optional mod
- HTTP range-based download resumption

The source-type and server-type models are already generalized
(`curseforge`/`modrinth`/`manual-upload`/`direct-url`/`existing-directory`;
`minecraft-java-modded`/`minecraft-java-vanilla`/`minecraft-bedrock-vanilla`) so these
can be added without redesigning the schema.

**Never planned:** Docker or containers, billing, public registration, browser shell
access, automatic network configuration, required telemetry.

---

## Acknowledgements

Bonghos learns from the usability and reliability of BisectHosting, Hostinger, Apex
Hosting, Shockbyte, Crafty Controller and Pterodactyl. It copies none of their branding,
proprietary assets or layouts.

It grew out of a hand-rolled tmux autostart script — which is exactly what the
systemd + supervisor architecture replaces.

---

## License

AGPL-3.0-only. See [`LICENSE`](LICENSE).

If you run a modified Bonghos as a network service, the AGPL requires you to offer your
users the corresponding source code.
