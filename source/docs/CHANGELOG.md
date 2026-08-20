# Changelog

All notable changes to Bonghos are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Schedules can be duplicated into a prefilled new-schedule form without
  changing the original schedule.
- Once, hourly, daily, weekly, and monthly schedules now accept seconds while
  remaining compatible with existing minute-only schedules.
- Truncated player names now scroll into view while hovered or focused, and
  mobile users can toggle the ticker by tapping the player row.
- Owners and delegated Admins can rename a Bonghos-linked Playit agent from
  the Settings editor, including while the integration is switched off.
- Player managers can whitelist or unwhitelist observed players directly from
  the Players page, with current whitelist state shown in each player row.
  Operator and ban actions also switch between Op/Deop and Ban/Unban according
  to the player's current state.

### Fixed

- Managed Playit startup now retries when the official agent is installed
  after Bonghos starts, using the same service reconciliation path as the
  Settings switch instead of requiring an off/on toggle.
- Bonghos now conservatively adopts its single matching remote Playit tunnel
  when the local tunnel ID is missing, including during startup and after an
  incompatible create response. Pending tunnels are refreshed until their
  public address becomes available; unrelated or duplicate tunnels are never
  adopted automatically.
- A uniquely matching Playit tunnel created manually can now be adopted by
  linked agent, tunnel type, and active local port without requiring Bonghos's
  generated tunnel name. Provider failures during tunnel and agent operations
  are logged safely, and rename rejections are returned as conflicts instead
  of generic gateway failures.
- The managed Playit account details now show the configured public address in
  addition to the existing Overview display.
- Playit API failures now preserve the provider's structured error category
  even when Playit responds with a non-success HTTP status, so expired links,
  account restrictions, unsupported operations, and provider failures produce
  actionable messages. Sanitized validation details identify rejected fields
  without exposing credentials. Web UI error toasts also always show a fallback
  message when an upstream or proxy response has no usable error text.
- Playit claims now register the installed official agent version instead of
  the Bonghos version, allowing Playit to recognize supported tunnel types.
  Existing incompatible links receive a relink instruction, and failed
  Playit actions no longer trigger a secondary browser error. Version detection
  uses the official CLI's `version` command, and Overview resize handling no
  longer fails on its player-name selector helper. Playit API tests are also
  isolated from agents installed on the machine running the suite.
- Backup integrity is shown beside the backup ID instead of consuming a
  separate table column.
- Playit now cancels abandoned managed claims when switching to an external
  deployment, preserves disabled configuration consistently in demo mode,
  keeps relink approval visible, stops stale managed services on disabled
  startup, and no longer advertises missing, pending, or disabled tunnels.
- Updating a tunnel that was deleted remotely now creates a replacement instead
  of leaving Bonghos permanently attached to the missing tunnel identifier.
- The in-process Playit fallback now restarts an unexpectedly exited agent after
  five seconds when a managed systemd user service is unavailable.
- Bot-card Remove actions now match the height of the other card actions.
- Player action buttons use stable desktop slots, so state-dependent labels and
  offline rows no longer shift the action columns.
- User, backup, and schedule actions now use stable desktop button columns.

## [0.3.0-rc.1] - 2026-08-19

### Added

- Optional Playit.gg player networking can be selected during new setup and
  configured above Bots in Settings. Owners may link an account or guest agent,
  provision one Minecraft Java tunnel for the active project, show its public
  address, or record an existing externally managed agent. Agent credentials
  are encrypted, existing systemd/Docker/Podman/process deployments are
  detected read-only, and a separate on-demand user service runs the official
  `playitd` executable. Upgraded installations remain on direct/manual
  networking until explicitly enabled, and Owners can delegate the new
  `playit.manage` permission to Admin.
- Owners can optionally protect every password and passkey sign-in with a
  Cloudflare Turnstile Managed challenge without requiring member email
  addresses. Widget secrets are encrypted at rest, tokens are validated
  server-side against hostname and action, the strict CSP allows only
  Cloudflare's challenge origin, and a local CLI command can disable a broken
  configuration to recover from lockout.
