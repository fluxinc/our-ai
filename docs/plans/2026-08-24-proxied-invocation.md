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
**launcher**: locally it supplies trusted organization inputs and the human
subject; remotely an administrator publishes stable inputs and a personal
launcher submits selections only. In both modes it requests an invocation,
starts the harness bound to it, and revokes it on exit.

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
  future vendor-supported contract and is not implemented or experimentally
  exercised by this plan.
- **S7 — the same launch against a central or on-prem proxy.** `proxy.mode:
  remote` with an HTTPS URL and `auth_ref` for a revocable per-person issuer
  token; an administrator first publishes the manifest-derived organization
  membership/loadout bundle; laptops hold no provider credentials; cllama
  binds each personal token to the immutable person subject and applies
  server-side `mayAssume(subject, role)` against that current bundle.
  A deployment may replace the token authenticator through cllama's interface.
  Proxy unavailability, revoked issuer, failed role authorization, or an
  unsupported harness adapter fails closed with remediation and never falls
  back to a direct provider.

## Decisions

1. **`my` compiles typed inputs; cllama compiles the effective context.** The
   wire operations use cllama's public v1 JSON format. `my` keeps a tiny stdlib
   HTTP client and pins cllama's JSON conformance fixtures; it does not import a
   Go module or duplicate final projection logic. Existing
   `internal/launchplan` and `my compile` remain the organization/control-plane
   compiler. A new `internal/invocationrequest` exposes three explicit outputs:

   - `BuildTrustedCreate` for the local sidecar: stable person subject, role,
     purpose, expiry, organization contract, role guidance, policies/data
     notes, bounded session startup context, granted skill entries, tool specs,
     memory reference/view, rules, model policy, budget, and channels;
   - `BuildOrganizationBundle` for a remote administrator: the complete stable
     `members` map plus organization and every role's loadout, excluding
     subject, purpose, expiry, and session-local context; tool/provider entries
     contain only deployment-owned `server://<organization>/<name>` aliases;
   - `BuildCreateSelection` for a remote personal launch: organization, role,
     narrowing purpose, harness/protocol, and expiry only. It cannot serialize
     skills, tools, context modules, memory, model policy, budget, channels, or
     delegation.

   cllama assembles the final provider-visible request in every case. These
   builders map into trusted input or selection types; none is called a
   projection.
2. **Issuer credential, not PKI.** Sidecar mode: `my proxy ensure` starts a
   pinned cllama with a freshly minted control credential held by the `my`
   process and stored in the OS keychain; local state stores only the nonsecret
   keychain locator. An explicitly documented 0600
   `~/.local/state/my-cli/proxy/<manifest>/control.token` is the fallback when
   no supported keychain is available. Remote v1 mode: `proxy.auth_ref`
   resolves a per-person issuer token from the OS keychain or 1Password for the
   control request only. cllama maps it to the subject and organization;
   current role authorization comes from the published organization bundle.
   `my` never sends a caller-selected subject, puts a raw credential in a
   manifest, or exposes one to the harness. Other central authenticators can
   replace this through cllama's authorizer interface.
3. **The manifest gains the identity-to-role map.** `members: [{subject:
   "github:<immutable-id>", roles: [...]}]`; `roles[]` gains `models`, `budget`,
   `harnesses`, `labels`, `purpose_default`. `my admin members add|remove` uses the governed
   manifest-edit flow, so a launcher cannot register itself into a role. The
   sidecar reads `members` from the synced manifest cache to answer
   `mayAssume`; remote deployments receive the same map and every role loadout
   in one atomic bundle from `my proxy publish-loadouts`.
4. **Harness adapters are tiny and explicit.** `internal/harness` gains an
   adapter per supported harness: command; launch-scoped config directory;
   proxy base-URL + bearer binding; one of cllama's v1 protocols; capability
   declaration and version probe; cleanup and revoke. Claude Code uses
   `anthropic-messages` through `ANTHROPIC_BASE_URL`, `ANTHROPIC_AUTH_TOKEN`, with
   `ANTHROPIC_API_KEY` and `CLAUDE_CODE_OAUTH_TOKEN` cleared. Codex: temporary
   `CODEX_HOME` whose config selects a `model_providers.<cllama>` profile with
   `base_url`, an `env_key` naming a launch-only bearer variable, and Responses
   wire API. The temporary home prevents fallback to the person's normal
   login; direct-provider variables are cleared. Guaranteed matrix: Claude
   Code and Codex. OpenCode, Antigravity, Grok and Cursor are reported
   **unsupported** when a proxy is configured until their adapter is proven;
   `my` never routes around cllama.
5. **Launch sequence and revoke-on-exit.** After `launchTarget` and the
   governed policy gate, before `runHarness`: resolve role (`--role`, else
   `state.json` `selected_role`, else error when the manifest declares roles)
   → build the trusted local create or remote personal selection → `POST`
   create → exec the harness under the adapter →
   on exit (any path) `DELETE` the invocation. `my session finish` revokes live
   invocations whose purpose is that session. A configured proxy that is
   unreachable is `proxy_unreachable` with remediation `my proxy ensure`.
