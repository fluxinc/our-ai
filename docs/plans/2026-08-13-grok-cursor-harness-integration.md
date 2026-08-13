# Grok and Cursor harness integration

Status: **shipped (v0.38.0)**, 2026-08-13. Drafted and debated by Grok and Codex
through Talking Stick. Independent Grok review: AGREE.

## Evidence and correction

The original premise was that Cursor CLI may have merged into Grok. Current
runtime and vendor evidence contradicts that premise:

- Grok Build ships `grok`, stores state under `~/.grok`, reads `AGENTS.md`, and
  discovers project skills through `.grok/skills` (plus compatibility seams).
- Cursor CLI remains a distinct product. Current documentation names its
  executable `agent`; existing installs also expose `cursor-agent`. Cursor
  reads `AGENTS.md` and project `.agents/skills` or `.cursor/skills`.
- Both products may expose a generic executable named `agent`. A harness
  launcher cannot treat that name as product identity.

Sources:

- <https://docs.x.ai/build/overview>
- <https://docs.x.ai/build/features/skills-plugins-marketplaces>
- <https://cursor.com/docs/cli/installation>
- <https://cursor.com/docs/skills>

## Decisions

1. **Separate identities.** Add canonical `grok` and `cursor` harnesses. Never
   parse Cursor as Grok or silently substitute one product for the other.
2. **Native user paths.** Install the bundled `my-cli` self-skill under
   `~/.grok/skills` and `~/.cursor/skills` respectively.
3. **Launch-scoped organization skills.** Keep `.agents/skills` as the portable
   composition center. Mirror it into `.grok/skills` for Grok; Cursor consumes
   `.agents/skills` directly and needs no duplicate mirror.
4. **Executable collision safety.** Grok launches only as `grok`. Cursor launch
   resolution prefers `cursor-agent` when installed. If only the current
   official `agent` command exists, inspect its help before execution and accept
   it only when it identifies Cursor Agent rather than Grok Build.
5. **My AI session boundary.** `my session` remains the worktree/content unit.
   Both harnesses launch in the resolved My AI directory; My AI does not map its
   session ids onto either vendor's conversation-session or worktree flags.
6. **Portable context boundary.** Generated `AGENTS.md`, umbrella `.mcp.json`,
   launch skills, working directory, initial prompt, and pass-through harness
   arguments are the integration. Do not add vendor hooks, plugins, or mutable
   config as hidden setup work.
7. **Detection discipline.** Include installed executables in onboarding
   detection. Grok's `auth.json` is a best-effort login marker. Do not treat
   Cursor's `cli-config.json` as proof of login because it exists for logged-out
   installs too.
8. **Harness recursion guard.** Recognize `GROK_SESSION_ID` and `CURSOR_AGENT`
   so a nested `my` call does not perform user-global startup maintenance.

## Verification contract

- Unit coverage for names, aliases, stable ordering, native paths, prompt
  delivery, login hints, launch capabilities, and Cursor executable collision.
- CLI coverage for launch commands, global self-skills, launch-scoped skills,
  onboarding detection, missing executables, and harness environment guards.
- `gofmt`, `go test ./...`, `go vet ./...`, and `git diff --check`.
- Documentation site build.
- Sandboxed stub launches proving cwd, arguments, Grok's mirror, and Cursor's
  direct `.agents/skills` path.
- Live read-only inspection with `grok inspect --json`, Cursor status/help, and
  resolved executable paths. Authentication or vendor configuration is never
  changed as part of verification.
