#!/usr/bin/env bash
# SessionStart hook: if the current repository has a compass, nudge the agent to warm up.
# Cheap and silent otherwise — it only injects a short reminder, it does NOT auto-run warm-up
# (warm-up reads files and would cost tokens; the choice to load context stays deliberate).

set -euo pipefail

if [ -d ".compass" ]; then
  context="This repository has a .compass/ agent-memory directory. Before substantial work, use the warm-up skill to load context cheaply (read .compass/INDEX.md and .compass/context/current-state.md first). At the end of a coherent task, use the wrap-up skill to capture knowledge back into .compass/ (human-reviewed, never silent)."

  if [ -f "graphify-out/graph.json" ]; then
    context="$context A graphify structural graph is also available at graphify-out/ -- prefer running graphify query over grepping for mechanical structural lookups (what calls what, where things live)."
  fi

  context_escaped="${context//\\/\\\\}"
  context_escaped="${context_escaped//\"/\\\"}"

  printf '{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"%s"}}\n' "$context_escaped"
fi