6. **Skills and MCP are delivered by the proxy.** Skill entries travel in the
   trusted local create or admin-published remote bundle; harnesses load bodies
   through cllama's managed `load_skill`. The
   v0.27 launch-root file composition becomes opt-in (`--skills-files`).
   `.mcp.json` is written only with `--mcp-files` or `proxy.mode: none`.
7. **Provider and tool credentials stay at the proxy.** In sidecar mode,
   manifest `services[].auth_ref` values (`op://`, `env://`) resolve into the
   sidecar process environment at `my proxy ensure`. Remote bundle publication
   accepts only `server://<organization>/<name>` aliases already provisioned in
   the central deployment; local-only, unknown, or cross-organization refs fail
   before publication. `my` never reads or transmits a central raw secret.
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
   download with checksum verification (reuse `internal/selfupdate`); keychain
   control credential with explicit 0600 fallback; `auth_ref` resolution into
   process env.
3. **Invocation request and launch.** `internal/invocationrequest`, harness adapters
   (Claude Code, Codex), launch sequence, revoke-on-exit, fail-closed, doctor
   rows. Ships S1.
4. **Deferred slice — person-bound subscription credentials.** Reserved; not
   scheduled until a vendor-documented gateway contract exists.
5. **Skill/MCP delivery defaults.** `--skills-files`, `--mcp-files`.
6. **Remote mode.** `proxy.mode: remote`, fail-closed, plus admin-only
   `my proxy publish-loadouts --auth-ref <ref> [--dry-run]`. The command compiles
   the complete organization `members` map and every role through
   `BuildOrganizationBundle`, atomically publishes the credential-free
   bundle to cllama, and reports its digest. The admin credential is resolved
   for this command only; it is neither shared manifest state nor a launch
   credential. Ships S7.

## Spike tests (delivery gates; hermetic first, live as extra evidence)

- `TestSpikeLaptopInvocation` (`internal/e2e`, tag `spike`, credential-free):
  builds `my`; uses the required `CLLAMA_BIN` built from the pinned release in
  CI with an
  httptest upstream; temp HOME with the acme manifest carrying `members`; fake
  `claude` and `codex` binaries that capture environment and config and issue
  one real protocol request each through the proxy. Asserts every S1
  acceptance item, including unmapped-role refusal, unsupported-harness
  refusal, role-disallowed-harness refusal, and revoke on normal exit,
  non-zero exit and SIGTERM. A fake
  keychain proves a second `my` process resolves the stored sidecar credential
  from a nonsecret locator; a no-keychain case proves the explicit fallback is
  mode 0600; neither credential reaches the harness. A missing `CLLAMA_BIN`
  fails the CI spike target rather than skipping it.
- `TestSpikePinnedHarnessCompatibility` (credential-free delivery gate):
  uses cached, version- and checksum-pinned distributions of the real Claude
  Code and Codex CLIs,
  runs `claude -p` and `codex exec` through a real cllama with fake Anthropic
  Messages and OpenAI Responses providers, and asserts request shape, streaming
  response handling, one managed-tool loop, launch-config isolation, and exit
  cleanup. A missing artifact or checksum mismatch fails the job. This proves
  the actual harness adapters without provider spend and is mandatory in the
  release workflow; PRs retain the mandatory fake-harness spike without adding
  an ambient network-install dependency.
- `TestSpikeLaptopInvocationLive` (extra release evidence; skips without
  provider credentials): real `claude -p` and `codex exec` through the sidecar
  to supported provider APIs.
- `TestSpikeOrgProxy` (credential-free): cllama behind a TLS test proxy with
  one admin and two personal scoped issuer-token records plus a mock
  OpenAI-compatible "on-prem" provider. `my proxy publish-loadouts` publishes
  the complete bundle; a personal token cannot publish or inspect it; create
  before publication and for a missing role fails `loadout_missing`; `my` in
  remote mode proves immutable subject binding, bundle-governed allowed role,
  member/role removal blocks future creates without mutating existing
  Invocations, cross-subject/cross-role denial, issuer
  revocation, recorded bundle/role digests, server-only resolution of one valid
  tool/provider alias, rejection of raw/local-only/unknown/cross-organization
  refs, rejection of every capability field in a personal create selection,
  no issuer or provider/tool token in the harness, fail-closed behavior, and
  on-prem routing.

## Sequencing

Blocked on a tagged cllama release shipping the v1 invocation types and
client, the control API with read endpoints, `mayAssume`, and the conformance
fixture. Slices 1–2 land before that tag; 3–6
after. `TestSpikeLaptopInvocation` and `TestSpikeOrgProxy` are required PR and
release checks; all three credential-free spikes are required by the release
workflow. The README roadmap and `docs/plans/README.md` are updated with each
slice.
