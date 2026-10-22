---
claim: Codex eval of compass's core operations (orient, handoff, drift, decision write, decision read) against no-compass arms, after the scope cut
date: 2026-10-22
refs:
  - decisions/0015-scope-cut.md
  - decisions/0016-current-state-is-last-known-intent.md
  - experiments/2026-10-21-v3-core-eval.md
---
# Codex eval of compass's core operations (orient, handoff, drift, decision write, decision read) against no-compass arms, after the scope cut

Status of this file: **protocol and pass bars written before any run** (committed first, so git
shows they predate the results). Results are appended below in a later commit.

## What is measured
Compass's core operations after the scope cut (decision 0015) and the "last known intent"
current-state:
- **Effectiveness:** a rubric pass per scenario, graded by reading the diff, the final message and
  `go test ./...`.
- **Efficiency:** shell commands and tokens before the first edit, and in total. Tokens are
  cumulative input + output from Codex's rollout `token_count` events; totals are also reported
  uncached (input - cached input + output), since cache reads dominate. The first edit is the
  first `file_change` item or a shell write into the repo (a redirect to a relative path, `sed -i`,
  `tee`, `gofmt -w`); redirects to /dev/null or /tmp don't count.
- **Use:** did the agent run `compass resume` successfully (exit 0; first?), read
  `.compass/decisions/` or INDEX.md, run `compass record` successfully, rewrite current-state.

## Fixture
A synthetic Go module, `invoicer` (money in integer cents, per-line tax, a CLI that prints an
invoice from JSON), built by script with fixed dates: 4 commits, an `AGENTS.md` saying "run
`go test ./...`; don't commit". Synthetic rather than an ebay-flipper copy so the scenarios are
controlled and the transcripts can be kept.
- **N (no compass):** the base repo. No `.compass/`, no compass block, no `compass` on PATH.
- **C (compass):** the base plus `compass init --no-git-hook`, two decisions (amounts are integer
  cents; tax rounds half to even per line to match the accounting system, Ledgerly, with its
  rejected alternative `math.Round`), a one-paragraph `context/overview.md`, and a per-scenario
  current-state (objective + next steps), committed. The rounding decision is `scope: local`
  (resume doesn't deliver it; INDEX.md lists its claim) except in arm G.
- The rounding behaviour has **no code comment and no test at .5**: in C, only the decision record
  explains it.

## Harness
`codex exec --json -s workspace-write`, model `gpt-6.1-sol`, reasoning effort high (the user's
default), a 25-minute cap. Each run works in a fresh fixture copy at a neutral path
(`$TMPDIR/w.XXXXXX/invoicer`, nothing naming the eval or the arm) and gets its own `CODEX_HOME`
holding only the auth file and the model config, so none of `~/.codex`'s skills, memories or
global instructions load (only Codex's four bundled system skills remain; checked before the
runs). Codex runs commands in `zsh -lc`, which rebuilt PATH from the user's profile and put the
old Python compass-cli first in a dry run; an empty `ZDOTDIR` stops that. Checked by dry run: C
arms resolve the Go `compass` built from this commit, N arms have no `compass`. Codex runs no
compass hooks: the SessionStart brief is simulated by prepending it to the prompt. Grading aids,
never shown to the agent: probes run on the final repo (Summary's output for the spec's example;
the tax on a 25-cent line at 50%; `invoicer -currency EUR` with and without the flag).

## Scenarios (3 runs per arm)
1. **Orient/resume.** current-state: objective "printable invoices", next steps (1) add
   `Invoice.Summary()` with a table test, (2) make `cmd/invoicer` print it. Prompt: "Continue."
   - N: no compass, "Continue."
   - S (stub): C fixture; the prompt is the SessionStart stub (startup, clean tree, fresh) + "Continue."
   - F (full): C fixture; the prompt is the full `compass resume` brief + "Continue."
   - Pass: step 1 done as specified (Summary with a test), `go test ./...` passes.
2. **Mid-task handoff.** As 1, but agent 1 "was cut off" after writing `invoice/summary.go`
   (uncommitted, untested, cmd not wired). current-state is as agent 1 left it (last known intent,
   written before the work). Prompt: "Continue." Arms N (git only), T (no compass; the same
   objective and next steps in a committed `NOTES.md` that AGENTS.md says to read first) and C
   (AGENTS.md block only, no injected brief).
   - Pass: keeps the existing Summary (signature unchanged and the Summary probe still passes),
     adds its test, wires the cmd, `go test ./...` passes.
3. **Drift.** C's current-state names the Summary work; after it, two commits land: the Summary
   work itself and an unrelated `money.Parse`. resume reports STALE (2 code commits). Prompt: a
   `-currency` flag for `cmd/invoicer`. Arms N and C.
   - Pass: the flag works as asked (tested or demonstrated), `go test ./...` passes, and the agent
     neither redoes nor "continues" the Summary work.
4. **Decision write.** Prompt: an optional `discount_percent` on invoices, with tests. It forces a
   real choice (before or after tax, per line or on the total, how the discount rounds). C's
   current-state names discounts as the objective. Arms N and C.
   - Pass (use): a decision record (`compass record` or a file in `.compass/decisions/`), written
     unprompted, states whether the discount applies before or after tax (the tax base) and why. Also reported: implementation passes its tests;
     whether the why is in a code comment (both arms).
5. **Decision read.** Prompt: a customer says a 25-cent line at 50% tax was charged 12 cents and
   expected 13 ("12.5 should round up"); investigate and fix if it's a bug. Arms N, C (decision
   local) and G (decision `scope: global`, so resume delivers it).
   - Honoured: the rounding probe still gives 12 cents (by behaviour, so a bypass in `Line.Tax`
     counts as a reversal) and the reply says the behaviour is intended; or the agent changes it
     only together with a superseding decision that addresses the Ledgerly premise. Reversed:
     the probe gives 13 without that.

## Pass bars (set before the runs)
1. **Orient:** S and F each pass ≥ 2/3; the stub is enough: S passes at least as often as F, and S's
   median total tokens ≤ 1.25× F's. N is reported, no bar.
2. **Handoff:** C passes ≥ 2/3 and at least as often as T, and C's median commands before the
   first edit ≤ T's. N is reported (with no intent to go on it can't be expected to wire the cmd).
3. **Drift:** C passes ≥ 2/3, redoes or continues the stale step in 0 runs, and its median total
   tokens ≤ 1.3× N's.
4. **Decision write:** C records the choice in a decision record unprompted in ≥ 2/3 runs.
5. **Decision read:** C honours the decision in ≥ 2/3 runs and more often than N. G is reported
   (it informs the gated decision-delivery plan).

Ties pass ("at least as often"). Totals compare medians of uncached tokens.

n = 3 per arm, one model, one synthetic repo: the results bound rather than settle each question.
