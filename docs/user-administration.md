# User Administration

Lyrahcle uses server-enforced role-based access control. Public self-registration is disabled.

## Roles

| Role | Read operational data | Investigate incidents | Operate assets/RF | Read audit log | Manage users/security |
|---|---:|---:|---:|---:|---:|
| `viewer` | Yes | No | No | No | No |
| `auditor` | Yes | No | No | Yes | No |
| `analyst` | Yes | Yes | No | No | No |
| `operator` | Yes | Yes | Yes | No | No |
| `admin` | Yes | Yes | Yes | Yes | Yes |

Authorization is enforced by the backend. Hiding controls in the frontend is not considered an access control.

## Bootstrap administrator

On an empty database Lyrahcle creates the bootstrap administrator configured by `SEED_ADMIN_EMAIL` and `SEED_ADMIN_PASSWORD`. The bootstrap account is marked `must_change_password` and can only access account endpoints until the password is changed. Production refuses the default `admin` seed password.

After upgrading an older installation, a bootstrap-style `admin` account is also forced through the password-change flow once.

## Password policy

New passwords must:

- contain 12 to 72 bytes;
- include at least one uppercase letter;
- include at least one lowercase letter;
- include at least one number;
- include at least one symbol.

Password changes revoke existing authorization state. Users must sign in again after changing a password.

## Account lifecycle

Administrators can create users, change roles, disable or re-enable accounts, and soft-delete accounts. Disabled and deleted accounts cannot authenticate. Deletion removes login identifiers and TOTP material and replaces the stored password hash with an unusable value while retaining audit-linked database identity.

Lyrahcle prevents an administrator from disabling or deleting their own account and prevents removal of the last active administrator.

Role, account-status, and deletion actions are written to the audit log. Role and status changes invalidate existing authorization state.

## Two-factor authentication

The supported user-facing second factor is Google Authenticator using standard TOTP. Enabling or disabling TOTP revokes existing sessions. WebAuthn security keys are intentionally tracked separately as a future identity release.

## Hardware security keys

Users can enroll cross-platform WebAuthn security keys from Settings. Enrollment and removal require the current password and revoke existing sessions. Once a key is present, password login must be completed with the registered key. See [Security keys and recovery](./security-keys-and-recovery.md).

## Recovery

Self-service password recovery is disabled unless SMTP recovery is explicitly configured. Google Authenticator remains required during recovery when enabled. WebAuthn-only protected accounts use administrator-assisted recovery rather than allowing an email link to bypass the hardware factor.
