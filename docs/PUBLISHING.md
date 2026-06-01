# Publishing & Mattermost Marketplace submission

## 1. Prepare the public repository

1. Create a **public** GitHub repo (e.g. `mrvnklm/mattermost-plugin-clockwork`) and push `main`.
2. Confirm `plugin.json` fields are correct: `id`, `name`, `description`,
   `homepage_url`, `support_url`, `icon_path`, `min_server_version`.
3. Keep `LICENSE` (Apache-2.0) and `README.md` with screenshots at the repo root.

## 2. Cut a release

The build embeds the version from the git tag (`make dist` reads `git describe`).

```sh
git tag -a v0.1.0 -m "v0.1.0"
git push origin v0.1.0
```

The bundled CI (`.github/workflows/ci.yml`) builds and attaches
`dist/com.vsjwl.mm-time-tracking-<version>.tar.gz` to the GitHub Release.
Verify the release asset downloads and installs cleanly via **System Console →
Plugins → Management**.

## 3. Pre-submission checklist

- [ ] Public repo, Apache-2.0 license, README with screenshots.
- [ ] Signed/attached release `.tar.gz` for a tagged version.
- [ ] `make check-style` and `make test` green in CI.
- [ ] SVG `icon_path` present and renders in the Plugins list.
- [ ] `min_server_version` matches what you actually tested (9.0.0+).
- [ ] Tested on a clean server with **both Postgres and MySQL** if you advertise both.
- [ ] No secrets in the repo; server authorizes every endpoint.

## 4. Submit to the Marketplace

Mattermost curates the in-product Marketplace from
[`mattermost/mattermost-marketplace`](https://github.com/mattermost/mattermost-marketplace).

1. Read its `CONTRIBUTING`/README for the current submission format.
2. Open a PR adding a plugin entry that points at your GitHub release(s)
   (repository, release tag, and the signed bundle URL).
3. Address review feedback from the Mattermost team; once merged, the plugin
   appears in the in-product Marketplace.

Until it's listed, users can always install the release `.tar.gz` manually.
