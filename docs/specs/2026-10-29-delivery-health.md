# Spec: compass notices when Claude Code won't deliver it

Date: 2026-10-29 · Status: approved (2026-10-29: report only; e2e in CI pinned; v2 hooks to backups)

## Goal

When Claude Code will not load the compass plugin or its hooks, compass says so, plainly, in a
place the agent and the user see during normal work, instead of failing silently as it did for 5 of
8 ebay-flipper sessions (2026-10-28 to 10-29, a dangling `~/.claude/settings.json` symlink). And a
fresh machine that follows the README gets a working install, proven by an end-to-end test that
starts a real Claude Code session.

## Non-goals

- compass does not write outside repositories: no `--fix` that edits `~/.claude/settings.json` or
  deletes files there. It names the fix; the user (or their setup tool) applies it.
- No session-policy features (context nudges, routing guards): decision 0015 cut those.
- No changes to ebay-flipper's records (the 27 cut-off claims, the 203-character claim in an
  in-flight session's untracked file). Reported, not fixed here.
- No fix to `ai-dev-setup`, which owns `~/.claude/settings.json` on this machine. Reported.
- The field-week judge pass and replays (2026-10-30) are separate work.

## Constraints

- Data: reads Claude Code's settings files only: `$CLAUDE_CONFIG_DIR` or `~/.claude`
  (`settings.json`, `plugins/installed_plugins.json`), and the project's `.claude/settings.json`
  and `.claude/settings.local.json`. Read-only. Formats are Claude Code's, undocumented in part:
  every check must degrade to a note, never a crash or a false "problem", when a file is missing or
  shaped unexpectedly.
- Forbidden: network calls from `doctor`, `resume` or hooks; new Go dependencies; anything that
  slows the SessionStart hook (it has a 10 s timeout and runs every session).
- Environment: macOS and Linux, arm64 and amd64. The plugin binaries stay reproducible and
  byte-checked (decision 0013): rebuild with `make plugin` and commit.
- Irreversible operations: none in compass. On this machine, v2 leftovers in `~/.claude/hooks/`
  are moved to `~/.claude/backups/`, not deleted.

## Behavior

1. **One delivery check** (`internal/compass`), used by three callers. It reports problems:
   - user settings unreadable: a dangling symlink, a read error, or invalid JSON (Claude Code then
     loads no hooks and no plugins);
   - `disableAllHooks: true` in any settings file that applies;
   - the compass plugin not enabled (`enabledPlugins["compass@compass"]` not true in any applicable
     settings file) or not installed (`installed_plugins.json` has no `compass@compass`), or its
     install path is missing `hooks/hooks.json`;
   - v2 leftovers: hook commands naming `tool-routing-guard`, `session-context-nudge` or
     `compass-session-start` in any settings file (a name match covers the missing or
     non-executable v2 file that failed 639 times; a general hook-file audit is not compass's job).
   A check whose input is missing or unparseable for reasons other than the above yields a note.
2. **`compass doctor`** prints every delivery problem with its fix (the file, and the command or
   edit to make), and exits 1 when any exists, as it does now for its other problems.
3. **`compass resume`**, when run inside Claude Code (`CLAUDECODE=1` in the environment), prints
   a `warning:` line to stderr per delivery problem and points to `compass doctor`. In a dark
   session the agent runs `resume` from the AGENTS.md block, so the warning reaches it and the user.
4. **`compass hook pre-commit`** prints the same warnings to stderr (never blocks a commit for them).
5. **The AGENTS.md block** adds one line: in Claude Code the resume brief arrives at session start;
   if it didn't, run `compass doctor` and tell the user what it says. (`agents-block-drift` then
   asks existing repos to re-run `compass init`.)
6. **`compass doctor` recognises lefthook:** a `lefthook.yml` (or `.lefthook.yml`) whose pre-commit
   runs `compass hook pre-commit` counts as the git hook, so ebay-flipper stops getting a false note.
