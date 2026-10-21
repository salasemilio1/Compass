# current state

Date: 2026-10-21
Commit: 82ecb32

## Where things stand
The v3 plan is settled and trimmed to a core; no v3 code exists yet. The Go implementation runs in
its own session on branch `go-rewrite`.
- [positioning.md](positioning.md): the goal (disposable sessions, cross-agent resume), the
  staleness rule (nothing maintained in place; expiry by event, not age), and **the core**: format,
  `compass check`, `compass index`, `compass record`, `compass resume`, the `AGENTS.md` block, and
  Claude Code hooks as an accelerator.
- [delivery-plan.md](delivery-plan.md): decision delivery (file/symbol, prompt and caller
  triggers), **gated** on a four-arm trigger eval with design and held-out incident sets.
- [decisions/0010](../decisions/0010-go-binary-as-claude-code-plugin.md): one Go binary, shipped as
  a plugin and through GitHub releases / `go install`; MCP deferred.
- [decisions/0011](../decisions/0011-dedup-by-index-judge-not-embeddings.md): conditional on
  learnings surviving as a category.
- [v3-direction.md](v3-direction.md): superseded except §1 (portability).

The v2 tools (`install.sh`, `scripts/compass-check.sh`, `tools/`, the Python `cli/`) still work
and stay until their Go replacements land.

## Known issues / things to remember
- The dedup experiment lives on branch `claude/dedup-experiment-436ce8` (`ded4590`), **not merged**.
- **On migration, uninstall the Python `compass-cli`.** It sits earlier on `PATH` and shadows the
  Go `compass`.
- ebay-flipper is `~/Dev/Misc/personal-dashboard`; evals use a **copy** of its `.compass/`, never
  the original.
- Session economics (token nudges, hard ceiling, stats) moved out of compass to
  `tools/session-context/`.

## Next steps
1. **Go core** (session on `go-rewrite`): scaffold, `check`, `index`, `record`, `resume`, `init`,
   `doctor`, plugin hooks, release pipeline (releases, `go install`, plugin binaries), tests.
2. **Evaluate the core:** resume cost and discovery overhead on a copy of ebay-flipper's `.compass/`,
   plus synthetic agent sessions, including a mid-task handoff between two agents.
3. Decide whether to merge the dedup-experiment branch into `main`.
4. Only after the core ships: build the trigger eval from delivery-plan.md.
