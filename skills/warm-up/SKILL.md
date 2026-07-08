---
name: warm-up
description: At the start of a session, cheaply rebuild the context needed for the task by reading the repo's .compass/ — index first, then only the relevant slices and resumable state. Read-only. Use when the user starts work in a repo that has a .compass/, says "warm up", "load context", "catch up", or "where were we".
---

# warm-up

Prime a fresh session from the repository instead of from conversation history. Cheap and **read-only** —
warm-up never writes. The goal: a cold session becomes useful in seconds without the user re-explaining.

## Steps

1. **Locate `.compass/`.** If the repo has none, say so and offer `init-compass`. Stop.

2. **Quick system health check.** Two cheap, deterministic checks, independent of the repo:
   - `claude mcp list 2>/dev/null | grep -q "context-mode.*Connected"` — is ctx (context-mode MCP) up?
   - `command -v graphify >/dev/null` — is the graphify CLI installed?
   Fold the result into a single status line (e.g. `System: ctx ✓ connected · graphify ✓ installed`)
   in the final report. Don't investigate failures here — just note the status; if the user wants a
   fix, that's a separate task.

3. **Read `INDEX.md` first.** It's the map. Use it to decide what else to load — do **not** load every
   file.

4. **Read the resume layer:**
   - `context/current-state.md` if present — where the work stands and the next steps. This is the most
     important file for resuming an in-progress project. Stop here for repeat sessions.
   - `context/overview.md` and `architecture/overview.md` only if: the user is new to the repo,
     explicitly asks for orientation, or the task involves structural changes (new files, new layers,
     bootstrap modifications). Otherwise skip — current-state.md is enough to resume.

5. **Load task-relevant slices only.** Infer from the user's first request, then pull the matching
   entries: relevant `decisions/`, recent `logs/`, and any `experiments/` that bear on the task.
   Skip the rest — loading everything defeats the purpose.

6. **Check for a graphify graph.** If `graphify-out/graph.json` exists, it's an auto-generated structural
   map (what calls what, where things live). Prefer `graphify query "<question>"` over grepping/reading
   raw files for mechanical structural questions. This is advisory, not required — compass's own
   `architecture/overview.md` remains the source of curated intent. Say nothing about graphify if
   `graphify-out/` doesn't exist.

7. **Report the primed context.** Lead with the system health status line from step 2, then a brief
   summary: what the project is, where it stands, what's next, and which entries you loaded. Then
   proceed with the user's task.

## Notes
- Prefer breadth-via-index over depth: read the one-line index rows, open only what's relevant.
- If `current-state.md` and the code disagree, trust the code and flag the staleness to the user.
- warm-up is the payoff for wrap-up discipline; if context is thin, that's a signal the last session
  wasn't wrapped up — mention it.
