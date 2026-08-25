---
layout: home

hero:
  name: "> my ai"
  text: Your team's AI, set up once
  tagline: One organization manifest gives every AI harness generated guidance, mounts, launch profiles, and local operating context — on any machine, with one command.
  actions:
    - theme: brand
      text: Install
      link: /guide/quickstart
    - theme: alt
      text: View on GitHub
      link: https://github.com/fluxinc/my-cli

features:
  - icon: "01"
    title: One manifest
    details: Skills, mounts, catalog, tools, and generated guidance flow from a single organization source of truth.
  - icon: "02"
    title: Every harness
    details: Claude Code, Codex, OpenCode, Antigravity, Grok, and Cursor launch from one manifest-defined profile, using the best available skill path for each harness.
  - icon: "03"
    title: Local umbrella
    details: Every operator gets the same deterministic workspace, with synced content, catalog repositories, state, and scratch.
  - icon: "04"
    title: Operational by default
    details: Read, inspect, launch, diagnose, and materialize local skills without mutating the shared source of truth.
  - icon: "05"
    title: Admin when explicit
    details: Manifest authoring, mounts, and content writes live under the admin surface with clear write targets.
  - icon: "06"
    title: Public mechanism
    details: The CLI stays generic and open. Organization knowledge lives in your private manifest and workspace repos.
---

## Install

```sh
curl -fsSL https://my-cli.com/install.sh | sh
```

The installer needs no Go or Node, persists the user-local binary path, and
starts guided onboarding on a fresh interactive install. Run `my update` for
later updates; re-running the installer also works.

## First Run

Install and authenticate one supported AI harness first; it guides the rest of
setup. Other prerequisites are Git, `curl`, and a logged-in GitHub CLI (`gh`)
for organizations hosted on github.com; the installer and `my doctor` tell you
what is missing. On Windows, use the Linux CLI
and Git together in [one WSL distribution](/guide/windows-wsl), with the
umbrella under `/home`.

Create a new organization manifest and local umbrella:

```sh
my init acme --name "Acme"
cd "$(my root)"
my setup
my doctor
my ai codex
```

To join an existing organization, use the organization-provided one-liner:

```sh
curl -fsSL https://my-cli.com/install.sh | sh -s -- --manifest acme git@github.com:acme/manifest.git
```

The agent continues from the first missing prerequisite through private GitHub
authentication, manifest sync, setup, required organization tools, and
verification.

`my init` creates a private manifest control plane plus a content workspace,
all local and working offline. Preview publication with `my publish --print`,
then run `my publish` when the repositories should be created and pushed.
See the [complete Quick Start](/guide/quickstart) for prerequisites, harness
choices, guided onboarding, daily sync, sessions, and publication.

## The Operating Shape

```
~/acme/
├── .my-cli/          # workspace identity and local state
├── workspace/      # manifest-declared content mount (its own repo)
├── repos/          # opted-in catalog repositories
├── personal/       # local-only scratch
├── .mcp.json       # generated local MCP config
├── AGENTS.md       # generated root guidance
└── CLAUDE.md       # compatibility pointer when supported
```

The organization manifest lives in its own private repository outside the
umbrella — the workspace is a mount of things the manifest defines, and
day-to-day work never edits the manifest itself.

## Part of a Toolchain

`my` is the organization layer of a broader agentic stack: org context and
knowledge for every agent and human, from one manifest. It composes with
[gnit](https://github.com/mostlydev/gnit) (git-native multi-repo workspaces,
the umbrella's publish substrate) and
[clawdapus](https://github.com/mostlydev/clawdapus) (governed agent
containers whose cognition is mediated by the cllama proxy) — manifest roles
compile into contained fleet agents that carry the `my` CLI as a governed
work surface. Gated organization services (credential brokers,
human-reviewed communications) are declared in the manifest and consumed the
same way by human and AI operators.

Start with the [quickstart](/guide/quickstart), then read
[the model](/guide/the-model) for the boundary between the public CLI and a
private organization manifest.
