# install.sh's context-mode install command referenced a nonexistent npm package

`install.sh`/README documented `claude mcp add context-mode npx @anthropic-ai/mcp-server-context-mode`
as the install command. That package returns a 404 on the npm registry — it never existed. The real
package is just `context-mode` (by mksglu, `npm install -g context-mode`), confirmed by checking what
was actually on `PATH` (`/opt/homebrew/bin/context-mode` -> `context-mode@1.0.169` in `npm ls -g`) and
cross-referencing the working `~/.claude.json` entry (`"command": "context-mode"`, no npx wrapper).

**Why it matters:** this had gone unnoticed because context-mode was already installed correctly by
some other means before `install.sh`'s command was ever exercised for real on a machine that didn't
already have it — same failure shape as `learnings/decision-installed-claim-vs-actual-deployment.md`
and `learnings/verify-flagged-installs-directly.md`: docs describing an install step doesn't mean that
exact command has ever been run and verified to work.

**How to apply:** when auditing any compass-documented install command for a third-party dependency,
resolve the real binary's location (`which`/`command -v`, follow symlinks) and check what package
actually provides it (`npm ls -g`, `pip show`, etc.) rather than trusting the doc's command to be
accurate just because the tool ends up working.
