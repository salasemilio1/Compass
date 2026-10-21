# current state

Date: 2026-10-21
Commit: 449dd4f6e91f18ddea385049836e04dd2ac8a1c9

## Objective
Ship the compass v3 core (positioning.md, "The core"; decision 0010) as one Go binary, and decide
from the eval what comes next. The core is built and evaluated on branch
`claude/go-rewrite-74db42`; it is not pushed, released or merged.

## Just done
- Go core: `check`, `index`, `record`, `resume`, `init`, `doctor`, hook handlers; unit tests,
  a smoke test of the shipped binaries (macOS arm64, Linux arm64/amd64 in Docker, no Go on PATH).
- Plugin in `plugin/` with committed, byte-reproducible binaries (decision 0013); CI and release
  workflows written, **never run** (nothing pushed).
- Live: SessionStart reached the model; Stop blocked once and the agent rewrote current-state;
  install/uninstall left Claude Code's registries byte-identical; Codex used AGENTS.md alone.
- Eval (experiments/2026-10-21-v3-core-eval.md): brief 93% smaller than v2's front door; no
  discovery reduction on specified code-local tasks; Claude-to-Codex handoff carried the task but
  wasn't cheaper than re-pasting the prompt.
- This repo's `.compass/` migrated to v3 (claims, generated INDEX, v3 AGENTS.md block).

## Files in play
- `internal/compass/` — the core (the only code that touches .compass/)
- `internal/hook/hook.go` — Claude Code hook adapters
- `plugin/` — what the marketplace installs
- `Makefile` — test, plugin, verify-plugin, dist
- `.github/workflows/ci.yml` — the go job (unverified until pushed)
- `.compass/experiments/2026-10-21-v3-core-eval.md` — the evidence

## Next steps
1. **Emilio:** review, then push the branch and open a PR; watch CI's first run (verify-plugin on
   Linux, smoke on ubuntu and macos, `check --require-fresh`).
2. Tag `v0.3.0` after merge, so `go install ...@latest` and the release archives exist; test both.
3. **Emilio:** `uv tool uninstall compass-cli` on this Mac (it shadows the Go binary), then retire
   `install.sh`, `scripts/`, `cli/` and the v2 skills/hooks from `~/.claude`.
4. Decide on the brief from the eval: it costs ~3.1k tokens a session and went unused on specified
   tasks; options are keeping it, injecting it only on `resume`/`compact`, or only when work is
   uncommitted. Test on unspecified "continue" tasks before choosing.
5. Mid-task cuts don't refresh current-state (no agent followed "before a long step"); only
   git status/diff carried the work. Decide whether that is enough.
6. Merge or drop the dedup-experiment branch (`claude/dedup-experiment-436ce8`); delivery stays gated.

## Known issues
- Decision 0010 claims `stats`, `--json` on every command, and CI-built binaries; the core has no
  `stats`, `init`/`index` have no `--json`, and binaries are built locally and verified by CI.
- `context/overview.md` (v2, hand-maintained) predates the Go layout: `layout-drift` warns.
- v2's pre-commit hook in the main checkout still runs `scripts/compass-check.sh`.
