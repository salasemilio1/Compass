# current state

Date: 2026-07-06
Commit: f01c3c9e8b0bb594a53c2d2e9d662361aab82952

## Where things stand
Two threads are live on the `graphify-integration` branch (unmerged, `main` untouched):

1. **graphify pilot** -- integration complete (ADR 0001, deterministic-only mode). Still mid-pilot:
   using `graphify query` day-to-day to see if it earns its keep before merge/revert.
2. **Deterministic hardening of compass itself** -- an audit found five places prompt-only discipline
   could let compass drift toward bloat/staleness. Closed via `scripts/compass-check.sh` (INDEX
   archival, ADR supersession, current-state Commit: anchor), `install.sh`'s canonical-source guard, a
   staleness check in the SessionStart hook, and a three-stage learnings-dedup filter. See decisions/0002
   and logs/2026-07-06-close-audited-gaps.md.
3. **Learnings-dedup calibrated against real content** -- 10 hard-case tests built from this repo's
   actual `.compass/learnings/*.md` content found two real (not hypothetical) false negatives in the
   "confidently novel, skip the model" short-circuit. Dropped that short-circuit -- stage 3 now runs on
   every candidate stage 1/2 doesn't already confidently resolve. See decisions/0003 and
   logs/2026-07-06-dedup-calibration-and-wrapup-ux.md. All tests passing (28 + 12 bash checks, 35 pytest
   cases).

## In flight
- The graphify pilot itself (unchanged from before).
- **This repo's own local `.git/hooks/pre-commit` backstop is not yet installed** -- the auto-mode
  permission classifier blocked writing it during the hardening session, even though it was named in
  the approved plan. Needs an explicit go-ahead; the exact command is in the prior session's summary
  (or just re-ask for it).
- **wrap-up's review flow was streamlined** (skills/wrap-up/SKILL.md step 7): summarize each artifact
  in 2-3 sentences, one consolidated write/commit/push question round, no full-content dump by default.
- **The global `new-test`/`audit-tests` skills gained a 6th rule** (assert correct behavior, not
  observed behavior) in the separate dotfiles repo (`$DOTFILES`, not this checkout) -- those edits are
  uncommitted there as of this session; that repo's own commit is out of scope for this repo's wrap-up.

## Known issues / things to remember
- Global `~/.claude/skills/*` and the `SessionStart` hook are currently symlinked to *this* repo on the
  `graphify-integration` branch (from running `install.sh` here during the pilot) — not wherever your
  usual dotfiles-managed compass install normally lives. Backups of the previous symlinks exist
  (`*.backup-20260704100310`) if you need to revert before merging. Note: `install.sh` will now refuse
  to re-run from here without `--dev`, since this checkout isn't `$DOTFILES/compass`.
- `graphify-out/` is gitignored and regenerable — a fresh clone needs one `graphify update .` to
  bootstrap it before the auto-refresh hook has anything to refresh.
- `.compass/learnings/.embedding-cache.json` is gitignored and regenerable (dedup tool's vector cache).

## Next steps
- Decide on the `.git/hooks/pre-commit` backstop (install it, or explicitly skip it).
- Commit the `new-test`/`audit-tests` SKILL.md edits in the dotfiles repo (separate repo, separate
  commit) if they haven't been already.
- Continue the graphify pilot: use `graphify query` in real sessions for a few days, then merge or
  revert per the original plan.
