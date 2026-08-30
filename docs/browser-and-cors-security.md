# Browser, CSP, and CORS security

Lyrahcle's default deployment is same-origin: the browser connects to the frontend
Nginx service and `/api/` is reverse-proxied internally to the backend. CORS is
therefore disabled by default.

For intentionally split frontend/backend origins, configure an exact allowlist:

```text
CORS_ALLOWED_ORIGINS=https://console.example.com,https://soc.example.com
```

Wildcards are rejected. Production origins must use HTTPS. The middleware permits
the Authorization and Content-Type headers required by the bearer-token API and
responds only to allowlisted preflight requests.

The frontend CSP is emitted with `always` by both HTTP and TLS Nginx configs, so it
applies to successful responses and Nginx error/fallback responses. The pre-paint
theme initializer is now an external same-origin script (`/theme-init.js`); the
application does not require `unsafe-inline` in `script-src`.

`sessionStorage` reduces token persistence but remains readable by JavaScript.
Consequently CSP and XSS prevention remain part of the authentication boundary.

When the API is intentionally moved to another origin, update the frontend CSP
`connect-src` directive to include that exact HTTPS API origin as well as enabling
the backend CORS allowlist.

## Production secrets

The backend refuses production startup with a default/unset `JWT_SECRET` or
`SEED_ADMIN_PASSWORD`. Grafana is a separate process, so `docker-compose.yml`
performs its own production startup check and refuses `lyrahcle-dev`, `admin`, or
an empty Grafana admin password when `ENV=production`.
