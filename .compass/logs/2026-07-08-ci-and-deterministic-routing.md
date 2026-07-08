# CI + deterministic tool-routing guard

Date: 2026-07-08
Commits: e905c23..HEAD

## Why
Working through the fault-tolerance audit backlog (logs/2026-07-07-fault-tolerance-audit.md) in
priority order. Gap #1 (no CI) was first. While reviewing decisions/0004's tool-usage data with the
user, confirmed `graphify query` has never once been called across 6 real sessions and `ctx_execute`
fired only 5 times against 243 plain `Bash` calls — the prompt-level nudge
(`feedback_treat_context_as_cache.md`) isn't moving behavior, exactly the failure mode already solved
once before for `learnings/` dedup. User asked to move straight to deterministic routing.

## What changed
- **`.github/workflows/ci.yml`** — runs `tests/run_tests.sh` (bash) and `uv run --with pytest pytest
  tests/` (python) on every push/PR. Closes audit gap #1. Verified locally first (47 bash checks + 43
  pytest tests passing) before relying on the new workflow.
- **`hooks/tool-routing-guard.py`** (new `PreToolUse` hook, matched on `Bash`) — denies command shapes
  known to be heavy (full `git diff`, unbounded `git log`, test/build runners) or broad
  (recursive grep/ripgrep/unrestricted `find`), redirecting toward `ctx_execute` or `graphify query`
  respectively. See decisions/0005 for the full design and the safety-valve rationale.
- **`scripts/register_hook.py`** — added optional `matcher` argument (`PreToolUse` needs one; the
  existing `SessionStart`/`Stop` hooks didn't).
- **`install.sh`** — links the new hook, registers it with `matcher: "Bash"`.
- **Tests** — `tests/test_tool_routing_guard.py` (27 cases: classification rules + the stdin/stdout
  contract + the retry safety valve + fail-open on malformed input), plus new assertions in
  `tests/test_install_guard.sh` for the new hook's linking/registration.

## Assumptions changed
Tool routing is no longer purely agent discretion for the command shapes the guard covers. Everything
else (small greps, git status, etc.) is untouched.

## Learnings
- `pretooluse-hooks-fail-closed.md` — PreToolUse hooks block on hook-script crash/timeout, unlike
  SessionStart/Stop; discovered while designing the guard's error handling.

## Breaking changes
None — the guard fails open on any internal error, and each rule only fires once per session before
allowing through unconditionally (so an unavailable `ctx_execute`/`graphify` can't deadlock a session).
Not yet observed running for real in a fresh session (hooks require a session restart to activate,
per `learnings/hooks-require-session-restart.md`).

## Next steps
- Start a fresh session to confirm the `PreToolUse` hook actually fires and denies as designed.
- After a handful of real sessions accumulate under the gate, re-run `tool_efficiency.py --all` to see
  whether `ctx_execute`/`graphify` usage actually increased.
- Continue the audit backlog: gap #3 (defensive transcript parsing in `session_status.py` /
  `tool_efficiency.py`) is next.
