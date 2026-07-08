# feat: extract init-compass's mechanical scaffolding into a real `compass` CLI

Date: 2026-07-08
Commits: 76b4d9c..HEAD (uncommitted at time of writing)

## Why
While explaining a known-deferred backlog item (`init-compass`'s template-path search --
`$DOTFILES/compass/template` then `~/Dev/*/compass/template` then `~/compass/template`, only working
on this machine because a persistent `~/Dev/compass` checkout happens to exist), the user asked to
weigh modeling `compass init` on `potluck` (an internal KUNGFU.AI tool): a real installed CLI with
`potluck init`, rather than an agent interpreting a markdown skill. Inspected potluck's actual
install on this machine (`pip install -e`, setuptools `console_scripts` entry point, Click +
InquirerPy, templates bundled as package data) as the convention to mirror.

Decided this fits `init` specifically: it's purely mechanical (dirs, gitignore, AGENTS.md), and
compass has repeatedly replaced agent judgment with deterministic code for exactly this kind of
correctness-relevant work (decisions/0001, 0002, 0003, 0005). `warm-up`/`wrap-up` were judged a worse
fit -- their value is synthesis/judgment an LLM has to do -- and were left as skills. See
`decisions/0007-compass-init-as-cli.md`.

## What changed
- New `cli/` package (`compass-cli`, Click-only dependency) with a single `compass init` command
  (`cli/compass_cli/commands/init.py`). Bundles the `.compass/INDEX.md` template inside the package
  (`cli/compass_cli/template/`, loaded via `importlib.resources`) instead of searching the
  filesystem -- this closes the template-path bug by construction, not just by relocating the search.
- `compass init [--mode committed|branch-local] [--path DIR]`: refuses to clobber an existing
  `.compass/`, creates the seven category directories, seeds `INDEX.md`, wires `.gitignore`
  (`.compass/scratch/` vs. whole-`.compass/` by mode), creates/appends `AGENTS.md` (spec-version
  stamped) and a `CLAUDE.md` pointer -- all idempotent re-runs are safe (checked via marker text
  before appending).
- Moved the old top-level `template/INDEX.md` into `cli/compass_cli/template/INDEX.md` (single
  source, no drift risk). Updated `SPEC.md`'s pointer accordingly.
- `install.sh` gained `install_compass_cli()`, following the exact uv/pipx/pip fallback pattern
  already established for graphify -- installs from this checkout's `cli/` path, always reinstalls
  (same "install.sh is also the update mechanism" convention as `install_path`). Never counts toward
  `INSTALL_FAILURES` -- it's an optional convenience layer, same tier as graphify/context-mode.
- `skills/init-compass/SKILL.md` rewritten: steps 3/4/7/8 (scaffolding) collapse into "run
  `compass init`"; steps 4-5 (seeding `INDEX.md`'s description, `context/overview.md`) remain agent
  judgment, done after the CLI runs. README/SPEC updated to match.
- CI (`.github/workflows/ci.yml`) gained a step running `cli/tests` via `uv run --project cli`.
- `tests/test_install.sh` gained 9 checks for `install_compass_cli`'s three branches (uv available +
  installed binary is runnable; no uv/pipx/pip -- warns, doesn't fail; installer command fails --
  warns, doesn't fail), isolated via `UV_TOOL_DIR`/`UV_TOOL_BIN_DIR` so the real global `uv tool`
  state is never touched.

## Assumptions / scope
- `compass` as the distribution/command name is fine for a local-only, non-PyPI-published install;
  flagged to revisit naming if this is ever published (graphify's own `graphifyy`-vs-`graphify` split
  is the precedent for why that matters).
- Did not build `warm-up`/`wrap-up` CLI equivalents -- explicitly scoped out, see decisions/0007.
- Left `compass init`'s mode-selection heuristic (committed vs. branch-local) to the agent calling
  it, not the CLI -- the CLI defaults to prompting with `committed` as the default choice, since
  detecting "guest repo" signal requires context (whose repo is this) the CLI doesn't have.

## Learnings
See `learnings/pretooluse-guard-must-exempt-backgrounded-commands.md` (from the same session, unrelated
finding) and `decisions/0007-compass-init-as-cli.md` for the extraction-boundary reasoning.

## Breaking changes
None for existing installs -- `install.sh` additively installs the new CLI; the old markdown-only
`init-compass` behavior still works standalone if `compass` isn't installed (documented fallback in
the skill).
