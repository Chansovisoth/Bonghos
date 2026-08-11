# Security Policy

## Supported versions

Bonghos is early software. Security fixes land on the latest release.

| Version | Supported |
|---|---|
| 0.2.x prereleases | ✅ |
| 0.1.x | ✅ |
| < 0.1 | ❌ |

## Reporting a vulnerability

**Please do not open a public issue for a security vulnerability.**

Report it privately through GitHub's
[private vulnerability reporting](https://github.com/Chansovisoth/Bonghos/security/advisories/new),
or contact the maintainer directly.

Please include:

- What kind of issue it is (authentication bypass, path traversal, SSRF, etc.)
- Affected files or components
- Steps to reproduce, ideally with a proof of concept
- The impact you believe it has
- Your Bonghos version and Linux distribution

**Never include `secret.key`, session cookies, recovery codes or passwords in a
report.**

You can expect an acknowledgement within a few days, an assessment shortly
after, and credit in the advisory unless you prefer to stay anonymous.

## Areas of particular concern

If you are looking for somewhere to dig, these are the highest-risk surfaces:

- Authentication, TOTP verification, session handling and CSRF
- Role and permission enforcement (especially Member and Viewer limits)
- Archive extraction: traversal, symlinks, hard links, decompression bombs
- Path containment across the file manager, backups, restore, imports and icons
- SSRF protection on server-side URL downloads, including redirect handling
  and DNS rebinding
- Command construction for Minecraft console and player actions
- The supervisor's local Unix socket
- Anything that could reach a Linux shell

## Threat model

Bonghos assumes:

- It runs as a **normal, non-root** Linux user
- It binds to `127.0.0.1` by default; exposing it is the operator's decision
- Anyone with a valid session is at most as privileged as their role allows
- Anyone with filesystem access as the Bonghos user has effectively full
  control — this is by design; Bonghos is not a sandbox against its own owner

Out of scope:

- Attacks requiring existing root or Bonghos-user shell access
- Vulnerabilities in Minecraft, Java, mods or modpacks themselves
- Denial of service from an operator's own misconfiguration (for example
  setting `Xmx` higher than the machine's RAM)
- Consequences of the operator deliberately binding to a public address
  without a firewall

## Good practice for operators

- Keep the Web UI on `127.0.0.1` and reach it through an SSH tunnel
- Keep Bonghos updated with `./setup.sh --update --pull`
- Back up `system/config/secret.key` somewhere safe and private
- Give people the lowest role that does the job — Member and Viewer exist for
  a reason
- Review the audit log at `system/logs/audit.log` periodically
