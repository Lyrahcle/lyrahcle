# Runtime command-execution security contract

Lyrahcle v1 treats runtime operating-system command execution as a prohibited application capability unless a future feature receives a dedicated security review.

## Current audit

Application source under `apps/` and `packages/` was reviewed for Go process execution (`os/exec`, `exec.Command`, `os.StartProcess`, `syscall.Exec`), Node `child_process`/spawn/exec APIs, and shell `-c` invocation. No runtime application path currently requires an external subprocess. The SDR agent uses in-process Go abstractions for device interaction and does not shell out to SDR utilities.

Operational scripts in `scripts/` and static container entrypoints are intentionally separate from this application-source contract. They are not invoked with request-controlled command strings by the application.

## CI enforcement

`scripts/check-command-execution.py` scans application source on every CI run and fails if a prohibited process or shell API is introduced. Its own regression tests are run in CI as well.

If external process execution becomes unavoidable, do not bypass the contract silently. The design must first define a narrow adapter with fixed executable selection, argument arrays (never shell command concatenation), strict input allowlists, explicit environment and working directory, timeouts/resource limits, audit logging, and negative command-injection tests. The CI rule can then be changed only for that reviewed adapter.
