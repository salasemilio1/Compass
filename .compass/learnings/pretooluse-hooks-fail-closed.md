# PreToolUse hooks fail closed on crash/timeout

Date: 2026-07-08

`PreToolUse` hooks block the tool call (deny) if the hook script errors or times out —
the opposite of `SessionStart`/`Stop` hooks in this repo, which just produce no effect
on failure. A hook wired to `PreToolUse` that can deny a call must be fast and defensive
on every code path (never crash, never hang), or a bug in it turns into an outage
(every matching tool call silently blocked) instead of a missed nudge. See
`hooks/tool-routing-guard.py` (decisions/0005): it wraps its entire `main()` in a bare
`except Exception: return 0` specifically because of this asymmetry.
