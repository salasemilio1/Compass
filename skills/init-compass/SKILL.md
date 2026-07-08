---
name: init-compass
description: Scaffold the compass agent-memory protocol (.compass/) into the current repository. Use when a repo doesn't yet have a .compass/ directory and the user wants to adopt the in-repo knowledge protocol, or says "init compass" / "add compass" / "set up project memory".
---

# init-compass

Scaffold the `compass` protocol into the current repository, then seed it with what's already
knowable about the project. The full protocol is defined in `SPEC.md` in the compass repo.

The mechanical scaffolding (directories, `INDEX.md`, `.gitignore`, `AGENTS.md`, `CLAUDE.md`) is
handled deterministically by the `compass` CLI (`cli/`), not by agent judgment — this closed a real
bug where the old markdown-only version searched the filesystem for a template directory
(`$DOTFILES/compass/template`, `~/Dev/*/compass/template`, `~/compass/template`) that often didn't
exist post-install. Only the parts that require actually reading and understanding the repo are
left to you.

## Steps

1. **Refuse to clobber.** If `.compass/` already exists, stop and report it — `compass init` also
   refuses, but check first so you don't need to parse its error.

2. **Choose the visibility mode** — ask the user, defaulting by signal:
   - **`committed`** — repos the user owns/maintains. Shared categories (`context`, `architecture`,
     `decisions`) are committed and reach `main`.
   - **`branch-local`** — inherited/guest repos. The whole `.compass/` is gitignored so it never lands
     on `main` unless specific files are deliberately promoted.

   Signal for the default: if the user is on a feature branch of a repo they didn't create, lean
   `branch-local`; otherwise `committed`. Confirm before proceeding.

3. **Run the CLI**: `compass init --mode <committed|branch-local>`. This creates the seven category
   directories, seeds `INDEX.md` from the bundled template, wires `.gitignore`, and creates/appends
   `AGENTS.md` (stamped with the compass spec version so `scripts/compass-check.sh` can later warn on
   drift) and a `CLAUDE.md` pointer. If the `compass` command isn't installed, tell the user to run
   `install.sh` from the compass repo, or fall back to generating the structure directly from
   `SPEC.md` (the category set and conventions are fully specified there).

4. **Seed `INDEX.md`'s project description** by inspecting the repo (read README, package manifests,
   top-level layout). Fill in the one-line description and stack line that `compass init` left as
   placeholders. Leave category sections empty (`_none yet_`) — entries are earned by real work, not
   invented.

5. **Optionally seed `context/overview.md`** from the README/manifests if there's enough to write a
   faithful, non-speculative overview. Keep it short. If unsure, leave it for the user.

6. **Report** what was created and the chosen mode, and remind the user that capture happens via
   `wrap-up` / `new-experiment` (human-reviewed) — compass is never written silently.

## Notes
- Follow `coding-style.md` conventions for any prose written into seeded files.
- Do not fabricate context, decisions, or experiments. Empty is correct until real work fills it.
