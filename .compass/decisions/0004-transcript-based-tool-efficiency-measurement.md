# 0004 — measure context-mode/graphify efficiency by attributing token deltas to tool calls in the transcript

Date: 2026-07-07

## Context

decisions did not yet exist for `session_status.py` (built this session, log to follow) but it
established the core technique this decision reuses: every assistant turn in a session's transcript
JSONL already carries an exact, API-reported `usage` object (`input_tokens` +
`cache_creation_input_tokens` + `cache_read_input_tokens`) — ground truth context consumption, no
tokenization or estimation needed.

Separately, an audit of this session's own tool usage found the agent had been defaulting to plain
`Bash` (with manual `tail`/`head` truncation) for large or unpredictable-size output, and never
invoking `graphify query` at all, despite both being installed specifically to keep context usage
down. The user's framing: context-mode and graphify are a free way to keep the context window clean,
and the agent should default to using them whenever doing so doesn't degrade response quality — not
just when reminded. That behavioral fix is tracked as a memory
(`feedback_treat_context_as_cache.md`), not this decision.

What's still missing is a way to check, deterministically, whether that behavioral fix (or the
tools themselves) actually earns their keep — i.e., whether routing a call through `ctx_execute`
or `graphify query` measurably costs fewer tokens than the equivalent `Bash`/`Read`/`Grep` call
would have, using this specific repo's real sessions as the evidence, not a synthetic benchmark.

## Decision

Extend the transcript-reading approach from `session_status.py` to attribute per-turn token growth
to the tool call(s) that preceded it, then aggregate by tool name:

1. Parse the same session transcript JSONL already used by `session_status.py`, but walk
   `tool_use`/`tool_result` blocks (present in the `assistant`/`user` message content arrays) instead
   of just the trailing `usage` object.
2. For each tool_result, record: the invoking tool name (`Bash`, `mcp__context-mode__ctx_execute`,
   `mcp__context-mode__ctx_search`, `graphify` invocations via Bash, `Read`, `Grep`, etc.), the byte
   size of the tool_result content, and the token delta on the *next* assistant turn's `usage.input_tokens
   + cache_creation_input_tokens` versus the previous turn's cache-read baseline.
3. Aggregate: average and total context-token cost per tool name, across one or more sessions.
   Report only — no policy, no threshold, no verdict baked in, matching `session_status.py`'s
   pure-measurement stance (decisions here are for a human or a future skill to draw, not the tool).

This stays a read-only, deterministic report over real transcripts — no LLM judgment, no synthetic
workload, same philosophy as `session_status.py` and the gap-closing scripts in decisions/0002.

## Alternatives considered

- **Synthetic A/B benchmark** (run the same task twice, once forced through Bash, once through
  context-mode, compare token counts) — more controlled, but doesn't answer the question that
  matters here: what is *actually happening* in real compass sessions. Real transcripts are already
  sitting on disk and are ground truth; a synthetic benchmark would be extra work to approximate
  something we can just measure directly.
- **Self-reported estimate from the agent** (e.g., the agent guesses at end-of-session how much
  context-mode saved) — rejected on the same grounds every other measurement decision in this repo
  has been: an estimate is worse than a number that's already sitting in the transcript.

## Consequences

- Gain: a deterministic, repo-native answer to "is this stack actually saving tokens," usable to
  validate or invalidate the behavioral change in `feedback_treat_context_as_cache.md` after a few
  real sessions accumulate.
- Cost/limitation: token delta attribution is inherently approximate at the margins — a turn can
  contain multiple tool calls before the next `usage` snapshot, and prompt caching means the
  "next turn's cost" isn't a clean 1:1 diff against a single preceding tool_result. The report should
  surface this (e.g. attribute multi-tool turns to the turn as a whole, or flag them as
  low-confidence) rather than silently averaging over it.
- This is additive to `tools/session-context/session_status.py`, not a replacement — likely lands
  as a sibling script in the same directory (e.g. `tool_efficiency.py`) reusing its transcript-loading
  helpers.