- Performance now includes shared host-side TCP, DNS, and HTTPS connectivity
  checks driven by the selected Performance update interval, failure-smoothed
  offline detection, and recent reliability history. Automatic checks run only
  while Performance is open. A separately authorized manual
  Cloudflare speed test reports approximate download and upload Mbps only after
  a bandwidth-impact warning is confirmed; it is serialized, audited, and
  never runs automatically.
- Owners can configure persisted permissions for the Admin, Member, and Viewer
  roles from a new **Role permissions** manager on the Users page. Owner access
  remains immutable. Admins receive role-management access only when an Owner
  grants it, may edit Member and Viewer only, and cannot grant capabilities
  they do not hold. The server-provided catalog defines assignment limits and
  prerequisites; customized roles use fail-closed snapshots, revision checks
  prevent lost updates, exact changes are audited, and affected live sessions
  are refreshed immediately. Viewer remains structurally read-only: only view
  permissions can be selected, while action and management permissions are
  disabled in the editor and rejected by the API.
- Accounts with no server or administration permission retain access to a
  focused Account page, personal security, and appearance settings.

### Fixed

- Java process memory no longer compares RSS to `-Xmx` as though the heap limit
  capped the entire JVM. The meter now shows RSS as a share of machine memory
  and presents `-Xms` and `-Xmx` separately as configured heap limits.
- Overview now returns and renders performance, backup, player, and schedule
  details only when the signed-in role holds the matching permission. Live
  Overview telemetry uses a separate permission-gated WebSocket topic, and
  historical metrics require `server.performance.view` as well.
- Role-permission drafts survive role-tab changes, unchanged saves no longer
  create revisions or disconnect users, and delegated managers cannot apply a
  misleading partial default profile.
- Accounts with backup-view access no longer see verify, protect, or delete
  controls unless they also hold backup-management access.
- Bonghos's periodic internal `list` command and player-count reply no longer
  appear in either the live Console stream or its persisted page history.
  Player tracking still receives the reply, while an operator-issued `list`
  remains visible.

## [0.2.0] - 2026-08-16

### Added

- `bonghos web start|stop|restart|status|logs|enable|disable` provides ordinary
  background Web-panel management without requiring users to remember
  `systemctl` or `journalctl` syntax.
- The Web-panel service now restarts after both failed and unexpected clean
  exits. Explicit `bonghos web stop` and machine shutdown remain intentional
  stops. `web enable` and `web status` also report whether systemd lingering is
  enabled for automatic startup before login after reboot.
- Backup storage can be inspected and relocated with `bonghos backup storage
  show|set|move|reset`. Moves copy and hash-verify every archive, switch the
  configuration only after validation, and require the Web panel to be
  stopped. External archives remain excluded from the literal BONGHOS_HOME
  disk-size metric and are identified separately on the Backups page.
- Backup lists now reflect archives physically present in the active storage
  directory. Moving an archive away hides and disables it without discarding
  its recovery metadata; returning it to its original relative path makes it
  available for verification and restore again.
- Telegram and Discord bot settings include an optional resolver IP for
  networks where the host DNS cannot resolve the provider. It applies only to
  that bot's connections, uses port 53, and is never activated automatically.
- Administrators can view, add, edit, test, invite, enable, and remove
  notification bots through a dedicated bot-management permission without
  receiving Owner-only account-security access.
- Bot invite actions use a joined accent button with a separate one-click
  control for copying the validated Telegram or Discord invite URL.
- Performance shows a Back to Overview button when it was opened from an
  Overview metric card.
- Bot destination cards and edit dialogs show when each Telegram group or
  Discord server was first detected. The timestamp remains stable across
  subsequent provider refreshes.
- Player skins link to the matching NameMC profile when Minecraft supplied a
  valid account UUID.
