# Publishing & Mattermost Marketplace submission

## How Marketplace listing actually works

You do **not** open a pull request against `mattermost/mattermost-marketplace`, and you
do **not** submit a pre-built `.tar.gz`. Mattermost **reviews, builds, and code-signs the
plugin itself from source**. The author's deliverable is a clean public repo + a tagged
`>= v1.0.0` release + a filled-out submission issue.

The flow:

1. You open a **GitHub issue** on
   [`mattermost/mattermost-marketplace`](https://github.com/mattermost/mattermost-marketplace)
   using the **`add_plugin` issue template**, linking the public repo and a release
   tag/commit. (The link must point at source, not a pre-built artifact.)
2. Mattermost staff do a license/legal review, fork the repo into the Mattermost org,
   review the code, **build and code-sign it**, and run their generator to add an entry to
   `plugins.json`. Servers verify that signature at download time. You never generate a
   signature or edit `plugins.json`.

## 1. Cut the release

The build embeds the version from the git tag (`make dist` reads `git describe`).
Marketplace requires **>= v1.0.0** (anything `0.x` can only enter as a *Beta* plugin).

```sh
git tag -a v1.0.0 -m "v1.0.0"
git push origin v1.0.0
```

The release workflow (`.github/workflows/release.yml`) builds and attaches
`dist/com.vsjwl.mm-time-tracking-<version>.tar.gz` to the GitHub Release (Mattermost
rebuilds from source anyway). Verify it installs cleanly via **System Console → Plugins →
Management**.

## 2. Pre-submission checklist

From the Marketplace `add_plugin` template:

**Product**
- [ ] OSI-approved open-source license (Apache-2.0 ✔).
- [ ] Public Git repository with a public issue tracker, linked via `support_url` ✔.
- [ ] A published changelog, linked via **`release_notes_url`** in `plugin.json` ✔.
- [ ] **Released at `>= v1.0.0`** (out of Beta).
- [ ] All configuration accessible via the Mattermost UI (`settings_schema`) ✔.
- [ ] Reverse-DNS, non-colliding plugin ID (`com.vsjwl.mm-time-tracking`) ✔.

**Documentation** (reviewed by a Mattermost Technical Writer; `homepage_url` should point
to it)
- [ ] README with: prerequisites, installation, configuration, usage, troubleshooting,
      **at least one screenshot**, and a development guide ✔.

**Functional / technical**
- [ ] `min_server_version` set (`9.0.0`) ✔; works on all server versions `>=` it.
- [ ] No local-only state that breaks High Availability (state lives in the MM DB) ✔.
- [ ] Important events logged at appropriate levels.

**Security**
- [ ] No exploitable vulnerabilities; every endpoint authorizes the caller ✔.
- [ ] A security contact provided in the submission issue (email or Community handle).

## 3. Submit

1. Make the repo public at `github.com/mrvnklm/mattermost-plugin-clockwork`.
2. Optionally join the **Plugin Marketplace** channel on community.mattermost.com to
   coordinate.
3. Open the **`add_plugin`** issue on `mattermost/mattermost-marketplace`; fill the
   checklist; link the repo + the `v1.0.0` tag/commit; include the security contact.
4. Respond to reviewer feedback. Once approved, Mattermost builds, signs, and lists it.

Reference links:
- Community plugin marketplace: <https://developers.mattermost.com/integrate/plugins/community-plugin-marketplace/>
- `add_plugin` template: <https://github.com/mattermost/mattermost-marketplace/blob/master/.github/ISSUE_TEMPLATE/add_plugin.md>
- Manifest reference: <https://developers.mattermost.com/integrate/plugins/manifest-reference/>
- Developer workflow (`make dist`/`make deploy`): <https://developers.mattermost.com/integrate/plugins/developer-workflow/>

## Alternative: self-hosted distribution (no Marketplace)

Ship the `make dist` `.tar.gz` as a **GitHub release asset**; admins install it via
**System Console → Plugins → Management → Upload Plugin** (requires `EnableUploads`).
No Mattermost review or signing needed — the zero-friction path.
