# current state

Date: 2026-07-08
Commit: 083360e52ad5ccae50ff3f14e3d2e2d627e128e8

## Where things stand
Compass is `main`, steady-state, maintenance-mode. This session was a proactive audit ("make sure the
last changes went smooth, patch issues before they hit me") plus one feature built from a design
discussion that came out of it.

1. **Audit finding, fixed:** `hooks/tool-routing-guard.py` denied backgrounded dev-server commands
   (`npm run dev` + `run_in_background: true`) on first use every session, redirecting them toward
   `ctx_execute` -- nonsensical for a process that never exits. Fixed by exempting any
   `run_in_background` call from all routing rules. See
   `logs/2026-07-08-tool-routing-guard-background-fix.md`,
   `learnings/pretooluse-guard-must-exempt-backgrounded-commands.md`.
2. **New: `compass init` CLI (`cli/`).** Extracted `init-compass`'s mechanical scaffolding (dirs,
   `INDEX.md`, `.gitignore`, `AGENTS.md`, `CLAUDE.md`) out of the markdown skill into a real,
   lightweight Click CLI, installed via `install.sh` (uv/pipx/pip fallback, same pattern as
   graphify). This fixes the previously-deferred template-path bug by construction -- the template is
   now bundled inside the package (`importlib.resources`), not searched for on disk. `warm-up`/
   `wrap-up` were evaluated for the same treatment and explicitly scoped out (judgment-heavy, weak
   fit for a CLI). See `decisions/0007-compass-init-as-cli.md`,
   `logs/2026-07-08-compass-init-cli.md`.

Verified this session (not yet re-verified after next steps below):
- Confirmed the real global install (`~/.claude/`) on this machine matches HEAD byte-for-byte (every
  hook, tool, skill diffed) -- the copy-based `install.sh` from decisions/0006 is genuinely deployed,
  not just claimed.
- `graphify-integration` branch is already gone (local and remote) -- decisions/0006's cleanup step
  is done.
- Full suite green: bash (`tests/run_tests.sh`, now 29 checks in `test_install.sh`), root python (71
  tests), and the new `cli/tests` (12 tests).

## Known issues / things to remember
- `install.sh` hard-requires `python3` (hook registration) and now also benefits from
  `uv`/`pipx`/`pip` (compass CLI, graphify) and `node` (context-mode) -- all three of the latter
  degrade to a warning, never a hard failure.
- `graphify-out/` and `.compass/learnings/.embedding-cache.json` are gitignored/regenerable.
- `.git/hooks/pre-commit` backstop (runs `compass-check.sh` check-only) is installed in this repo.
- The `compass` command/distribution name isn't checked for PyPI collisions -- fine for a local-only
  install; revisit if `compass-cli` is ever published (see decisions/0007's naming note).

## Next steps
1. **Commit this session's changes** (uncommitted at time of writing: the tool-routing-guard fix, the
   entire `cli/` package, install.sh/README/SPEC/skill updates, and this wrap-up's `.compass/`
   entries).
2. Watch a few ordinary sessions to confirm `compass init` behaves well when an agent drives it
   (not just the direct-terminal smoke test done this session) -- particularly whether the
   mode-selection prompt or `--mode` flag path gets exercised correctly from inside Claude Code.
3. Re-run `install.sh` on this machine (or confirm it already ran) so the fixed
   `tool-routing-guard.py` and the new `compass` CLI are actually live, not just committed to the
   repo -- diff `~/.claude/hooks/tool-routing-guard.py` against HEAD to confirm.
4. Revisit decisions/0007's follow-up question opportunistically, not proactively: is a
   `compass status` CLI command worth it given `compass-session-start.sh` already covers the same
   deterministic floor for warm-up.
5. From here on, compass stays maintenance-mode: fix real breakage as it's found, don't proactively
   expand scope beyond what this session already added.
