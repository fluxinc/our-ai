# Windows with WSL

My AI's supported Windows operating path is the Linux CLI inside Windows
Subsystem for Linux (WSL). Keep `my`, Git, the AI harness, and the umbrella in
one Linux environment. Native Windows CLI portability remains a separate
follow-up.

## Install WSL

Open PowerShell as Administrator:

```powershell
wsl --install
```

Restart Windows when prompted, launch Ubuntu, and create the Linux user account.
Microsoft documents `wsl --install` as the one-command path on current Windows
10 and Windows 11; if WSL already exists without a distribution, use
`wsl --list --online` followed by `wsl --install -d Ubuntu`.

## Install prerequisites inside Ubuntu

Run the remaining commands in the Ubuntu/WSL shell, not PowerShell:

```sh
sudo apt update
sudo apt install -y ca-certificates curl git tar coreutils

git config --global user.name "Your Name"
git config --global user.email "you@example.com"
```

Install at least one supported **Linux** harness CLI in this same distribution.
If private Git authentication uses the GitHub CLI, install and authenticate the
Linux `gh` package here as well.

## Install and verify My AI

```sh
curl -sSL https://raw.githubusercontent.com/fluxinc/my-cli/master/install.sh | sh
export PATH="$HOME/.local/bin:$PATH"

my version
```

Persist the PATH line in `~/.profile` or your shell's startup file if the
installer reports that `~/.local/bin` is not already available.

Continue with the [Quickstart](./quickstart): either produce a new manifest
with `my init` or register an existing manifest with `my manifests add`, then
run `my setup` and `my ai <harness>`.

## Keep one filesystem and one Git

Create umbrellas under the WSL home filesystem, for example
`/home/alex/acme`. Do not put the umbrella under `/mnt/c/Users/...` and do not
run Windows `git.exe` or a Windows `my.exe` against its repositories.
Worktrees record paths in shared Git metadata; mixing Windows and Linux path
semantics makes cleanup and recovery ambiguous.

Microsoft likewise recommends storing projects in the WSL filesystem when
working from a Linux command line for better performance and fewer
cross-filesystem surprises. Windows applications can still browse the files
through `\\wsl$` or by running `explorer.exe .` from WSL.

## Official WSL references

- [Install WSL](https://learn.microsoft.com/windows/wsl/install)
- [Working across Windows and Linux file systems](https://learn.microsoft.com/windows/wsl/filesystems)
