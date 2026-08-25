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
harness cannot bypass, and audited. That process is cllama. `my` becomes the
organization-control-plane client/compiler and a human **launcher**. It compiles
one organization bundle, publishes that same shape to a local sidecar or a
central proxy, then submits a role selection. In both modes it requests an
invocation, starts the harness bound to it, and revokes it on exit.

## Operator stories

- **S1 — a person launches a supported interactive harness.**
  `my ai --role support --purpose triage claude` (or `codex`) starts the harness
  against the configured local sidecar or remote cllama. Acceptance: the
  harness environment and launch-scoped config hold only the proxy base URL and
  a short-lived invocation bearer; cllama audits every request with invocation
  id, stable subject, role and purpose; only granted skills, tools and models
  are presented; memory view is `support`; a subject not authorized for the
  role is refused before the harness starts, with `{error, message,
  remediation}`; each launch is a distinct invocation; `my ai --session <id>`
  or `--flux-task <id>` carries only bounded late continuation context and
  never reuses a bearer; the bearer is revoked on normal exit, non-zero exit
  and signal.
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
- **S8 — a role onboards a person or a claw.** A declared
  `billing-reconciliation` role contains an approved effective contract,
  playbook, nested skill dependencies, managed business tools, memory/rule
  policy, and Flux/Flux Gate tools. A laptop launch and an
  organization-managed Clawdapus launch select the same role-entry digest.
  Identity-specific runtime envelopes and external principals differ without
  changing the role package.

## Decisions

1. **`my` compiles typed inputs; cllama compiles the effective context.** The
   wire operations use cllama's public v1 JSON format. `my` keeps a tiny stdlib
   HTTP client and pins cllama's JSON conformance fixtures; it does not import a
   Go module or duplicate final projection logic. Existing
   `internal/launchplan` and `my compile` remain the organization/control-plane
   compiler. A new `internal/invocationrequest` exposes two explicit outputs:

   - `BuildOrganizationBundle`: the complete stable
     `members` and `controllers` maps plus organization and every role's loadout, excluding
     subject, purpose, expiry, and session-local context; tool/provider entries
     contain only deployment-owned `server://<organization>/<name>` aliases;
   - `BuildCreateSelection`: organization, role, narrowing purpose,
     harness/protocol, expiry, and bounded invocation-local startup/work
     references only. It cannot serialize stable contract text, skills, tools,
     rules, memory, model policy, budget, channels, or delegation.

   Local sidecar mode publishes the current bundle through the private local
   admin channel before creating by selection; remote mode publishes through
   the explicit admin command. cllama assembles the final provider-visible
   request in every case. There is no `BuildTrustedCreate` or parallel
   complete-input authority.
2. **Scoped issuer credentials, not PKI.** Sidecar mode: `my proxy ensure`
   starts a pinned cllama with two distinct credentials held by `my`: a
   bundle-admin credential that can publish but cannot create, and a launcher
   issuer bound to the local person that can create/revoke its Invocations but
   cannot publish. On first bootstrap, it starts with the admin credential,
   publishes the bundle containing that person, and only then creates the
   person-bound launcher issuer. Both use separate OS-keychain entries; local
   state stores only nonsecret locators. Explicitly documented 0600 files under
   `~/.local/state/my-cli/proxy/<manifest>/` are separate fallbacks when no
   supported keychain is available. Remote v1 mode: `proxy.auth_ref`
   resolves a per-person issuer token from the OS keychain or 1Password for the
   control request only. cllama maps it to the subject and organization;
   current role authorization comes from the published organization bundle.
   `my` never sends a caller-selected subject, puts a raw credential in a
   manifest, or exposes one to the harness. Other central authenticators can
   replace this through cllama's authorizer interface.
