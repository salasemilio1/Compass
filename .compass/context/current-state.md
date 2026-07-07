# current state

Date: 2026-07-07
Commit: 088c2d4dbcf7899eb494c31431f8b9e14fa9f7da

## Where things stand
Four threads are live on the `graphify-integration` branch (unmerged, `main` untouched):

1. **graphify pilot** -- integration complete (ADR 0001, deterministic-only mode). Still mid-pilot:
   using `graphify query` day-to-day to see if it earns its keep before merge/revert. As of this
   session, actual usage has been low -- see thread 4.
2. **Deterministic hardening of compass itself** -- closed via `scripts/compass-check.sh`, install
   guard, staleness hook, learnings-dedup filter. See decisions/0002, 0003 and prior logs. Stable,
   no open work.
3. **Context-usage nudge, built and installed this session** -- replaced the ADE repo's turn-count
   `Stop` hook with a token-based one (`hooks/session-context-nudge.py` +
   `tools/session-context/session_status.py`), reading the transcript's real API `usage` data (no
   tokenization). Nudges once at 75% of a 200k context window. Installed for real in
   `~/.claude/settings.json` (not just this repo's checkout). See logs/2026-07-07-context-usage-nudge.md.
   **Needs a fresh session to activate** -- Claude Code loads hook config at session start and can't
   hot-swap (learnings/hooks-require-session-restart.md).
4. **Agent's own context-mode/graphify usage was found to be low** -- an in-session audit found the
   agent defaulting to plain `Bash` with manual truncation instead of `ctx_execute`, and never calling
   `graphify query` at all despite a structural-lookup opportunity. Fixed via a standing behavioral
   memory (`feedback_treat_context_as_cache.md`, in agent memory, not this repo) instructing the agent
   to default to these tools whenever it wouldn't degrade response quality. decisions/0004 designs a
   deterministic way to verify this actually saves tokens, using real transcripts -- not yet built.

## In flight
- **decisions/0004's `tool_efficiency.py` is designed but not implemented.** Attributes per-turn token
  deltas to the tool calls that preceded them (Bash vs. ctx_execute vs. graphify), aggregated by tool
  name, reusing `session_status.py`'s transcript-loading helpers. Next concrete task if picking this
  thread back up.
- **Uncommitted**: everything from this session (session-context-nudge feature + install.sh/jq->python3
  swap + decisions/0004 + this file) is still working-tree changes as of this wrap-up, pending the
  user's commit/push decision.
- The graphify pilot itself continues in parallel -- now with a behavioral nudge to actually use it,
  worth revisiting whether usage picks up in future sessions.
- **This repo's own local `.git/hooks/pre-commit` backstop is still not installed** (carried over from
  a prior session, unrelated to this one's work) -- needs an explicit go-ahead.

## Known issues / things to remember
- Global `~/.claude/skills/*` and hooks are symlinked to *this* repo on the `graphify-integration`
  branch (from running `install.sh --dev` here) -- not the usual `$DOTFILES`-managed install.
  `install.sh` refuses to re-run from here without `--dev` since this checkout isn't `$DOTFILES/compass`.
- `install.sh` now hard-requires `python3` (added this session, replacing a `jq` dependency for hook
  registration) in addition to `git`/`claude`.
- `graphify-out/` and `.compass/learnings/.embedding-cache.json` are gitignored/regenerable, as before.

## Next steps
- Start a new session and confirm the token-based `Stop` hook actually fires near the 75% threshold.
- Continue the graphify pilot; revisit merge/revert once both the pilot and the new behavioral nudge
  have had a real trial run.
- **Design deterministic Bash-vs-ctx_execute routing.** `tool_efficiency.py` (built this session, see
  decisions/0004's caveat addendum) surfaced that relying on the agent's own judgment
  (`feedback_treat_context_as_cache.md`, a system-prompt-level nudge) is inherently non-deterministic
  and unmeasurable cleanly -- the agent decides per-call whether to truncate via Bash or route to
  ctx_execute, which is exactly what makes the tool_efficiency comparison apples-to-oranges. Direction
  floated, not yet decided: a deterministic gate instead of agent discretion -- e.g. an
  allowlist of small/cheap Bash commands (quick git status/log, single-file greps, short lookups) that
  stay on Bash, with everything else defaulting to ctx_execute; possibly a file-size/line-count
  threshold (e.g. more than a few dozen lines forces ctx). Goal: eliminate the biggest muddying
  factors in context deterministically, not via agent-prompt judgment calls. Needs its own ADR once a
  concrete mechanism is chosen.

## Completed this session (2026-07-07, second pass)
- Installed the local `.git/hooks/pre-commit` backstop (runs `compass-check.sh` check-only) --
  closes the item decisions/0002 left pending; user explicitly approved.
- Built `tools/session-context/tool_efficiency.py` per decisions/0004: attributes per-turn token
  deltas to tool calls across all transcripts, reusing `session_status.py`'s `context_total` helper.
  Ran against all 5 real sessions in this project -- see decisions/0004 for the measurement-bias
  caveat found while interpreting its output.
