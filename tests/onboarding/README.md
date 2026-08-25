# First-run container matrix

Run the public-safe onboarding stories in fresh Ubuntu containers:

```sh
tests/onboarding/run.sh
```

The default matrix covers `linux/arm64` and `linux/amd64`. Every scenario gets a
new container and a new home directory. Fixtures use local bare Git repositories
and fake harness/provider adapters; the matrix never reads a private manifest or
writes to GitHub.

The scenarios cover agent-backed AUTHOR and JOIN through setup, required and
optional tool decisions, doctor, root, first launch, and explicit completion;
same-URL continuation after missing prerequisites; HTTPS provider checks and
global Git-config preservation; durable PATH; manifest conflicts; corrupt-state
repair; the deterministic no-agent path (installer `--no-onboarding`, `my setup`
self-heal of an unsynced manifest, doctor `prereq` rows, `my onboarding
--no-agent`); and wrapper download failure.

Raw console logs and structured per-command JSONL are generated under
`transcripts/` and ignored by Git. `transcripts/summary.json` is the small,
reviewable result index retained with the candidate.

Override the matrix when diagnosing one seam:

```sh
MYCLI_ONBOARDING_PLATFORMS=linux/arm64 \
MYCLI_ONBOARDING_SCENARIOS='join resume' \
tests/onboarding/run.sh
```