3. **The manifest gains governed subjects and effective roles.** `members: [{subject:
   "github:<immutable-id>", roles: [...]}]`; `roles[]` gains `models`, `budget`,
   `harnesses`, `purpose_default`, ordered `includes`, approved organization and
   role `contract_revision` references, and role playbook/skill/tool/service
   grants. `my admin members add|remove` uses the governed
   manifest-edit flow, so a launcher cannot register itself into a role. The
   compiler rejects unknown includes, cycles, unresolved parent conflicts, and
   missing or cyclic skill/tool dependencies; shared diamond ancestry is
   deduplicated deterministically by source digest. It emits only declared,
   flattened effective roles. `controllers: [{id, member_namespace, roles,
   pod_members_feed?: {base_url, auth_ref}, policy_exempt?}]`
   is the canonical authorization for organization-managed pods. A central
   admin may provision a controller issuer only for an ID in that map; each
   create is checked against the current map, so the pod cannot self-scope.
   `policy_exempt: true` is valid only for an isolated
   `policy-evaluator/` namespace whose sole allowed role is the internal
   `policy-evaluator`; issuer administration cannot add the flag. The
   optional feed entry is the only controller-specific runtime endpoint:
   central mode requires an administrator-reviewed HTTPS origin and
   organization-scoped `server://` credential alias. A controller selection
   can enable that reserved feed but cannot name or override any feed, memory,
   or tool endpoint. The
   sidecar and remote deployments receive the same map and every role loadout
   in one atomic bundle.
   Every composite records its current parent-revision digests. Bundle
   compilation fails `role_revision_stale` when any parent changed. Accepting a
   parent instruction stages re-reconciliation of all transitive dependent
   composites in the same governed change.
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
   → ensure the current bundle is published locally or already published
   remotely → build the personal selection → `POST`
   create → exec the harness under the adapter →
   on exit (any path) `DELETE` the invocation. `my session finish` revokes live
   invocations whose purpose is that session. A configured proxy that is
   unreachable is `proxy_unreachable` with remediation `my proxy ensure`.
   `my ai --flux-task <id>` is an explicit human action after the task appears
   in Flux My work; `my` is not a wake daemon. `my proxy revoke --role <role>
   --auth-ref <admin-ref> [--bundle-digest <digest>]` uses the admin credential
   to list redacted live Invocations, resolves the matching role-entry digests,
   and invokes cllama's digest-filtered admin bulk-revoke operation for an
   urgent published change. The ordinary launcher credential never reaches an
   admin route.
6. **Skills and MCP are delivered by the proxy.** Skill entries travel in the
   published bundle; harnesses load bodies
   through cllama's managed `load_skill`. The
   manifest's existing `skills[].requires` grows a `skill:<id>` dependency next
   to `workspace:`, `service:`, and `tool:`. Bundle compilation computes the
   transitive closure and rejects missing/cyclic dependencies; the v1 index
   preserves `requires` for discovery and provenance.
   The
   v0.27 launch-root file composition becomes opt-in (`--skills-files`).
   `.mcp.json` is written only with `--mcp-files` or `proxy.mode: none`.
7. **Provider and tool credentials stay at the proxy or effect broker.** In sidecar mode,
   manifest `services[].auth_ref` values (`op://`, `env://`) resolve into the
   sidecar process environment at `my proxy ensure`. Remote bundle publication
   accepts only `server://<organization>/<name>` aliases already provisioned in
   the central deployment; local-only, unknown, or cross-organization refs fail
   before publication. Managed tool declarations name a principal policy:
   standing role-service alias, invoking-person local `auth_ref`, or a
   short-lived centrally delivered Flux Gate binding. A member cannot select an
   invoking-person binding. `my` never reads or transmits a central raw secret,
   and cllama is not a standing per-person credential vault.
8. **Doctor and JSON.** `my doctor` reports proxy mode and health, control
   credential presence, the current subject's role mapping, and adapter
   support per harness. Every new verb takes `--json`.
9. **Role contracts evolve through one governed authoring flow.** The
   organization control plane owns an append-only directive ledger and approved
   `ContractRevision` artifacts. The manifest Git repository is the reference
   backend, not a runtime mount requirement. `my admin role instruct <role>
   <text>` invokes MGL as a processor through its read-only control-plane
   source adapter: it stages reconciliation against the
   current revision, shows conflicts/supersession and provenance, and prepares
   one reviewed organization-control-plane change. Only an authorized human
   publishes it by writing the canonical files/records itself. MGL emits a
   staged artifact and never writes the organization control plane.
   Agent-originated instructions enter the same staging path as
   proposals and never self-approve. MGL's database and indexes are disposable
   staging/cache state, not another canonical store.
