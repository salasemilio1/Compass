# hardening pass + install topology rework, merge to main

Date: 2026-07-08
Commit: 76b4d9ce61a373ef29630e741831f719242d6292

## What changed
Ran a full audit (code/fault-tolerance + usability/docs) to move compass from "actively developed" to
"steady-state, out of the way." No new features -- every fix converts a silent failure into a loud but
contained one, or fixes a concrete correctness bug.

- `session_status.py`/`tool_efficiency.py`: malformed JSONL lines no longer truncate the rest of a
  transcript's usage data.
- `register_hook.py`: atomic temp-file + `os.replace` write for `settings.json`.
- `compass-check.sh`: INDEX.md archival/supersession rewrites assert entry counts before overwriting;
  `Commit:` stamp now verified to actually resolve via `git`, not just shape-matched -- this immediately
  caught current-state.md's own stamp being a stale/wrong hash.
- `session-context-nudge.py`: surfaces a "context tracking degraded" warning instead of silently
  skipping the nudge when a non-trivial transcript parses to zero turns.
- `tool-routing-guard.py`: closed a check-then-create race on the per-session marker file (`O_EXCL`).
- `compass-session-start.sh`: background `graphify update` is now time-bounded.
- `check_similarity.py`: `UNAVAILABLE` now names the actual exception type.
- Docs: graphify no longer described as "piloting"; warm-up defines a fallback for missing
  current-state.md; wrap-up's dedup contract made explicit (best-effort, reason shown to user).
- Merged `graphify-integration` into `main` (fast-forward, then reconciled two commits that had landed
  directly on `origin/main` -- an old jq-based install.sh fix, superseded by this branch's
  `register_hook.py` rewrite, and a README fix, kept).
- **Install topology rework (decisions/0006, supersedes 0002):** clarified that the real install
  workflow clones a dotfiles repo, runs its bootstrap, then deletes the clone -- incompatible with
  symlinking global skills/hooks to the source. Switched `install.sh` to copy (always overwrite)
  instead, removed the now-pointless `$DOTFILES/compass` canonical-source guard and `--dev` flag. Plain
  `install.sh`, no flags, verified working end-to-end from a temp clone that's deleted immediately after.

## Verification
- Both test suites pass throughout (`bash tests/run_tests.sh`, `uv run --with pytest pytest tests/`).
- Manual `install.sh` smoke tests: symlink-failure -> now install-failure summary (simulated with a
  read-only target dir, confirmed `ln`'s/`cp`'s exit code is actually checked -- an existing bug in
  `create_symlink` that swallowed failures silently was also fixed along the way).
- Manual end-to-end test: cloned this repo to a temp dir, ran `install.sh` with no flags against an
  isolated `COMPASS_HOME`, deleted the clone, confirmed the installed skill content was still present
  and readable.
- `tests/test_install_guard.sh` renamed to `tests/test_install.sh` and rewritten for copy semantics
  (asserts real files not symlinks, survives source deletion, re-install updates content).

## Correction
The initial hardening commit (b9aa9b2, already merged to main) "fixed" `graphifyy` -> `graphify` in
`install.sh`'s installer commands, believing it was a repeated typo -- an Explore-agent audit flagged it
as one, and the double-`y` did look wrong on sight. It's actually correct: `graphifyy` is the real PyPI
package name (installs a `graphify` command), independently documented in
learnings/graphify-cli-quirks.md. Reverted in this session before it broke anyone's install; see
learnings/verify-flagged-installs-directly.md, which had already recorded this exact fact and would have
caught the mistake if consulted first.

## Learnings
- The `$DOTFILES/compass` canonical-source guard (decisions/0002) was designed against the wrong threat
  model -- it assumed a persistent dev checkout that might be *mutated*, not one that's *deleted by
  design* as a normal part of every install. Symlinking and "delete the source after install" are
  fundamentally incompatible; only copying works with an ephemeral source.
- An audit finding that "looks obviously right" (a doubled letter reads as a typo) still needs a check
  against existing `.compass/learnings/` before acting -- two separate learnings already documented that
  `graphifyy` was intentional, and skipping that check nearly reintroduced a real bug.
