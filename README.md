# compass

compass keeps a project's working memory in the repo, in `.compass/`, so any agent session is
disposable: a fresh agent (Claude Code, Codex, Cursor, anything that reads `AGENTS.md`) runs
`compass resume` and knows where the work stands, what was decided, and what is uncommitted,
without a discovery phase. When one agent hits its usage limit, another picks up from there.

compass v3 is one static Go binary. The v2 installer, checker script and Python `compass-cli`
are gone; see [v2 leftovers](#v2-leftovers) for what remains in the repo.

## Install (v3)

**Claude Code plugin** (binaries are committed; no Go, Python or network needed after install):

```bash
claude plugin marketplace add salasemilio1/Compass
```

```bash
claude plugin install compass@compass
```

The plugin's `SessionStart` hook injects the resume brief (a short stub with the objective and next
step when current-state is current and the tree is clean on a fresh start), `Stop` asks for an updated
current-state once when code was committed after it, and `PreCompact` asks once before a manual
`/compact`. Inside Claude Code the agent's Bash tool has the plugin's `compass` on PATH, and the
git pre-commit hook finds the plugin's copy on its own. For your own terminal, install a release
binary as below (the plugin's cache path changes with every plugin version, so don't link to it).

**Without Claude Code**, from a release (macOS and Linux, arm64 and amd64):

```bash
gh release download --repo salasemilio1/Compass --pattern "compass_*_$(uname -s | tr A-Z a-z)-$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/').tar.gz"
```

Unpack it and put `compass` on your PATH. A file downloaded by a browser on macOS is quarantined;
clear it with `xattr -d com.apple.quarantine compass`. Or, with Go installed:

```bash
go install github.com/salasemilio1/Compass/cmd/compass@latest
```

**Uninstall:** `claude plugin uninstall compass@compass` (and `claude plugin marketplace remove
compass`), or delete the binary. Nothing else is written outside your repositories.

## Use (v3)

```bash
compass init        # once per repo: .compass/, the AGENTS.md block, a git pre-commit hook;
                    # fill in the current-state template before your first commit (check rejects it)
compass resume      # what a fresh agent reads first
compass record decision --claim "Prices come from the model's own fit" --premise "goal:quick-flip"
compass check       # caps, anchor, files in play, claims, supersedes, refs (pre-commit and CI)
compass index       # regenerate .compass/INDEX.md (pre-commit does this)
compass doctor      # install problems, such as an old compass shadowing this one
```

`compass help` lists every flag; `check`, `resume`, `record` and `doctor` take `--json`. Exit status: 0 ok, 1 findings or failure, 2 usage error.

**The format.** `.compass/context/current-state.md` is the front door: at most 80 lines
(objective, just done, files in play, next steps), rewritten whole at each boundary, with a
`Commit:` line set from `git rev-parse HEAD`. Decisions, goals and experiments are dated records
that are never edited; each has frontmatter:

```yaml
---
claim: one line, what holds (required, at most 200 characters)
date: 2026-10-21
scope: global          # optional: resume delivers live global records to every session
supersedes:            # optional: the records this one replaces; their status becomes superseded
  - "0006"
premises:              # optional: what the decision rests on, e.g. goal:<slug>
  - goal:portable-memory
refs:                  # optional: paths and commit:<hash> are checked; other schemes are left alone
  - internal/compass/check.go
---
```

`INDEX.md` is generated, never hand-edited. Status is derived from later records' `supersedes:`.
A Claude-memory-style `description:` is read as the claim.

**CI:** run `compass check --require-fresh` on pull requests with full history
(`actions/checkout` with `fetch-depth: 0`), so a PR that changes code fails if current-state was
not rewritten after it.

## Migrating from v2 and the Python `compass-cli`

1. Remove the Python CLI. It sits earlier on PATH and shadows the Go binary (the plugin's `bin/`
   is appended to PATH, so any other `compass` wins):

   ```bash
   uv tool uninstall compass-cli
   ```

   (or `pipx uninstall compass-cli`). `compass doctor` reports any `compass` shadowing this one,
   and the SessionStart brief warns the agent and names the plugin's binary until you do.
2. In each repo, run `compass init`. It keeps your files and adds what is missing: the v3 block in
   `AGENTS.md`, a one-line pointer in `CLAUDE.md`, the pre-commit hook (only if you have none;
   otherwise add `compass hook pre-commit` to yours). Delete the old v2 instructions it reports
   (the `compass-spec-version` text in `AGENTS.md`, "Read `.compass/INDEX.md` first" in
   `CLAUDE.md`).
3. Run `compass check` and work through it: add a `claim:` to each record (or keep
   `description:`), restate current-state under 80 lines, split records over 300 lines. Folders
   v3 doesn't use (`logs/`, `learnings/`) are left alone and counted in the index.
4. If v2's `install.sh` set you up, remove its hook entries from `~/.claude/settings.json`
   (`compass-session-start.sh`, `session-context-nudge.py`, `tool-routing-guard.py`) before deleting
   the files (a registered hook whose file is gone errors), then delete v2's skills and hooks from
   `~/.claude` and any compass-workflow section in `~/.claude/CLAUDE.md`. Until then the v2
   SessionStart nudge and the plugin's brief both fire.

## v2 leftovers

`skills/` (`init-compass`, `warm-up`, `wrap-up`), `hooks/` and `tools/` are v2 pieces kept in the
repo, with their tests in `tests/`. Nothing installs them any more and the skills still call the
removed `scripts/compass-check.sh`, `install.sh` and `cli/`, so `wrap-up` and `init-compass` no
longer work. The compass commands above replace them.
