package compass

import (
	"fmt"
	"strings"
)

const (
	resumeMaxClaims      = 30
	resumeMaxStatusLines = 40
	resumeMaxDiffLines   = 30
	resumeMaxCommits     = 15
)

// Brief is what a fresh agent needs and cannot cheaply derive: where the work stands, the live
// project-wide decisions, and the uncommitted work. Sections are kept apart so the JSON output
// and the text output share one source.
type Brief struct {
	Project      string   `json:"project"`
	Branch       string   `json:"branch,omitempty"`
	Head         string   `json:"head,omitempty"`
	CurrentState string   `json:"current_state"`
	Freshness    string   `json:"freshness,omitempty"`
	CommitsSince []string `json:"commits_since,omitempty"`
	Claims       []string `json:"global_claims"`
	Status       []string `json:"status,omitempty"`
	DiffStat     []string `json:"diff_stat,omitempty"`
	Records      int      `json:"records"`
	Unparseable  int      `json:"unparseable_records,omitempty"`
}

// Resume gathers the brief. Git sections are empty outside a git repository.
func Resume(repo *Repo) (*Brief, error) {
	brief := &Brief{Project: repo.Name()}
	docs, problems, err := LoadDocs(repo.Dir)
	if err != nil {
		return nil, err
	}
	brief.Records = len(docs)
	brief.Unparseable = len(problems)
	var global []*Doc
	for _, doc := range docs {
		if doc.Live() && doc.Global() {
			global = append(global, doc)
		}
	}
	sortNewestFirst(global)
	for i, doc := range global {
		if i == resumeMaxClaims {
			brief.Claims = append(brief.Claims, fmt.Sprintf("… %d more: rg -l '^scope: global' .compass/", len(global)-i))
			break
		}
		brief.Claims = append(brief.Claims, fmt.Sprintf("%s — %s (.compass/%s)", doc.ID, truncate(oneLine(doc.Claim), ClaimMaxChars), doc.Path))
	}

	state, err := repo.ReadCurrentState()
	if err != nil {
		return nil, err
	}
	if state != nil {
		brief.CurrentState = strings.TrimRight(state.Text, "\n")
	}
	if !repo.IsGit() {
		return brief, nil
	}
	brief.Branch, _ = repo.Git("branch", "--show-current")
	brief.Head, _ = repo.Git("log", "-1", "--format=%h %s")
	if state != nil {
		freshness := repo.Freshness(state)
		switch {
		case !freshness.Known:
			brief.Freshness = "unknown (anchor missing or not in history, and current-state.md never committed)"
		case freshness.CodeCommits == 0:
			brief.Freshness = "current: no code commits since it was written"
		default:
			brief.Freshness = fmt.Sprintf("STALE: %d code commit(s) since it was written; trust the code and git log over it, and rewrite it", freshness.CodeCommits)
		}
		if freshness.Known {
			if out, err := repo.Git("log", "--format=%h %s", fmt.Sprintf("-%d", resumeMaxCommits+1), freshness.Since+"..HEAD"); err == nil && out != "" {
				brief.CommitsSince = capLines(strings.Split(out, "\n"), resumeMaxCommits, "git log "+short(freshness.Since)+"..HEAD")
			}
		}
	}
	if out, err := repo.Git("status", "--short", "--untracked-files=normal"); err == nil && out != "" {
		brief.Status = capLines(strings.Split(out, "\n"), resumeMaxStatusLines, "git status")
	}
	if repo.Head() != "" {
		if out, err := repo.Git("diff", "--stat=100", "HEAD"); err == nil && out != "" {
			lines := strings.Split(out, "\n")
			// Keep the "N files changed" summary even when the per-file lines are cut.
			if len(lines) > resumeMaxDiffLines+1 {
				summary := lines[len(lines)-1]
				lines = append(capLines(lines[:len(lines)-1], resumeMaxDiffLines, "git diff --stat HEAD"), summary)
			}
			brief.DiffStat = lines
		}
	}
	return brief, nil
}

