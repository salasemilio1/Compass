# 0007 — extract init-compass's mechanical scaffolding into a real CLI

Date: 2026-07-08

## Context

`init-compass` was a markdown skill: an agent reads `SKILL.md` and performs scaffolding steps
(create directories, seed `INDEX.md`, wire `.gitignore`, write `AGENTS.md`/`CLAUDE.md`) by following
prose instructions. This had a live bug: step 3's template-directory resolution
(`$DOTFILES/compass/template`, then search `~/Dev/` for `compass/template`, then `~/compass/template`)
assumed a persistent local checkout of compass exists on disk -- an assumption decisions/0006 already
invalidated by design (the real install flow clones a dotfiles repo ephemerally and deletes it after
bootstrap). It only "worked" on the primary dev machine because a `~/Dev/compass` checkout happens to
exist there for compass's own development, not because the design guaranteed it.

Discussing the fix, the user proposed modeling this on `potluck` (an internal KUNGFU.AI tool): a real
installed CLI (`potluck init`) with an interactive menu, rather than an agent interpreting markdown.
Potluck's actual mechanism (inspected on this machine): a `pip`-installed package with a
`console_scripts` entry point, Click for the CLI surface, InquirerPy for interactive prompts,
templates bundled as package data.

## Decision

Build `compass init` as a real, lightweight CLI (`cli/`, Click only, no InquirerPy) that performs
**only the mechanical scaffolding** -- directories, `INDEX.md` skeleton, `.gitignore`, `AGENTS.md`,
`CLAUDE.md` pointer -- deterministically, with the template bundled inside the package
(`importlib.resources`, no filesystem search). `install.sh` installs it via the same uv/pipx/pip
fallback chain already used for graphify.

`skills/init-compass/SKILL.md` becomes a thin wrapper: call `compass init --mode <mode>`, then do the
parts that require reading and understanding the repo (seeding `INDEX.md`'s one-line description,
optionally `context/overview.md`) -- work a CLI can't do without an LLM.

**Scope explicitly limited to `init`.** `warm-up` and `wrap-up` were evaluated and rejected for the
same full-CLI treatment:
- `warm-up`'s value is judgment -- "load task-relevant slices, infer from the user's request" --
  which needs an LLM. Its deterministic floor (concatenate `INDEX.md` + `current-state.md` + the
  staleness check) is *already* covered by `hooks/compass-session-start.sh`; a CLI subcommand would
  just duplicate that.
- `wrap-up`'s core output is prose synthesized from a git diff into decision/log entries -- the whole
  point is the agent's judgment about what's worth writing down. A CLI could at most hand it
  diff/dedup plumbing, which doesn't change the skill's shape enough to justify the build.

## Alternatives considered

- **Fix only the template-path search** (e.g. have `install.sh` copy `template/` to a fixed
  `$COMPASS_HOME/compass-template` location, keep the skill markdown-only). Rejected as
  under-fixing: it patches this one bug but leaves the rest of `init-compass`'s mechanical work
  (gitignore wiring, `AGENTS.md` append logic, refuse-to-clobber) as agent-interpreted prose, still
  exposed to the same class of bug (an LLM skipping/misreading a step) that decisions/0001-0005
  already established doesn't hold up for anything correctness-relevant.
- **Full potluck-style CLI for warm-up and wrap-up too, with an interactive menu (InquirerPy).**
  Rejected for now: interactive terminal menus don't drive well from an agent's Bash tool (no real
  TTY), so agent-invoked scaffolding would need a parallel non-interactive flag surface anyway;
  warm-up/wrap-up's actual value is LLM judgment a CLI can't replace. Potential future work if the
  init CLI proves out and a `--yes`/flag-driven non-interactive mode is needed elsewhere.

## Consequences

- Gain: the template-path bug is fixed by construction (bundled package data, not a filesystem
  search) for as long as the CLI exists, not just patched at today's known failure paths.
- Gain: `compass init` works from a plain terminal with zero Claude-Code dependency -- directly
  improves the cross-agent portability `learnings/compass-agent-portability-boundary.md` already
  flagged as a manual-translation gap.
- Cost: first real Python package in the repo (previously all hooks/scripts were standalone files
  run via `python3` directly) -- adds a `pyproject.toml`/build step and a new install target,
  though `install.sh`'s existing optional-tool pattern (graphify, context-mode) absorbed it without
  new install-time failure modes (never counts toward `INSTALL_FAILURES`).
- Follow-up: if `compass init` proves reliable in practice, revisit whether the same "deterministic
  floor + agent judgment on top" split is worth extracting for `warm-up` specifically (its
  deterministic floor already exists in `compass-session-start.sh` -- the open question is whether
  a `compass status` CLI command duplicating that hook is worth it, not whether the pattern works).