7. **End-to-end test** (`tests/e2e.sh`, `make e2e`, a CI job): with a temporary
   `CLAUDE_CONFIG_DIR` and HOME, add this checkout as a plugin marketplace, install
   `compass@compass` with the `claude` CLI, `compass init` a throwaway repo, start a headless
   session with hook events on, and assert the SessionStart hook ran and its output carries the
   brief. Then break the settings (dangling symlink) and assert `compass doctor` fails and names it.
   No API key needed: hook events fire before authentication.
8. **This machine and ebay-flipper:** after the change, `compass doctor` is clean in ebay-flipper
   (or lists only the known record errors), a fresh headless session there gets the brief, and v2
   leftovers in `~/.claude/hooks/` are moved to a backup.

## Acceptance criteria

- [ ] Unit tests: each delivery problem in Behavior 1 is detected from a fixture config dir
      (dangling symlink, invalid JSON, plugin disabled, plugin not installed, hooks.json missing,
      `disableAllHooks`, each v2 hook command); a clean fixture yields no problems; a missing
      `installed_plugins.json` or unexpected shape yields a note, not a problem.
- [ ] `compass doctor` exits 1 and names the settings file when it is a dangling symlink; exits 0
      on a clean fixture.
- [ ] `CLAUDECODE=1 compass resume` on a broken fixture prints a `warning:` to stderr and the
      brief to stdout unchanged; without `CLAUDECODE` it prints no warning.
- [ ] `compass hook pre-commit` on a broken fixture warns on stderr and still exits 0 for a clean
      `.compass/`.
- [ ] The AGENTS.md block carries the doctor line; `compass init` on an old block refreshes it.
- [ ] `compass doctor` in a repo whose lefthook.yml runs `compass hook pre-commit` gives no
      "pre-commit hook doesn't run compass" note.
- [ ] `make e2e` passes locally with the `claude` CLI on PATH, and fails if the plugin's
      SessionStart hook is removed from `plugin/hooks/hooks.json` (checked once by hand).
- [ ] CI runs the e2e job on ubuntu and it passes.
- [ ] The SessionStart hook's added cost: none (it does not call the delivery check).
- [ ] `make test`, `make verify-plugin`, `sh tests/smoke.sh` and `compass check --require-fresh`
      pass.
- [ ] In ebay-flipper on this machine: a headless session gets the brief; `compass doctor` shows
      no delivery problem.

## Open questions

1. CI installs the `claude` CLI from npm (`@anthropic-ai/claude-code`) for the e2e job: a network
   dependency on a third-party package in CI. Default: yes, pinned to an exact version.
2. The SessionStart hook can't report its own absence, so detection lives in resume, pre-commit and
   doctor. Default: accept that a session that neither runs `resume` nor commits stays dark.
3. Whether the field-week bars should treat days 1–4 as void for 1a/3a given the outage. Default:
   report both with and without the dark sessions on 10-30.

<!-- plan-critique appends ## Plan, ## Assumptions, ## Critique below this line. -->

## Plan

**Approach.** One read-only function, `compass.Delivery(configDir, projectRoot) DeliveryReport`
(`internal/compass/delivery.go`), returns `Problems` and `Notes`, each problem a sentence that
names the file and the fix. Three callers: `Doctor` appends its problems/notes; `runResume` and the
pre-commit branch of `runHook` print each problem as `compass: warning: … (run compass doctor)` to
stderr, only when `CLAUDECODE=1` (git inherits it from the agent's Bash tool, so a terminal or
Codex user who never installed the plugin gets no noise). The SessionStart hook does not call it:
a hook that runs can't report its own absence, and it would cost every session. Alternative
rejected: a `doctor --fix` that edits `~/.claude` (the user chose report-only; README promises no
writes outside repos).

