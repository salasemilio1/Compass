# current state

Date: 2026-10-29
Commit: 01c126703c83c6f5bff4b289488b7ce3eb122578

## Objective
Make compass notice when Claude Code won't deliver it (the 10-28/29 ebay-flipper outage: a dangling
~/.claude/settings.json symlink turned off every hook and the plugin, silently), and prove a fresh
install works end to end. Spec, plan and critique: docs/specs/2026-10-29-delivery-health.md (branch
fix/hooks-go-dark). Field week on ebay-flipper continues to 2026-10-30.

## Next steps
1. Implementation is in the working tree, uncommitted: internal/compass/delivery.go, doctor/resume/
   pre-commit callers, AGENTS block line, version 0.3.1, tests/e2e.sh + CI job, decision 0020.
   Unit, smoke and e2e pass; fold in the break-pass findings, then commit.
2. Open the PR with pr-intent; after merge, update the plugin on this machine and confirm
   ebay-flipper gets the brief and `compass doctor` shows no delivery problem.
3. Friday 2026-10-30: field-retro over the week with `--former-path ~/Dev/Misc/personal-dashboard`
   (repo moved), judge pass and at most 5 replays; append to experiments/2026-10-26-v3-field-week.md,
   reporting 1a/3a with and without the 5 dark sessions. Interim read:
   context/2026-10-29-ebay-flipper-v3-days-1-4.md.
