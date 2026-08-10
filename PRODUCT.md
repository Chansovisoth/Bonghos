# Bonghos Product Context

## Product

Bonghos is a native-Linux, self-hosted Minecraft Java Edition server control panel for modded server owners and administrators.

It imports, configures, runs, monitors, schedules, backs up, restores, and manages Minecraft servers directly on a Linux host. It is local-first, free, open source, and does not require a cloud account, subscription, telemetry service, external database, Docker, or a separate frontend runtime.

## Users

Bonghos users are server owners and administrators who may operate the panel during normal maintenance, restarts, player moderation, imports, restore operations, backup checks, configuration changes, and incidents. They need to scan quickly, act safely, and read technical output for long sessions.

## Product Priorities

- Reliability and status clarity outrank decoration.
- Server lifecycle controls must be visible and predictable.
- Destructive actions must have clear confirmation language.
- Console output, paths, commands, JVM values, ports, checksums, sizes, and timestamps must be highly legible.
- The UI must be usable in dark environments and complete in light mode.
- Every displayed operational value must come from the real API or an honest unavailable state.
- The Web UI must never expose an unrestricted Linux shell.

## Platform Constraints

The frontend is dependency-free vanilla HTML, CSS, and JavaScript under `source/web/src/`. The production binary embeds copied frontend files from `source/cmd/bonghos/webdist` via Go `embed`.

No JavaScript package manager, bundler, framework, remote font service, or runtime third-party asset dependency should be added unless the production embedding path remains intact and the operational value clearly justifies the supply-chain cost.

