# Contributing to Clockwork

Thanks for your interest! This is a Mattermost plugin (Go server + React/TypeScript
webapp) bundled into one artifact.

## Prerequisites

- Go toolchain (see `go.mod`)
- Node — the version pinned in `.nvmrc` (`nvm install`)
- A Mattermost server `>= 9.0.0` (Postgres recommended; MySQL also supported) for manual
  testing

## Development loop

```sh
make check-style      # Go vet/lint + webapp eslint
make test             # go test ./server/... + webapp jest
make test-integration # SQL store/migrations against real Postgres + MySQL (see docs/TESTING.md)
make dist             # build dist/com.mrvnklm.clockwork-<version>.tar.gz
make deploy           # build + install onto a dev server (see README → Deploy)
```

## Guidelines

- **Match the existing style.** Keep the layered structure (`store` ⇄ `api`/`command`;
  webapp `Client.ts` ⇄ components). Reuse existing helpers rather than adding new ones.
- **Authorization first.** Derive the user from the `Mattermost-User-ID` header, never the
  body; gate admin endpoints on `PermissionManageSystem`.
- **Localise** every user-facing string (German + English) via `t()` in
  `webapp/src/i18n.ts`.
- **Keep the contract in sync.** If you change the REST surface, update
  [`docs/API_CONTRACT.md`](docs/API_CONTRACT.md).
- **Add tests** for new logic; **add a `CHANGELOG.md` entry** under `[Unreleased]`.

## Pull requests

Open a PR against `main` with a clear description and the checklist in the PR template
filled in. CI (lint + tests + build) must be green.

## Reporting issues

Use the bug/feature templates. For security issues, follow [`SECURITY.md`](SECURITY.md)
— do not open a public issue.
