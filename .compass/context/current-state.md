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
   and logs/2026-07-06-close-audited-gaps.md. All tests passing (28 + 12 bash checks, 25 pytest cases).

## In flight
- The graphify pilot itself (unchanged from before).
- **This repo's own local `.git/hooks/pre-commit` backstop is not yet installed** -- the auto-mode
  permission classifier blocked writing it during the hardening session, even though it was named in
  the approved plan. Needs an explicit go-ahead; the exact command is in the prior session's summary
  (or just re-ask for it).

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
- **Write additional dedup calibration tests rooted in real compass snippets** (actual learnings/,
  logs/, decisions/ content -- not synthetic fixtures) representative of cases the system will actually
  see, to validate/tune `LEXICAL_HIGH` (0.6), `LEXICAL_LOW` (0.1), and `EMBEDDING_THRESHOLD` (0.80) in
  `tools/learning-dedup/check_similarity.py` against real near-duplicate and genuinely-distinct pairs.
- Continue the graphify pilot: use `graphify query` in real sessions for a few days, then merge or
  revert per the original plan.
