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
	filesInPlay    = regexp.MustCompile(`(?i)^#{2,3}\s+files in play\s*$`)
	codeSpan       = regexp.MustCompile("`([^`]+)`")
	markdownLink   = regexp.MustCompile(`\]\(([^)\s]+)\)`)
	anyHeading     = regexp.MustCompile(`^#{1,3}\s`)
	schemePrefixed = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)
)

// CurrentState is the parsed front door.
type CurrentState struct {
	Text  string
	Lines int
	// Anchor is the Commit: value as written ("" when the line is missing).
	Anchor      string
	HasAnchor   bool
	FilesInPlay []string
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

// ParseCurrentState extracts the anchor and the paths listed under "## Files in play". A path
// is the first `code span` or link target on a list item; globs and URLs are skipped.
func ParseCurrentState(text string) *CurrentState {
	state := &CurrentState{Text: text}
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	state.Lines = len(lines)
	inFiles := false
	for _, line := range lines {
		if m := anchorLine.FindStringSubmatch(line); m != nil && !state.HasAnchor {
			state.HasAnchor = true
			state.Anchor = m[1]
			continue
		}
		if filesInPlay.MatchString(line) {
			inFiles = true
			continue
		}
		if anyHeading.MatchString(line) {
			inFiles = false
			continue
		}
		trimmed := strings.TrimSpace(line)
		if !inFiles || !(strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ")) {
			continue
		}
		path := ""
		if m := codeSpan.FindStringSubmatch(trimmed); m != nil {
			path = m[1]
		} else if m := markdownLink.FindStringSubmatch(trimmed); m != nil {
			path = m[1]
		}
		path = strings.TrimSpace(path)
		if path == "" || strings.ContainsAny(path, "*?[ ") || schemePrefixed.MatchString(path) {
			continue
		}
		state.FilesInPlay = append(state.FilesInPlay, path)
	}
	return state
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
	// AnchorResolves is false when the anchor is not in this repository's history, which is what
	// a squash merge or rebase does to a branch commit.
	AnchorResolves bool
}

// Freshness measures current-state.md against HEAD. It needs git; without it Known is false.
func (r *Repo) Freshness(state *CurrentState) Freshness {
	var result Freshness
	if state == nil || !r.IsGit() || r.Head() == "" {
		return result
	}
	since := ""
	if state.AnchorWellFormed() && r.CommitExists(state.Anchor) {
		result.AnchorResolves = true
		since = state.Anchor
	}
	statePath := ".compass/" + CurrentStatePath
	dirty, _ := r.Git("status", "--porcelain", "--", statePath)
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
