Claude Code's per-project session directory (`~/.claude/projects/<slug>/<session_id>.jsonl`) derives
`<slug>` by replacing *every* non-alphanumeric character in the absolute project path with `-`, not
just `/`. A path like `/Users/emilio.kungfu/Dev/compass` becomes `-Users-emilio-kungfu-Dev-compass` —
the dot in the username also becomes a dash. Reconstructing this path from `$CLAUDE_PROJECT_DIR` (e.g.
for a standalone CLI tool that isn't handed `transcript_path` directly by a hook) needs
`re.sub(r"[^A-Za-z0-9]", "-", path)`, not a plain `.replace("/", "-")` — the latter silently produces a
non-existent path whenever the project path contains any other punctuation.
