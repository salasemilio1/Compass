# Pre-commit backstop + tool_efficiency.py

Commits: babd0fa..HEAD

## Why
Two carried-over threads: the local `.git/hooks/pre-commit` backstop (decisions/0002) was blocked by
the permission classifier in an earlier session and left pending an explicit go-ahead; `tool_efficiency.py`
was designed in decisions/0004 but not yet built, waiting on real transcript data to aggregate.

## What changed
- Installed `.git/hooks/pre-commit` (local, untracked) running `compass-check.sh` check-only — closes
  decisions/0002's residual item. User gave explicit go-ahead this session.
- Built `tools/session-context/tool_efficiency.py`: attributes per-turn token deltas to the
  tool_use/tool_result pairs that produced them, aggregated by tool name across one or more session
  transcripts, reusing `session_status.py`'s `context_total` helper. Supports `--transcript` (repeatable)
  or `--all` (scan this project's transcript directory). Flags multi-tool-call windows as
  `low_confidence` since the delta can't be cleanly attributed to one tool.
- Ran it against all 5 real transcripts in this project.

## Assumptions changed
- decisions/0004 assumed raw per-tool token averages could be compared directly. Running it for real
  surfaced a selection-bias caveat (documented as an addendum in decisions/0004): Bash's own
  self-truncation (`head`/`tail`/`-n`) hides the tokens it would have cost without truncation, while
  `ctx_execute`/`graphify` are reached for on harder, larger jobs by construction — so raw averages
  structurally favor Bash regardless of which tool is actually more context-efficient.
- Floated (not yet decided, captured in `context/current-state.md`'s Next steps): replace the current
  system-prompt-level agent judgment call (`feedback_treat_context_as_cache.md`) with a deterministic
  routing rule — an allowlist of small Bash commands, or a line/size threshold forcing `ctx_execute` —
  both to actually eliminate context bloat and to make future `tool_efficiency.py` comparisons clean.
  Needs its own ADR once a concrete mechanism is chosen.

## Learnings
- learnings/compass-agent-portability-boundary.md

## Breaking changes
None.
