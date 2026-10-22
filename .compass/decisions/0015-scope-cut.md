---
claim: "compass owns only the why, the why-not and where-we-are: v2 skills, hooks, session tools, learnings and layout-drift are cut; restates 0014 as built"
date: 2026-10-22
supersedes:
  - 0004-transcript-based-tool-efficiency-measurement
  - 0005-deterministic-tool-routing-guard
  - 0007-compass-init-as-cli
  - 0008-graphify-becomes-optional-add-on
  - 0009-environment-category-and-repo-layout-check
  - 0011-dedup-by-index-judge-not-embeddings
  - 0014-go-binary-as-built
premises:
  - goal:disposable-sessions
refs:
  - context/positioning.md
  - decisions/0013-plugin-is-a-subdirectory-with-reproducible-binaries.md
---
# compass owns only the why, the why-not and where-we-are: v2 skills, hooks, session tools, learnings and layout-drift are cut; restates 0014 as built

## Context
positioning.md's scope ("Scope: what a compass does", 2026-10-22) gives compass three questions
nobody else answers: why the system is shaped this way, why not the alternatives, and where the
work is and what's next. Discovery, task tracking and session policy belong to other tools. 0014
recorded the Go binary as built but kept v2's `skills/`, `hooks/` and `tools/` in the repo,
installed by nothing.

## Decision
Cut what falls outside that scope:
- **Session policy:** `hooks/session-context-nudge.py`, `hooks/tool-routing-guard.py`,
  `tools/session-context/`, `tools/usage-retro/` (supersedes 0004 and 0005). Token nudges, routing
  guards and usage retros are a separate tool, not memory.
- **v2 workflow:** the `warm-up`, `wrap-up` and `init-compass` skills, `hooks/compass-session-start.sh`
  and SPEC Part B (supersedes 0007). `compass resume`, `compass init` and the plugin's hooks replace them.
- **Learnings:** the category, `.compass/learnings/` here and `tools/learning-dedup/` (supersedes
  0011; dedup is moot). A learning routes to a rule in AGENTS.md, a code comment, or the decision it
  affects. Git history keeps the old files.
- **`layout-drift`:** a repo-layout map is a "where" question the code tree owns (supersedes 0009,
  whose `environment/` category v3 never read either).
- **graphify and session-tooling notes** in this repo's AGENTS.md (supersedes 0008): graphify or an
  LSP is a view of the code that agents pick up on their own; compass doesn't integrate it.

Kept: doctor's and the SessionStart hook's Python-shadow detection. The v2 `compass-cli` still
sits first on PATH on the maintainer's machine, so removing the warning would hide a real fault.

Restated from 0014, which this supersedes because its "skills/hooks/tools stay" line no longer holds:
- compass is one Go binary: the CLI (`init`, `check`, `index`, `record`, `resume`, `doctor`) and
  the hook handler (`compass hook <event>`) over one core in `internal/compass/`. No `stats`
  command; measures come from evals in `experiments/`.
- `--json` only on `check`, `resume`, `record` and `doctor`.
- Packaging and binaries follow 0013 (`make plugin`, committed, byte-checked by `verify-plugin`).
- A session cut off by a usage limit fires no Stop hook; `compass resume`'s git status and diff
  carry its work, and compass doesn't checkpoint mid-task.
- compass claims continuity (a fresh or different agent continues the work), not less discovery.

## Alternatives
- Keep the v2 pieces as reference material: rejected, nothing installs or tests them against v3 and
  they name deleted scripts.
- Keep learnings with 0011's judge-based dedup: rejected, a learning that matters already has an
  owner (a rule, a comment or a decision).
- Drop the Python-shadow detection now: rejected until the v2 CLI is uninstalled where it runs.

## Consequences
- The repo is the Go core, its tests, the plugin and `.compass/`; CI runs only the Go suite, smoke
  and `compass check`.
- The global copies of the v2 skills and session tools under `~/.claude` lose their source repo;
  removing or moving them is the maintainer's call.