- Telegram and Discord bot cards now provide an **Invite Bot** action using the
  providers' standard group/server installation links. Discord destinations
  retain and display the server name, server icon, and selected `#channel`.
- Bot edit dialogs refresh command-connected destinations automatically and
  reload Settings when closed, so a destination added with `/bonghos here`
  remains visible without saving unrelated form fields.
- Bot edit dialogs now show Telegram groups and Discord servers as soon as the
  bot joins them. They remain marked **Not configured** until an administrator
  runs `/bonghos here` in the intended topic or channel.
- Discord notification bots now accept a token without a preconfigured channel
  ID. Bonghos registers `/bonghos` slash commands and maintains an outbound
  Gateway connection, so server administrators can connect up to three channels
  with `/bonghos here`, inspect one with `/bonghos where`, disconnect one with
  `/bonghos disconnect`, and see command help with `/bonghos help`. Replies are
  private to the administrator and permissions are verified server-side.
- The loopback development relay now accepts a Discord bot token without a
  preconfigured channel ID, derives and validates its application identity,
  registers `/bonghos` slash commands, and receives interactions through an
  outbound Discord Gateway connection. Server administrators can connect up to
  three channels with `/bonghos here`, inspect them with `/bonghos where`, and
  remove them with `/bonghos disconnect`; replies are ephemeral and permissions
  are enforced again by the relay.
- A loopback-only native Node development relay can load one Telegram bot and
  one Discord bot from the Git-ignored `.env.development` file, allowing real
  notification tests from `?demo&debug-bots` on Windows without exposing bot
  tokens to browser JavaScript.
- Telegram notification destinations are connected from Telegram itself: a
  group administrator runs `/bonghos here` in the topic that should receive
  broadcasts. `/bonghos where` reports the current group destination,
  `/bonghos disconnect` removes it, and `/bonghos help` lists the commands.
  Bonghos consumes commands once using a durable Telegram update cursor and
  supports up to three groups without retaining stale topic-discovery lists.
  The upgrade clears destinations saved by the former dropdown flow once; an
  administrator reconnects each intended group topic with `/bonghos here`.

### Changed

- Running `bonghos` without a subcommand now starts the Web panel in the
  background, matching `bonghos web start`. The explicit `bonghos serve`
  command remains available for foreground and debugging use.
- Builds now select Go 1.26.6, which includes standard-library security fixes
  required by the vulnerability scan.
- `bonghos help` now groups the complete public command surface by purpose and
  spells out server-import, scoped restore/export, and advanced-service forms.
  Internal supervisor and compatibility-alias commands remain intentionally
  hidden from the everyday reference.
- `bonghos owner create` is now the canonical command for creating the first
  Owner account. The former `bonghos admin create` spelling remains as a
  compatibility alias and prints a rename notice.
- Installations now support up to two Telegram and two Discord notification
  bots, with four bots total and up to three destinations per bot. Optional
  replacement-token and DNS fields are collapsed under Show more when
  editing a bot.
- Telegram invite links add bots to groups without requesting administrator
  permissions; only the user running configuration commands must be a group
  administrator.
- Overview metrics and the sidebar's live player count now refresh every four
  seconds. Disk usage remains a cached snapshot measured only when Performance
  is opened or its storage Refresh button is used.

### Fixed

- Settings no longer renders a literal `null` below Theme for accounts that
  cannot manage notification bots. The detailed Performance page, its
  WebSocket topic, and its configuration/storage endpoints are now limited to
  Owners and Admins instead of appearing for Members and Viewers.
- When Minecraft stops, crashes, restarts, or leaves stale supervisor state,
  Bonghos now closes every active player session, reports zero online players,
  rejects stale Java PID and uptime data, and records the terminal lifecycle
  event once. Overview player counts and faces also refresh immediately on
  joins, leaves, reconciled lists, and shutdown.
