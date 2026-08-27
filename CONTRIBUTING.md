# Contributing to Lyrahcle

Thanks for considering a contribution. This document covers local setup, coding
conventions, and the pull request workflow for the open-core components
(`apps/`, `packages/`). Enterprise modules (`enterprise/`) are not open for
external contribution.

## Local setup

Requires Docker and Docker Compose.

```bash
git clone git@github.com:mlko01/lyrahcle.git
cd lyrahcle
docker compose up --build
```

- Frontend: http://localhost:5173
- Backend: http://localhost:8080

For backend-only work without Docker, install Go 1.22+ and a local Postgres, then:

```bash
cd apps/backend
export DATABASE_URL="postgres://lyrahcle:lyrahcle@localhost:5432/lyrahcle?sslmode=disable"
go run ./cmd/server
```

For frontend-only work:

```bash
cd apps/frontend
npm install
npm run dev
```

See [docs/configuration.md](./docs/configuration.md) for the full list of
environment variables.

## Coding conventions

- **No comments in code.** This project's convention is self-documenting code:
  clear names, small functions, no `//`, `#`, or `--` comments in `.go`, `.js`,
  `.jsx`, or `.sql` files. Configuration files (YAML, Dockerfiles) may use
  comments where they clarify non-obvious setup steps.
- **English only.** All code, identifiers, commit messages, and code comments
  (where unavoidable, e.g. generated config) are in English, regardless of the
  contributor's or reviewer's first language.
- **Go**: run `gofmt -l .` and `go vet ./...` before committing; both must be
  clean. Prefer table-driven tests.
- **React**: functional components, hooks, Tailwind utility classes. Keep
  components focused — one concern per file under `src/components/`.
- **SQL migrations**: additive and forward-only under
  `apps/backend/migrations/`, numbered sequentially (`000N_description.sql`).
  Never edit a migration that has already been merged; add a new one instead.

## Testing

See [README.md#testing](./README.md#testing) for exact commands. In short:

```bash
cd apps/backend && go test ./... && go vet ./...
cd apps/frontend && npm test
```

New backend logic that parses external or untrusted input (protocol decoders,
API request bodies) should include a fuzz test where practical — see
`apps/backend/internal/tmtc/fuzz_test.go` for the pattern. Integration tests
that need Postgres check for `TEST_DATABASE_URL` and skip cleanly if it's
unset; don't make a test fail just because no database is configured.

## Security-sensitive changes

Changes touching authentication, authorization, input validation, or anything
in [SECURITY_ROADMAP.md](./SECURITY_ROADMAP.md) should:

- Add or update tests that exercise the security property directly (e.g. "a
  revoked key stops working", "a role below the minimum is rejected") — not
  just the happy path.
- Update `SECURITY_ROADMAP.md` and the relevant README section in the same PR.
- Avoid weakening an existing default (e.g. don't make an insecure fallback
  silent — it should log a warning, matching the existing pattern for
  `JWT_SECRET` and `TLS_CERT_FILE`).

Report vulnerabilities privately per [SECURITY.md](./SECURITY.md) — do not open
a public issue for those.

## Pull requests

1. Fork or branch from `main`.
2. Keep PRs focused — one logical change per PR is easier to review than a
   large mixed one.
3. Make sure `go vet`, `go test ./...`, and `npm test` all pass locally before
   opening the PR; CI runs the same checks (see `.github/workflows/ci.yml`).
4. Describe what changed and why in the PR description. For anything touching
   the database, mention the migration number.
5. Update `CHANGELOG.md` with a short entry under the current in-progress
   phase.

## Commit messages

Conventional, imperative mood: `feat: add satellite pass prediction`,
`fix: correct AX.25 SSID bit mask`, `docs: expand deployment guide`,
`test: cover agent key revocation`. This isn't strictly enforced but keeps
`CHANGELOG.md` and release notes easy to generate later.
