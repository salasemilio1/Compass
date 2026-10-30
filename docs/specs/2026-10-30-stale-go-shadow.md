# Spec: say when an older Go compass on PATH shadows the plugin's

Date: 2026-10-30 · Status: approved (Emilio, 2026-10-30: "yes, do both")

## Goal

Inside Claude Code the agent's shell finds `compass` on PATH before the plugin's copy. On the
maintainer's machine that was a `go install`ed 0.3.0 while the plugin was 0.3.1, so agents ran an
old CLI that lacks the 0.3.1 delivery warnings, and nothing said so. compass should notice an older
Go compass ahead of the plugin's and say which binary to run and how to update the old one.

## Non-goals

- No PATH edits, no deleting or replacing the user's binary (decision 0020: report, never fix).
- No warning when the PATH copy is the same version or newer than the plugin's binary.
- No network version check (what the latest release is).

## Constraints

- The SessionStart hook runs every session (10 s timeout): at most one extra process, only when the
  first `compass` on PATH is not the plugin's own, with a short timeout; a timeout or unparseable
  output means no note.
- Doctor may be the old binary itself (it's first on PATH); it can only compare against what it can
  read: its own version and the installed plugin's version in `installed_plugins.json`.
- Plugin binaries rebuilt and committed (decision 0013); version 0.3.2.

## Behavior

1. **SessionStart hook** (run by the plugin's binary): if the first `compass` on PATH is a Go
   compass outside `CLAUDE_PLUGIN_ROOT` whose `compass version` is older than the hook's own, the
   brief is the full one and its preface says: the `compass` on PATH is `<path>` at `<version>`,
   older than the plugin's `<version>`; run `"<plugin root>/bin/compass"` for compass commands, and
   tell the user to update or remove `<path>`.
2. **`compass doctor`**: a problem when the first Go compass on PATH is older than the doctor binary
   itself, or older than the installed compass plugin's version (from the registry). Fix text: update
   it (`go install github.com/salasemilio1/Compass/cmd/compass@latest` or a release) or remove it.
3. Version comparison is numeric by dotted part (`0.3.10` > `0.3.9`); anything that doesn't parse
   compares as unknown and never warns.

## Acceptance criteria

- [ ] Unit: `CompareVersions` orders `0.3.0 < 0.3.1 < 0.3.10`, equal for equal, unknown for junk.
- [ ] Unit: SessionStart with an older fake Go compass first on PATH (a script printing
      `compass 0.0.1 (format 3)`) gives the full brief with the note naming its path and the plugin
      binary; with the same version, a newer one, the plugin's own launcher first, or a script that
      hangs past the timeout, no note.
- [ ] Unit: doctor reports a problem for an older Go compass first on PATH, using the registry's
      plugin version; none when equal or newer.
- [ ] e2e: a fake older `compass` placed ahead of the plugin on PATH yields the note in the brief.
- [ ] `make test`, `verify-plugin`, smoke, e2e, `compass check` pass; CI green.
- [ ] On this machine, after the update, a headless session in ebay-flipper has no shadow note (the
      `go install`ed copy is current).

## Open questions

None.

<!-- plan-critique appends ## Plan, ## Assumptions, ## Critique below this line. -->

## Plan

- `internal/compass/doctor.go`: `CompareVersions(a, b string) (int, bool)`; `GoVersion(path)` runs
  `<path> version` (1 s timeout) and parses `compass X (format N)`; `ShadowingOlderGo(pathList,
  pluginRoot, own string) (path, version string)`: first PATH entry only; skipped when its resolved
  path is inside `pluginRoot` (the launcher and libexec), or when it's Python (existing note);
  returns it when `GoVersion` < own.
- `internal/hook/hook.go` SessionStart: call it after `ShadowingPython`; a hit forces the full brief
  and adds the note. `needsFullBrief` takes the combined shadow.
- `Doctor`: after `FindOnPath`, for `OnPath[0]` of kind go with a parsed version, compare with
  `Version` and with each installed compass plugin entry's `version`; problem when older than either.
  `installedCompass` gains the registry `version` field.
- Tests in `internal/hook/hook_test.go`, `internal/compass/delivery_test.go` (doctor), e2e step 4.
- Version 0.3.2 in `init.go` and both manifests; `make plugin`; SPEC.md and decision 0020's area:
  a SPEC line under Delivery checks (no new decision: it applies 0020).

## Assumptions

1. The hook's `PATH` is the agent's PATH (Claude Code passes its environment to hooks), so the first
   `compass` it finds is the one the agent's Bash tool finds. Observed: this session's Bash PATH has
   `~/go/bin` before the plugin's `bin`.
2. `CLAUDE_PLUGIN_ROOT` is set for plugin hooks (used today for the Python note).
3. A Go compass prints `compass <semver> (format <n>)` for `version` (true since 0.3.0).

## Critique

Fresh-context Sonnet critic, 2026-10-30. Each finding and its resolution:

1. **No release step; `go install …@latest` resolves to v0.3.0, the only tag.** Real, and wider than
   this change: a fresh machine following the README's Go or release route gets 0.3.0 while the plugin
   is 0.3.1. → `release.yml` also runs on pushes to `main` and publishes `v$Version` when that tag
   doesn't exist yet (`gh release create` makes the tag), so merging a version bump releases it. Tag
   pushes keep working; an existing tag is skipped.
2. **Python first, old Go second shows only the Python note.** Intentional: the Python note already
   tells the agent to run the plugin's binary.
3. **Hook PATH may differ from the Bash tool's.** Checked: a session launched from a shell gives its
   hooks the same PATH (`~/go/bin` at 17, plugin `bin` at 19 of 19 in both). Desktop-app hook env not
   checked; stated as not verified.
4. **`PathEntry.Version` is the whole line.** → doctor parses it with the same regexp.
5. **An older Go compass with another format is "other".** Accepted: doctor already reports it as "not
   this compass".
6. **Prefix test for "inside pluginRoot" breaks on `/plugins/compass2`, `/var` vs `/private/var`.** →
   `EvalSymlinks` both sides, then `filepath.Rel` without `..`.
7. **Empty pluginRoot makes everything "outside".** → no check without a plugin root.
8. **Registry version may be a sha or empty.** → compares as unknown, never warns; tested.
9. **`sh` + `sleep` keeps the pipe open past the timeout.** → `cmd.WaitDelay`; tested with a hanging
   script.
10. **A relative PATH entry (`node_modules/.bin`) would run repo code at session start.** → only
    absolute PATH entries are run.
11. **A non-compass `compass` (MongoDB Compass) launched with `version`.** → run only a Mach-O or ELF
    binary, or a script whose first 256 bytes mention compass.
12. **Several registry versions.** → compare with the highest.
13. **`FindOnPath` twice in the hook.** Accepted: one `stat` per PATH dir, microseconds.
14. **Doctor already runs `version`; don't spawn twice.** → doctor parses the version it already has.
15. **Carry the note in the stub instead of forcing the full brief.** Declined: same as the Python
    note; the case is rare and the full brief tells the agent where things stand.

## Verification

| Acceptance criterion | Check | Result |
|---|---|---|
| `CompareVersions` order, equality, unknowns | `TestCompareVersions` (incl. numeric shas, `+3`, pre-releases, 4 parts) | pass |
| SessionStart notes an older Go compass; no note when same, newer, unparseable, hanging, the plugin's own, no plugin root, relative entry, inside the project, flooding stdout | `TestSessionStartNotesAnOlderGoCompassFirstOnPath` (fails with the comparison broken: checked), `TestSessionStartNoOlderGoNote` (each case under 1.5 s) | pass |
| No child process outlives the probe | `TestOlderGoCheckLeavesNoChildBehind` (fails without the group kill: checked) | pass |
| Output capped | `TestCappedBufferKeepsOnlyItsLimit` (the flood case passes uncapped too, at 4.3 GB RSS, so the cap is tested directly) | pass |
| doctor flags an older first Go compass, using the registry's highest real version | `TestDoctorFlagsAnOlderGoCompassFirstOnPath`, `TestDeliveryPluginVersionIsTheHighestRealOne` | pass |
| e2e: an older compass ahead on PATH yields the note | `tests/e2e.sh` step 3 | pass |
| `make test`, `verify-plugin`, smoke, e2e, `compass check` | run 2026-10-30 | pass |
| Releases follow merges | `release.yml` on `main`; YAML parses | pending: first main push after merge |
| This machine: no shadow note in ebay-flipper | after merge, release v0.3.2, `go install …@latest` | pending |

**Break pass** (fresh Sonnet), each reproduced and fixed with a test: the probe buffered a flooding
`compass`'s output (4.3 GB RSS) → stdout capped at 4 KB; a forking script left an orphan per session
and stretched the hook to 2.1 s → own process group, killed whole, `WaitDelay` 100 ms; a numeric
git sha (`20261030`) parsed as a version and would make doctor call a healthy PATH copy stale →
versions are 2–3 parts of at most 4 digits, `+3` and pre-releases never warn; a `compass` inside the
project (direnv `$PWD/bin`) would run at session start → skipped; release runs could race, and a tag
without a release at another commit would get this commit's binaries → `concurrency: release`, and
that case fails instead of publishing.

**Not verified:** the desktop app's hook PATH (checked for shell-launched sessions only); the
release workflow until it runs on `main`; a pre-release shadow (never warns, by design).
