# Open-issue simplicity sweep

Status: shipped in v0.41.0 (2026-08-25)

This sweep re-evaluated every issue open on 2026-08-25 against current master.
The governing constraint is operational simplicity: fix current defects and
proven friction, but do not add speculative nouns or large schema/refactor
surfaces merely because an old request remains technically possible.

| Issue | Disposition | Evidence or change |
|---|---|---|
| #36 | implemented | `my customers list --identity --json` provides a deterministic, least-privilege identity projection and fails distinctly for unavailable, dirty, or stale sources. |
| #34 | implemented | A deliberately remoteless control root is informational; auto sync routes exact members through guarded built-in publication while forced Gnit still fails closed. |
| #33 | implemented | Contract, role, service, tool, and policy reads use the existing TTL-bounded auto-refresh without changing JSON result shapes. |
| #31 | closed, not planned | Native Windows is not a supported execution target; WSL uses the Linux install path. Revisit with native support instead of preloading schema and compile complexity. |
| #30 | implemented | Publish and land holds compare exact base/session paths; disjoint work proceeds and overlapping or unprovable work remains held. |
| #28 | closed, completed | The existing hold message already sequences dirty-base cleanup before session finish; structural path handling is covered by #30. |
| #24 | implemented | Public docs and resume help distinguish persistent My work sessions from harness conversations. |
| #18 | closed, not planned | File-size-only mechanical splitting adds churn without operator value; command groups continue to split organically when behavior changes. |
| #17 | implemented | Push/PR CI now runs formatting, vet, build, tests, and whitespace checks. |
| #12 | closed, not planned | No bulk-import need was demonstrated; agents can loop the existing validated add command without adding another ingestion surface. |

Release gates are full Go tests and vet, formatting and whitespace checks,
site build, independent peer review of the exact commit, then both-agent
approval before any tag or push.
