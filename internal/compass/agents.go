package compass

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	agentsBegin = "<!-- compass:begin (managed by `compass init`; edits inside are overwritten) -->"
	agentsEnd   = "<!-- compass:end -->"
)

// AgentsBlock is everything compass writes into AGENTS.md. It is the baseline for agents without
// hooks: they learn to run `compass resume` from here.
const AgentsBlock = agentsBegin + `
## Project memory (compass)
- **Start:** run ` + "`compass resume`" + ` before anything else. It prints where the work stands
  (` + "`.compass/context/current-state.md`" + `), the live project-wide decisions, and uncommitted work.
  Don't rediscover what it already tells you; ` + "`.compass/INDEX.md`" + ` maps the rest.
- **Rewrite current-state** (whole, never patched, at most 80 lines: objective, just done, files in
  play, next steps) after a commit, before a long step, and before you stop. Set its
  ` + "`Commit:`" + ` line to the output of ` + "`git rev-parse HEAD`" + `; never type a hash from memory.
  Usage limits hit mid-task: the next agent resumes from it.
- **Record a decision when it happens:** ` + "`compass record decision --claim \"...\" --premise \"...\"`" + `.
  Records are never edited; a later record supersedes one (` + "`--supersedes <id>`" + `).
- **Decisions are the current best answer, not law.** To change one, propose it and address its
  premises.
- **Comment the why** in code; compass holds what code and git can't.
- Run ` + "`compass check`" + ` before committing.
` + agentsEnd

var agentsBlockPattern = regexp.MustCompile(`(?s)<!-- compass:begin[^>]*-->.*?<!-- compass:end -->`)

// v2 stamped AGENTS.md with this marker and its own instructions.
var legacyAgentsMarker = regexp.MustCompile(`<!-- compass-spec-version: [0-9]+ -->`)

// UpsertAgentsBlock writes or refreshes the managed block in AGENTS.md, leaving the rest of the
// file as the user wrote it. It reports whether the file changed.
func UpsertAgentsBlock(root string) (bool, error) {
	path := filepath.Join(root, "AGENTS.md")
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	old := string(data)
	var updated string
	switch {
	case agentsBlockPattern.MatchString(old):
		updated = agentsBlockPattern.ReplaceAllLiteralString(old, AgentsBlock)
	case old == "":
		updated = "# Agent instructions\n\n" + AgentsBlock + "\n"
	default:
		updated = strings.TrimRight(old, "\n") + "\n\n" + AgentsBlock + "\n"
	}
	if updated == old {
		return false, nil
	}
	return true, writeFileAtomic(path, []byte(updated))
}

// claudePointer imports AGENTS.md into Claude Code's memory, so the block reaches Claude even
// without the plugin.
const claudePointer = "@AGENTS.md"

// EnsureClaudePointer makes CLAUDE.md import AGENTS.md, unless it already mentions AGENTS.md.
func EnsureClaudePointer(root string) (bool, error) {
	path := filepath.Join(root, "CLAUDE.md")
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	old := string(data)
	if strings.Contains(old, "AGENTS.md") {
		return false, nil
	}
	updated := "# CLAUDE.md\n\n" + claudePointer + "\n"
	if old != "" {
		updated = strings.TrimRight(old, "\n") + "\n\n" + claudePointer + "\n"
	}
	return true, writeFileAtomic(path, []byte(updated))
}

// checkAgentsBlock (E) warns when AGENTS.md lacks the block or carries an older one. Detection
// only, so a user's hand edit never fails a commit.
func checkAgentsBlock(repo *Repo) []Finding {
	data, err := os.ReadFile(filepath.Join(repo.Root, "AGENTS.md"))
	text := string(data)
	message := ""
	switch {
	case err != nil || (!agentsBlockPattern.MatchString(text) && !legacyAgentsMarker.MatchString(text)):
		message = "no compass block, so agents without hooks never learn to run `compass resume`"
	case !agentsBlockPattern.MatchString(text):
		message = "carries the v2 compass instructions"
	case agentsBlockPattern.FindString(text) != AgentsBlock:
		message = "compass block differs from this version's"
	default:
		return nil
	}
	return []Finding{{Severity: SeverityWarn, Code: "agents-block-drift", Path: "AGENTS.md",
		Message: message + " (refresh: compass init)"}}
}