func capLines(lines []string, max int, more string) []string {
	if len(lines) <= max {
		return lines
	}
	return append(lines[:max:max], fmt.Sprintf("… %d more lines: %s", len(lines)-max, more))
}

// Text renders the brief as markdown. maxChars > 0 bounds the output (Claude Code shows the model
// only a preview of SessionStart context over 10,000 characters); current-state is cut first,
// from the middle out, keeping its head and its tail where the next steps live.
func (b *Brief) Text(maxChars int) string {
	var head, tail strings.Builder
	fmt.Fprintf(&head, "# compass resume — %s", b.Project)
	if b.Branch != "" {
		fmt.Fprintf(&head, " (branch %s)", b.Branch)
	}
	head.WriteString("\nThis brief already contains current-state.md and the live global records: don't re-read them. .compass/INDEX.md is for lookup, not for reading at start.\n")
	if b.Head != "" {
		fmt.Fprintf(&head, "HEAD: %s\n", b.Head)
	}
	head.WriteString("\n## Current state (.compass/context/current-state.md)\n")
	if b.Freshness != "" {
		fmt.Fprintf(&head, "Freshness: %s\n", b.Freshness)
	}
	head.WriteString("\n")

	writeSection := func(title string, lines []string, empty string) {
		fmt.Fprintf(&tail, "\n## %s\n", title)
		if len(lines) == 0 {
			tail.WriteString(empty + "\n")
			return
		}
		tail.WriteString(strings.Join(lines, "\n") + "\n")
	}
	if len(b.CommitsSince) > 0 {
		writeSection("Commits since current-state was written", b.CommitsSince, "")
	}
	claims := make([]string, len(b.Claims))
	for i, claim := range b.Claims {
		claims[i] = "- " + claim
	}
	writeSection("Live global records (decisions and goals with scope: global)", claims, "(none recorded with scope: global)")
	if b.Branch != "" || b.Head != "" {
		writeSection("Uncommitted work (git status)", b.Status, "(clean)")
		if len(b.DiffStat) > 0 {
			writeSection("Diff stat vs HEAD", b.DiffStat, "")
		}
	}
	fmt.Fprintf(&tail, "\nMap of all %d records: .compass/INDEX.md\n", b.Records)
	if b.Unparseable > 0 {
		fmt.Fprintf(&tail, "%d record(s) don't parse and were skipped: run `compass check`\n", b.Unparseable)
	}

	state := b.CurrentState
	if state == "" {
		state = "(no current-state.md yet: write one before you stop)"
	}
	if maxChars > 0 {
		room := maxChars - head.Len() - tail.Len()
		if room < 200 {
			room = 200
		}
		state = cutMiddle(state, room, "… current-state cut here to fit; read .compass/context/current-state.md for all of it …")
	}
	out := head.String() + state + "\n" + tail.String()
	if maxChars > 0 && len(out) > maxChars {
		out = strings.ToValidUTF8(out[:maxChars-3], "") + "…"
	}
	return out
}

// cutMiddle keeps the start and end of text, by whole lines, within max bytes.
func cutMiddle(text string, max int, marker string) string {
	if len(text) <= max {
		return text
	}
	lines := strings.Split(text, "\n")
	budget := max - len(marker) - 2
	var front, back []string
	used := 0
	for i, j := 0, len(lines)-1; i <= j; {
		// Alternate, two lines of head for every line of tail: the objective comes first.
		if len(front) <= 2*len(back) {
			if used+len(lines[i])+1 > budget {
				break
			}
			used += len(lines[i]) + 1
			front = append(front, lines[i])
			i++
		} else {
			if used+len(lines[j])+1 > budget {
				break
			}
			used += len(lines[j]) + 1
			back = append([]string{lines[j]}, back...)
			j--
		}
	}
	return strings.Join(front, "\n") + "\n" + marker + "\n" + strings.Join(back, "\n")
}