- The strict script CSP no longer blocks theme initialization, and an optional
  Discord REST fallback uses the legacy API hostname only after safe transport
  failures, without retrying ambiguous message POSTs that could cause
  duplicates. Safe DNS and timeout details are retained without exposing bot
  credentials, and REST and Gateway endpoints can be overridden for restricted
  networks. Testing a bot before connecting a destination now returns a
  conflict instead of misreporting local setup as an upstream gateway failure.
- Start, stop, and restart actions use a continuously animated two-orb loading
  indicator with isolated SVG filter identifiers, while still completing the
  current animation cycle before displaying the final action icon.
- Disabled notification bots cannot send test messages; the Web UI disables
  the action and the API independently rejects direct requests.
- Server-stop bot notifications are sent as soon as graceful shutdown begins,
  while fully-started notifications remain tied to Minecraft's ready signal.
  The later process exit does not send a duplicate notification, and an
  unexpected crash still produces a stop alert.
- Discord automatic server detection now requests the standard `GUILDS`
  Gateway intent required for initial `GUILD_CREATE` backfill. Telegram also
  remembers a group from any group update the bot can receive, without
  requiring `/bonghos here` to configure it first.
- Destination refreshes preserve their display order and first-discovered
  timestamp, Telegram discovery works before any destination is configured,
  and provider command failures no longer expose backend details in group chat.
- Removing the bot from a Telegram group or Discord server now also removes
  unreachable broadcast targets from that container instead of leaving stale
  destinations enabled.

- The generated control-plane systemd user service no longer applies
  capability-derived kernel and control-group hardening that restricted hosts
  reject with `218/CAPABILITIES`. It remains an unprivileged service with
  `NoNewPrivileges`, `RestrictSUIDSGID`, and a private temporary directory.
- Settings no longer crashes when no notification bots exist: the Bots API now
  returns an empty JSON array and the Web UI also tolerates a legacy `null`
  response.

## [0.2.0-rc.1] - 2026-08-11

### Security

- Password, authenticator and recovery-code changes require fresh verification
  with the current password plus TOTP or a one-time recovery code, or with a
  user-verified passkey. Five-minute action grants are single-use, purpose-bound
  and tied to the exact browser session. Password and TOTP changes revoke every
  other session; TOTP replacement keeps the old secret active until the new
  authenticator is confirmed and then rotates all recovery codes.
- Login now has a distinct recovery-code mode instead of forcing one-time
  recovery codes through the six TOTP cells. Numeric-only hexadecimal recovery
  codes are preserved correctly, and a code is consumed only after the password
  and account state also pass authentication.
- WebSocket subscriptions now use an explicit topic allowlist. Activity,
  backup, schedule and console-command events require their matching backend
  permissions; unknown topics are rejected instead of inheriting
  `server.view`.
- Live WebSocket sessions are disconnected immediately after logout, session
  revocation, account disable/delete or a role change, and are revalidated
  every five seconds to cover out-of-process administration.
- External `.tar.xz`, `.7z` and `.rar` extractors are no longer used for
  untrusted imports because they wrote to disk before Bonghos could validate
  archive paths. In-process `.zip`, `.tar`, `.tar.gz` and `.tar.zst` imports
  remain supported with traversal, link, size, file-count and disk-reserve
  checks.
- Session and CSRF cookies now retain secure attributes behind a same-origin
  HTTPS reverse proxy or tunnel; the CSRF cookie is HttpOnly, JSON request
  bodies reject trailing values, and the fixed player-avatar proxy refuses
  redirects.
- Builds now select Go 1.26.5, `golang-jwt/jwt` is updated to 5.2.2 and
  `klauspost/compress` to 1.18.7. The audit found no reachable or package-level
  advisories; the remaining inactive module advisory was removed by the
  `klauspost/compress` update.

### Fixed (reported from a live modded server)

- The supervisor looked up its active project in an `app_state` table that does
  not exist in the schema, so it could not find the project to run. A new test
  now prepares every SQL statement in the tree against a migrated database, so
  this whole class of drift fails at `go test` rather than in front of a user.
