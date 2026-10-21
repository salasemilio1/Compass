---
claim: "install.sh copies (not symlinks) skills/hooks so an ephemeral dotfiles clone can be deleted after install; drops the now-pointless $DOTFILES canonical-source guard and --dev flag (supersedes 0002)"
---
# 0006 -- install.sh copies skills/hooks instead of symlinking; drop the $DOTFILES canonical-source guard

Date: 2026-07-08
Supersedes: 0002

## Context
Hardening compass for steady-state use (post graphify-integration merge to main) surfaced that the
actual install workflow -- clone a dotfiles repo fresh, run its bootstrap (which invokes
`compass/install.sh`), then delete the clone -- was incompatible with 0002's canonical-source guard.
That guard assumed a *persistent* dev checkout and protected against a destructive git operation on it
(reset --hard, branch delete, force-push) breaking the global symlinks. It never anticipated the source
being deleted outright as a normal, expected step of every install.

## Decision
- **`install.sh` now copies** (`cp -R`, always overwriting) skills/hooks/`tools/session-context` into
  `$COMPASS_HOME` instead of symlinking them. The installed copy has no dependency on the source checkout
  after `install.sh` finishes -- deleting the clone immediately afterward is safe.
- **Removed the `$DOTFILES/compass` canonical-source guard and `--dev` flag entirely.** Copying means the
  risk that guard existed to prevent (global breakage from a mutated/deleted source) no longer exists.
  Plain `install.sh`, no flags, now works from any temporary clone location.
- **Re-running `install.sh` doubles as the update mechanism.** Since installs always overwrite, getting a
  newer compass onto an already-set-up machine is just: clone, run `install.sh`, delete the clone again.
  No separate update script.

## Alternatives considered
- Keep symlinking but require the dotfiles clone to persist at a stable path -- rejected; contradicts the
  actual, already-established dotfiles workflow (ephemeral clone, deleted after bootstrap), and would
  require changing the dotfiles repo's behavior instead of compass's.
- Keep the canonical-source guard as a warn-only advisory even after switching to copies -- rejected; once
  copying removes the actual risk, the guard has no remaining purpose and is dead weight to maintain.

## Consequences
- Gain: `install.sh` genuinely works standalone, with no environment precondition (`$DOTFILES`) and no
  flag. Matches the "clone, install, delete" workflow exactly.
- Gain: re-running `install.sh` is now a real update path, not just an idempotent no-op.
- Lose: no built-in warning if someone still runs `install.sh` from a long-lived dev checkout and later
  mutates it -- irrelevant under the copy model (the installed files are already independent), so this
  isn't a real loss.
- `tests/test_install_guard.sh` (canonical-source guard tests) replaced by `tests/test_install.sh`
  (copy-based install tests: real files not symlinks, survives source deletion, re-install updates
  content).
