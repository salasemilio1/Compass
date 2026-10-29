---
claim: "v3 field days 1–4 on ebay-flipper: the hooks went dark for 5 of 8 sessions (a dangling ~/.claude/settings.json symlink), nothing noticed; AGENTS.md fallback carried orientation"
date: 2026-10-29
refs:
  - experiments/2026-10-26-v3-field-week.md
  - context/2026-10-15-ebay-flipper-retro.md
  - tools/field-retro/retro.py
---
# Field retro: compass v3 on ebay-flipper, days 1–4 (2026-10-26 to 10-29)

Date: 2026-10-29. Interim read of the field week ([protocol](../experiments/2026-10-26-v3-field-week.md));
the judge pass and replays stay on 2026-10-30.
Source: `~/Dev/Personal Projects/personal-dashboard` (moved from `~/Dev/Misc/` on 2026-10-28), 8 main
sessions and 44 subagents, compared with the 11 v2 sessions of 2026-10-15 to 10-23.
Method: `tools/field-retro/retro.py --former-path ~/Dev/Misc/personal-dashboard` plus a one-off
cost-by-context-band scan and a hook-event scan of the transcripts. Costs are list-price estimates,
relative only. Four days, different work in each window: read the comparisons loosely.

## 1. The hooks went dark, and nothing noticed
| Delivery at session start | Sessions |
|---|---|
| v3 SessionStart brief (plugin hook) | 1 of 8 |
| v2 brief (`compass-session-start.sh`; plugin not yet installed) | 2 |
| **No hook of any kind fired** (2026-10-28 14:28 → 10-29 ~21:00 UTC) | **5** |

- **Root cause:** `~/.claude/settings.json` was a symlink to `~/Dev/ADE/claude/settings.json`. The
  folder reorganisation on 2026-10-28 removed `~/Dev/ADE`, so the link dangled. Claude Code then read
  no user settings: no `hooks`, and no `enabledPlugins`, so the compass plugin was off. RTK's
  rewrite hook and the v2 hooks went silent with it. An `ai-dev-setup` run on 2026-10-29 replaced the
  link with a real file; a headless session in the repo then got the brief again.
- **Nothing in compass could see it.** `compass doctor` checks the binary on PATH and the git hook,
  not whether Claude Code will load the plugin. The Stop and PreCompact staleness guards were off for
  the same 5 sessions.
- **The AGENTS.md block was the fallback that worked:** 4 of the 5 dark sessions ran
  `compass resume` themselves because the block says to.

## 2. v2 leftovers ran beside v3 until the settings broke
- `~/.claude/hooks/tool-routing-guard.py` (not executable) failed with "Permission denied" 639 times
  in v3 sessions (main and subagents): a non-blocking error on almost every Bash call.
- `session-context-nudge.py` kept suggesting `/wrap-up` → `/warm-up` at 90% of an assumed 200k window.
- v3's install and migration never removed v2's hook registrations, and `compass doctor` does not
  look for them.

## 3. Field-week measures (deterministic part; judge pass on 10-30)
| # | Measure | Days 1–4 | Bar |
|---|---|---|---|
| — | v3 brief injected at start | 1/8 | tracked |
| 1a | Continue-style sessions that oriented first | 0/0 (no session opened with a continue prompt) | ≥ 80% |
| 1b | Sessions with a commit that rewrote current-state | 5/6 (83%) | ≥ 90% |
| 2a | Flagged edits made after a records search | 12/12 | ≥ 80% |
| 3a | Compass share of tool I/O, median per session | 15.6% | ≤ 10% (kill > 16.8%) |
| — | `compass record` calls | 4 | read by hand |
| 2c | Human catches | 0 | each read by hand |

The retro script needed three fixes to measure this (in the same change): it counted the AGENTS.md
instructions attachment as a brief (7/10 → 1/8), missed current-state rewrites done by shell
heredoc (0/7 → 5/6), and could not follow a moved repo (`--former-path`).

## 4. Cost and context
| | v2 (10-15..10-23) | v3 (10-26..10-29) |
|---|---|---|
| Main $ per session | 40 | 33 |
| Peak context, max | 756k | 580k |
| Main cost above 300k context | 67% | 55% |
| Main cost above 600k context | 11% | 0% |
| Subagent share | 9% | 15% |

Likely the one-problem-one-PR-one-session habit from the 10-15 retro rather than v3 itself, which
has no context nudge (decision 0015).

## 5. The records
- `INDEX.md` 5.9 KB (was 103 KB); no v3 session read it. Brief 6–9 KB.
- `rg -il … .compass/` ran 3 times; agents mostly opened files the brief and plans named.
- `compass record` ran 4 times in 8 sessions; most new records were hand-written.
- `compass check` fails in the main checkout: one claim at 203 characters (an uncommitted
  experiment) and a stale index.
- 27 of 154 claims end in "…": the migration cut first lines at 200 characters. 0117's claim repeats
  its number.
- 11 records and plans sit untracked in the main checkout, invisible to worktree sessions.
- 8 of 26 commits touched only `.compass/`; all landed on their feature branches (wrap-up into the
  PR works), 4 of them current-state refreshes.

## 6. What changes
1. `compass doctor` checks what Claude Code will actually load: settings.json readable (a dangling
   symlink or bad JSON is a problem), the plugin enabled, its hooks file present, and no v2 hook
   registrations left. The SessionStart hook is the one place every session passes through, so a
   brief that cannot be delivered must be loud somewhere else: the AGENTS.md block tells agents to
   run `compass doctor` when the brief is missing.
2. An end-to-end install test on a clean HOME: install the plugin from the repo, start a headless
   session, assert the brief arrives.
3. The 10-30 judge pass runs as planned on the fixed script.
