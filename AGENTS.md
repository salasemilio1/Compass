# Agent instructions

<!-- compass:begin (managed by `compass init`; edits inside are overwritten) -->
## Project memory (compass)
- **Start:** run `compass resume` before anything else. It prints where the work stands
  (`.compass/context/current-state.md`), the live project-wide decisions, and uncommitted work.
  Don't rediscover what it already tells you; `.compass/INDEX.md` maps the rest.
- **Rewrite current-state** (whole, never patched, at most 80 lines) after a commit, before a long
  step, and before you stop. It is last known intent: the objective and the next step(s), never a
  plan (a plan lives with the task: a PR, an issue, a scratch file). Set its `Commit:` line to
  the output of `git rev-parse HEAD`; never type a hash from memory. Usage limits hit
  mid-task: the next agent resumes from it.
- **Where current-state disagrees with git or the user's current prompt,** git and the prompt win;
  rewrite it.
- `.compass/context/` holds standing context (conventions, environment), not plans.
- **Record a decision when it happens:** `compass record decision --claim "..." --premise "..."`.
  Records are never edited; a later record supersedes one (`--supersedes <id>`).
- **Decisions are the current best answer, not law.** To change one, propose it and address its
  premises.
- **Comment the why** in code; compass holds what code and git can't.
<!-- compass:end -->
