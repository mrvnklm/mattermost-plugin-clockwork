# Clockwork — Mattermost Time Tracking (Arbeitszeiterfassung)

[![CI](https://github.com/mrvnklm/mattermost-plugin-clockwork/actions/workflows/ci.yml/badge.svg)](https://github.com/mrvnklm/mattermost-plugin-clockwork/actions/workflows/ci.yml)

A native Mattermost plugin for tracking working hours directly inside Mattermost —
clock in/out with breaks, a weekly timesheet, manual corrections, and CSV export.
Built compliance-first for German *Arbeitszeiterfassung* (records start, end,
breaks and net daily hours), with an optional project/description per entry.

- **Plugin id:** `com.vsjwl.mm-time-tracking`
- **Min server version:** 9.0.0 (self-hosted; uses the plugin SQL store — Postgres or MySQL)
- **License:** Apache-2.0 · German + English UI

## Features

- **Right-hand-side panel** (clock button in the channel header):
  - Live running timer with **Start / Stop / Break** (pause & resume).
  - Optional **project** and **note** per entry, with **autocomplete** from your previously-used values.
  - **Today** list and a **weekly timesheet** (start, end, breaks, net hours, week total).
  - Add / edit / delete entries manually (timezone-correct, DST-safe).
- **Slash command** `/track in|out|break|status` for fast clocking.
- **Team report** (full-page, system-admins only — Main Menu → “Clockwork — Team report”):
  date-range filter, per-user filter, totals, and **CSV export** across all users.
- **CSV export** for your own records, and for admins across the team (formula-injection safe).
- Net hours = `end − start − breaks`; days are grouped in each user's Mattermost timezone.

| Timer (RHS) | Team report | Edit entry |
|---|---|---|
| ![Timer](docs/screenshots/timer.png) | ![Team report](docs/screenshots/team-report.png) | ![Edit entry](docs/screenshots/edit-entry.png) |

## Architecture

A single bundle: a **Go server** component (slash command, REST API, SQL persistence
in the Mattermost database) and a **React/TypeScript webapp** (RHS panel + full-page
admin report). One table, `timetracking_entries`; a running entry is a row with
`end_at IS NULL`. The server authorizes every request via the `Mattermost-User-ID`
header; admin endpoints require `PermissionManageSystem`.

```
plugin.json                 manifest (id, min_server_version, icon)
server/
  plugin.go                 wiring: store + command + router
  api.go                    REST handlers, auth, CSV/summary, admin
  command/command.go        /track slash command
  store/store.go            TimeEntry + Store interface (contract)
  store/sqlstore.go         Postgres/MySQL implementation
  store/migrations.go       idempotent schema (+ partial unique index)
webapp/src/
  index.tsx                 RHS + header button + admin route registration
  client/Client.ts          typed REST client (CSRF-aware)
  components/rhs/RHSView.tsx, components/WeeklyTimesheet.tsx, components/EntryEditModal.tsx
  components/admin/AdminConsole.tsx, components/admin/AdminTable.tsx
  utils/time.ts, styles.ts, icons.tsx, i18n.ts
docs/API_CONTRACT.md        the server⇄webapp REST contract
```

## Build

Requires the Node version in `.nvmrc` (`nvm i`) and the Go toolchain.

```sh
make dist        # → dist/com.vsjwl.mm-time-tracking-<version>.tar.gz
make check-style # lint (Go + webapp)
make test        # go test ./... + webapp jest
```

## Deploy to a dev server

Enable plugin uploads + local mode in the server `config.json`:

```json
"PluginSettings": { "EnableUploads": true },
"ServiceSettings": { "EnableLocalMode": true, "LocalModeSocketLocation": "/var/tmp/mattermost_local.socket" }
```

```sh
export MM_SERVICESETTINGS_SITEURL=http://localhost:8065
export MM_ADMIN_TOKEN=<token>     # or MM_ADMIN_USERNAME/MM_ADMIN_PASSWORD
make deploy                        # build + install via pluginctl
MM_DEBUG=1 make deploy             # unminified webapp + debuggable server
```

Or upload `dist/*.tar.gz` via **System Console → Plugins → Management**.

## Usage

- Click the **clock icon** in the channel header to open the panel; **Start** a timer
  (optionally type a project/note), **Break** to pause, **Stop** to clock out.
- `/track in` · `/track out` · `/track break` · `/track status`.
- System admins: **Main Menu → “Clockwork — Team report”** for the team dashboard + export.

## Tests

- **Go unit tests**: net-hours math, break accumulation, validation (`server/store`).
- **Manual / integration E2E**: see [`docs/API_CONTRACT.md`](docs/API_CONTRACT.md) and the
  verification checklist. The plugin has been exercised end-to-end against Postgres
  (start/break/stop, manual add/edit, weekly totals, self & admin CSV export, permission
  gates, and the admin team report).

## Publishing / Marketplace

See [`docs/PUBLISHING.md`](docs/PUBLISHING.md) for the release + Mattermost Marketplace
submission steps.

## Compliance note

This plugin records start, end, break and net daily working time and supports CSV
export for retention — building blocks for German *Arbeitszeiterfassung* (BAG ruling
13.09.2022). It is provided as-is and is **not legal advice**; confirm your own
retention and approval requirements.
