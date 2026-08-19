# Bonghos

**A free, open-source, self-hosted web control panel for modded Minecraft servers on Linux.**

**Bonghos** (បង្ហោះ) is the Khmer word for **hosting**.

Bonghos lets you import, configure, run, monitor, schedule, back up, restore, and manage Minecraft Java Edition servers from a browser. It runs directly on your Linux machine with no Docker, cloud account, subscription, or required telemetry.

> **Status: v0.3.0-rc.1** — current prerelease. Keep independent backups and test restores before using it with an important world.

For implementation details, security design, data layout, development instructions, and current limitations, see [TECHNICAL.md](TECHNICAL.md).

## Key features

- Manage multiple server projects, with one active server at a time
- Stay fully self-hosted with no paid tier, Bonghos account, advertisements, or external database
- Import an existing directory, upload an archive, or download a server pack from a URL
- Detect common startup scripts, modloader/game versions, Java installations, and JVM memory arguments
- Start, gracefully stop, restart, or force-stop a server
- Use the live console from the Web UI or terminal
- Monitor CPU, memory, disk, load, temperatures, and player activity
- Manage players, operators, bans, and the whitelist
- Create scheduled actions, announcements, saves, and backups
- Make full, world-only, or configuration-only backups; verify and restore them
- Browse and edit server files, properties, icons, and startup settings; copy or move selections safely between managed projects
- Optionally publish the active Minecraft server through a Playit.gg tunnel without router port forwarding
- Connect up to two Telegram and two Discord bots for selected server and player notifications; Owners and Admins choose up to three destinations per bot with `/bonghos here`
- Reconcile player sessions, Java PID, uptime, and Overview counts immediately when Minecraft stops, crashes, or restarts
- Invite users with Owner, Admin, Member, or Viewer access and configure Admin, Member, and Viewer permissions
- Protect accounts with mandatory TOTP, recovery codes, and optional passkeys
- Move the self-contained Bonghos runtime to another disk or Linux machine

## Installation

### Requirements

- A 64-bit Linux system
- Java 17 or 21 for Minecraft
- Git, Go, and a C compiler for building from source
- `tmux` only if you want the optional terminal console session

Run the installer as your normal Linux user, not as root.

### From Git (recommended)

```bash
git clone https://github.com/Chansovisoth/Bonghos.git ~/bonghos-source
cd ~/bonghos-source
./setup.sh
```

### From an extracted release archive

```bash
cd ~/Downloads/Bonghos-0.3.0-rc.1
chmod +x setup.sh
./setup.sh
```

The installer checks dependencies, builds and tests Bonghos, creates the runtime directory, installs the `bonghos` command, guides you through the first Owner account, and can install the systemd user services.

By default, source and runtime data stay separate:

```text
~/bonghos-source/    source checkout
~/bonghos/           servers, backups, configuration, and Bonghos data
```

To use another runtime location:

```bash
./setup.sh --home /mnt/storage/bonghos
```

If `bonghos` is not found immediately after installation, open a new login shell or run:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

## Setup

During guided setup, you will:

1. Choose the Bonghos runtime directory.
2. Create the first Owner username and password.
3. Scan a TOTP QR code and enter the six-digit code from your authenticator app.
4. Save the one-time recovery codes somewhere safe.
5. Choose Playit.gg (recommended) or direct/manual player networking; Playit can use an account or guest setup.
6. Decide whether to install the recommended systemd user services.

Start the Web UI service:

```bash
bonghos
bonghos web status
```

Running `bonghos` without a command is the convenient form of `bonghos web
start`. To start now and enable automatic startup after reboot, run `bonghos
web enable` once.

To let the user service start after boot without waiting for an interactive login, enable lingering once:

```bash
sudo loginctl enable-linger "$USER"
```

There is no service time limit. The managed Web-panel unit restarts after an
unexpected exit with a five-second delay. An explicit `bonghos web stop` or
machine shutdown remains stopped normally; after the next boot, the enabled
unit starts automatically when lingering is enabled.

If systemd user services are unavailable, run Bonghos in the foreground:

```bash
bonghos serve
```

Open <http://127.0.0.1:8080>, sign in, then:

