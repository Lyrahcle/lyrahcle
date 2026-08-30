# Contributing

## Setup

Requirements: Docker, Docker Compose, Go 1.25 for backend/CVE worker development, and Node.js for frontend work.

```bash
git clone https://github.com/Lyrahcle/lyrahcle.git
cd lyrahcle
docker compose up --build
```

For focused development:

```bash
cd apps/backend && go test ./... && go vet ./...
cd apps/frontend && npm install && npm test && npm run build
cd apps/sdr-agent && go test ./...
cd apps/cve-sync && go test ./...
```

## Code conventions

- Code, identifiers, commit messages, and technical documentation are written in English.
- Source code is self-documenting: prefer clear names and small functions instead of comments.
- Go must be `gofmt` clean and pass `go vet`.
- React uses functional components and focused modules.
- SQL migrations are additive, forward-only, and sequentially numbered under `apps/backend/migrations`.
- Do not edit an already-merged migration; add a new migration.

## Security changes

Authentication, authorization, cryptography, parsing, agent identity, rate limiting, and other trust-boundary changes require regression tests for the security property being changed. Do not weaken secure defaults to make a test pass.

Report vulnerabilities privately as described in [SECURITY.md](./SECURITY.md).

## Pull requests

Keep each PR focused. Before opening it:

```bash
cd apps/backend && go test ./... && go vet ./...
cd apps/frontend && npm test && npm run build
```

Update documentation when behavior, configuration, APIs, migrations, deployment requirements, or security guarantees change.

Use concise Conventional Commit-style messages, for example:

```text
feat: add satellite pass prediction
fix: preserve SSE streaming through metrics middleware
docs: align API reference with current routes
test: cover agent key revocation
```