- The Activity page failed with `no such column: created_at`; the column is
  `occurred_at`. Caught by the same test, plus an API test for the endpoint.
- Console history never replayed. History was bounded by lines (500) while the
  frame limit is bytes (64 KiB), so a modded boot overflowed one frame and the
  write error was ignored, leaving connecting clients with no backlog at all.
  History is now bounded by bytes as well as lines and replayed in chunks.
- Console output carried raw PTY escape sequences into the browser and the
  stored history. They are now stripped, including carriage-return overwrites
  from progress bars; Minecraft colour codes are preserved and the raw log file
  keeps everything.
- JVM detection treated a generated `user_jvm_args.txt` as authoritative for
  ServerPackCreator packs, so edits were discarded when the pack regenerated it
  at launch. Detection now recognises a script that regenerates its argument
  file and uses the variable that actually owns the settings.
- `-Xms(\S+)` swallowed the closing quote in `JAVA_ARGS="-Xmx4G -Xms4G"`,
  corrupting the assignment when memory was saved.
- `UpdateJVMArgFile` applied the second substitution to the original line,
  discarding the first, so with both values on one line the Xms change was
  silently lost.
- CPU samples could record about 1.8e19 when the process tick counter went
  backwards. The calculation now guards counter resets, PID reuse, unreadable
  `/proc`, and NaN, and clamps to the available cores.
- Player polling issued `list` every twelve seconds into the operator's
  console. Bookkeeping commands are now suppressed from the console stream
  while still being parsed and logged. The suppressible list is closed, so
  nothing can be run invisibly.
- Generated systemd units omitted hardening present in their reference
  templates and produced invalid `WorkingDirectory` values for custom Bonghos
  homes containing spaces. Unit values are now safely escaped and the generated
  services pass `systemd-analyze verify`.

### Added

- The Security page now supports changing the current account password,
  replacing its TOTP authenticator, and viewing recovery-code creation and use
  metadata without exposing stored code hashes. Recovery codes can be replaced
  as a complete set whose plaintext is shown once.
- The installer now creates a managed `~/.local/bin/bonghos` command, including
  the selected runtime home so custom `--home` installations can use short CLI
  commands. Update and repair recreate it; uninstall removes only Bonghos's own
  launcher and never overwrites an unrelated command.
- WebAuthn passkeys can now be enrolled, renamed and removed from Security, then used
  for username-free sign-in. Enrollment requires password and TOTP
  re-verification, discoverable credentials and user verification; the native
  browser prompt supports the current device, cross-device sign-in and
  hardware security keys without exposing private key material to Bonghos.
- Settings can now manage encrypted Telegram and Discord notification bots,
  including a master enable switch, independent ready/stopped/player-join/
  player-leave switches, a test-message action, and removal. Ready alerts are
  emitted only after Minecraft reports that it is fully started, and every
  lifecycle or player alert names the active server pack.
- TOTP enrolment now shows a scannable QR code: block characters in the
  terminal during `bonghos setup` and `bonghos admin create`, and an SVG on the
  Web UI activation page. The QR encodes the same `otpauth://` URI as before.
  Rendering is best-effort — a non-interactive or narrow terminal, or any
  encoding failure, falls back to the secret and URI and never interrupts
  account creation. The QR is generated server-side, so the browser needs no
  JavaScript QR library and enrolment keeps working offline. The secret and URI
  are still never written to the audit trail or application logs.
- A durable server event timeline (`server_events`, `GET /api/events`)
  recording starting, startup progress, ready, stop, force stop, crashes,
  backups and restores, plus recognised failures such as an unaccepted EULA, a
  port already in use, the wrong Java version or too large a heap. Startup
  phases are recorded once per run rather than once per matching log line.
- Overview is now the single live dashboard: server state, Java PID, uptime,
  CPU, process and host memory, disk, load, players, service status, CPU and
  memory trends, and the recent timeline. Host and Performance are demoted to
  detail views and no longer clutter the navigation.
