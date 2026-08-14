# Detect unclosed mount worktrees and teach a real first-run

Status: **shipped (v0.39.0)**, 2026-08-14. Debated on Talking Stick by Claude,
Codex, and Grok; independently verified by all three before release.

## Problem

Agents create extra Git worktrees on umbrella checkouts with raw
`git worktree add` (Claude EnterWorktree, Codex, superpowers, scratch siblings).
Those trees are invisible to `my doctor` and `my session status`, which only
see the **session registry**. Unlanded session work is already reported;
unregistered leftovers are not. The work never "makes it up to the umbrella"
because nothing tells the operator or the next agent to land or close it.

Separately, public first-run docs assume `curl | sh` works, never name
prerequisites, have no Windows/WSL path, and do not walk a new operator from
"no organization" to a working umbrella.

## Decisions

1. **Same noun.** No `my worktrees`. Inspect is `my session leftovers [--json]`.
   Explicit close is `my session close-worktree <path> --yes`. `my doctor`
   gains a leftovers section that prints the same inventory and names
   `my session leftovers`.
2. **Inventory is porcelain, not a directory walk.** For each scanned checkout,
   run `git worktree list --porcelain -z`. Dedupe checkouts by canonical
   `git-common-dir`. Report worktrees whose paths sit outside the umbrella
   (`~/.claude-worktrees`, personal dirs, temp) — those are the leak.
3. **Scan set.** Umbrella content-mount checkouts plus every catalog repo
   whose `umbrella.RepoPath` (`repos/<id>` or leftover `products/<id>`) is
   actually a Git checkout, selected or not. Deselected-but-retained clones
   stay in scope. Plain leftovers and doctor do **not** scan the registered
   manifest checkout (intentional admin/governance topic trees); explicit
   `leftovers --all` includes it for administrative inspection. Do **not** walk
   arbitrary umbrella Git dirs.
4. **Classification** of each porcelain worktree:
   - `base` — the managed checkout itself
   - `registered-active` — path matches an active session registry mount
   - `registered-finished-residue` — path matches a finished/discarded session
     record that still has a worktree
   - `prunable` — Git still lists it, directory is gone
   - `leftover` — anything else, with a `locked` flag when Git reports locked
5. **Close is discard-the-worktree, keep the branch.** `close-worktree`
   resolves the path against the current inventory, then runs ordinary
   `git worktree remove` (no `--force`). It refuses `base`,
   `registered-active`, locked, missing/prunable, detached, and dirty
   (including untracked). Detached is refused because there is no branch to
   preserve and remove can orphan commits. Dirty leftovers print exact git
   instructions; there is no force flag in this release.
6. **Confirmation.** Non-TTY always requires `--yes`. TTY without `--yes` may
   prompt. Merged vs unmerged (leftover HEAD ancestor of the scanned
   checkout's current HEAD) changes only the warning text; unmerged prints
   `commits remain on branch <name>`. Squash-merged work that fails the
   ancestor check still closes with `--yes`; the branch remains.
7. **Registered sessions stay session-owned.** Active sessions close only via
   `my session finish --land|--publish|--discard`. Finished residue uses
   `close-worktree` / printed `git worktree remove`, never `finish` again.
8. **`--fix` does not grow.** Doctor `--fix` stays fast-forward of clean stale
   checkouts plus derived reconcile. It does **not** prune, remove, merge, or
   discard worktrees. A missing path can be a temporarily unmounted
   WSL/external disk; pruning can also discard the only reference to a detached
   commit. Prunable rows print a read-only porcelain inspection command, not a
   prune command.
9. **Inform, never hold; no launch scan.** Sync adds an informational result
   only when a content-mount leftover has commits absent from the base checkout.
   It does not change eligibility, pull/publish behavior, or exit status.
   Registered-session holds are unchanged, and `my ai` does not scan.
10. **Guidance.** Baseline `AGENTS.md` and the bundled `my-cli` self-skill get
    one contract: prefer `my session`; if you used raw `git worktree add`, run
    `my session leftovers` and close or land before you stop.
11. **Docs in the same release.** README and `site/guide/quickstart.md` name
    real prerequisites (`git`, `curl`, `tar`, `sha256sum`/`shasum`; `gh` only
    on `my publish` and private-HTTPS join; at least one harness CLI to
    launch). Windows path is WSL: install WSL + Ubuntu, install those tools
    **inside WSL**, run the Unix `install.sh`, keep the umbrella on the WSL
    filesystem, never mix Windows Git. First-run walks both `my init` (produce
    a local org) and `my manifests add` + `my manifests sync` + `my setup`
    (join an existing manifest) through the first `my ai`.

## Surfaces

```
my session leftovers [--all] [--json] [--home DIR] [--umbrella DIR] [--manifest NAME]
my session close-worktree <path> [--yes] [--json] [--home DIR] [--umbrella DIR]
my doctor                 # leftovers section; inspect only
```

Text leftovers output names, for each row: repo path, worktree path, class,
and the exact next command (`my session finish …`, `my session close-worktree
<path> --yes`, or a read-only porcelain inspection / `git worktree unlock`).
Inspect-only and a declined prompt leave every tree intact.

## Implementation notes

- Keep porcelain parse + classify as a small testable package or
  `worksession` helpers. Do not copy quarantine's "skip paths outside the
  allowed root" filter — the opposite is required.
- Pin "merged" to the scanned checkout's current HEAD only. Never "any
  local or remote-tracking tip".
- `close-worktree` must refuse to target a path that is not in the current
  inventory (no closing random Git worktrees).
- Catalog code leftovers have no session-land path (undesigned). Instructions
  say: commit/push/PR in that worktree using the repo's normal workflow, or
  discard the worktree and keep the branch.

## Tests (must drive shipped functions)

Fixture umbrella with one content mount and one catalog-repo clone. Plant
(a) an active `my` session worktree and (b) an extra `git worktree add`
leftover that is **not** in the session registry. Inspect must list both,
with repo path, worktree path, and a concrete close/land/discard command.
`--fix` / inspect-only / declined close must not delete the leftover. Explicit
`close-worktree --yes` on a dedicated clean leftover, then re-inspect: that
row is gone and its branch still exists.

Also: spaces in paths, external paths, locked, detached (refused),
dirty/untracked (refused), clean-unmerged (closes with `--yes`, branch kept),
clean-merged, common-dir duplicate inventory, prunable/missing, porcelain `-z`,
repo with zero extra worktrees, close of a path not in inventory.

Launch the real CLI twice against the same kind of fixture. Both runs must
print leftover content and the close instruction, not merely exit 0.

## Docs / release

- README + `site/guide/quickstart.md` (link a short WSL section if the
  quick start would get too long). Refresh `site/guide/sessions.md` and
  `site/guide/sync-and-doctor.md` only as needed for the new verbs.
- Next public release after `v0.38.0` per `AGENTS.md` Releasing.

## Non-goals

- Native Windows (non-WSL) installer or making `go test ./...` green on Windows.
- Silently garbage-collecting or auto-landing dirty worktrees.
- Repo-inclusive sessions / landing catalog **code** through `my session finish`.
- Editing any private flux/org manifest.
- `my ai` launch-time leftover scan (possible later TTL-piggyback).
- Closing detached worktrees in this release.
