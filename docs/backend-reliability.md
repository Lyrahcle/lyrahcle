# Backend reliability

## Error chains

The backend reliability gate audits every production `fmt.Errorf` call. A call that receives an error value must preserve a causal chain with `%w`; calls that construct a new validation or configuration error do not need wrapping.

At the start of this work the backend contained 166 production `fmt.Errorf` calls. Eighty-nine used `%w`. The difference was not a count of broken chains: most non-wrapping calls had no underlying error argument.

Redis connection failures are a special security boundary. Raw Redis client errors can contain connection credentials, so Lyrahcle does not expose the raw error in the public chain. The client redacts the message and wraps a safe sentinel error. Callers can distinguish invalid targets and unavailable Redis with `errors.Is` without exposing credentials.

## PostgreSQL context propagation

Every production store method that performs PostgreSQL work receives `context.Context` from its caller and passes that context to pgx. Store code must not replace the caller context with `context.Background` or `context.TODO`.

This keeps request cancellation connected to PostgreSQL work and prevents abandoned requests from leaving avoidable database work running.

Pool session cleanup and migration advisory-lock release are detached cleanup operations because they must still run after the initiating context is cancelled. Both use a bounded two-second cleanup context so cleanup cannot wait indefinitely.

## Gate

`scripts/check-backend-reliability.py` enforces:

- error arguments passed to `fmt.Errorf` retain a `%w` chain;
- errors are not converted to `err.Error()` inside `fmt.Errorf`;
- database-backed store methods accept `context.Context`;
- store PostgreSQL operations use the caller `ctx`;
- store code does not replace the caller context;
- PostgreSQL cleanup contexts are bounded;
- Redis redacted failures remain classifiable with `errors.Is`.

The contract is part of `scripts/check-contracts.sh` and therefore runs from `scripts/test-local-v1.sh` and CI.
