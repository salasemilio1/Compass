compass's `.compass/`, `AGENTS.md`, and `SPEC.md` are plain git-tracked files, so they're agent-agnostic
by construction — any coding agent that can read the repo (Cursor, etc.) gets them for free, no porting
step needed. What does NOT port: `skills/*/SKILL.md` (Claude Code's Skill mechanism — content is
portable as prose, but not as an invokable command), `hooks/*` registered via `~/.claude/settings.json`
(a Claude-Code-only hook system), and `tools/session-context/*.py` (read Claude Code's own transcript
JSONL format, so they have no data under a different agent). When moving compass-adopted work to a
non-Claude-Code agent, only the skill *content* and hook *intent* need manual translation — the
knowledge itself already travels with the repo.