**What `Delivery` checks** (config dir = `$CLAUDE_CONFIG_DIR` or `~/.claude`):
1. `settings.json`: `os.Lstat` + `os.Stat` to tell a dangling symlink ("points to X, which doesn't
   exist") from a missing file (a note: no user settings, so no user-scope plugin either); read
   error and invalid JSON are problems.
2. Settings that apply, lowest to highest precedence: user, `<project>/.claude/settings.json`,
   `<project>/.claude/settings.local.json`. `enabledPlugins` keys matching `compass@*`: the last
   file that sets one decides; none true → problem "the compass plugin is not enabled". Any
   `disableAllHooks: true` → problem.
3. `plugins/installed_plugins.json` (`{"plugins": {"compass@…": [{"installPath": …}]}}`): no entry →
   problem "not installed" with the two `claude plugin` commands; an entry whose
   `installPath/hooks/hooks.json` is missing → problem. Unreadable or unexpected shape → note.
4. v2 leftovers: the raw `hooks` value of each applicable settings file contains
   `tool-routing-guard`, `session-context-nudge` or `compass-session-start` → problem naming the
   file and the entry to delete.
Project settings that are missing are fine; malformed project settings are a problem (Claude Code
reports those too, but the agent may not see it).

**Files and seams.**
- `internal/compass/delivery.go` + `delivery_test.go` (fixtures built in `t.TempDir()`).
- `internal/compass/doctor.go`: call `Delivery`; lefthook recognition (`lefthook.yml`,
  `.lefthook.yml`, `lefthook.yaml` containing `compass hook pre-commit` counts as the git hook).
- `cmd/compass/main.go`: `runResume` gains `stderr`; a helper `warnDelivery(stderr, root)` used by
  resume and pre-commit.
- `internal/compass/agents.go`: one line in `AgentsBlock`: "In Claude Code, the resume brief
  arrives at session start (`compass: this is the resume brief`). If it didn't, run `compass doctor`
  and tell the user what it reports."
- `internal/compass/init.go`: `Version` 0.3.0 → 0.3.1 (the plugin cache is keyed by version, so
  without the bump `claude plugin update` keeps the old binary).
- `plugin/libexec/*`: `make plugin`, committed. `plugin/.claude-plugin/plugin.json` version bump.
- `tests/e2e.sh`, `Makefile` target `e2e`, CI job `e2e` (ubuntu, `npm i -g
  @anthropic-ai/claude-code@2.1.300`).
- `SPEC.md` (doctor/resume/pre-commit behaviour, AGENTS block), `README.md` (doctor after install,
  update commands), `.compass/` current-state and a decision record for "report, don't fix".

**e2e.sh.** Temp `HOME` and `CLAUDE_CONFIG_DIR`; `claude plugin marketplace add <checkout>`;
`claude plugin install compass@compass`; a throwaway git repo with `compass init` and a filled
current-state; `claude -p hi --output-format stream-json --verbose --include-hook-events
--max-turns 1`; assert a `hook_response` for `SessionStart` whose output contains `# compass resume`
or the stub marker. Then replace settings.json with a dangling symlink and assert: no compass
SessionStart response; `compass doctor` exits 1 naming the symlink; `CLAUDECODE=1 compass resume`
warns on stderr. Restore and assert `compass doctor` has no delivery problem.

## Assumptions
1. Claude Code reads user settings from `$CLAUDE_CONFIG_DIR/settings.json`, else
   `~/.claude/settings.json`; an unreadable file means no user hooks and no user-scope plugins
   (observed on 2026-10-28/29: every user hook and the plugin went silent together).
2. `CLAUDECODE=1` is set in the agent's Bash tool even when settings are broken (it's set by the
   process, not by settings), and git hooks inherit it.
3. Hook events fire before authentication, so e2e needs no API key (observed locally with an empty
   config dir: `hook_response` then "Not logged in").
4. `claude plugin marketplace add <dir>` works on a local checkout and writes
   `extraKnownMarketplaces` + `enabledPlugins` into the config dir's settings.json (observed).
