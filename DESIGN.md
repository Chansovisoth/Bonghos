# Bonghos Design System

## Direction

Bonghos is an Operate surface: a dense administration interface for a native-Linux Minecraft host. It should feel technical, precise, reliable, dark-first, and calm during long operational sessions. Minecraft context should be present through language, server state, console behavior, and project metadata rather than childish motifs.

## Shape

The default radius is `0`. Buttons, inputs, cards, panels, dialogs, tables, tabs, toast messages, status labels, dropzones, console regions, file rows, and OTP cells are square. Status is expressed through small squares, labels, borders, and text, not glowing dots or soft badges.

Avoid nested cards. Prefer flat regions, hairline dividers, tables, grids, clear labels, and alignment.

## Color Tokens

The implementation uses semantic OKLCH CSS tokens in `source/web/src/style.css`.

- Background: warm near-black in dark mode; neutral off-white in light mode.
- Surfaces: warm charcoal/graphite in dark mode; white and warm gray in light mode.
- Text: near-white or near-black primary text with neutral secondary text.
- Accent: amber-gold primary, deeper amber-orange for hover and selected operational emphasis.
- Success: restrained green.
- Warning: amber-orange.
- Danger: restrained red.
- Info: muted blue only when semantically needed.

Do not use purple-to-blue gradients, generic cyan, glassmorphism, glow effects, bokeh fields, gradient text, or colored shadow decoration.

## Typography

Preferred font stacks are defined as tokens:

- Display: `Alumni Sans`, then condensed/system fallbacks.
- Interface: `Albert Sans`, then system UI fallbacks.
- Technical: `JetBrains Mono`, then system monospace fallbacks.

No font files are currently vendored. This preserves the offline, dependency-free build. If Bonghos later vendors Alumni Sans, Albert Sans, or JetBrains Mono, place the assets under the embedded frontend source and document their source/license here.

Use tabular numerals for metrics, timestamps, memory, storage, players, ports, JVM values, and technical metadata.

## Theme Behavior

The default preference is System. First load applies the resolved system theme before the stylesheet loads. Users may explicitly choose System, Dark, or Light in Settings. System reacts to operating-system theme changes. Browser-native controls follow the active `color-scheme`.

## Components

Core primitives live in vanilla CSS classes and small JavaScript helpers:

- `pageHeader` for consistent page title, subtitle, and primary actions.
- `btn`, `primary`, `ghost`, `danger`, and `small` for commands.
- Form fields through native `input`, `select`, and `textarea`.
- `status-label`, `status-square`, and `tag` for state.
- `card`, `metric`, `table-wrap`, `kv`, `notice`, `alert`, `toast`, `modal`, `dropzone`, `progress`, `console-shell`, `console`, `breadcrumb`, `editor`, and `theme-choice`.

Buttons remain text-first in this dependency-free build because no icon library is shipped. Do not add decorative icon tiles.

## Page Patterns

Overview prioritizes current project state, lifecycle controls, player count, uptime, CPU, RSS, host memory, disk, backups, next schedule, trends, recent events, and project metadata.

Console behaves like an operational terminal with search, pause autoscroll, copy, clear, connection status, lifecycle actions, quick commands, command history, and stopped/read-only disabled states.

Data pages use tables or dense structured lists with horizontal overflow on small screens. Forms use settings rows and grouped sections rather than dashboard-card grids.

