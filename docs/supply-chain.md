# Software Supply Chain

## SBOM

Lyrahcle generates CycloneDX JSON SBOMs for the backend, CVE worker, SDR agent, and frontend.

Run locally with current Go and Node toolchains:

```bash
./scripts/generate-sbom.sh
```

Output is written to `artifacts/sbom/` with a `SHA256SUMS` manifest. CI uploads the directory as a workflow artifact. Tagged releases regenerate the same module SBOM set and attach it to the GitHub Release.

The generator versions are deliberately pinned in `scripts/generate-sbom.sh` defaults and can be overridden through `CYCLONEDX_GOMOD_VERSION` and `CYCLONEDX_NPM_VERSION` during controlled upgrades.

## Dependency gates

Pull requests run GitHub dependency review. Go modules additionally run `govulncheck` and `gosec`; the frontend runs `npm audit`; release container images are scanned with Trivy for high and critical vulnerabilities.

## Remaining pre-v1 work

- pin third-party GitHub Actions to immutable commit SHAs
- sign release containers and SBOMs
- publish build provenance
- define an exception process for accepted vulnerabilities and licenses
