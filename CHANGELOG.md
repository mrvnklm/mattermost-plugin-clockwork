# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/), and the project aims to follow
[Semantic Versioning](https://semver.org/).

## [Unreleased]

## [1.0.0] - 2026-06-19

First stable release: adds the optional approval workflow and configuration, overhauls
the admin report, and hardens the data layer and test coverage.

### Added
- **Clockwork as a full product** in the Mattermost product switcher (next to
  Channels/Playbooks): a full-page time report available to **every user** for their own
  tracked time (range, totals, per-project breakdown, CSV export, submit/withdraw), with a
  **My time ↔ Team report** toggle for system admins. The quick-timer RHS panel stays.
- **Optional approval workflow** (`EnableApproval`, off by default): employees submit a
  week for approval; admins approve (locks entries), reject (reopens), or reopen an
  approved range. Entry lifecycle `open → submitted → approved`, exposed as a `status`
  field; `locked` is now derived from it. Owners may edit/delete only `open` entries.
- **Plugin settings** (System Console): `EnableApproval`, and `DefaultReportDays` to
  override the report/export lookback window. New `GET /config` endpoint feeds the webapp.
- **Per-project breakdown** in the admin Team report (hours/entries per project).
- **Accessibility**: the entry editor is now a proper dialog (focus trap, `Escape`,
  labelled controls), with a delete confirmation; icon-only buttons gained labels.
- **Weekly navigation** (previous/next week) in the timesheet, and an RHS loading state.
- **DB integration test suite** (Postgres + MySQL) plus Go handler tests and webapp
  unit tests (timezone/DST, REST client).

### Fixed
- **Compatibility:** replaced React 18's `useId()` (unavailable to plugins on the React
  the Mattermost host injects) with a React-17-safe id helper — it was crashing the entry
  editor and the entire admin report on the supported server. Verified end-to-end on a
  live Mattermost 10.5.
- Made the RHS weekly timesheet fit the narrow sidebar when the approval STATUS column is
  shown (compact layout + scroll fallback) so the hours column is no longer clipped.
- Admin **CSV export now respects the selected user** — previously it always exported
  the whole team regardless of the on-screen filter.
- `Delete` is now transactional (`FOR UPDATE`), closing a TOCTOU window that `Update`
  already guarded against.
- Admin entry lists are bounded (hard `LIMIT` + warning) so a wide range can't load an
  unbounded result set into memory.
- Manual entries are validated against sane bounds (max duration, not in the future).
- Localised the previously hard-coded RHS/menu registration strings (DE + EN).

### Changed
- Removed dead code (unused report-summary client, duplicate icon, the obsolete `locked`
  SQL column) and fixed CI/docs inaccuracies (`ci.yml` vs `release.yml`, signing claims);
  release builds now gate on a green test run.
- **PostgreSQL is now the primary database** (Mattermost's standard since v8.0 and the
  only DB from v11). MySQL remains supported and integration-tested but is deprecated
  upstream.
- Server bundle now builds for **Linux amd64 + arm64** only (the platforms Mattermost
  servers run on); the unused darwin/windows binaries were dropped to shrink the bundle.

## [0.1.0] - 2026-06-01

### Added
- Clock in/out with break (pause/resume) via an RHS panel and a
  `/track in|out|break|status` slash command.
- Optional project + note per entry, with autocomplete from previously-used values.
- Today list and weekly timesheet (start, end, breaks, net hours, week total).
- Manual add/edit/delete of entries (timezone-correct, DST-safe).
- Self CSV export and admin CSV export across all users (spreadsheet formula-injection safe).
- Full-page admin **Team report**: per-user summary (entries + total hours, click to
  drill in), date range, per-user filter, totals, show/hide entries, and CSV export.
- German + English UI.

### Database
- Postgres (recommended) and MySQL both supported. At most one running entry per user
  is enforced by a unique index — a Postgres partial index and, on MySQL, a
  generated-column unique key.

### CI
- GitHub Actions release workflow: builds and attaches the plugin bundle to a GitHub
  Release on every `v*` tag.

[Unreleased]: https://github.com/mrvnklm/mattermost-plugin-clockwork/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/mrvnklm/mattermost-plugin-clockwork/compare/v0.1.0...v1.0.0
[0.1.0]: https://github.com/mrvnklm/mattermost-plugin-clockwork/releases/tag/v0.1.0
