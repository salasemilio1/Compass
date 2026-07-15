# environment category + repo-layout drift check

Date: 2026-07-15
Commits: 230017b..HEAD

## Why

User wants machine/infra quirks (SSH access, non-default ports, pinned versions tied to a specific
box) reliably captured so agents stop re-hitting the same environment gotcha. Separately, while
investigating why agents couldn't find `scripts/compass-check.sh`, found `context/overview.md`'s
repo-layout list was stale and nothing checked for that. See decisions/0009 for the full reasoning.

## What changed

- `SPEC.md`: new `environment/` category (directory layout, category table, sharing table, INDEX.md
  format, entry conventions, lifecycle); spec version 1 → 2.
- `cli/compass_cli/commands/init.py`: `environment` added to `CATEGORIES` (scaffolded automatically);
  `SPEC_VERSION` bumped to 2. `cli/tests/test_init.py` updated to assert the version dynamically.
- `cli/compass_cli/template/INDEX.md`: `## environment` section + folder-guide row.
- `skills/warm-up/SKILL.md`: step 5 now checks `environment/<host>.md` before touching a named remote
  target.
- `skills/wrap-up/SKILL.md`: new artifact question for environment quirks (separate from
  `learnings/`), dedup check extended to `environment/`, visibility note updated.
- `scripts/compass-check.sh`: new gap F, `check_repo_layout` — warns (never blocks) if a git-tracked
  top-level directory isn't mentioned in `context/overview.md`. `tests/test_compass_check.sh` covers
  it (no-overview no-op, stale-overview warns, already-mentioned dir not flagged).
- This repo's own `.compass/`: added `environment/` dir, `INDEX.md` sections/folder-guide row, bumped
  `AGENTS.md`'s spec-version stamp to 2, and fixed `context/overview.md`'s repo layout (was missing
  `scripts/`, `cli/`, `tests/`, `tools/`; also still listed the pre-decisions/0007 `template/` path).

## Assumptions changed

- `environment/` sits in the same sharing tier as `logs`/`experiments`/`learnings` (private-first,
  promotable) rather than getting its own gitignore rule — consistent with how this repo already
  treats similarly specific operational detail in `learnings/` (e.g. `graphify-cli-quirks.md`).

## Learnings

- None new — see decisions/0009 for the two gaps this closes; no additional gotchas surfaced during
  implementation.

## Breaking changes

- Spec version 2. Repos scaffolded under version 1 aren't broken (compass-check.sh's version-drift
  check is warn-only), but their `AGENTS.md` stamp will now show as behind until manually reviewed.
