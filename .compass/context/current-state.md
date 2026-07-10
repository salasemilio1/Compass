# current state

Date: 2026-07-10
Commit: cfa70eb5db7d4c9073b725f111fedb1ad073c1e3

## Where things stand
Compass is `main`, steady-state, maintenance-mode. The prior session's `compass init` CLI
(decisions/0007) and tool-routing-guard fix are committed and confirmed live. This session was
verification + presentation prep, not new development: confirmed the CLI actually works end-to-end,
built diagrams explaining compass's workflow, and — under user pressure-testing — found one real,
previously-unverified gap in the learnings-dedup embedder (see
`learnings/dedup-stage3-never-verified-with-real-embedder.md`).

## Known issues / things to remember
- **Stray backup artifacts in `~/.claude`** (global, not repo state): `skills/wrap-up.backup-20260704100310`,
  `skills/init-compass.backup-20260704100310`, `skills/warm-up.backup-20260704100310`, and
  `hooks/compass-session-start.sh.backup-20260704100310` are leftover symlinks from before
  decisions/0006 switched install.sh to copying. Not referenced by `settings.json`, not a live risk,
  but dead weight. **Delete these next session.**
- **Learnings-dedup stage 3 has never fired for real** — calibrated only against mock embedder
  vectors, never the real `fastembed`/MiniLM model. Worth an opportunistic real-world test (run
  `check_similarity.py` with a genuine near-duplicate pair and confirm a real `POSSIBLE_DUPLICATE`
  verdict) but not urgent — the design is sound even if unconfirmed, and dedup is a best-effort
  filter, not a gate.
- `install.sh` hard-requires `python3` (hook registration) and benefits from `uv`/`pipx`/`pip`
  (compass CLI, graphify) and `node` (context-mode) — all three of the latter degrade to a warning,
  never a hard failure.
- `graphify-out/` and `.compass/learnings/.embedding-cache.json` are gitignored/regenerable.
- `.git/hooks/pre-commit` backstop (runs `compass-check.sh` check-only) is installed in this repo.

## Next steps
1. **Commit this session's `AGENTS.md` change** (documents `session_status.py` for token questions).
2. Delete the four stray `*.backup-20260704100310` files under `~/.claude/` (see above).
3. Opportunistically verify learnings-dedup stage 3 against a real `fastembed` run, not proactively.
4. Continue watching `compass init` behave well when agent-driven across ordinary sessions.
5. From here on, compass stays maintenance-mode: fix real breakage as it's found, don't proactively
   expand scope.
