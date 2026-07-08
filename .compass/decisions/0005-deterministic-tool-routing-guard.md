# 0005 — deterministic PreToolUse guard for Bash-vs-ctx/graphify routing

Date: 2026-07-08

## Context

decisions/0004 measured tool usage across real sessions and found `graphify query` had **never
been called, in any session**, and `ctx_execute`/`ctx_batch_execute` had only fired 5 times total
against 243 plain `Bash` calls. The existing fix for this (`feedback_treat_context_as_cache.md`, a
behavioral memory telling the agent to default to these tools) is a prompt-level suggestion, and the
data shows it isn't moving behavior. This is the same failure mode compass has already solved once
before for `learnings/` dedup (decisions/0002, 0003): leaving a correctness-relevant judgment call
to agent discretion doesn't reliably work, and the fix there was to replace judgment with a
deterministic filter. This decision applies the same fix to tool routing.

## Decision

A `PreToolUse` hook (`hooks/tool-routing-guard.py`), matched on the `Bash` tool only, classifies the
raw command string against a small, explicit rule set and can **deny** the call outright — not
suggest, actually block it — with a reason fed back to the model:

1. **graphify-first**: a repo-wide symbol search (recursive `grep`/`ripgrep`, or an unrestricted
   `find`) is denied toward `graphify query`, but only when `graphify-out/graph.json` already exists
   for the repo.
2. **ctx-first**: a command shape known to produce large output regardless of content — a full
   `git diff` (no `--stat`), an unbounded `git log`, a test/build runner (`pytest`, `npm test`,
   `make`, etc.), or the same broad grep/find as above when no graphify graph exists — is denied
   toward `ctx_execute`.

Classification is purely shape-based (regex on the command string), matching the standing rule for
everything else in this repo: no heuristic guess about the size of *this specific* call's output,
only known-heavy/known-broad command shapes.

**Safety valve:** each rule fires at most once per session (tracked via a temp marker file, same
mechanism as `hooks/session-context-nudge.py`'s per-threshold marker). A second Bash call matching
the same rule in the same session is allowed through unconditionally. This exists because
`context-mode`'s MCP server was observed disconnected mid-session while building this — a hard,
always-on deny would turn an unrelated infrastructure hiccup into a dead end where the agent can
neither run the original command nor reach the tool it's being redirected to. The guard also never
denies on its own internal error (parse failure, missing fields) — it fails open, since Claude
Code's hook runtime already fails *closed* on a hook crash/timeout, and compounding that would make
an ordinary `git status` unable to run.

## Alternatives considered

- **Keep it a prompt-level nudge, just phrase it more strongly.** Rejected — decisions/0004's data is
  the direct evidence this doesn't work; a stronger sentence in a memory file isn't a different
  mechanism.
- **Hard deny, no safety valve.** More literally "determinism as enforcement," but fails badly the
  moment the alternate tool is genuinely unavailable (observed live during this session). Rejected as
  incompatible with the fault-tolerance goal this whole audit thread is about.
- **`ask` instead of `deny`** (defer to a human permission prompt each time). Considered and rejected
  by the user: it reintroduces a manual step for something that should resolve itself, and the
  retry-once-then-allow valve already provides the human escape hatch implicitly (nothing gets stuck).
- **Broader command coverage** (also flag `cat`/`head`/`tail` of arbitrary files). Deferred — output
  size for those is genuinely unknown from the command shape alone, unlike `git diff`/test runners/
  broad greps, which are heavy by construction. Adding it now risked false positives on legitimately
  tiny `cat` calls; narrow-but-precise was prioritized over broad-but-noisy.

## Consequences

- Gain: tool routing is now enforced the same way `compass-check.sh` enforces its invariants —
  mechanically, independent of whether the agent remembers to follow the suggestion.
- Cost: this is the first compass hook that can block a *live* tool call mid-session, not just a git
  commit. Blast radius is global once installed via `install.sh` (every project, every session). The
  safety valve and fail-open-on-error are load-bearing, not optional hardening — see the "PreToolUse
  hooks fail closed on crash/timeout" learning this decision produced.
- Follow-up: `tool_efficiency.py` (decisions/0004) can now be re-run after enough sessions accumulate
  under this gate to see whether `ctx_execute`/`graphify` usage actually increased — the thing the
  prompt-only nudge failed to move.
