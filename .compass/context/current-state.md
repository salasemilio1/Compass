# current state

Date: 2026-07-15
Commit: 230017b

## Where things stand
Compass is `main`, steady-state, maintenance-mode. This session added a new `environment/` category
(decisions/0009) — one file per machine/service, keyed by host, overwritten in place — for SSH
access/ports/pinned-version quirks that had no home before, since `learnings/` is topic-keyed and
dedups by text similarity rather than by host. Also added a warn-only `compass-check.sh` check (gap
F) that a repo's top-level directories are mentioned in `context/overview.md`, after finding this
repo's own copy was stale (missing `scripts/`, `cli/`, `tests/`, `tools/`; still referenced the
pre-decisions/0007 `template/` path) — which was the actual reason agents couldn't find
`scripts/compass-check.sh`. Spec version bumped 1 → 2.

## Known issues / things to remember
- `environment/` is private-first/promotable, same tier as `experiments`/`logs`/`learnings` — not
  gitignored by default in committed mode, since that mirrors how this repo already treats similarly
  specific detail in `learnings/`. It often names real hosts/usernames, so promotion to shared should
  be deliberate.
- `scripts/compass-check.sh`'s new repo-layout check is warn-only (never blocks) and matches on a
  literal `` `dirname/ `` substring in `context/overview.md` — it only flags directories git actually
  tracks, so gitignored/generated dirs (`graphify-out/`, `.venv`, etc.) never trigger it.
- Every repo scaffolded under spec version 1 will now show as spec-version-behind (informational
  only) until its `AGENTS.md` stamp is bumped — this is expected, not a regression.
- graphify remains opt-in only (`COMPASS_WITH_GRAPHIFY=1 ./install.sh`); no change this session.
- `.git/hooks/pre-commit` backstop (runs `compass-check.sh` check-only) is installed in this repo.

## Next steps
1. Commit this session's changes (environment category, repo-layout check, spec v2, overview.md fix)
   — currently staged, not yet committed.
2. No other backlog items remain from prior sessions — this session closed the two gaps it found.
3. Compass stays maintenance-mode: fix real breakage as it's found, don't proactively expand scope.
   If `environment/` sees real multi-host usage, revisit whether a flat `environment/<slug>.md` is
   still the right shape (see decisions/0009's follow-up note).
