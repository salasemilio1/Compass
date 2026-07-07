# fault-tolerance audit of compass itself

Date: 2026-07-07
Commits: 00d0add..00d0add (review only, no code changed)

## Why
Stepped back to audit compass as a piece of software meant to keep working for years with
minimal upkeep, not just as a working prototype. Goal: confirm the "deterministic filters,
prompts as suggestions" pattern (exemplified by `tools/learning-dedup/check_similarity.py`'s
fastembed-only, no-API-call design) is followed consistently, and find where the system is
still fragile at its edges.

## Verdict
The pattern holds where it matters: `scripts/compass-check.sh` is 100% mechanical, the dedup
filter's stages 1-2 are dependency-free and stage 3 fails soft to `UNAVAILABLE`, and the
context-usage nudge reads real API `usage` fields with no estimation. Gaps found are at the
edges where compass touches things it doesn't control (Claude Code's undocumented transcript
format, a third-party CLI, this machine's own install topology) — not in the core logic.

## Gaps found (none fixed yet — this entry is the record; each needs its own pickup)

**Tier 1 — real fault-tolerance risk:**
1. No CI. `tests/run_tests.sh` (1007 lines of tests across `tests/`) is never invoked
   automatically — a regression can be committed silently. *(Being fixed next — see
   upcoming log entry.)*
2. Global skills/hooks on this machine are symlinked via `install.sh --dev` from the
   `graphify-integration` branch, not `$DOTFILES/compass`. A destructive git op on that
   branch breaks compass globally with no detection mechanism (no `compass doctor`).
3. `tools/session-context/session_status.py` and `tool_efficiency.py` call `json.loads(line)`
   per transcript line with no try/except — a corrupted/partial JSONL line (process killed
   mid-write) throws uncaught inside code reachable from the `Stop` hook, the one path that's
   supposed to never crash a turn. Contrast with `check_similarity.py`'s `except Exception` ->
   `UNAVAILABLE` pattern, which these should mirror.
4. `scripts/register_hook.py` / `install.sh`'s `settings_has_hook` mutate/probe
   `settings.json` via blind `setdefault`-append and a raw substring `grep` for a marker —
   no post-write validation, no protection against a future hooks-schema shape change.

**Tier 2 — silent long-term drift:**
5. `.compass/learnings/.embedding-cache.json` is keyed by content hash only, not
   `(model_name, hash)`. Retuning `MODEL_NAME`/`EMBEDDING_THRESHOLD` later would silently
   reuse stale vectors from the old model space.
6. No inventory of which repos have a compass — when `SPEC.md` bumps its version, nothing
   proactively flags every already-scaffolded repo; `AGENTS.md` drift only surfaces
   reactively, next time that specific repo is opened.
7. No `.compass/`-directory-level version marker; layout changes (vs. content-convention
   changes) have nothing to gate on directly — `compass-check.sh` infers version drift one
   hop removed, via `AGENTS.md`'s stamp, which is itself optional.

**Tier 3 — minor:**
8. `graphify` has no `UNAVAILABLE`-equivalent signal if its CLI output format changes
   upstream — unlike fastembed's explicit failure path, a broken `graphify query` could
   return garbage indistinguishable from a real empty answer.
9. `logs/archive.md` / `experiments/archive.md` grow unbounded once the 15-entry cap
   archives into them — no cap on the archive itself.

## Next steps
Tackle in rough priority order: #1 (CI) now, #3 (defensive transcript parsing) next since
it's small and protects the same hook path #1 exists to catch regressions in. #2, #5-9 are
one-time fixes or low-urgency ADR candidates at current single-user/single-machine scale.
