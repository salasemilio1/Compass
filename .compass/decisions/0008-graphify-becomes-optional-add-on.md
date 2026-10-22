---
claim: "graphify moves from default-installed+enforced to opt-in only (`COMPASS_WITH_GRAPHIFY=1`); a week under decisions/0005's forced-routing guard produced under 5 real uses despite the tool working well…"
supersedes:
  - "0001"
---
# 0008 — graphify becomes an opt-in add-on, not part of the default install

Date: 2026-07-10

## Context

decisions/0005 built a `PreToolUse` guard specifically to force `graphify query` usage after
decisions/0004 found it had never been called. A week later, with that guard live the whole time,
real usage was still fewer than 5 calls. Live-testing `graphify query`/`explain` this session
confirmed the tool itself works well for cross-language call-graph questions (e.g. bash `install.sh`
calling into Python `init.py` in one shot) — the problem isn't capability, it's that compass's own
scale rarely produces a question where the graph beats a couple of greps. `graphify-out/` is also
~3.2MB, roughly half of the entire repo's disk footprint, entirely for a feature that fires rarely.
Doubling down on enforcement (broadening the guard's trigger conditions) would mean forcing a tool
into cases it wasn't earning, which cuts against orthogonality more than removing it would.

## Decision

graphify moves from installed-by-default to an explicit opt-in:

- `install.sh` skips `install_graphify` unless `COMPASS_WITH_GRAPHIFY=1` is set.
- `hooks/tool-routing-guard.py` drops its `graphify`-routing rule entirely (was: redirect broad
  greps/finds toward `graphify query` when a graph exists). Broad searches now always route to
  `ctx_execute`, regardless of whether a graph happens to exist.
- The `SessionStart` hook's auto-refresh (`graphify update .`) and `warm-up`'s "prefer graphify
  query" guidance are left as-is — both are already conditional on `graphify-out/graph.json`
  existing, so they're inert unless a repo owner opts in and builds a graph themselves.
- README/AGENTS.md/SKILL.md updated to describe graphify as opt-in, not part of the standard setup.

This supersedes decisions/0001 (deterministic-only graphify wiring) and amends decisions/0005 (drops
its graphify-first rule) — deterministic-only remains the right *mode* if graphify is used at all,
but "wired in and enforced by default" is reversed.

## Alternatives considered

- **Broaden the guard's trigger conditions** so more commands get redirected toward graphify.
  Rejected — the guard already had a full week of unconditional enforcement on its existing
  trigger (broad grep/find) and still produced under 5 uses; the gap is demand, not trigger
  coverage.
- **Remove graphify entirely**, including the hook/skill wiring. Rejected — the tool proved
  genuinely useful when tested live; the existing infra (SessionStart refresh, warm-up guidance)
  is already correctly conditional and costs nothing when unused, so keeping it as an opt-in
  costs nothing either.
- **Keep it default-installed but drop only the enforcement guard.** Rejected — without
  enforcement, decisions/0004/0005's own data says it won't get used organically, so keeping it
  default-installed would just restore the original problem (footprint for no payoff) without the
  guard's downside of forcing bad-fit redirects.

## Consequences

- Gain: compass's default footprint drops back to `.compass/` scale (~350KB) instead of carrying a
  3.2MB structural graph that's rarely queried; the tool-routing guard no longer forces a redirect
  into a tool that may not have earned it for this specific repo.
- Cost: anyone who wants graphify's navigation layer now has to explicitly opt in
  (`COMPASS_WITH_GRAPHIFY=1 ./install.sh` or a direct `uv tool install graphifyy && graphify
  install`) instead of getting it automatically.
- Follow-up: none planned — this is a settled call, not a re-open-later experiment. If a future
  repo turns out large/complex enough that graphify's call-graph queries would earn their keep
  routinely, opt in there; don't re-add it to compass's default path without new evidence.
