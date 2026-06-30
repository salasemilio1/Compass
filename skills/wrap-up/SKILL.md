---
name: wrap-up
description: At the end of a task or before ending a long session, extract its durable knowledge into the repo's .compass/ — implementation log, experiments, decisions, learnings, resumable state — then update the index. Human-reviewed, never silent. Use when the user says "wrap up", "wrap this up", "archive this session", "capture this before I log off", or has clearly finished a coherent unit of work.
---

# wrap-up

Turn a finished (or paused) session into durable repository memory, so the conversation can be discarded
without losing anything. Capture is **human-gated**: draft, present for review, write only after approval.
Never write `.compass/` silently.

## Preconditions
- The repo must have a `.compass/` (run `init-compass` first if not — offer it).
- For huge sessions, prefer **several focused passes** (architecture, then decisions, then current-state)
  over one mega-extraction — higher quality, index stays clean.

## Steps

1. **Find the anchor commit.** Check `.compass/INDEX.md` for the most recent log entry — it contains a
   `Commits:` field with the hash range it covered. Use the end of that range as the anchor. If no prior
   log exists, use the last 20 commits (`git log --oneline -20`) as context instead.

2. **Read the diff, not the codebase.** Run these three commands:
   ```
   git log <anchor>..HEAD --oneline          # commit messages = the "why"
   git diff <anchor>..HEAD --stat            # what files changed (summary)
   git diff <anchor>..HEAD                   # the actual changes (ground truth)
   ```
   **If context-mode is available (strongly preferred):** run all three via `ctx_batch_execute` or
   sequential `ctx_execute` calls so the raw output is sandboxed — only a compressed summary lands in
   context. This prevents a large diff from consuming thousands of tokens.
   **Fallback (context-mode not installed):** run via Bash directly; token cost will be higher.
   Draft from the diff output — do **not** re-read source files or conversation history.

3. **Draft the artifacts** that apply (not all sessions need all of them):
   - **`logs/YYYY-MM-DD-slug.md`** — for completed tasks: include `Commits: <anchor>..<HEAD hash>` so
     the next wrap-up has a clean anchor. Cover: why · what changed · assumptions changed · learnings ·
     breaking changes.
   - **`experiments/YYYY-MM-DD-slug.md`** — if ML runs happened. Or invoke `new-experiment`.
   - **`decisions/NNNN-slug.md`** — for notable choices: context · decision · alternatives · consequences.
   - **`learnings/slug.md`** — durable gotchas worth never repeating.
   - **`context/current-state.md`** — **for in-progress projects**: where the work stands, what's in
     flight, known issues, and concrete next steps. Living doc — overwrite, don't append.

4. **Update `INDEX.md`** — add a one-line row for each new entry (newest first for `logs`/`experiments`),
   refresh the `Updated:` date, update `Start here` pointers if they changed.

5. **Respect visibility** (see `compass/SPEC.md`): `context`/`architecture`/`decisions` are shared;
   `experiments`/`logs`/`learnings` are private-first; `scratch` is gitignored.

6. **Present every draft for review.** Show files and contents. Apply edits. Only then write to disk.
   If mode is `committed`, offer to stage (not commit unless asked).

7. **Keep it concise.** Retrieval aids, not transcripts. Follow `coding-style.md` conventions in prose.

## Output
A short summary of what was captured and where, so the user can confidently end the session.
