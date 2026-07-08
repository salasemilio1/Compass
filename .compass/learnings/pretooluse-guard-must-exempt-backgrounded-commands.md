# PreToolUse routing hooks must exempt backgrounded commands, not just narrow the regex

A hook that redirects heavy-output Bash commands (broad greps, `git diff`, test/build runners) to a
capture-output alternative (`ctx_execute`, `graphify query`) must check the tool call's execution
mode, not only the command's shape. `hooks/tool-routing-guard.py`'s `TEST_BUILD_RUNNER` regex matched
any `npm run <script>`, including `npm run dev` -- a dev server started with `run_in_background:
true`. Routing that through `ctx_execute` is nonsensical: the whole premise of the redirect is
"capture and summarize a finished result," which doesn't apply to a process that's meant to keep
running. Narrowing the regex further wouldn't have generalized -- the same problem exists for `make
serve`, `gradle bootRun`, or any other build/run alias a future rule might catch. The fix is a single
exemption on `tool_input.run_in_background` before classification runs, correct for every rule at
once. See `.compass/decisions/0005-deterministic-tool-routing-guard.md` for the guard's original
design and `.compass/logs/2026-07-08-tool-routing-guard-background-fix.md` for the fix.
