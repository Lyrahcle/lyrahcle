# Releasing

Lyrahcle uses Semantic Versioning. Before v1.0, feature releases increment the minor version and bug-only releases increment the patch version.

## Release prerequisites

- `main` is green in CI.
- `CHANGELOG.md` describes the release.
- Documentation matches the current API and configuration.
- No known critical/high release-blocking vulnerabilities remain.

## Create a release

```bash
git checkout main
git pull origin main

git tag -a v0.9.0 -m "v0.9.0"
git push origin v0.9.0
```

A `v*.*.*` tag triggers `.github/workflows/release.yml`. The workflow builds, scans, and publishes four GHCR images, then creates a GitHub Release.

The registry owner is derived from the repository owner automatically:

```text
ghcr.io/<owner>/lyrahcle-backend:<version>
ghcr.io/<owner>/lyrahcle-frontend:<version>
ghcr.io/<owner>/lyrahcle-cve-sync:<version>
ghcr.io/<owner>/lyrahcle-sdr-agent:<version>
```

The Docker image version omits the leading `v`. A Git tag `v0.9.0` therefore publishes image tag `0.9.0` plus `latest`.

Example for this repository:

```bash
docker pull ghcr.io/Lyrahcle/lyrahcle-backend:0.9.0
```

If the package is private, authenticate to GHCR with credentials that have `read:packages` access.

## Failed releases

Do not move an existing tag. Fix the problem on `main`, choose the next patch version, and publish a new tag so the release history remains auditable.
