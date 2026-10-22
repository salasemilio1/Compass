package compass

import (
	"fmt"
	"regexp"
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
// only a preview of SessionStart context over 10,000 characters); current-state is cut first, by
// section priority (cutStateSections), and line 2 then says the brief holds only part of it.
func (b *Brief) Text(maxChars int) string {
	writeHead := func(cut bool) string {
		var head strings.Builder
		fmt.Fprintf(&head, "# compass resume — %s", b.Project)
		if b.Branch != "" {
			fmt.Fprintf(&head, " (branch %s)", b.Branch)
		}
		if cut {
			head.WriteString("\nThis brief contains current-state.md truncated to fit (read .compass/context/current-state.md for the cut parts) and the live global records: don't re-read the records. .compass/INDEX.md is for lookup, not for reading at start.\n")
		} else {
			head.WriteString("\nThis brief already contains current-state.md and the live global records: don't re-read them. .compass/INDEX.md is for lookup, not for reading at start.\n")
		}
		if b.Head != "" {
			fmt.Fprintf(&head, "HEAD: %s\n", b.Head)
		}
		head.WriteString("\n## Current state (.compass/context/current-state.md)\n")
		if b.Freshness != "" {
			fmt.Fprintf(&head, "Freshness: %s\n", b.Freshness)
		}
		head.WriteString("\n")
		return head.String()
	}
	var tail strings.Builder
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
	head := writeHead(false)
	if maxChars > 0 {
		// The -1 is the newline joining state to the tail.
		if room := maxChars - len(head) - tail.Len() - 1; len(state) > room {
			head = writeHead(true)
			room = max(maxChars-len(head)-tail.Len()-1, 0)
			state = cutStateSections(state, room)
		}
	}
	out := head + state + "\n" + tail.String()
	if maxChars > 0 && len(out) > maxChars {
		out = strings.ToValidUTF8(out[:maxChars-3], "") + "…"
	}
	return out
}

// stateSectionRanks orders current-state sections by which to trim first: history and issues
// go before files in play, unrecognised sections next, then the objective; the last next steps
// section is kept.
// Titles match case-insensitively at H2 or H3, by prefix ("## Next steps (in order)").
var stateSectionRanks = []struct {
	heading *regexp.Regexp
	rank    int
}{
	{regexp.MustCompile(`(?i)^#{2,3}\s+just done\b`), 0},
	{regexp.MustCompile(`(?i)^#{2,3}\s+known issues\b`), 1},
	{regexp.MustCompile(`(?i)^#{2,3}\s+files in play\b`), 2},
	{regexp.MustCompile(`(?i)^#{2,3}\s+objectives?\b`), 4},
	{nextSteps, rankNextSteps},
}

var stateHeading = regexp.MustCompile(`^#{2,3}\s`)

const (
	rankPreamble  = -1 // title, Date:, Commit:
	rankUnknown   = 3
	rankObjective = 4
	rankNextSteps = 5
)

type stateSection struct {
	lines   []string // lines[0] is the heading, except in the preamble
	keep    int      // lines kept, from the top
	rank    int
	dropped bool // the whole section, heading included, is cut
}

func cutLinesMarker(count int) string { return fmt.Sprintf("… %d lines cut", count) }

// cutStateSections fits current-state into max bytes, cutting the least useful text first:
// the bottoms of Just done, Known issues, Files in play and unrecognised sections, then those
// sections whole, then the bottom of the objective, then everything but the next steps from the
// middle. Next steps are cut only when they alone exceed max. Text with no recognised heading
// falls back to cutMiddle.
func cutStateSections(text string, max int) string {
	if len(text) <= max {
		return text
	}
	sections := []*stateSection{{rank: rankPreamble}}
	known := false
	lines := strings.Split(text, "\n")
	fenced := fencedLines(lines)
	for i, line := range lines {
		if !fenced[i] && stateHeading.MatchString(line) {
			rank := rankUnknown
			for _, candidate := range stateSectionRanks {
				if candidate.heading.MatchString(line) {
					rank, known = candidate.rank, true
					break
				}
			}
			sections = append(sections, &stateSection{rank: rank})
		}
		last := sections[len(sections)-1]
		last.lines = append(last.lines, line)
		last.keep++
	}
	// Only the last next-steps section is protected: an earlier one is an archive or a draft, and
	// keeping it whole could starve the live list.
	last := true
	for i := len(sections) - 1; i >= 0; i-- {
		if sections[i].rank == rankNextSteps {
			if !last {
				sections[i].rank = rankUnknown
			}
			last = false
		}
	}
	if !known {
		return fitBytes(cutMiddle(text, max, "… current-state cut here to fit …"), max)
	}
	render := func(include func(*stateSection) bool) string {
		var out []string
		droppedLines, markerAt := 0, -1
		for _, section := range sections {
			if !include(section) {
				continue
			}
			if section.dropped {
				if markerAt < 0 {
					markerAt = len(out)
					out = append(out, "")
				}
				droppedLines += len(section.lines)
				continue
			}
			out = append(out, section.lines[:section.keep]...)
			if cut := len(section.lines) - section.keep; cut > 0 {
				out = append(out, cutLinesMarker(cut))
			}
		}
		if markerAt >= 0 {
			out[markerAt] = cutLinesMarker(droppedLines)
		}
		return strings.Join(out, "\n")
	}
	all := func(*stateSection) bool { return true }
	size := len(text)
	// trimBody drops the fewest bottom lines that fit, or the whole body when no partial cut
	// does, and only when the cut saves more than its marker costs.
	trimBody := func(section *stateSection) {
		saved := 0
		for count := 1; count < len(section.lines); count++ {
			saved += len(section.lines[len(section.lines)-count]) + 1
			net := saved - len(cutLinesMarker(count)) - 1
			if size-net <= max || (count == len(section.lines)-1 && net > 0) {
				section.keep = len(section.lines) - count
				size = len(render(all))
				return
			}
		}
	}
	for rank := 0; rank <= rankUnknown && size > max; rank++ {
		for _, section := range sections {
			if section.rank == rank && size > max {
				trimBody(section)
			}
		}
	}
	for rank := 0; rank <= rankUnknown && size > max; rank++ {
		for _, section := range sections {
			if section.rank == rank && size > max {
				section.dropped = true
				size = len(render(all))
			}
		}
	}
	for _, section := range sections {
		if section.rank == rankObjective && size > max {
			trimBody(section)
		}
	}
	if size <= max {
		return render(all)
	}
	// Last resort: keep the next steps whole and fit everything else around them.
	next := render(func(section *stateSection) bool { return section.rank == rankNextSteps })
	rest := render(func(section *stateSection) bool { return section.rank != rankNextSteps })
	marker := "… current-state cut here to fit …"
	room := max - len(next) - 1
	if next == "" {
		room = max
	}
	if room < len(marker)+2 {
		return fitBytes(next, max)
	}
	rest = cutMiddle(rest, room, marker)
	if next == "" {
		return fitBytes(rest, max)
	}
	return fitBytes(rest+"\n"+next, max)
}

// fitBytes cuts text to at most max bytes without splitting a UTF-8 sequence.
func fitBytes(text string, max int) string {
	if len(text) <= max {
		return text
	}
	return strings.ToValidUTF8(text[:max], "")
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
