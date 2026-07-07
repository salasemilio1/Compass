# build a token-based context-usage nudge, replacing ADE's turn-based one

Date: 2026-07-07
Commits: 088c2d4..HEAD

## Why
Agents have no visibility into how much of their context window is used, so long sessions either
sprawl unnoticed or the user has to eyeball the Claude Code UI and interrupt manually. An existing
global `Stop` hook (`session-nudge.py`, symlinked from the separate ADE repo) approximated this with
a plain turn counter — nudging every 15 turns regardless of how much context those turns actually
cost. A ChatGPT-drafted plan proposed a "measurement tool" that tokenizes the exact prompt sent to
the model plus a calibrated overhead for hidden context; auditing that plan against how Claude Code
actually works found the premise unnecessary — every assistant turn in a session's transcript JSONL
already carries the API's own exact `usage` object, so no tokenization or calibration is needed at
all. This also surfaced that the agent itself had drifted from its own stated policy of preferring
`ctx_execute`/`graphify` for large or unpredictable output, prompting a separate behavioral fix
(see `feedback_treat_context_as_cache.md` in agent memory, not this repo).

## What changed
- `tools/session-context/session_status.py` (new) — pure-measurement module + CLI. Reads a transcript's
  real `usage` data (no tokenization), excludes subagent/sidechain turns, returns raw
  (`used_tokens`/`remaining_tokens`/`percent_used`) and derived (trailing-5-turn growth rate,
  `estimated_turns_remaining`) metrics. No policy — that's left to callers.
- `hooks/session-context-nudge.py` (new) — `Stop` hook. Reads `transcript_path` directly from the
  hook's stdin payload (Claude Code hands it over; no path-guessing needed), imports
  `session_status.py`, and emits one `systemMessage` nudge when usage crosses a threshold (default 75%
  of 200k, configurable via `COMPASS_CONTEXT_NUDGE_THRESHOLD`/`COMPASS_CONTEXT_WINDOW`), then
  self-suppresses for the rest of the session via a `/tmp` marker file.
- `scripts/register_hook.py` (new) — replaces `jq` for `install.sh`'s settings.json hook registration.
  `python3` is already a hard dependency of the hooks themselves, so this removes a second tool
  dependency rather than adding one.
- `install.sh` — added `python3` to `check_prerequisites`; generalized `register_hook` into
  `register_event_hook` (parameterized by event/command/marker) so both `SessionStart` and the new
  `Stop` hook share one registration path; `link_hook` now also symlinks the new hook and
  `tools/session-context/`. Registration failure (e.g. no `python3`) now exits 1 with a clear error
  instead of silently skipping, which is what the old `jq`-missing path did.
- `tests/test_install_guard.sh` — extended the install smoke test to cover the new symlinks and Stop
  hook registration; added a case asserting registration fails loudly (not silently) without `python3`.
- `tests/test_session_status.py` (new) — 8 cases covering usage extraction, sidechain exclusion,
  empty-turns error, trailing-window growth rate, zero-growth edge case, env-based path resolution.
- `.compass/decisions/0004-transcript-based-tool-efficiency-measurement.md` — follow-on design decision
  (not yet built): attribute per-turn token deltas to the tool calls that preceded them, to measure
  whether `ctx_execute`/`graphify` are actually saving tokens in real sessions.
- Operationally: removed the ADE repo's turn-based `session-nudge.py` from `~/.claude/settings.json`'s
  `Stop` hooks and registered the new token-based hook in its place (real global config, not just this
  repo's checkout).

## Assumptions changed
The ChatGPT-drafted plan's core assumption — that context usage must be estimated by tokenizing the
outgoing prompt plus a calibrated overhead — was wrong for this platform. The transcript JSONL already
contains the API's exact `usage` object per turn; measurement should read that, not reconstruct it.

## Learnings
- `learnings/hooks-require-session-restart.md`
- `learnings/claude-project-dir-slug-replaces-non-alphanumeric.md`

## Breaking changes
- Global `~/.claude/settings.json`: the `Stop` hook that previously fired every 15 turns
  (`session-nudge.py`, ADE repo) no longer fires — replaced by the token-based hook here. Anyone still
  relying on the turn-based nudge cross-project would need it reinstated separately; the user explicitly
  requested this swap.
- `install.sh` now requires `python3` as a hard prerequisite (previously only `git`/`claude`) and fails
  the install step if missing, rather than silently degrading.
- The new `Stop` hook only takes effect in sessions started after `install.sh` registers it — Claude
  Code loads hook config once at session start and cannot hot-swap (see the hooks-require-restart
  learning above).
