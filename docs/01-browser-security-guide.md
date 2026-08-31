# Browser Security Guide

## Authentication Model

Lyrahcle Core authenticates browser API requests with a JWT sent explicitly in the `Authorization: Bearer` header. The backend does not use a browser authentication cookie for API authorization.

### Token Storage

The web client stores the access token and role in `sessionStorage`, not `localStorage`. This removes persistent bearer credentials when the browser tab/session ends and reduces exposure from long-lived browser storage. It does not make bearer tokens immune to XSS: code executing in the Lyrahcle origin can still access `sessionStorage`.

Legacy `lyra_token` and `lyra_role` values in `localStorage` are purged on successful login and logout.

### Token Lifecycle Requirements

- Keep access-token lifetimes short and continue server-side token/session revocation.
- Never place bearer tokens in URLs, logs, analytics events, or error reports.
- Server-side revocation invalidates tokens immediately upon password/TOTP/role changes.
- Security regression tests must continue to reject persistent localStorage JWT storage and ambient cookie credentials for authenticated API requests.

## CSRF Assessment

Because authenticated state-changing API requests require an explicit bearer header, the browser does not attach Lyrahcle authentication ambiently to cross-site requests. Under this model, classical cookie-based CSRF is not the primary browser-authentication risk.

**Do not add `credentials: include` or migrate API authentication to cookies without adding an explicit CSRF defense at the same time.**

Login and WebAuthn-login responses return a token in JSON. Cross-origin callers cannot place that token into Lyrahcle origin storage under the same-origin policy.

## CORS and Same-Origin Deployment

Lyrahcle's default deployment is same-origin: the browser connects to the frontend Nginx service and `/api/` is reverse-proxied internally to the backend. CORS is therefore disabled by default.

### Split Frontend/Backend Origins

For intentionally split frontend/backend origins, configure an exact allowlist:

```text
CORS_ALLOWED_ORIGINS=https://console.example.com,https://soc.example.com
```

**Important:** Wildcards are rejected. Production origins must use HTTPS. The middleware permits only the Authorization and Content-Type headers required by the bearer-token API and responds only to allowlisted preflight requests.

### Dynamic Origins

When the API is intentionally moved to another origin, update the frontend CSP `connect-src` directive to include that exact HTTPS API origin as well as enabling the backend CORS allowlist.

## Content Security Policy (CSP)

The frontend CSP is emitted with `always` by both HTTP and TLS Nginx configs, so it applies to successful responses and Nginx error/fallback responses.

### Key CSP Directives

- **script-src:** The pre-paint theme initializer is an external same-origin script (`/theme-init.js`). The application does not require `unsafe-inline` in `script-src`.
- **connect-src:** Limited to same-origin by default; extended only to allowlisted CORS origins if split deployment is configured.
- **img-src, font-src:** Restricted to same-origin and data: URIs only.

CSP and XSS prevention remain part of the authentication boundary. `sessionStorage` reduces token persistence but remains readable by JavaScript.

## Production Secrets

The backend refuses production startup with a default/unset `JWT_SECRET` or `SEED_ADMIN_PASSWORD`. Grafana is a separate process, so `docker-compose.yml` performs its own production startup check and refuses `lyrahcle-dev`, `admin`, or an empty Grafana admin password when `ENV=production`.

## Future Cookie-Based Authentication

Treat any future HttpOnly-cookie authentication migration as a separate security design requiring:

- SameSite policy (Strict or Lax)
- Origin/Referer validation
- An anti-CSRF token for state-changing requests
- Full re-audit of browser attack surface
- Updated security regression tests

## Checklist for Deployment

- [ ] `CORS_ALLOWED_ORIGINS` explicitly configured (if split deployment)
- [ ] `JWT_SECRET` set to a strong random value
- [ ] `SEED_ADMIN_PASSWORD` changed on first login
- [ ] TLS enabled for all production endpoints
- [ ] CSP headers present in all responses (verify with browser DevTools)
- [ ] No bearer tokens in request/response logs
- [ ] sessionStorage verified in browser (not localStorage)