5. `--include-hook-events` exists in 2.1.300 (pinned for CI); verified on 2.1.241 locally.
6. Managed settings (`/Library/Application Support/ClaudeCode/`, `/etc/claude-code/`) can disable
   plugins too; not checked (none on this machine), stated as a limit.
7. Plugin key may be `compass@<any marketplace name>`; matching `compass@` covers renamed
   marketplaces.

## Risks
- A false "problem" in doctor on a healthy machine (unknown settings shape) would train people to
  ignore it: every unrecognised shape is a note, and the e2e's healthy phase asserts no delivery
  problem on a real install.
- Warning fatigue in resume/pre-commit: only when `CLAUDECODE=1` and only real problems.
- CI flakiness from npm or the CLI changing hook-event output: pinned version; the assertion looks
  for the hook name and the brief marker only.
- Version bump: users on 0.3.0 keep the old behaviour until `claude plugin update`; README says so.

## Verification
Unit tests per acceptance criterion 1–6; `make test`, `make verify-plugin`, `tests/smoke.sh`,
`make e2e` locally, then once by hand with SessionStart removed from `plugin/hooks/hooks.json` to
see e2e fail; CI green on the PR; a headless session in ebay-flipper after updating the plugin; a
break pass by a fresh subagent.

## Critique

Fresh-context Opus critic, 2026-10-29. Each finding and its resolution:

1a. v2 hook file missing/not executable unchecked → spec amended: the name match covers the case
    seen; auditing every hook's file is session tooling (0015), not compass.
1b. Behaviour 8 not in plan → v2 hooks already moved to `~/.claude/backups/compass-v2-hooks-2026-10-29/`;
    ebay-flipper doctor + headless session are explicit verification steps after the plugin update.
1c. `marketplace.json` and `version_test.go` → bump both manifests; the test guards it.
2a. AGENTS line quoted only the full-brief marker; the stub starts "compass: current-state is
    current…" → the line says "a message starting `compass:`".
2b. Malformed project settings were a problem in the plan, a note in the spec → note (Claude Code
    skips the file; user scope still delivers).
2c. `--include-hook-events` unverified on 2.1.300 → verified (`claude -p --help` on 2.1.300).
3a. Non-Claude users would fail doctor forever → delivery problems only with evidence of intent
    (a `compass@` key in settings, an installed entry, or `CLAUDECODE=1`); otherwise notes.
3b. "Last file decides" mixed keys → precedence per key; pass if any key resolves true;
    non-boolean values are notes.
3c. `installed_plugins.json` scopes ignored → entries count when `scope` is user, or project/local
    with `projectPath` equal to the project root or cwd; others ignored.
3d. `--plugin-dir`/`--settings` and launch dir ≠ compass root → project settings read from both
    cwd and repo root; `--plugin-dir`/`--settings` stated as a limit (doctor can't see a CLI flag).
3e. Wrong config dir → doctor prints the config dir it checked.
3f. Torn settings write during `claude plugin install` → re-read once after 100 ms before calling
    JSON invalid.
3g. e2e could pass while broken → assert on problem text, never doctor's exit code; `make e2e`
    depends on `verify-plugin`; run nested `claude` with `env -u CLAUDECODE -u ANTHROPIC_API_KEY`
    and the `CLAUDE_CODE_*` vars unset; automate the negative check (a temp copy of the plugin with
    SessionStart removed must yield no brief). `CLAUDECODE=1` in the agent's Bash tool is observed
    in this session, not tested by e2e (stated).
3h. main_test resume/hook tests would read the real `~/.claude` → `warnDelivery` takes env and
    config dir as arguments; tests set `CLAUDE_CONFIG_DIR` to a temp dir.
4a. git hook fallback hard-codes `cache/compass/compass` → glob `cache/*/compass/*/` so a renamed
    marketplace still finds the binary.
4b. Lefthook detection in a third place, substring too loose → one helper shared by doctor and
    init; ignores commented lines.
4c. Doctor returns early outside a project → user-level delivery checks run before that return.
5.  Simpler option: none found.
