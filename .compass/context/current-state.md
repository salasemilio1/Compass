# current state

Date: 2026-07-08
Commit: 76b4d9ce61a373ef29630e741831f719242d6292

## Where things stand
Compass is merged to `main` and considered steady-state software: not under active development,
hardened to run unattended. `graphify-integration` was fast-forwarded/merged into `main` and can be
deleted once you're satisfied the reinstall from `main` works.

A hardening pass (this session) closed the fault-tolerance audit backlog and one newly-surfaced
architectural gap:

1. **Silent-failure fixes** -- one malformed transcript JSONL line no longer truncates all later usage
   data (`session_status.py`/`tool_efficiency.py`); `register_hook.py` writes `settings.json` atomically
   (temp file + `os.replace`); `compass-check.sh`'s INDEX.md archival/supersession rewrites now assert
   entry counts before overwriting; `compass-check.sh`'s `Commit:` stamp is now verified to actually
   resolve via git, not just shape-matched (this caught current-state.md's own stamp being stale/wrong);
   `tool-routing-guard.py`'s per-session marker file is now created atomically (`O_EXCL`), closing a
   double-fire race; `compass-session-start.sh`'s background `graphify update` is now time-bounded.
2. **Install topology rebuilt (decisions/0006, supersedes 0002).** The real install workflow turned out
   to be: clone a dotfiles repo, run its bootstrap (which invokes `compass/install.sh`), delete the
   clone. That's incompatible with symlinking global skills/hooks to the source checkout -- deleting the
   clone would silently break everything. `install.sh` now **copies** (always overwrites) instead of
   symlinking, so installed files have no dependency on the source after install finishes. This also
   made the `$DOTFILES/compass` canonical-source guard and `--dev` flag pointless (the risk they guarded
   against no longer exists), so both were removed. Plain `install.sh`, no flags, now works from any
   temp clone -- verified end-to-end (see logs/2026-07-08-hardening-and-install-rework.md).
3. **graphify** -- integration complete (decisions/0001, deterministic-only mode), no longer described
   as "piloting" anywhere (AGENTS.md updated). Organic usage was zero; addressed via decisions/0005's
   deterministic tool-routing guard rather than more nudging.
4. **Context-usage nudge** (`hooks/session-context-nudge.py`) and **tool-routing guard**
   (`hooks/tool-routing-guard.py`) are both installed via the new copy-based `install.sh` on this
   machine. Not yet confirmed firing across a full real session post-hardening -- worth watching for in
   the next few sessions, not worth a dedicated verification task on its own.

## Remaining backlog (from logs/2026-07-07-fault-tolerance-audit.md), explicitly deferred
Evaluated during hardening and intentionally left alone -- low payoff for a solo user, revisit only if
the pain actually shows up:
- Embedding cache (`.compass/learnings/.embedding-cache.json`) keyed by content hash only, not
  `(model_name, hash)`.
- `logs/archive.md`/`experiments/archive.md` growing unbounded once entries overflow into them.
- No `.compass/`-directory-level version/layout marker.
- No inventory of which repos have a compass, to push `SPEC.md` version bumps out to.
- graphify has no `UNAVAILABLE`-equivalent signal if its CLI output format changes upstream.
- End-to-end tests of the warm-up/wrap-up skill markdown itself; dedicated tests for
  `register_hook.py`/`session-context-nudge.py` beyond what's covered today.
- `skills/init-compass/SKILL.md`'s template-path assumption -- low-frequency skill, fix opportunistically
  if it's ever touched for another reason.

## Known issues / things to remember
- `install.sh` hard-requires `python3` (used for hook registration, including the JSON `matcher` field
  `PreToolUse` hooks need) in addition to `git`/`claude`.
- `graphify-out/` and `.compass/learnings/.embedding-cache.json` are gitignored/regenerable.
- `.git/hooks/pre-commit` backstop (runs `compass-check.sh` check-only) is installed in this repo.
- `tests/test_install_guard.sh` was renamed to `tests/test_install.sh` and rewritten for the copy-based
  install (no more canonical-source guard to test).

## Next steps
1. Confirm the copy-based `install.sh` works from a real dotfiles-bootstrap flow (not just the manual
   temp-clone smoke test done this session), then delete the `graphify-integration` branch.
2. Watch a few ordinary sessions to confirm the `Stop`/`PreToolUse` hooks fire without issue under the
   hardened code paths -- no dedicated task needed unless something looks wrong.
3. From here on, compass is maintenance-mode: fix real breakage as it's found, don't proactively expand
   scope.
