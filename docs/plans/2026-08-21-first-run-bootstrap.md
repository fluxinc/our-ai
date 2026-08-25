# One-command first-run bootstrap

Status: **shipped (v0.40.0)**, 2026-08-25. Jointly implemented and independently reviewed by Codex and Claude over Talking Stick; proven on the `tests/onboarding` cross-arch container matrix.

## Problem

The deterministic pieces existed, but a new machine still required a person to
assemble an installer, PATH repair, GitHub CLI setup, manifest registration and
sync, workspace setup, organization tool installation, and agent launch from
several pages. Common failures were especially hostile:

- source installation introduced an unnecessary Go prerequisite and placed the
  binary in a second user-local directory;
- `~/.local/bin` was correct for release binaries but was not persisted for the
  next shell;
- copied prose comments were interpreted as commands by interactive shells;
- `gh auth login` could prove GitHub API access while an HTTPS Git clone still
  fell through to an obsolete username/password prompt;
- a registered-but-unsynced manifest made model-driven onboarding fail before
  the agent could help;
- required and optional organization tools were reported together instead of
  being walked one decision at a time.

## Product contract

The organization handout is one copy-paste-safe command:

```sh
curl -fsSL https://my-cli.com/install.sh | sh -s -- --manifest acme <git-url>
```

The operator installs and authenticates one supported agent first. The script
then installs the checksum-verified My AI release without Go or Node, persists
the standard user-local binary path, registers the organization manifest, and
starts agent-guided onboarding. A bare installer command takes the new-
organization AUTHOR path. `--no-onboarding` retains a quiet automation seam.

Bootstrap is resumable rather than transactional. Its ordered states are:

1. release binary installed;
2. shell path durable;
3. agent harness available;
4. organization manifest registered;
5. Git and provider authentication ready;
6. manifest synced and valid;
7. umbrella setup and role selected;
8. required organization tools reviewed and installed with consent;
9. doctor clean enough for the first launch;
10. local onboarding completion explicitly recorded.

An interruption leaves completed states intact. Re-running the installer or
`my onboarding --manifest <name>` continues from the first incomplete state.

## Safety boundaries

- The installer never installs an agent harness, accepts policy, publishes a
  repository, or silently installs manifest-declared tools.
- Required tools are handled one at a time after setup; optional tools remain
  choices and never block onboarding.
- GitHub HTTPS Git commands receive `gh auth git-credential` through child-
  process environment only. My AI does not call `gh auth setup-git` or modify
  global Git configuration.
- SSH Git URLs continue to use normal SSH configuration and keys.
- Installer examples contain executable commands only; prose stays outside
  pasted shell blocks.
- `my setup`, `my manifests sync`, `my doctor`, and `my onboarding --no-agent`
  remain deterministic automation and diagnostic seams.

## Implemented seams

- Prompt-free Git with destination-specific authentication remediation and an
  invocation-local `gh auth git-credential` helper shared by manifest and mount
  operations. An integration test drives real `git credential fill`, observes
  the helper call, and proves the global Git config is byte-identical.
- Deterministic self-heal: `my setup` and `my onboarding --no-agent` sync a
  registered manifest before loading it. Agent onboarding keeps JOIN_BOOTSTRAP
  and adds JOIN_REPAIR for corrupt or invalid local checkout state.
- `my doctor` prerequisite rows for Git, `gh` login, durable binary PATH, and
  installed harnesses.
- Observable tool readiness (`required|optional`, `present|missing`, and path)
  plus explicit `my onboarding --complete|--status` state. Required tools block
  completion; optional tools do not.
- Fail-closed installer manifest conflicts, an idempotent durable zsh/bash/fish
  PATH update, redirect-based latest-release discovery without the anonymous
  REST API, and a stable wrapper that propagates download failures.

## User-story verification

`tests/onboarding/run.sh` builds the candidate from source and runs every story
in a separate clean `ubuntu:24.04` container on both `linux/arm64` and
`linux/amd64`. It uses public-safe local Git fixtures and fake harness/provider
adapters; it never touches a private or live GitHub repository. Each command,
exit status, and output is retained as JSONL under
`tests/onboarding/transcripts/`.

| Story | End-to-end acceptance |
|---|---|
| Existing organization | One URL → JOIN_BOOTSTRAP agent → real manifest clone → setup/role → required consent and retry → optional left missing → doctor → root → first `my ai` → completion |
| New organization | Bare URL → AUTHOR agent → `my init` → validation/setup → doctor → root → first `my ai` → completion, with no publication |
| Interrupted prerequisites | Missing harness, Git, and `gh` each produce an exact continuation; the identical URL resumes after the prerequisite appears |
| HTTPS authorization | Provider identity/permission checks succeed, Git clones through the invocation environment, and global Git config remains byte-identical |
| Safe retries | PATH line stays singular, a new zsh login finds `my`, completed onboarding stays quiet, and a manifest name/URL conflict preserves the original registration |
| Broken inputs | Invalid manifest checkout launches JOIN_REPAIR; stable-wrapper download failure remains nonzero |

The 2026-08-21 candidate matrix passed all 20 architecture/scenario runs. The
remaining gate is an independent Talking Stick review of the final diff and
transcripts, followed by the normal release process.
