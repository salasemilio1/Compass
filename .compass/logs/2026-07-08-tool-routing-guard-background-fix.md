# fix: tool-routing-guard denied backgrounded dev-server commands

Date: 2026-07-08
Commits: 76b4d9c..HEAD (uncommitted at time of writing)

## Why
A steady-state audit of the deployed hooks (prompted by "make sure the last changes went smooth")
found that `hooks/tool-routing-guard.py`'s `TEST_BUILD_RUNNER` regex (`npm\s+(test|run)|...`) matches
any `npm run <script>`, not just tests -- including `npm run dev`, `npm run start`, `make serve`,
`gradle bootRun`, etc. The hook also never looked at the Bash tool's `run_in_background` field. Net
effect: the first time a session started a dev server (a near-certain first move for anyone using
this machine's `/run` skill), the guard would deny it and redirect toward `ctx_execute` -- which
can't meaningfully capture the output of a process that's meant to keep running.

## What changed
`hooks/tool-routing-guard.py`'s `main()` now exempts any call with `tool_input.run_in_background`
truthy from all routing rules, before classification runs. This is correct for every existing rule,
not just the test/build one: none of graphify-first, ctx-first-search, git-diff, or git-log routing
makes sense for a command that isn't run to produce a captured, finished result.

Added `tests/test_tool_routing_guard.py::test_main_allows_backgrounded_heavy_command` covering the
new exemption with `npm run dev` + `run_in_background: true`.

## Assumptions / scope
- Did not narrow `TEST_BUILD_RUNNER`'s regex itself -- `npm run build` is already an intentional,
  tested match (decisions/0005 wants build output routed too), and the real fix is execution-mode
  awareness, not a better regex.
- Found no dotfiles-repo-in-the-wild to test decisions/0006's install flow end-to-end; verified
  instead by diffing every installed file under `~/.claude/` against this repo's HEAD (byte-identical)
  and re-running the full test suite (bash + python), all green.

## Learnings
See `learnings/pretooluse-guard-must-exempt-backgrounded-commands.md`.
