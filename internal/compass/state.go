package compass

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// CurrentStateMaxLines caps current-state.md: it is the hot window, rewritten whole at each
// boundary, and it is delivered into every session.
const CurrentStateMaxLines = 80

var (
	anchorLine     = regexp.MustCompile(`^Commit:\s*(\S*)\s*$`)
	anchorHash     = regexp.MustCompile(`^[0-9a-f]{7,40}$`)
	nextSteps      = regexp.MustCompile(`(?i)^#{2,3}\s+next steps\b`)
	anyHeading     = regexp.MustCompile(`^#{1,3}\s`)
	schemePrefixed = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)
	// placeholder is a line holding only a template's `<...>`, optionally as a list item: an
	// unfilled template must not pass for a written objective or next step.
	placeholder = regexp.MustCompile(`^\s*(?:(?:[-*+]|\d+[.)])\s+)?<[^<>:@]*>\s*$`)
)

// CurrentState is the parsed front door.
type CurrentState struct {
	Text  string
	Lines int
	// Anchor is the Commit: value as written ("" when the line is missing).
	Anchor    string
	HasAnchor bool
	// HasNextSteps is true when a "## Next steps" heading has at least one line under it that is
	// neither blank nor a template placeholder.
	HasNextSteps bool
}

// ReadCurrentState returns nil, nil when the file does not exist.
func (r *Repo) ReadCurrentState() (*CurrentState, error) {
	data, err := os.ReadFile(filepath.Join(r.Dir, filepath.FromSlash(CurrentStatePath)))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return ParseCurrentState(string(data)), nil
}

// ParseCurrentState extracts the anchor and whether next steps are written. Headings inside ```
// or ~~~ fences are content, not headings.
func ParseCurrentState(text string) *CurrentState {
	state := &CurrentState{Text: text}
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	state.Lines = len(lines)
	fenced := fencedLines(lines)
	inNext := false
	for i, line := range lines {
		inFence := fenced[i]
		if m := anchorLine.FindStringSubmatch(line); m != nil && !state.HasAnchor {
			state.HasAnchor = true
			state.Anchor = m[1]
			continue
		}
		if !inFence && nextSteps.MatchString(line) {
			inNext = true
			continue
		}
		if !inFence && anyHeading.MatchString(line) {
			inNext = false
			continue
		}
		if inNext && strings.TrimSpace(line) != "" && fenceMarker(line) == "" && !placeholder.MatchString(line) {
			state.HasNextSteps = true
		}
	}
	return state
}

// fenceMarker returns "```" or "~~~" when line opens or closes a fenced code block, else "".
func fenceMarker(line string) string {
	trimmed := strings.TrimSpace(line)
	for _, marker := range []string{"```", "~~~"} {
		if strings.HasPrefix(trimmed, marker) {
			return marker
		}
	}
	return ""
}

// fencedLines marks the lines inside fenced code blocks, fence lines included, so a `##` there
// isn't taken for a heading. A fence closes only with its own marker; when one is still open at
// the end, the fences are ignored rather than hiding every heading after a typo.
func fencedLines(lines []string) []bool {
	fenced := make([]bool, len(lines))
	open := ""
	for i, line := range lines {
		marker := fenceMarker(line)
		opens := open == "" && marker != ""
		if opens {
			open = marker
		}
		fenced[i] = open != ""
		if !opens && marker == open {
			open = ""
		}
	}
	if open != "" {
		return make([]bool, len(lines))
	}
	return fenced
}

// AnchorWellFormed reports whether the Commit: value looks like an abbreviated or full hash.
func (s *CurrentState) AnchorWellFormed() bool { return anchorHash.MatchString(s.Anchor) }

// Freshness says how far current-state.md lags the code.
type Freshness struct {
	// Since is the commit freshness is measured from: the anchor, or the last commit that changed
	// current-state.md when that is later (an agent that commits code and current-state together
	// stamps the parent, and that commit must not count against it).
	Since string
	// CodeCommits counts commits after Since that change anything outside .compass/.
	CodeCommits int
	// Known is false when no base could be established (no git, unresolvable anchor and the file
	// never committed).
	Known bool
	// StateDirty is true when current-state.md has uncommitted changes, so its last commit can't
	// stand in for an anchor outside HEAD's history.
	StateDirty bool
	// AnchorResolves is false when the anchor is not in HEAD's history, which is what a squash
	// merge or rebase does to a branch commit.
	AnchorResolves bool
}

// Freshness measures current-state.md against HEAD. It needs git; without it Known is false.
func (r *Repo) Freshness(state *CurrentState) Freshness {
	var result Freshness
	if state == nil || !r.IsGit() || r.Head() == "" {
		return result
	}
	since := ""
	// An anchor left unreachable by a squash merge or rebase still exists as an object, so
	// existence isn't enough: it must be in HEAD's history to measure from.
	if state.AnchorWellFormed() && r.LookupCommit(state.Anchor) == CommitFound && r.isAncestor(state.Anchor, "HEAD") {
		result.AnchorResolves = true
		since = state.Anchor
	}
	statePath := ".compass/" + CurrentStatePath
	dirty, _ := r.Git("status", "--porcelain", "--", statePath)
	result.StateDirty = dirty != ""
	if dirty == "" {
		if last, err := r.Git("log", "-1", "--format=%H", "--", statePath); err == nil && last != "" {
			if since == "" || r.isAncestor(since, last) {
				since = last
			}
		}
	}
	if since == "" {
		return result
	}
	out, err := r.Git("rev-list", "--count", since+"..HEAD", "--", ".", ":(exclude).compass")
	if err != nil {
		return result
	}
	count, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		return result
	}
	result.Since, result.CodeCommits, result.Known = since, count, true
	return result
}

func (r *Repo) isAncestor(ancestor, descendant string) bool {
	_, err := r.Git("merge-base", "--is-ancestor", ancestor, descendant)
	return err == nil
}

// StampAnchor rewrites the Commit: line to hash, inserting it after the Date: line (or at the
// top) when missing. It is the only edit compass makes to current-state.md.
func StampAnchor(text, hash string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if anchorLine.MatchString(line) {
			lines[i] = "Commit: " + hash
			return strings.Join(lines, "\n")
		}
	}
	for i, line := range lines {
		if legacyDate.MatchString(line) {
			out := append([]string{}, lines[:i+1]...)
			out = append(out, "Commit: "+hash)
			return strings.Join(append(out, lines[i+1:]...), "\n")
		}
	}
	return "Commit: " + hash + "\n" + text
}
