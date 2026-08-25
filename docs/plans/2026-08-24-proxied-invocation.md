# Proxied invocation: `my ai` launches harnesses through a governed proxy

Status: **active (design)**, 2026-08-24. Debated on Talking Stick by Claude and
Codex against the current code of my-cli v0.39.0, cllama v0.10.1, and
clawdapus v0.28.1. Contract authority is cllama's invocation plan
(`cllama/docs/plans/2026-08-24-invocations.md`); this plan consumes it and
does not re-decide it.

## Problem

`my ai` execs a harness with the inherited environment
(`internal/cli/launch.go`, `runHarness`). Nothing binds the harness to a proxy:
the harness talks to its provider with the person's own credentials, and every
governance mechanism `my` has is a file it wrote beforehand (`AGENTS.md`,
launch-root skill mirrors, `.mcp.json`). Roles exist in the manifest
(`manifest.Role` selects mounts, skills, tools, services, guidance) but role
selection is a local `state.json` choice, and the README admits the manifest
has no authoritative identity-to-role mapping.

The organization wants one mechanism for a person on a laptop and a claw in a
pod: at invocation, identity, role, purpose, context, skill and tool grants,
memory view, model policy and budget are bound once, enforced by a process the
harness cannot bypass, and audited. That process is cllama. `my` becomes a
**launcher**: it supplies trusted organization inputs and the human subject,
requests an invocation, starts the harness bound to it, and revokes it on exit.

## Operator stories

- **S1 — a person launches a supported interactive harness.**
  `my ai --role support --purpose triage claude` (or `codex`) starts the harness
  against the configured local sidecar or remote cllama. Acceptance: the
  harness environment and launch-scoped config hold only the proxy base URL and
  a short-lived invocation bearer; cllama audits every request with invocation
  id, stable subject, role and purpose; only granted skills, tools and models
  are presented; memory view is `support`; a subject not authorized for the
  role is refused before the harness starts, with `{error, message,
  remediation}`; each launch is a distinct invocation and explicit resume is
  the only reuse path; the bearer is revoked on normal exit, non-zero exit and
  signal.
- **Deferred: person-bound subscription tokens.** `claude setup-token` and Codex
  ChatGPT sign-in both mint credentials that bill a subscription and both
  harnesses can be pointed at a gateway, but neither vendor documents that a
  gateway may hold and replay those tokens upstream. v1 guarantees API and
  custom-provider mediation only. `my proxy login <harness>` is reserved for a
  vendor-supported contract; cllama keeps a credential plugin seam (kind +
  harness binding + subject binding) and an operator-run live spike exists to
  test the Claude path against the operator's own account without making it a
  delivery gate.