1. Add a server project from an archive, URL, or existing directory.
2. Select its startup script and Java version.
3. Set its memory limits and accept the Minecraft EULA.
4. Start the server.

If Playit was selected during setup, open **Settings > Playit.gg**. Install the
official Linux agent from <https://packages.playit.gg/>, link it using the
browser approval flow, then create the tunnel. Bonghos encrypts the agent
credential, waits for the official agent to report that it is ready, and uses
a separate on-demand user service. Tunnel status and the agent version are
shown in Settings; the linked agent can be renamed and tunnels can be updated
or deleted there. The public player address is shown on Overview. Existing installations
upgraded from a version without Playit remain on direct/manual networking until
an Owner enables the integration.

Bonghos listens on `127.0.0.1` by default. To reach it securely from another computer, create an SSH tunnel from that computer:

```bash
ssh -L 8080:127.0.0.1:8080 user@your-server
```

Then open <http://127.0.0.1:8080> locally. Bonghos does not configure routers,
firewalls, or reverse proxies for panel access; optional Playit support is only
for the Minecraft player connection.

For a panel intentionally published through Cloudflare Tunnel, an Owner can
enable Cloudflare Turnstile under **Security > Login protection**. Create a
Managed Turnstile widget for the panel hostname, then enter its public site key
and private secret key. Member accounts do not need email addresses. Bonghos
encrypts the secret in its database and validates every password and passkey
sign-in with Cloudflare. Keep the current Owner session open while testing a
private window. If a bad configuration blocks login, run:

```bash
bonghos security turnstile disable
```

For a complete walkthrough, see [Tutorial.txt](Tutorial.txt).

## Updating

### Git installation

Retrieve and install the latest source in one command:

```bash
cd ~/bonghos-source
./setup.sh --update --pull
```

Or review the source update before installing it:

```bash
cd ~/bonghos-source
git pull --ff-only
./setup.sh --update
```

### Extracted release archive

Extract the new archive into a new source directory, then run:

```bash
cd ~/Downloads/Bonghos-NEW_VERSION
./setup.sh --update
```

Updates build and test in a temporary area before replacing the installed executable. Server projects, backups, configuration, accounts, secrets, database records, and logs are preserved. If validation or health checks fail, the installed version is left in place or restored automatically.

## Commands

Run `bonghos help` or `./setup.sh --help` on the server for the current built-in help.

### Installer and updater

| Command | Purpose |
|---|---|
| `./setup.sh` | Guided production installation |
| `./setup.sh --dev` | Prepare and verify development dependencies |
| `./setup.sh --build` | Build and test without installing |
| `./setup.sh --update` | Install the source currently in this directory |
| `./setup.sh --update --pull` | Fast-forward a clean Git checkout, then update |
| `./setup.sh --repair` | Repair the installation, services, and portable paths |
| `./setup.sh --uninstall` | Remove services and the executable while keeping data |
| `./setup.sh --home DIR` | Use a custom runtime directory |
| `./setup.sh --yes` | Accept supported installer prompts non-interactively |
| `./setup.sh --help` | Show every installer option |

### Web panel

| Command | Purpose |
|---|---|
| `bonghos` | Start the Web panel in the background (same as `bonghos web start`) |
| `bonghos serve` | Run the Web UI and API in the foreground |
| `bonghos web start` | Start the Web panel in the background |
| `bonghos web stop` | Stop the background Web panel |
| `bonghos web restart` | Restart the background Web panel |
| `bonghos web status` | Show Web panel service status |
| `bonghos web logs [--follow]` | Show or follow Web panel logs |
| `bonghos web enable` | Start now and automatically after reboot |
| `bonghos web disable` | Disable automatic startup without stopping the panel |
| `bonghos version` | Print the installed version |

### Minecraft server control

| Command | Purpose |
|---|---|
| `bonghos server list` | List server projects |
| `bonghos server import <directory> [display name]` | Copy an existing server into Bonghos |
| `bonghos server select <slug-or-id>` | Select the active project |
| `bonghos server start` | Start the active server |
| `bonghos server stop` | Save and stop the active server gracefully |
| `bonghos server restart` | Save, stop fully, then start again |
| `bonghos server force-stop` | Kill a stuck server immediately; recent changes may be lost |
| `bonghos console` | Create or attach the optional tmux console session |
| `bonghos console --direct` | Attach directly without tmux |

