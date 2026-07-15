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
   - **`decisions/NNNN-slug.md`** — for notable choices: context · decision · alternatives ·
     consequences. If this decision replaces an earlier ADR, add a `Supersedes: NNNN` line under
     `Date:` (see `compass/SPEC.md` §5) — `compass-check.sh` (step 4a) marks the old one automatically.
   - **`learnings/slug.md`** — durable gotchas worth never repeating.
   - **`environment/<host>.md`** — ask this **separately** from learnings: did anything about the
     *environment itself* (not the code) surprise you — SSH access, a non-default port/path, a version
     that had to be pinned differently than usual? If so, write or update the file for that host/service
     in place (living doc, like `current-state.md` — don't create a second file for the same box).
   - **`context/current-state.md`** — **for in-progress projects**: where the work stands, what's in
     flight, known issues, and concrete next steps. Living doc — overwrite, don't append. Don't hand-write
     the `Commit:` line — `compass-check.sh --fix` (step 4a) stamps it.

4. **Before finalizing a new `learnings/` or `environment/` entry, run the dedup check** (guarded by
   `command -v uv`):
   ```
   uv run tools/learning-dedup/check_similarity.py .compass "<the drafted learning's text>"
   ```
   `DUPLICATE`/`POSSIBLE_DUPLICATE` → surface it in the review presentation ("this looks like/similar to
   `learnings/X.md` (score) — update that instead of a new file?") instead of silently adding a
   duplicate. `UNAVAILABLE: <reason>` (missing dependency, offline, etc.) → show the reason to the user
   once and proceed normally; this check is a best-effort gentle filter, never a gate.
   For `environment/`, a near-duplicate hit against the *same host's* existing file isn't a dedup case —
   it means an existing quirk almost got silently re-discovered. Say so plainly instead of just merging
   the text: that's the "don't repeat the same mistake twice" signal actually firing.

5. **Update `INDEX.md`** — add a one-line row for each new entry (newest first for `logs`/`experiments`),
   refresh the `Updated:` date, update `Start here` pointers if they changed.

6. **Respect visibility** (see `compass/SPEC.md`): `context`/`architecture`/`decisions` are shared;
   `experiments`/`logs`/`learnings`/`environment` are private-first; `scratch` is gitignored.
   `environment/` often names real hosts/usernames — treat promotion to shared deliberately.

7. **Summarize, don't dump — then ask everything at once.** Wrap-up usually runs when the user is
   already low on context or ready to end the session; don't drag that out. Do not paste full file
   contents into the conversation by default. Instead:
   - Give a **2-3 sentence summary** per artifact (what it captures, not its full text) — e.g. "1 log
     entry: closed the pre-commit backstop gap. 1 learning: `.git/hooks` writes need explicit
     go-ahead even after plan approval. `current-state.md` rewritten to reflect that."
   - Ask **one consolidated round of questions** covering write / commit / push together (e.g. via
     `AskUserQuestion`, one question per action with sensible defaults), instead of asking after each
     step in sequence. If mode isn't `committed`, drop the commit/push questions entirely.
   - Only show a file's full contents if the user asks to see it, or flags something as suspicious.
   - Execute whatever combination was approved, then stop — don't re-open the round for confirmation
     on each individual git command (`add`, `commit`, `push`) once the batch answer covers it.

8. **Run `scripts/compass-check.sh --fix .compass`** — mandatory, not optional, run immediately after
   writing to disk (before the commit step, if committing). This archives any `logs`/`experiments`
   overflow past the cap, marks superseded ADRs, and stamps `current-state.md`'s `Commit:` line.
   Deterministic — no judgment call left for the skill to make here.

9. **Keep it concise.** Retrieval aids, not transcripts. Follow `coding-style.md` conventions in prose.

## Output
A short summary of what was captured, committed, and/or pushed (per step 7's batch decision), so the
session can end without further back-and-forth.
