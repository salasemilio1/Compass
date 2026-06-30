#!/usr/bin/env bash
# SessionStart hook: if the current repository has a compass, nudge the agent to warm up.
# Cheap and silent otherwise — it only injects a short reminder, it does NOT auto-run warm-up
# (warm-up reads files and would cost tokens; the choice to load context stays deliberate).

set -euo pipefail

if [ -d ".compass" ]; then
  cat <<'EOF'
{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"This repository has a .compass/ agent-memory directory. Before substantial work, use the warm-up skill to load context cheaply (read .compass/INDEX.md and .compass/context/current-state.md first). At the end of a coherent task, use the wrap-up skill to capture knowledge back into .compass/ (human-reviewed, never silent)."}}
EOF
fi
