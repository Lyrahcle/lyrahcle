# Browser authentication security

Lyrahcle Core authenticates browser API requests with a JWT sent explicitly in the
`Authorization: Bearer` header. The backend does not use a browser authentication
cookie for API authorization.

## CSRF assessment

Because authenticated state-changing API requests require an explicit bearer header,
the browser does not attach Lyrahcle authentication ambiently to cross-site requests.
Under this model, classical cookie-based CSRF is not the primary browser-authentication
risk. Do not add `credentials: include` or migrate API authentication to cookies without
adding an explicit CSRF defense at the same time.

Login and WebAuthn-login responses return a token in JSON. Cross-origin callers cannot
place that token into Lyrahcle origin storage under the same-origin policy.

## Token storage

The web client stores the access token and role in `sessionStorage`, not
`localStorage`. This removes persistent bearer credentials when the browser tab/session
ends and reduces exposure from long-lived browser storage. It does not make bearer
tokens immune to XSS: code executing in the Lyrahcle origin can still access
`sessionStorage`.

Legacy `lyra_token` and `lyra_role` values in `localStorage` are purged on successful
login and logout.

## Required controls

- Keep access-token lifetimes short and continue server-side token/session revocation.
- Maintain a restrictive Content Security Policy and avoid unsafe HTML injection.
- Never place bearer tokens in URLs, logs, analytics events, or error reports.
- Treat any future HttpOnly-cookie authentication migration as a separate security
  design requiring SameSite policy, Origin/Referer validation, and an anti-CSRF token
  for state-changing requests.
- Security regression tests must continue to reject persistent localStorage JWT storage
  and ambient cookie credentials for authenticated API requests.