- **S7 — the same launch against a central or on-prem proxy.** `proxy.mode:
  remote` with an HTTPS URL; laptops hold no provider credentials; the remote
  deployment's own human authenticator and server-side `mayAssume(subject,
  role)` decide roles; proxy unavailability, failed role authorization, or an
  unsupported harness adapter fails closed with remediation and never falls
  back to a direct provider.

## Decisions

1. **`my` submits trusted inputs; cllama compiles the effective context.** The
   invocation request (cllama public v1 wire format, JSON, stdlib-only client
   published by cllama and vendored as a conformance fixture — no Go module
   dependency) carries: stable subject `{kind: person, id: github:<immutable
   id>}`; role; purpose; expiry; ordered **input modules by kind** — organization
   contract (baseline fleet-work contract + manifest `contract`), role guidance
   fragments, policies index, data-binding domain notes, session startup
   context; skill entries (id, description, digest, body for granted static
   skills); tool specs from manifest `services` of kind `mcp` and `http`
   (`auth_ref` references only, never secrets); the single memory service
   reference (kind `memory`); model policy and budget from the role. `my`
   never assembles the system prompt. `internal/launchplan` is renamed
   `internal/projection`; `my compile` becomes `my projection [--role R]
   [--json]` (alias kept one release).
2. **Issuer credential, not PKI.** Sidecar mode: `my proxy ensure` starts a
   pinned cllama with a freshly minted control credential held by the `my`
   process and stored at `~/.local/state/my-cli/proxy/<manifest>/control.token`
   (0600). Remote mode: the deployment authenticates the human (its configured
   authenticator); `my` carries that identity and never self-asserts a role.
3. **The manifest gains the identity-to-role map.** `members: [{subject:
   "github:<immutable-id>", roles: [...]}]`; `roles[]` gains `models`, `budget`,
   `labels`, `purpose_default`. `my admin members add|remove` uses the governed
   manifest-edit flow, so a launcher cannot register itself into a role. The
   sidecar reads `members` from the synced manifest cache to answer
   `mayAssume`; remote deployments obtain it from their own manifest sync.
4. **Harness adapters are tiny and explicit.** `internal/harness` gains an
   adapter per supported harness: command; launch-scoped config directory;
   proxy base-URL + bearer binding; protocol (`anthropic-messages` or
   `openai-responses`); capability declaration and version probe; cleanup and
   revoke. Claude Code: `ANTHROPIC_BASE_URL`, `ANTHROPIC_AUTH_TOKEN`, with
   `ANTHROPIC_API_KEY` and `CLAUDE_CODE_OAUTH_TOKEN` cleared. Codex: temporary
   `CODEX_HOME` with a `model_providers.<cllama>` profile (`base_url`,
   `env_key`, Responses wire API) selected with `-c`. Guaranteed matrix:
   Claude Code and Codex. OpenCode, Antigravity, Grok and Cursor are reported
   **unsupported** when a proxy is configured until their adapter is proven;
   `my` never routes around cllama.
5. **Launch sequence and revoke-on-exit.** After `launchTarget` and the
   governed policy gate, before `runHarness`: resolve role (`--role`, else
   `state.json` `selected_role`, else error when the manifest declares roles)
   → build the request → `POST` create → exec the harness under the adapter →
   on exit (any path) `DELETE` the invocation. `my session finish` revokes live
   invocations whose purpose is that session. A configured proxy that is
   unreachable is `proxy_unreachable` with remediation `my proxy ensure`.
6. **Skills and MCP are delivered by the proxy.** Skill entries travel in the
   request; harnesses load bodies through cllama's managed `load_skill`. The
   v0.27 launch-root file composition becomes opt-in (`--skills-files`).
   `.mcp.json` is written only with `--mcp-files` or `proxy.mode: none`.
7. **Provider credentials never touch disk.** Manifest `services[].auth_ref`
   (`op://`, `env://`) resolve into the sidecar process environment at `my
   proxy ensure`.
8. **Doctor and JSON.** `my doctor` reports proxy mode and health, control
   credential presence, the current subject's role mapping, and adapter
   support per harness. Every new verb takes `--json`.

## Non-goals

- No central proxy implementation, PKI, attester, mTLS or ABAC engine in
  `my`; remote mode is configuration plus fail-closed behaviour.
- No subscription-token mediation in v1 (Claude `setup-token`, Codex ChatGPT
  mode) — deferred to a vendor-documented contract.
- No change to sync, sessions, records, governance publication, or the
  public/private repository boundary. No Go dependency on cllama or clawdapus.

## Slices

1. **Manifest schema.** `members`, role loadout fields, validation, admin
   verbs, docs, acme fixture.
2. **Sidecar management.** `my proxy ensure|status|stop`; pinned cllama
   download with checksum verification (reuse `internal/selfupdate`); control
   credential; `auth_ref` resolution into process env.
3. **Projection and launch.** `internal/projection`, harness adapters
   (Claude Code, Codex), launch sequence, revoke-on-exit, fail-closed, doctor
   rows. Ships S1.
4. **Deferred slice — person-bound subscription credentials.** Reserved; not
   scheduled until a vendor-documented gateway contract exists.
5. **Skill/MCP delivery defaults.** `--skills-files`, `--mcp-files`.
6. **Remote mode.** `proxy.mode: remote`, fail-closed. Ships S7.

## Spike tests (delivery gates; hermetic first, live as extra evidence)

- `TestSpikeLaptopInvocation` (`internal/e2e`, tag `spike`, credential-free):
  builds `my`; real cllama binary from `CLLAMA_BIN` (skips when unset) with an
  httptest upstream; temp HOME with the acme manifest carrying `members`; fake
  `claude` and `codex` binaries that capture environment and config and issue
  one real protocol request each through the proxy. Asserts every S1
  acceptance item, including unmapped-role refusal, unsupported-harness
  refusal, and revoke on normal exit, non-zero exit and SIGTERM.
- `TestSpikeLaptopInvocationLive` (skips without `ANTHROPIC_API_KEY`): real
  `claude -p` and `codex exec` through the sidecar in API-key mode.
- `TestSpikeSubscriptionCredentialOperatorOnly` (skips without
  `CLAUDE_CODE_OAUTH_TOKEN`; not a delivery gate): probes whether the
  operator's own setup-token can be replayed by the sidecar; documents the
  result for the deferred slice.
- `TestSpikeOrgProxy` (credential-free): cllama behind a TLS test proxy with a
  mock authenticator, a mock OpenAI-compatible "on-prem" provider, `my` in
  remote mode; proves fail-closed and on-prem routing.

## Sequencing

Blocked on a tagged cllama release shipping the v1 invocation types and
client, the control API with read endpoints, the credential plugin seam, `mayAssume`, and the conformance fixture. Slices 1–2 land before that tag; 3–6 after. The README roadmap and
`docs/plans/README.md` are updated with each slice.
