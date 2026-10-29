# current state

Date: 2026-10-29
Commit: 1a02fb640690095fd6a117def07e32f9ee357988

## Objective
Make compass notice when Claude Code won't deliver it (the 10-28/29 ebay-flipper outage: a dangling
~/.claude/settings.json symlink turned off every hook and the plugin, silently), and prove a fresh
install works end to end. Spec and plan: docs/specs/2026-10-29-delivery-health.md (branch
fix/hooks-go-dark). Field week on ebay-flipper continues to 2026-10-30.

## Next steps
1. Fold the plan critique into docs/specs/2026-10-29-delivery-health.md, then implement:
   internal/compass/delivery.go (read-only checks), doctor/resume/pre-commit callers, AGENTS block
   line, version 0.3.1, tests/e2e.sh + CI job (claude CLI 2.1.300).
2. Verify (unit, smoke, e2e, break pass), open the PR with pr-intent.
3. Friday 2026-10-30: field-retro over the week with `--former-path ~/Dev/Misc/personal-dashboard`
   (repo moved), judge pass and at most 5 replays; append to experiments/2026-10-26-v3-field-week.md,
   reporting 1a/3a with and without the 5 dark sessions. Interim read:
   context/2026-10-29-ebay-flipper-v3-days-1-4.md.
