# Agent instructions

<!-- compass:begin (managed by `compass init`; edits inside are overwritten) -->
## Project memory (compass)
- **Start:** run `compass resume` before anything else. It prints where the work stands
  (`.compass/context/current-state.md`), the live project-wide decisions, and uncommitted work.
  Don't rediscover what it already tells you; `.compass/INDEX.md` maps the rest.
- **Before changing behaviour,** search the records: `rg -il <words> .compass/` lists the files that mention the words.
- **Rewrite current-state** (whole, never patched, at most 80 lines) after a commit, before a long
  step, and before you stop. It is last known intent: the objective and the next step(s), never a
  plan (a plan lives with the task: a PR, an issue, a scratch file). Set its `Commit:` line to
  the output of `git rev-parse HEAD`; never type a hash from memory. Usage limits hit
  mid-task: the next agent resumes from it.
- **Where current-state disagrees with git or the user's current prompt,** git and the prompt win;
  rewrite it.
- `.compass/context/` holds standing context (conventions, environment), not plans.
- **Record a decision** when you pick between real alternatives and a later agent could
  reasonably undo it (the rejected option looks simpler, or the reason lies outside the code):
  `compass record decision --claim "..." --premise "..." --body-file -`, the body saying why and
  which alternatives were rejected. A choice the code or a comment explains is not a decision.
  One choice per record, and its claim names that choice.
- **Supersede only to replace.** Records are never edited once committed. Pass
  `--supersedes <id>` only when that record's claim is no longer true as written; the new claim
  then states the whole current answer (keep `--global` if the old one had it). A choice that
  adds to or builds on a decision still in force is a new record citing it (`--premise decision:<id>`).
- **Decisions are the current best answer, not law.** To change one, propose it and address its
  premises.
- **Comment the why** in code; compass holds what code and git can't.
<!-- compass:end -->
