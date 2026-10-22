---
claim: compass is one Go binary (CLI + hook handler, no stats, --json only on check/resume/record/doctor, binaries per 0013); v2 installer/scripts/cli deleted; it claims continuity, not discovery
date: 2026-10-22
supersedes:
  - 0010-go-binary-as-claude-code-plugin
premises:
  - goal:disposable-sessions
refs:
  - decisions/0013-plugin-is-a-subdirectory-with-reproducible-binaries.md
  - experiments/2026-10-21-v3-core-eval.md
  - context/positioning.md
---
# compass is one Go binary (CLI + hook handler, no stats, --json only on check/resume/record/doctor, binaries per 0013); v2 installer/scripts/cli deleted; it claims continuity, not discovery

## Context
0010 described the Go binary before it was built. Three of its claims don't hold for what shipped:
a `stats` command, `--json` on every command, and binaries built by a CI release job. The core eval
(experiments/2026-10-21-v3-core-eval.md) also found two things 0010 didn't address: what carries a
session that is cut off mid-task, and whether compass reduces discovery at all.

## Decision
- compass is one Go binary: the CLI (`init`, `check`, `index`, `record`, `resume`, `doctor`) and the
  hook handler (`compass hook <event>`) over one core in `internal/compass/`. There is **no `stats`
  command**; measures come from evals recorded in `experiments/`. The MCP server stays deferred to
  the gated delivery plan.
- `--json` exists where it is implemented: `check`, `resume`, `record`, `doctor`. `init`, `index`,
  `hook` and `version` have none; add it when a caller needs it.
- Packaging and binaries follow [0013](0013-plugin-is-a-subdirectory-with-reproducible-binaries.md):
  the plugin is `./plugin`, binaries are built locally with `make plugin`, committed, and CI
  byte-checks them (`make verify-plugin`). CI does not build the shipped binaries.
- v2's `install.sh`, `scripts/` and `cli/` are deleted, with the tests and CI steps that covered
  them. `skills/`, `hooks/` and `tools/` stay in the repo with their tests, installed by nothing.
- **Cut-off sessions (eval issue 6):** a session stopped by a usage limit fires no Stop hook, and no
  agent rewrote current-state "before a long step". The resume brief's `git status` and
  `git diff --stat HEAD` are what carry cut-off work; compass does not try to checkpoint mid-task.
- **Continuity, not discovery (eval issue 12):** on specified, code-local tasks v3 showed no
  discovery reduction against no compass (78.5k vs 67.3k context at first edit, n=2); agents found
  plans by grep. compass claims that a fresh or different agent continues the work, not that it
  reads less. positioning.md's Measures say so.

## Consequences
- README and SPEC no longer point at the v2 installer or checker; SPEC Part B is history.
- The v2 skills still name the deleted checker, installer and CLI; they are reference material until replaced.
- Whether the SessionStart stub is enough on an unspecified "continue" task is still to be measured.