- The Configuration page names the file that actually controls the JVM
  settings, explains when a pack regenerates its argument file, and links
  straight to that file in the editor.
- Vendored `rsc.io/qr` (BSD-3-Clause) under `source/third_party/qr`, consistent
  with the other locally maintained dependency replacements. QR rendering does
  not add a browser-side or runtime network dependency.

### Changed

- The embedded Web UI has been expanded across Overview, Performance,
  Servers, Files, Configuration, Players, Backups, Schedules, Users, Activity,
  Security and Settings while preserving the compact Bonghos visual language.
  This includes responsive search/filter controls, table action menus, direct
  project navigation, light/dark themes and clearer mobile layouts.
- Overview and Performance now expose host and Java CPU/memory choices,
  per-core CPU detail, temperature, host/Bonghos storage breakdowns, explicit
  refresh controls and hoverable history charts. Disk scans occur on page
  entry or manual refresh rather than every metrics interval.
- Server management now includes rename, server icon crop/resize, inactive
  project file/config access, mod-loader and game-version detection, and
  configuration inputs for common `server.properties` values and JVM argument
  files.
- CI and release vulnerability scans target the application package roots
  explicitly, so ignored developer caches under `source/bin` cannot make a
  local release audit fail package discovery.
- CI now enforces ShellCheck warnings for both `setup.sh` and the guarded Web
  UI integration helper instead of discarding the result.
- The installer now removes stale embedded Web UI assets, verifies the new
  provider images, accepts only exact official Git remote URL forms without a
  warning, and selects the patched Go 1.26.5 toolchain declared by `go.mod`.
- Updates now stop database writers before taking a private SQLite snapshot,
  integrity-check and checkpoint without applying migrations, and restore the
  previous executable, database and configuration if post-migration validation
  fails. The snapshot also protects existing bot tokens and passkey metadata.

## [0.1.1] — 2026-08-03

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

### Fixed after the v0.1.1 review

- A world-only restore resolved the world directory from the *destination*
  server.properties. If `level-name` changed after the backup was taken, the
  restore looked for the new name, matched nothing in the archive and reported
  success while changing nothing. The archive's own server.properties is now
  authoritative, and a scoped restore that matches nothing fails loudly instead
  of silently doing nothing.
- URL downloads ignored `free_space_reserve_mb`: the option existed and the
  downloader enforced it, but the handler never passed it through. Archive
  extraction never checked free space at all. Both now respect the configured
  reserve.
- Source builds stamped `0.1.0` regardless of the tag, because the version is
  declared separately in setup.sh, main.go and the Makefile. CI now fails if
  those three disagree with each other, with the documentation, or with the tag
  being released.
- The restore dialog claimed the previous files were kept alongside as
  `.bonghos-pre-restore`, but those are removed once each replacement succeeds.
  The durable undo is the emergency backup, and the dialog now says so.
- A world-only restore put the archived world back under its own name but left
  `level-name` naming a different one, so the server kept loading the old world
  and the operator saw no change after an apparently successful restore.
  Restoring a world now repoints `level-name` at it, and the API, CLI and UI
  report the change. `Manager.Restore` returns a `RestoreResult` describing what
  it did instead of only an error.

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
- Selected Go dependencies are maintained as local replacements under
  `source/third_party/`; remaining modules are pinned in `go.sum`.
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

[Unreleased]: https://github.com/Chansovisoth/Bonghos/compare/v0.3.0-rc.1...HEAD
[0.3.0-rc.1]: https://github.com/Chansovisoth/Bonghos/compare/v0.2.0...v0.3.0-rc.1
[0.2.0]: https://github.com/Chansovisoth/Bonghos/compare/v0.2.0-rc.1...v0.2.0
[0.2.0-rc.1]: https://github.com/Chansovisoth/Bonghos/compare/v0.1.1...v0.2.0-rc.1
[0.1.1]: https://github.com/Chansovisoth/Bonghos/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/Chansovisoth/Bonghos/releases/tag/v0.1.0
