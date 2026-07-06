# 0001 — use graphify's deterministic mode only, skip the LLM layer

Date: 2026-07-06

## Context

We piloted [graphify](https://github.com/safishamsi/graphify) as a structural-navigation layer alongside
compass (see `context/overview.md` for the three-layer model). graphify has two extraction modes:

- **Deterministic**: `graphify update <path>` — tree-sitter for code, regex for markdown headers/links.
  No LLM, no API key, no network call, no cost.
- **Semantic**: `graphify <path> --backend <name>` — an LLM (cloud or local via Ollama) extracts
  concepts/relationships from prose files (`.md`, `.txt`, `.rst`) and names graph communities.

We initially configured the semantic layer against a local Ollama model (`qwen2.5-coder:14b`) to avoid
cloud cost. Testing it against this repo surfaced a real, if minor, data-quality issue: the local model's
output dropped a required field on one edge (schema violation), consistent with the general expectation
that a 7-14B local model is less reliable at structured extraction than a frontier model.

Separately, the semantic layer introduces a real staleness-refresh cost: unlike deterministic extraction
(near-instant, safe to run on every session start), semantic extraction takes real wall-clock time even
at $0, which would force a choice between slowing every session start or building advisory
staleness-detection logic into `warm-up`/`wrap-up`.

compass's stated goal is to be easy, orthogonal, and to make working in a codebase as efficient as
possible — stringing together small deterministic tools rather than absorbing complexity. An LLM backend
with local-model quality caveats and its own refresh-latency tradeoff works against that goal, for a
capability (inferred cross-document relationships, prettier community names) we don't have evidence we
need yet.

## Decision

compass's graphify wiring uses **only** `graphify update <path>` — the deterministic mode. No backend is
configured; no API key or Ollama setup is required or documented as part of compass's install.

This is enough to get: a real code call-graph, markdown document structure, and explicit cross-file
markdown links (confirmed via direct testing: `.compass/INDEX.md` → `context/overview.md` and → `SPEC.md`
extracted as real edges, zero LLM involved). Because it's fully deterministic, the `SessionStart` hook
refreshes it unconditionally in the background on every session start (see `hooks/compass-session-start.sh`)
— no cost/latency tradeoff to gate on.

## Alternatives considered

- **Local LLM (Ollama)** — zero API cost, but real latency per refresh and observed data-quality
  issues (dropped edge fields) even on a small corpus. Would also need advisory staleness logic in
  `warm-up`/`wrap-up` rather than an unconditional background refresh.
- **Cloud LLM backend** — better extraction quality, but reintroduces an API-key dependency and per-run
  cost, working against the "easy, orthogonal" design goal.

## Consequences

- Lose: inferred (non-explicit) relationships between documents, and LLM-generated readable community
  names (fallback names are the top node's raw identifier, e.g. `install.sh` instead of `"Installation
  Scripts"`).
- Gain: zero-config, zero-cost, zero-latency, offline-capable structural layer that needs no ongoing
  maintenance decision — it just stays fresh. If a future need for inferred cross-document relationships
  emerges, revisit this decision rather than defaulting the semantic layer back on.
