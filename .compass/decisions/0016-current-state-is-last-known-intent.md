---
claim: "current-state is last known intent: objective and next steps only, never a plan; git and the user's current prompt win over it"
date: 2026-10-22
premises:
  - goal:disposable-sessions
refs:
  - context/positioning.md
  - experiments/2026-10-21-v3-core-eval.md
---
# current-state is last known intent: objective and next steps only, never a plan; git and the user's current prompt win over it

## Context
current-state carried Objective, Just done, Files in play and Next steps. Just done restates git;
Files in play is a "where" the code tree and `git status` own, and in the core eval a restated
current-state invented five of them (`file-in-play-missing` caught them, at the cost of a check).
A step-by-step plan in current-state goes stale the moment the work turns.

## Decision
- current-state is **last known intent**: `## Objective` and `## Next steps` only, at most 80
  lines, never a plan. A plan lives with the task (a PR, an issue, a scratch file).
- Where it disagrees with git or the user's current prompt, they win, and the agent rewrites it;
  the STALE line, the Stop/PreCompact reason and the AGENTS.md block say so.
- `file-in-play-missing` is gone. A line that is only a template `<placeholder>` doesn't count as
  a next step (check warns `next-steps-missing`) and is never shown as the objective or next step.
- `.compass/context/` holds standing context (conventions, environment), not plans.
- The AGENTS.md block no longer says "run `compass check`": the pre-commit hook runs it.
- Older files keep working: Just done, Known issues and Files in play are trimmed first by resume.

## Alternatives
- Keep Files in play with its existence check: rejected, it guards a list that shouldn't exist.
- Keep a "done" section for handoff: rejected, git log and the uncommitted diff carried the Codex
  handoff in the core eval; current-state didn't.

## Consequences
- An unfilled template now passes `check` with a warning instead of failing on `<path>`.
- Repos with the old block see `agents-block-drift` until `compass init` is re-run.
