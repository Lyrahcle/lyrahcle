# Releasing

Lyrahcle follows [Semantic Versioning](https://semver.org/) (`MAJOR.MINOR.PATCH`)
once v1.0.0 ships. Before that, `0.x.y` tags are used the same way: `MINOR`
for anything that isn't a pure bug fix, `PATCH` for bug fixes only.

## What gets published

Pushing a tag matching `v*.*.*` (e.g. `v0.8.0`) triggers
[`.github/workflows/release.yml`](./.github/workflows/release.yml), which:

1. Builds and pushes four Docker images to the GitHub Container Registry
   (GHCR), each tagged with both the version and `latest`:
   - `ghcr.io/mlko01/lyrahcle-backend`
   - `ghcr.io/mlko01/lyrahcle-frontend`
   - `ghcr.io/mlko01/lyrahcle-cve-sync`
   - `ghcr.io/mlko01/lyrahcle-sdr-agent`
2. Creates a GitHub Release on the tag with auto-generated notes (commits
   since the last tag) plus a pointer to [CHANGELOG.md](./CHANGELOG.md).

GHCR is used because it needs no extra secrets — the workflow authenticates
with the repository's built-in `GITHUB_TOKEN`. `packages/` (the planned MIT
shared libraries) has no actual code yet, so there's nothing to publish to
npm; that step will be added once those packages exist.

## Cutting a release

```bash
# 1. Make sure main is green and CHANGELOG.md's "Unreleased" section
#    reflects everything going out in this release.
git checkout main && git pull

# 2. Move the CHANGELOG's "Unreleased" heading to a dated version heading,
#    e.g. "## [0.8.0] - 2026-09-01", and start a fresh empty "Unreleased"
#    above it.

git add CHANGELOG.md
git commit -m "chore: release v0.8.0"
git push origin main

# 3. Tag and push — this is what triggers the release workflow.
git tag -a v0.8.0 -m "v0.8.0"
git push origin v0.8.0
```

Then watch the **Release** workflow run in the Actions tab. Once it's green,
the images are live on GHCR and the GitHub Release page has the notes.

## Publishing to Docker Hub instead of (or in addition to) GHCR

Not enabled by default — it needs a Docker Hub account and an access token,
which only you can create. To add it:

1. Create a Docker Hub access token (Docker Hub → Account Settings →
   Security → New Access Token).
2. Add `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` as repository secrets
   (Settings → Secrets and variables → Actions).
3. In `.github/workflows/release.yml`, add a second `docker/login-action`
   step logged into `docker.io` with those secrets, and add
   `docker.io/<your-username>/lyrahcle-<image>:<tag>` entries to the `tags:`
   list in the build-and-push step.

## Pulling a released image

```bash
docker pull ghcr.io/mlko01/lyrahcle-backend:0.8.0
docker pull ghcr.io/mlko01/lyrahcle-backend:latest
```

GHCR images inherit the visibility of the repository; if the repository is
private, `docker login ghcr.io` with a personal access token
(`read:packages` scope) is required before pulling.
