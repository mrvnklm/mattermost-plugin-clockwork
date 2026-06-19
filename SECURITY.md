# Security Policy

## Reporting a vulnerability

Please **do not** open a public issue for security vulnerabilities.

Report them privately via [GitHub Security Advisories](https://github.com/mrvnklm/mattermost-plugin-clockwork/security/advisories/new)
(Security → Report a vulnerability), or by email to the maintainer listed on the GitHub
profile. Include a description, reproduction steps, affected version(s), and impact.

You can expect an initial acknowledgement within a few business days. Once a fix is
available we will publish a patched release and credit the reporter (unless anonymity is
requested).

## Scope

This plugin runs inside a self-hosted Mattermost server and stores time-tracking records
in the Mattermost database. Relevant areas: REST endpoint authorization (every request is
authorized via the `Mattermost-User-ID` header; admin endpoints require
`PermissionManageSystem`), entry ownership checks, the approval-workflow gating, and CSV
export (spreadsheet formula-injection is neutralised).

## Supported versions

The latest released version receives security fixes.
