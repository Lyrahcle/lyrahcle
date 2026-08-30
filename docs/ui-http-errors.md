# UI HTTP error handling contract

Lyrahcle keeps HTTP failure semantics in the frontend instead of flattening every failed request into an untyped JavaScript `Error`.

- `401` is a security boundary: bearer state is cleared and the user returns to sign-in with an explicit expired/revoked-session notice.
- `403` is shown as a permission problem; hiding a UI control is never treated as backend enforcement.
- `404` is described without claiming whether a tenant-scoped resource exists elsewhere.
- `409` tells the operator to refresh/reconcile state before retrying the mutation.
- `429` preserves `Retry-After`, pauses dashboard polling for that cooldown and disables manual retry until it expires.
- `5xx` preserves the current screen and offers a controlled manual retry.
- Network failures preserve the current screen, mark the backend unavailable and offer retry.

`X-Request-ID`, when returned by the backend, is surfaced in the global error banner so operators can correlate UI failures with backend logs.

Mutation components may continue to show their more specific local error text. The global banner supplies the cross-application status/action contract and must not replace backend authorization.
