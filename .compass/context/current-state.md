# current state

Date: 2026-10-26
Commit: 5accbe7832b0d68bb660b9470656f8d26aa9c960

## Objective
Make compass v3 trustworthy enough to rely on during a week of real work on ebay-flipper
(2026-10-26 to 2026-10-30), then judge it from that week's transcripts on Friday.
Decision 0019's search line now reaches Claude Code too (CLAUDE.md pointer and both brief forms);
before this it lived only in AGENTS.md, which Claude Code doesn't load.

## Next steps
1. Signal test with Claude sessions (`claude -p`, not Codex): legacy pointer vs new pointer on a
   copy of ebay-flipper with buried decisions.
2. Merge, tag `v0.3.0`, install the plugin, remove the Python `compass-cli` and v2 globals,
   migrate ebay-flipper.
3. Commit the field-retro protocol and script before the week starts; run it on 2026-10-30.
