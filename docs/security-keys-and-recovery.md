# Security keys and account recovery

## WebAuthn security keys

Lyrahcle supports WebAuthn cross-platform authenticators such as USB or NFC security keys. Enrollment requires an authenticated session, the current password, user verification on the authenticator, and a short-lived ceremony stored in Redis.

After a security key is enrolled, password login returns `webauthn_required` and the user must complete the WebAuthn assertion. The security key takes precedence over Google Authenticator for login. Registration and removal increment the database-backed authorization version and revoke existing sessions.

Production requirements:

- `WEBAUTHN_RP_ID` must match the application relying-party domain.
- `WEBAUTHN_ORIGINS` must contain the exact HTTPS frontend origins.
- Authentication traffic must use TLS.
- Redis must be available for one-time WebAuthn ceremonies.

## Password recovery

Password recovery is disabled by default. Enable it only when a trusted SMTP service is configured.

The request endpoint always returns an accepted response. Reset tokens contain 256 bits of randomness, are stored only as SHA-256 hashes, expire after 15 minutes, and are single-use. Completing a reset invalidates existing sessions through `auth_version` and Redis session revocation.

If Google Authenticator is enabled, the current TOTP code is required during recovery. If the account is protected by a WebAuthn key and TOTP is not enabled, self-service recovery is deliberately refused; an administrator must restore access so the password-reset channel cannot bypass the hardware factor.
