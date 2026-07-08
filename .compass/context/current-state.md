# current state

Date: 2026-07-08
Commit: e905c23fbc7e10adfe0f2b0eb0d15aeed5b4f38a

## Where things stand
On the `graphify-integration` branch (unmerged, `main` untouched). Three live threads:

1. **graphify pilot** -- integration complete (decisions/0001, deterministic-only mode). Usage has
   been zero across every real session measured so far (see thread 3) -- being addressed
   deterministically rather than by continuing to wait for organic adoption.
2. **Context-usage nudge** -- token-based `Stop` hook (`hooks/session-context-nudge.py` +
   `tools/session-context/session_status.py`) reading real API `usage` data, nudges at 75% of a 200k
   window. Installed globally. Not yet confirmed firing in a real session (needs a session start after
   install to activate -- learnings/hooks-require-session-restart.md).
3. **Fault-tolerance audit + deterministic tool routing (2026-07-07/08)** -- stepped back to review
   compass as long-lived software rather than a working prototype. Findings and full gap list:
   logs/2026-07-07-fault-tolerance-audit.md. Core pattern (deterministic filters, prompts as
   suggestions) holds; gaps are at the edges. Working the backlog below in priority order. First fix
   (CI) is done; second fix (deterministic tool-routing guard, decisions/0005) was built and committed
   but, it turned out, never actually deployed -- `install.sh --dev` hadn't been re-run since, so it had
   no symlink in `~/.claude/hooks/` and no `PreToolUse` entry in `settings.json`
   (learnings/decision-installed-claim-vs-actual-deployment.md). Fixed 2026-07-08 by re-running
   `install.sh --dev`; the guard is now genuinely installed but **still not yet confirmed firing in a
   real session** -- same activation caveat as thread 2, now for the first time actually testable.
4. **`warm-up` system health check (2026-07-08)** -- `skills/warm-up/SKILL.md` now runs a quick
   deterministic check (ctx MCP connected? graphify CLI installed?) as step 2 of every warm-up, reported
   as a one-line status. Added after confirming, in the course of thread 3's investigation, that
   `context-mode` MCP was down (unrelated system cause: a Homebrew `llhttp` version mismatch broke the
   local Node install; fixed by relinking, not a compass code change).

## Audit backlog (logs/2026-07-07-fault-tolerance-audit.md), in priority order
1. ~~No CI~~ -- **done.** `.github/workflows/ci.yml` runs both test suites on every push/PR.
2. ~~Tool routing left to agent judgment~~ -- **done, actually deployed 2026-07-08, unconfirmed live.**
   `hooks/tool-routing-guard.py` (decisions/0005) denies heavy/broad `Bash` calls toward
   `ctx_execute`/`graphify query`, with a per-session retry safety valve. Next: start a fresh session
   and verify it actually fires.
3. `session_status.py`/`tool_efficiency.py` don't defensively handle corrupted/partial transcript
   JSONL lines -- can throw uncaught inside the `Stop` hook path. **Next up.**
4. Global skills/hooks are symlinked via `install.sh --dev` from this branch, not `$DOTFILES/compass`
   -- no detection if the branch is later deleted/rebased.
5. `register_hook.py`/`install.sh`'s settings.json mutation has no post-write validation.
6. Embedding cache (`.compass/learnings/.embedding-cache.json`) keyed by content hash only, not
   `(model_name, hash)` -- stale vectors on retune.
7. No inventory of which repos have a compass, to push `SPEC.md` version bumps out to.
8. No `.compass/`-directory-level version marker (layout, not just content-convention, drift).
9. graphify has no `UNAVAILABLE`-equivalent signal if its CLI output format changes upstream.
10. `logs/archive.md`/`experiments/archive.md` grow unbounded once the entry cap archives into them.

## Known issues / things to remember
- Global `~/.claude/skills/*` and hooks are symlinked to *this* repo on the `graphify-integration`
  branch (from running `install.sh --dev` here) -- not the usual `$DOTFILES`-managed install. This is
  audit-backlog item 4 above, not yet fixed.
- `install.sh` hard-requires `python3` (used for hook registration, including the JSON `matcher`
  field `PreToolUse` hooks need) in addition to `git`/`claude`.
- `graphify-out/` and `.compass/learnings/.embedding-cache.json` are gitignored/regenerable.
- `.git/hooks/pre-commit` backstop (runs `compass-check.sh` check-only) is installed in this repo.

## Next steps
1. Start a fresh Claude Code session in this repo to confirm both the context-usage `Stop` hook and
   the new `tool-routing-guard.py` `PreToolUse` hook actually fire as designed.
2. After a handful of real sessions accumulate under the routing guard, re-run
   `tool_efficiency.py --all` to check whether `ctx_execute`/`graphify` usage actually increased --
   the thing the prompt-only nudge failed to move.
3. Continue the audit backlog: item 3 (defensive transcript parsing) next.
