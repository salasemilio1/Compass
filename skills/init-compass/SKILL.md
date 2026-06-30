---
name: init-compass
description: Scaffold the compass agent-memory protocol (.compass/) into the current repository. Use when a repo doesn't yet have a .compass/ directory and the user wants to adopt the in-repo knowledge protocol, or says "init compass" / "add compass" / "set up project memory".
---

# init-compass

Scaffold the `compass` protocol into the current repository by creating a `.compass/` directory from the
template, then seeding it with what's already knowable about the project. The full protocol is defined in
`SPEC.md` in the compass repo.

## Steps

1. **Refuse to clobber.** If `.compass/` already exists, stop and report it. Never overwrite an existing
   compass without explicit confirmation.

2. **Choose the visibility mode** — ask the user, defaulting by signal:
   - **`committed`** — repos the user owns/maintains. Shared categories (`context`, `architecture`,
     `decisions`) are committed and reach `main`.
   - **`branch-local`** — inherited/guest repos. The whole `.compass/` is gitignored so it never lands
     on `main` unless specific files are deliberately promoted.

   Signal for the default: if the user is on a feature branch of a repo they didn't create, lean
   `branch-local`; otherwise `committed`. Confirm before proceeding.

3. **Create `.compass/`.** Copy the structure from the compass template: `INDEX.md` plus the seven
   category directories (`scratch/` is always gitignored). Resolve the template directory in this
   order: `$DOTFILES/compass/template` (dotfiles users with compass as a submodule), then search
   `~/Dev/` for any directory named `compass/template`, then `~/compass/template`. If none is found,
   generate the structure directly from the compass SPEC (the category set and conventions are fully
   specified there) — the template is a convenience, not a hard dependency.

4. **Wire gitignore:**
   - Always add `.compass/scratch/` to the repo's `.gitignore`.
   - If mode is `branch-local`, add the whole `.compass/` to `.gitignore` instead.

5. **Seed `INDEX.md`** by inspecting the repo (read README, package manifests, top-level layout). Fill in
   the project name, one-line description, stack, mode, and today's date. Leave category sections empty
   (`_none yet_`) — entries are earned by real work, not invented.

6. **Optionally seed `context/overview.md`** from the README/manifests if there's enough to write a
   faithful, non-speculative overview. Keep it short. If unsure, leave it for the user.

7. **Create `AGENTS.md` in the repo root.** This makes the repo immediately usable by any agent
   (Claude, Gemini, Cursor, Copilot, etc.), not just Claude Code. The file should be short — point to
   compass and state the behavioral defaults:
   ```markdown
   # Agent instructions

   This repo uses the compass protocol for agent memory. Before substantial work:
   1. Read `.compass/INDEX.md` — the retrieval map.
   2. Read `.compass/context/current-state.md` — where the work stands and what's next.

   At task end, run the wrap-up workflow (git-diff-first, human-reviewed) to capture knowledge back
   into `.compass/`. Never write `.compass/` silently.

   Coding behavior and style: see `$DOTFILES/claude/AGENTS.md` or ask the repo owner for conventions.
   ```
   If `AGENTS.md` already exists, append the compass section rather than overwriting.

8. **Point CLAUDE.md at compass.** If the repo has a `CLAUDE.md`, add a short line directing agents to
   read `.compass/INDEX.md` first. If it doesn't, offer to create a minimal one that does so.

9. **Report** what was created and the chosen mode, and remind the user that capture happens via
   `wrap-up` / `new-experiment` (human-reviewed) — compass is never written silently.

## Notes
- Follow `coding-style.md` conventions for any prose written into seeded files.
- Do not fabricate context, decisions, or experiments. Empty is correct until real work fills it.
