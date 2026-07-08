#!/usr/bin/env bash
# SessionStart hook: if the current repository has a compass, nudge the agent to warm up.
# Cheap and silent otherwise — it only injects a short reminder, it does NOT auto-run warm-up
# (warm-up reads files and would cost tokens; the choice to load context stays deliberate).

set -euo pipefail

if [ -d ".compass" ]; then
  context="This repository has a .compass/ agent-memory directory. Before substantial work, use the warm-up skill to load context cheaply (read .compass/INDEX.md and .compass/context/current-state.md first). At the end of a coherent task, use the wrap-up skill to capture knowledge back into .compass/ (human-reviewed, never silent)."

  # Deterministic staleness check (gap D): compare current-state.md's stamped Commit: hash
  # against HEAD by commit count. No judgment here -- just a fact appended to context;
  # deciding what to do about it (re-warm-up vs. proceed) is left to the agent.
  if [ -f ".compass/context/current-state.md" ]; then
    stamped_hash="$(grep -m1 -oE '^Commit: [0-9a-f]{7,40}$' ".compass/context/current-state.md" 2>/dev/null | awk '{print $2}')" || true
    if [ -n "${stamped_hash:-}" ] && command -v git >/dev/null 2>&1 && git rev-parse --git-dir >/dev/null 2>&1; then
      behind="$(git rev-list --count "${stamped_hash}..HEAD" 2>/dev/null || echo "")"
      if [ -n "$behind" ] && [ "$behind" -gt 0 ] 2>/dev/null; then
        context="$context current-state.md is $behind commit(s) behind HEAD -- it may be stale."
      fi
    fi
  fi

  if [ -f "graphify-out/graph.json" ]; then
    context="$context A graphify structural graph is also available at graphify-out/ -- prefer running graphify query over grepping for mechanical structural lookups (what calls what, where things live)."

    # graphify's code-only "update" is fully deterministic (tree-sitter + markdown parsing,
    # no LLM, no network) so it's safe to refresh unconditionally in the background. Only
    # runs when graphify-out/ already exists -- the first build stays an explicit action.
    if command -v graphify >/dev/null 2>&1; then
      # Bounded so a hung graphify (bad grammar, symlink loop) can't leak an orphan
      # process on every session start. `timeout` isn't on stock macOS; skip the bound
      # there rather than fail the whole nudge over a missing coreutils package.
      if command -v timeout >/dev/null 2>&1; then
        ( timeout 30 graphify update . >/dev/null 2>&1 & disown ) 2>/dev/null || true
      else
        ( graphify update . >/dev/null 2>&1 & disown ) 2>/dev/null || true
      fi
    fi
  fi

  context_escaped="${context//\\/\\\\}"
  context_escaped="${context_escaped//\"/\\\"}"

  printf '{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"%s"}}\n' "$context_escaped"
fi
