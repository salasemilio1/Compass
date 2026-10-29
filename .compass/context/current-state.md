# current state

Date: 2026-10-29
Commit: d28e8eb2c443537eb09f7c120d977ae73110a9f4

## Objective
Make compass notice when Claude Code won't deliver it (the 10-28/29 ebay-flipper outage: a dangling
~/.claude/settings.json symlink turned off every hook and the plugin, silently), and prove a fresh
install works end to end. Spec, plan, critique and evidence: docs/specs/2026-10-29-delivery-health.md
(branch fix/hooks-go-dark, version 0.3.1, decision 0020).

## Next steps
1. Open the PR (pr-intent) and get the e2e CI job green on ubuntu.
2. After merge: `claude plugin marketplace update compass && claude plugin update compass@compass`,
   then in ebay-flipper run `compass init` (AGENTS block refresh) and confirm a headless session
   gets the brief and `compass doctor` shows no delivery problem.
3. Friday 2026-10-30: field-retro over the week with `--former-path ~/Dev/Misc/personal-dashboard`
   (repo moved), judge pass and at most 5 replays; append to experiments/2026-10-26-v3-field-week.md,
   reporting 1a/3a with and without the 5 dark sessions. Interim read:
   context/2026-10-29-ebay-flipper-v3-days-1-4.md.
