# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/), and the project aims to follow
[Semantic Versioning](https://semver.org/).

## [Unreleased]

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

[Unreleased]: https://github.com/mrvnklm/mattermost-plugin-clockwork/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/mrvnklm/mattermost-plugin-clockwork/releases/tag/v0.1.0