10. **Flux owns work; managed services own effects.** Flux and Flux Gate are
   ordinary declared services whose tools a role may receive. Flux owns tasks,
   waits, wakes, checkpoints, and approval requests. Flux Gate owns JIT access;
   accounting, mail, source-control, payment, and other services validate a
   release token for the effect-owner-defined proposal digest and perform their
   own effects. An effect owner or isolated executor reads the Flux approval
   record using its own credential, verifies the approved opaque digest against
   the canonical proposal, and only then signs the token; this is separate from
   Flux Gate's JIT grant ceremony. my-cli does not put this state in record
   domains or implement a workflow engine.

## Non-goals

- No central proxy implementation, PKI, attester, mTLS or ABAC engine in
  `my`; remote mode is configuration plus fail-closed behaviour.
- No free-form role union, label-conditioned authority, generic role-condition
  language, runtime context processor, business-work ledger, or approval
  engine in `my`.
- No subscription-token mediation in v1 (Claude `setup-token`, Codex ChatGPT
  mode) — deferred to a vendor-documented contract.
- No change to sync, sessions, records, governance publication, or the
  public/private repository boundary. No Go dependency on cllama or clawdapus.

## Slices

1. **Organization schema and compiler.** Directives/contract-revision
   references, `members`, `controllers`, flattened effective-role loadouts, `includes`, skill
   dependency closure, validation, admin verbs, docs, and acme fixture.
2. **Sidecar management.** `my proxy ensure|status|stop`; pinned cllama
   download with checksum verification (reuse `internal/selfupdate`); keychain
   separate bundle-admin and launcher credentials with explicit separate 0600
   fallbacks; `auth_ref` resolution into process env.
3. **Bundle publication, selection, and launch.**
   `internal/invocationrequest`, local-sidecar bundle publication, harness
   adapters (Claude Code, Codex), launch sequence, revoke-on-exit, fail-closed,
   doctor rows. Ships S1.
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
   `my proxy provision-controller <id> --auth-ref <admin-ref>` validates the
   controller against the just-published `controllers` map, creates the scoped
   issuer, and returns its raw token once to the deployment secret destination;
   my-cli does not place it in the manifest.
7. **Role authoring and orchestration adapters.** MGL-backed staged
   `role instruct`, Flux/Flux Gate service declarations, identity-aware tool
   bindings, and S8. The internal `rules-author` and `policy-evaluator` roles
   are ordinary published bundle roles, and a human author must appear in
   `members` for `rules-author`. This adds clients and compiler inputs, not
   another daemon.

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
  keychain proves a second `my` process resolves both stored sidecar credentials
  from nonsecret locators; a no-keychain case proves two separate fallbacks are
  mode 0600; the admin credential cannot create, the launcher cannot publish,
  and neither reaches the harness; initial bootstrap proves bundle publication
  precedes launcher-issuer creation. A missing `CLLAMA_BIN`
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
  one admin, two personal issuer records, one bundle-declared controller and
  its scoped issuer, plus a mock
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
  controller namespace/role/removal denial, issuer self-scoping denial, and
  on-prem routing. Personal list/get is limited to its subject; controller
  list/get is limited to its organization and member namespace; only admin can
  inspect across them. A controller can enable only its bundle-declared
  authenticated `pod-members` route, and an arbitrary URL is rejected before
  fetch.
- `TestSpikeRoleOnboarding` (credential-free, cross-repository): pins the
  canonical cllama conformance billing fixture and publishes its role with
  organization/role contract revisions, an ordered composite
  source, nested skill/tool requirements, fake accounting and browser-broker
  tools, fake Flux/Flux Gate, and a release-token-validating fake send service. The
  my-cli and Clawdapus launch paths select the same role entry. Assertions cover
  identical stable business loadout digests, distinct person/member principal
  bindings, secret starvation, missing/mismatched/forged/expired/replayed/
  wrong-audience release-token denial, a Flux wait, explicit human
  `--flux-task` resume, controller-adapter wake claim, and a new Invocation from
  bounded late task context.

## Sequencing

Blocked on a tagged cllama release shipping the v1 invocation types and
client, the control API with read endpoints, `mayAssume`, and the conformance
fixture. Slices 1–2 land before that tag; 3–7
after. `TestSpikeLaptopInvocation`, `TestSpikeOrgProxy`, and the
cross-repository role-onboarding spike are required PR and
release checks; all three credential-free spikes are required by the release
workflow. The README roadmap and `docs/plans/README.md` are updated with each
slice.
