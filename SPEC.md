# compass — protocol specification

**Spec version:** 1 — bump this when a convention below changes, so `scripts/compass-check.sh` can
detect a repo's `AGENTS.md` drifting behind the spec it was scaffolded from.

`compass` is the in-repo agent-memory protocol. A repo "has a compass" when it contains a `.compass/`
directory following this spec. Its purpose: **orient any agent in the codebase cheaply**, so sessions
can be disposable and knowledge lives in the repo instead of in a conversation.

This spec is the contract. `cli/compass_cli/template/` is the concrete scaffold that the `compass`
CLI's `init` command copies into a target repo (see `/init-compass`, which wraps it).

---

## 1. Directory layout

```
.compass/
  INDEX.md          # the retrieval map — read FIRST during warm-up (kept small)
  context/          # domain / client / project context
  architecture/     # how the system is built: components, data flow, boundaries
  decisions/        # ADRs — why a choice was made, alternatives, consequences
  experiments/      # ML runs: hyperparameters, configs, results, observations, failures
  logs/             # per-task implementation logs (small, searchable)
  learnings/        # gotchas and insights — "don't do X again"
  scratch/          # ephemeral working notes (never authoritative)
```

### Relationship to adjacent optional tools

A repo may also have a `graphify-out/` directory (from [graphify](https://github.com/safishamsi/graphify),
an auto-generated code structure graph) or context-mode sandboxing configured. compass never reads or
writes `graphify-out/` — it's read-only reference material an agent may consult for mechanical lookups
("what calls X"), same trust tier as reading source directly. compass stays the source of curated intent
("why was X built this way"); the two are complementary, not merged.

---

## 2. The categories

| Category | Holds | Answers |
| --- | --- | --- |
| `context` | Domain, client, and project facts; glossary; goals; constraints; **`current-state.md`** (resumable status + next steps for in-progress work) | "What is this project and why does it exist? Where are we and what's next?" |
| `architecture` | Components, data flow, module boundaries, key interfaces | "How is it built and where does X live?" |
| `decisions` | ADRs: decision, alternatives considered, consequences | "Why was it done this way?" |
| `experiments` | One file per run/sweep: config + results + what we learned | "What learning rate did we settle on for the BERT Mini run?" |
| `logs` | One file per completed task: why / files touched / assumptions / learnings / breaking changes | "What changed in this work and why?" |
| `learnings` | Durable gotchas extracted from logs/experiments | "What mistake should I not repeat?" |
| `scratch` | Temporary notes for in-flight work | nothing authoritative |

**Dedup note (`learnings`):** before a new `learnings/*.md` is written, `wrap-up` runs
`tools/learning-dedup/check_similarity.py` — a three-stage filter (exact hash → lexical overlap →
embedding similarity, only reaching the model for the genuinely ambiguous case). Stages 1–2 are exact
and lexical guarantees, not heuristics; the embedding stage narrows the true-paraphrase residual but
does not guarantee it, and no stage can tell you a learning has become outright false. That residual is
left open by design, not solved.

---

## 3. Sharing model — hard categories + easy promotion

Knowledge has two audiences: any agent/teammate working the codebase (shared), and me (private until
I choose to share). The split is by category, with explicit promotion.

| Category | Default visibility | Notes |
| --- | --- | --- |
| `context`, `architecture`, `decisions` | **Shared** — committed, reaches `main` | Legible to teammates' agents by design |
| `experiments`, `logs`, `learnings` | **Private-first**, promotable | Mine until promoted; useful to share when they help the team |
| `scratch` | **Private** — gitignored | Never committed |

**Promotion** moves a private-first entry into the shared set (and onto `main`). This is the
`promote` skill (Phase 2).

### Inherited repos (feature-branch + PR workflow)
When I'm a guest in someone else's repo, `.compass/` lives on my feature branches and is never *forced*
onto `main`. Two supported modes, chosen at `/init-compass` time:

- **`committed`** (repos I own/maintain): the shared categories are committed and meant to reach `main`.
- **`branch-local`** (inherited repos): the whole `.compass/` is added to `.gitignore` and kept on my
  branch only, so it never lands on `main` unless I deliberately stage and promote specific files.

> **Open question (settling here):** mechanism is **gitignore-based** rather than a separate local-only
> path — it keeps everything in one place and lets `git status` show me what I'd be sharing. `scratch/`
> is always gitignored regardless of mode.

---

## 4. `INDEX.md` format — small but sufficient (the make-or-break detail)

`INDEX.md` is the *map*, not the territory. warm-up reads it first to decide what else to load. It must
stay small: it summarizes and points; it never inlines category content.

Required structure:

```markdown
# compass index — <project name>

> <one-sentence description of what this project is and does>

**Stack:** <languages/frameworks>   ·   **Mode:** committed | branch-local   ·   **Updated:** <YYYY-MM-DD>

## Start here
- New to the project? Read: context/overview.md, architecture/overview.md
- Common tasks: <task> → <pointer>

## context
- [overview.md](context/overview.md) — <one line>

## architecture
- [overview.md](architecture/overview.md) — <one line>

## decisions
- [0001-<slug>.md](decisions/0001-<slug>.md) — <one line>

## experiments
- [<date>-<slug>.md](experiments/<date>-<slug>.md) — <one line: model + headline result>

## logs
- [<date>-<slug>.md](logs/<date>-<slug>.md) — <one line>

## learnings
- [<slug>.md](learnings/<slug>.md) — <one line>
```

Rules:
- **One line per entry**, linking to the file. The line is a retrieval hint, not a summary of contents.
- Newest entries first within `experiments`/`logs`.
- `INDEX.md` is regenerated/updated by `wrap-up` and `new-experiment`; it is the single thing warm-up is
  guaranteed to read, so every new artifact must add its line here.
- **Capped.** `logs` and `experiments` sections are capped at 15 entries each. Entries beyond the cap are
  moved verbatim to `<category>/archive.md` (never linked from "Start here") by
  `scripts/compass-check.sh --fix`, run automatically as part of `wrap-up`. This keeps warm-up's
  guaranteed read cheap regardless of how long a repo has been in use — see `scripts/compass-check.sh`
  for the mechanism.

---

## 5. Entry conventions

- **Filenames:** `kebab-case`. Dated entries (`experiments`, `logs`) are prefixed `YYYY-MM-DD-`. ADRs
  are numbered `NNNN-slug.md`.
- **Every entry** opens with a one-line summary and a `Date:` line so it's self-describing out of context.
- **Concise over complete.** These are retrieval aids, not transcripts. If an entry grows long, it's
  probably two entries.
- Entries follow my `coding-style.md` conventions for prose too: precise, no needless abbreviation.
- **ADR supersession:** an ADR that replaces an earlier one adds a `Supersedes: NNNN` line directly
  under its `Date:` line. `scripts/compass-check.sh` mechanically prefixes the superseded ADR's
  `INDEX.md` row with `[SUPERSEDED by NNNN]` — this is a deterministic string operation on explicit
  metadata, not an inference about whether two decisions conflict.
- **`current-state.md` anchor:** carries a `Commit: <hash>` line under its `Date:` line, stamped by
  `wrap-up` at the commit it was last written against — mirrors the `Commits: <range>` field already
  used in `logs/` entries. Lets `warm-up`/the `SessionStart` hook check staleness by commit count
  instead of by inference.

---

## 6. Lifecycle (how entries get created)

- `warm-up` — reads `INDEX.md`, `context/current-state.md`, + relevant slices to prime a session. Read-only.
- `wrap-up` — at task end, drafts a `logs/` entry (+ any ADRs, + `INDEX.md` update), I review, commit.
  Also runs `scripts/compass-check.sh --fix` (archives overflow, stamps `Commit:`) and, before writing a
  new `learnings/` entry, `tools/learning-dedup/check_similarity.py` (see dedup note above) — both
  deterministic steps, not judgment calls left to the skill.
- `new-experiment` — scaffolds an `experiments/` entry from a run's config/results.
- `promote` — moves a private-first entry into the shared set.

All capture is **command-triggered and human-reviewed**. Agents never silently write `.compass/`.
`scripts/compass-check.sh` (check-only, no `--fix`) also runs as this repo's own local `.git/hooks/pre-commit`
— a backstop that blocks a commit on a real violation independent of whether `wrap-up`'s steps were
followed correctly.
