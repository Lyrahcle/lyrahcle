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

## Signed releases and provenance

Tagged release images are signed keylessly with Sigstore Cosign using the GitHub Actions OIDC identity. The workflow also publishes GitHub artifact attestations for each image digest. Verification should target immutable digests rather than mutable tags.

## Remaining pre-v1 work

- pin third-party GitHub Actions to immutable commit SHAs
- sign standalone SBOM files in addition to container images
- define an exception process for accepted vulnerabilities and licenses
- enforce provenance at deployment admission time