### Accounts

| Command | Purpose |
|---|---|
| `bonghos setup` | Run first-account and service setup |
| `bonghos owner create` | Create the first Owner if none exists |
| `bonghos user list` | List accounts and their status |
| `bonghos user invite [admin\|member\|viewer]` | Create a single-use invitation |
| `bonghos user disable <username>` | Disable an account and revoke its sessions |
| `bonghos user enable <username>` | Re-enable an account |
| `bonghos user revoke-sessions <username>` | Sign an account out everywhere |
| `bonghos user reset-password <username>` | Set a new password and revoke existing sessions |

### Security

| Command | Purpose |
|---|---|
| `bonghos security turnstile status` | Show whether Turnstile and its credentials are configured |
| `bonghos security turnstile disable` | Disable Turnstile locally while retaining its saved credentials |

### Backups and portability

| Command | Purpose |
|---|---|
| `bonghos backup <world\|full\|configuration>` | Create a backup of the active project |
| `bonghos backup list` | List backups for the active project |
| `bonghos backup verify <backup-id>` | Check an archive and its checksum again |
| `bonghos backup restore <backup-id>` | Restore a backup while the server is stopped |
| `bonghos backup storage show` | Show the active backup directory |
| `bonghos backup storage move <directory>` | Move existing backups and use the new directory |
| `bonghos backup storage set <directory>` | Set an empty location when no backups exist |
| `bonghos backup storage reset` | Return an empty installation to the default location |
| `bonghos export --output <file.tar.zst>` | Create a portable export without account secrets |
| `bonghos export --include-secrets --output <file.tar.zst>` | Export accounts and secrets too; protect this archive |
| `bonghos import [--force] <archive.tar.zst>` | Import a portable Bonghos export |

Custom backup storage must be an absolute location outside `BONGHOS_HOME`.
Stop the Web panel before changing it. External archives appear on the Backups
page but are excluded from the Bonghos disk-size total, which counts only files
physically inside `BONGHOS_HOME`.

Backup lists reflect the archives currently present in the active storage
directory. If an archive is moved away manually, it disappears from normal
lists and cannot be restored. Returning it to the same relative path makes it
available again because Bonghos retains only the dormant recovery metadata.

Limit an export to one area with `--scope complete`, `configuration_only`, `system_data`, `servers`, or `backups`:

```bash
bonghos export --scope servers --output bonghos-servers.tar.zst
```

Restore only part of a backup with:

```bash
bonghos backup restore <backup-id> --scope world_only
bonghos backup restore <backup-id> --scope configuration_only
```

### Maintenance and services

| Command | Purpose |
|---|---|
| `bonghos doctor` | Diagnose the installation |
| `bonghos doctor --repair` | Apply safe automatic repairs |
| `bonghos doctor --fix-permissions` | Restore expected file permissions |
| `bonghos doctor --json` | Print diagnostic results as JSON |
| `bonghos database checkpoint` | Check SQLite integrity and checkpoint its WAL |
| `bonghos fix-permissions` | Restore expected file permissions |
| `bonghos service install` | Install the advanced user-service definitions |
| `bonghos service status` | Show control-panel, Minecraft, and managed Playit service status |
| `bonghos service repair` | Regenerate service files for the current runtime path |
| `bonghos service uninstall` | Remove the user services without deleting data |

Useful systemd commands:

```bash
systemctl --user status bonghos.service
systemctl --user restart bonghos.service
journalctl --user -u bonghos.service -f
journalctl --user -u bonghos-minecraft.service -f
journalctl --user -u bonghos-playit.service -f
```

Every CLI command supports a custom runtime directory through either form:

```bash
bonghos --home /mnt/storage/bonghos <command>
BONGHOS_HOME=/mnt/storage/bonghos bonghos <command>
```

## More documentation

- [Complete user walkthrough](Tutorial.txt)
- [Technical reference](TECHNICAL.md)
- [Changelog](source/docs/CHANGELOG.md)
- [Security policy](.github/SECURITY.md)
- [Contributing guide](.github/CONTRIBUTING.md)

## License

Bonghos is free and open-source software licensed under [AGPL-3.0-only](LICENSE).
