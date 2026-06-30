# compass — protocol specification

`compass` is the in-repo agent-memory protocol. A repo "has a compass" when it contains a `.compass/`
directory following this spec. Its purpose: **orient any agent in the codebase cheaply**, so sessions
can be disposable and knowledge lives in the repo instead of in a conversation.

This spec is the contract. The `template/` directory beside it is the concrete scaffold that
`/init-compass` copies into a target repo.

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

---

## 5. Entry conventions

- **Filenames:** `kebab-case`. Dated entries (`experiments`, `logs`) are prefixed `YYYY-MM-DD-`. ADRs
  are numbered `NNNN-slug.md`.
- **Every entry** opens with a one-line summary and a `Date:` line so it's self-describing out of context.
- **Concise over complete.** These are retrieval aids, not transcripts. If an entry grows long, it's
  probably two entries.
- Entries follow my `coding-style.md` conventions for prose too: precise, no needless abbreviation.

---

## 6. Lifecycle (how entries get created)

- `warm-up` — reads `INDEX.md`, `context/current-state.md`, + relevant slices to prime a session. Read-only.
- `wrap-up` — at task end, drafts a `logs/` entry (+ any ADRs, + `INDEX.md` update), I review, commit.
- `new-experiment` — scaffolds an `experiments/` entry from a run's config/results.
- `promote` — moves a private-first entry into the shared set.

All capture is **command-triggered and human-reviewed**. Agents never silently write `.compass/`.
