# Agent instructions

[graphify](https://github.com/safishamsi/graphify) is an optional, opt-in structural navigation layer
(not installed by default — see README.md). If `graphify-out/graph.json` happens to exist, prefer
`graphify query "<question>"` over grepping for "what calls X" / "where does Y live" style questions.
compass stays the source of curated intent; graphify is read-only reference material, never written or
modified by compass workflows.

If asked how many tokens the session has used, run
`python3 ~/.claude/tools/session-context/session_status.py` (installed globally, works from any repo)
instead of estimating — it reads the session's transcript for exact API-billed usage. A `Stop` hook
(`session-context-nudge.py`) already nudges toward `/wrap-up` once usage crosses 75%, once per session.

<!-- compass:begin (managed by `compass init`; edits inside are overwritten) -->
## Project memory (compass)
- **Start:** run `compass resume` before anything else. It prints where the work stands
  (`.compass/context/current-state.md`), the live project-wide decisions, and uncommitted work.
  Don't rediscover what it already tells you; `.compass/INDEX.md` maps the rest.
- **Rewrite current-state** (whole, never patched, at most 80 lines: objective, just done, files in
  play, next steps) after a commit, before a long step, and before you stop. Set its
  `Commit:` line to the output of `git rev-parse HEAD`; never type a hash from memory.
  Usage limits hit mid-task: the next agent resumes from it.
- **Record a decision when it happens:** `compass record decision --claim "..." --premise "..."`.
  Records are never edited; a later record supersedes one (`--supersedes <id>`).
- **Decisions are the current best answer, not law.** To change one, propose it and address its
  premises.
- **Comment the why** in code; compass holds what code and git can't.
- Run `compass check` before committing.
<!-- compass:end -->
