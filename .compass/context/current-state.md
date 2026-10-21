# current state

Date: 2026-10-21
Commit: c5a9521

## Where things stand
Compass is out of maintenance mode and its v3 direction is decided, but no v3 code exists yet.
This session set the direction:
- [positioning.md](positioning.md) covers what compass is for, how context gets delivered, the
  closed loop with no manual wrap-up, and the critique dispositions.
- [decisions/0010](../decisions/0010-go-binary-as-claude-code-plugin.md): one Go binary as CLI,
  hook handler and MCP server, shipped as a Claude Code plugin.
- [decisions/0011](../decisions/0011-dedup-by-index-judge-not-embeddings.md): dedup by an LLM judge
  over INDEX summaries, with embeddings dropped.

The v2 tools (`install.sh`, `scripts/compass-check.sh`, `tools/`, the Python `cli/`) still work
and stay until their Go replacements land.

## Known issues / things to remember
- The dedup experiment (code, protocol, results) lives on branch `claude/dedup-experiment-436ce8`
  (`ded4590`) and is **not merged**. 0011 cites it there.
- The spike plugin is in the session scratchpad only (it will be lost). The facts it established
  are recorded in 0010 and positioning.md.
- **On migration, uninstall the Python `compass-cli`.** It sits earlier on `PATH` and shadows the
  plugin's `compass`.
- Decision delivery (the shoulder tap) is **gated**: build it only if the trigger eval (current-state
  alone vs. why-comments vs. delivery) clears a bar that is written down before the eval runs.

## Next steps
1. Decide whether to merge the dedup-experiment branch into `main`.
2. **Scaffold the Go plugin:** repo layout (`cmd/`, `internal/`, `bin/`, `.claude-plugin/`), the
   launcher, the CI release job building 4 targets, and a `compass doctor` that checks for shadowing.
3. **Port `compass check`** from `scripts/compass-check.sh`: same checks A/B1/D/E/F, plus a check
   that every doc carries its one-line `claim`.
4. **Build the front door:** a SessionStart hook loading capped current-state; a Stop hook plus a CI
   check for a stale `Commit:` anchor; `compass index` in pre-commit (replaces manual wrap-up).
5. Build the trigger eval on ebay-flipper incidents before any delivery code.
