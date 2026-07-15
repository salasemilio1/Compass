# 0009 — add an `environment` category; check `context/overview.md` for repo-layout drift

Date: 2026-07-15
Supersedes: none

## Context

Two related gaps surfaced in one session. First: nothing in compass had a natural home for
machine/service-specific facts — SSH access, non-default ports, pinned tool versions tied to a
specific box. `learnings/` is the closest fit but is topic-keyed and deduped by text similarity,
not host-keyed, so it can't reliably answer "what's different about this specific box" and would
accumulate near-duplicate files as a host's facts change over time. Second: while investigating why
agents couldn't find `scripts/compass-check.sh`, `context/overview.md`'s "Repo layout" list turned
out to be stale — written 2026-07-04, before `scripts/` existed and before decisions/0007 moved
scaffolding into `cli/`. Nothing mechanically checks that this list stays current.

## Decision

1. Add `environment/` as a new category (`SPEC.md` §1, §2, §4, §5, §6): one file per machine/service,
   named after the host/alias, **overwritten in place** like `context/current-state.md` rather than
   accumulated like `logs/`. Sharing tier: private-first/promotable, grouped with
   `experiments`/`logs`/`learnings`, called out separately since it often names real hosts/usernames.
   `warm-up` loads the matching file when a task names a remote target; `wrap-up` asks about
   environment quirks as a question distinct from `learnings/`, and reuses the existing
   `learning-dedup` similarity check — a near-duplicate hit against the *same host's* file is treated
   as a near-miss signal worth surfacing, not just silently merged.
2. Add a warn-only check to `scripts/compass-check.sh` (gap F): every top-level directory tracked by
   git must appear as a `` `dirname/ `` mention somewhere in `context/overview.md`. Mirrors gap E's
   style (detection-only, never blocks, `git ls-files` makes it immune to gitignored/generated dirs
   like `graphify-out/`). Fixed the actual staleness this session found (`scripts/`, `cli/`, `tests/`,
   `tools/` were missing from this repo's own `overview.md`).
3. Bumped spec version 1 → 2 for the new category.

## Alternatives considered

- **Fold environment facts into `context/`.** Rejected — `context/` is for project/domain facts read
  by every session; environment facts are keyed by a specific machine and are irrelevant noise on
  every session that isn't touching that machine. A separate category keeps `warm-up` from having to
  read past them when they don't apply.
- **Auto-fix `context/overview.md`'s repo-layout drift instead of just warning.** Rejected — the
  correct one-line description of what a directory holds is a judgment call (see `init.py`'s own
  comment: compass never fabricates project narrative), not a mechanical operation like archiving or
  ADR-marking. Warn-only matches gap E's existing precedent for exactly this reason.
- **A stronger, code-enforced "don't repeat this mistake twice" gate.** Rejected — compass is
  deliberately deterministic-script-gated, not inference-gated; there's no reliable mechanical way to
  detect "the agent is about to repeat a specific past mistake." The realistic mechanism is
  `warm-up` reliably loading `environment/<host>.md` before the fact, not detection after it.

## Consequences

- Gain: a discoverable, host-keyed place for infra quirks that previously had nowhere consistent to
  live; a cheap, non-blocking backstop against `overview.md` repo-layout drift recurring.
- Cost: one more category for `warm-up` to be aware of (mitigated — it's conditional on the task
  naming a remote target, so it's inert otherwise, same pattern as graphify's conditional loading).
- Follow-up: none planned. If `environment/` sees real multi-host usage, revisit whether per-host
  subfolders (vs. flat `environment/<slug>.md`) are still the right shape.
