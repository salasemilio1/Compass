package compass

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// RecordMaxLines caps a record. A record past it is restated or split, so appends can't grow a
// file without bound.
const RecordMaxLines = 300

// Severity of a finding: errors fail `compass check`, warnings never do.
type Severity string

const (
	SeverityError Severity = "error"
	SeverityWarn  Severity = "warn"
)

// Finding is one problem `compass check` reports.
type Finding struct {
	Severity Severity `json:"severity"`
	// Code is a stable identifier for the rule ("anchor-missing"), for scripts and tests.
	Code    string `json:"code"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
	// Fixable means `compass check --fix` repairs it.
	Fixable bool `json:"fixable,omitempty"`
}

func (f Finding) String() string {
	location := ""
	if f.Path != "" {
		location = f.Path + ": "
	}
	suffix := ""
	if f.Fixable {
		suffix = " (fix: compass check --fix)"
	}
	return fmt.Sprintf("%s: %s%s [%s]%s", f.Severity, location, f.Message, f.Code, suffix)
}

// CheckOptions selects the optional rules.
type CheckOptions struct {
	// RequireFresh makes code commits after current-state's anchor an error instead of a warning.
	// CI sets it on pull requests; pre-commit doesn't, or every code commit would be blocked.
	RequireFresh bool
}

// Check runs every rule and returns findings sorted errors first, then by path.
func Check(repo *Repo, options CheckOptions) ([]Finding, error) {
	if info, err := os.Stat(repo.Dir); err != nil || !info.IsDir() {
		return nil, ErrNoCompass
	}
	docs, err := LoadDocs(repo.Dir)
	if err != nil {
		return nil, err
	}
	var findings []Finding
	add := func(severity Severity, code, path, format string, args ...any) {
		findings = append(findings, Finding{Severity: severity, Code: code, Path: path, Message: fmt.Sprintf(format, args...)})
	}

	// A, B1: INDEX.md is generated, so a stale one is the only way it can be wrong.
	generated, err := GenerateIndex(repo, docs)
	if err != nil {
		return nil, err
	}
	if current, err := os.ReadFile(filepath.Join(repo.Dir, IndexPath)); err != nil || string(current) != generated {
		findings = append(findings, Finding{Severity: SeverityError, Code: "index-stale", Path: ".compass/" + IndexPath,
			Message: "does not match the records (regenerate: compass index)", Fixable: true})
	}

	// Two agents recording at once can both take the next number; `supersedes: 0012` would then
	// be ambiguous.
	numbers := map[string][]string{}
	for _, doc := range docs {
		if number := doc.Number(); doc.Category == "decisions" && number != "" {
			numbers[number] = append(numbers[number], doc.ID)
		}
	}
	for number, ids := range numbers {
		if len(ids) > 1 {
			add(SeverityError, "decision-number-duplicate", ".compass/decisions/", "number %s is used by %s: renumber the newer one", number, strings.Join(ids, " and "))
		}
	}

	git := repo.IsGit()
	for _, doc := range docs {
		path := ".compass/" + doc.Path
		switch {
		case strings.TrimSpace(doc.Claim) == "":
			add(SeverityError, "claim-missing", path, "no `claim:` in frontmatter (one line: what this record holds)")
		case strings.ContainsAny(doc.Claim, "\n\r") || len([]rune(doc.Claim)) > ClaimMaxChars:
			add(SeverityError, "claim-too-long", path, "claim is %d characters, cap %d", len([]rune(doc.Claim)), ClaimMaxChars)
		}
		if doc.Fields["status"] {
			add(SeverityError, "status-handwritten", path, "`status:` is derived from later records' `supersedes:`; remove it")
		}
		if doc.Lines > RecordMaxLines {
			add(SeverityError, "record-too-long", path, "%d lines, cap %d: restate it keeping every fact, or split it", doc.Lines, RecordMaxLines)
		}
		if doc.Scope != "" && doc.Scope != "global" && doc.Scope != "local" {
			add(SeverityError, "scope-invalid", path, "scope %q, expected global or local", doc.Scope)
		}
		for _, target := range doc.Supersedes {
			old := Resolve(docs, target)
			switch {
			case old == nil:
				add(SeverityError, "supersedes-dangling", path, "supersedes %q, which is not a record", target)
			case old == doc:
				add(SeverityError, "supersedes-self", path, "supersedes itself")
			}
		}
		for _, ref := range doc.Refs {
			if message := checkRef(repo, doc, ref, git); message != "" {
				add(SeverityError, "ref-unresolved", path, "%s", message)
			}
		}
	}
	findings = append(findings, checkSupersessionCycles(docs)...)
	findings = append(findings, checkCurrentState(repo, options, git)...)
	findings = append(findings, checkAgentsBlock(repo)...)
	findings = append(findings, checkRepoLayout(repo, git)...)

	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Severity != findings[j].Severity {
			return findings[i].Severity == SeverityError
		}
		return findings[i].Path < findings[j].Path
	})
	return findings, nil
}

// HasErrors reports whether any finding is an error.
func HasErrors(findings []Finding) bool {
	for _, finding := range findings {
		if finding.Severity == SeverityError {
			return true
		}
	}
	return false
}

var commitRef = regexp.MustCompile(`^commit:([0-9a-f]{7,40})$`)

// checkRef validates the refs compass can resolve locally: "commit:<hash>" and paths. Anything
// else with a scheme (https:, mlflow:, issue:) belongs to another tool and is left alone.
func checkRef(repo *Repo, doc *Doc, ref string, git bool) string {
	ref = strings.TrimSpace(ref)
	if m := commitRef.FindStringSubmatch(ref); m != nil {
		if git && !repo.CommitExists(m[1]) {
			return fmt.Sprintf("ref %q: commit not in this repository", ref)
		}
		return ""
	}
	if schemePrefixed.MatchString(ref) {
		return ""
	}
	path, _, _ := strings.Cut(ref, "#")
	candidates := []string{
		filepath.Join(repo.Root, filepath.FromSlash(path)),
		filepath.Join(repo.Dir, filepath.FromSlash(filepath.Dir(doc.Path)), filepath.FromSlash(path)),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return ""
		}
	}
	return fmt.Sprintf("ref %q: no such path (relative to the repo root or to this record)", ref)
}

func checkSupersessionCycles(docs []*Doc) []Finding {
	var findings []Finding
	for _, start := range docs {
		seen := map[*Doc]bool{start: true}
		frontier := []*Doc{start}
		for len(frontier) > 0 {
			doc := frontier[0]
			frontier = frontier[1:]
			for _, target := range doc.Supersedes {
				next := Resolve(docs, target)
				if next == nil || next == doc {
					continue
				}
				if next == start {
					findings = append(findings, Finding{Severity: SeverityError, Code: "supersedes-cycle", Path: ".compass/" + start.Path,
						Message: "is part of a supersedes cycle, so neither side is live"})
					frontier = nil
					break
				}
				if !seen[next] {
					seen[next] = true
					frontier = append(frontier, next)
				}
			}
		}
	}
	return findings
}

func checkCurrentState(repo *Repo, options CheckOptions, git bool) []Finding {
	path := ".compass/" + CurrentStatePath
	state, err := repo.ReadCurrentState()
	if err != nil {
		return []Finding{{Severity: SeverityError, Code: "current-state-unreadable", Path: path, Message: err.Error()}}
	}
	if state == nil {
		return []Finding{{Severity: SeverityWarn, Code: "current-state-missing", Path: path,
			Message: "no current-state.md: a fresh agent has no front door (write one: objective, just done, files in play, next steps)"}}
	}
	var findings []Finding
	if state.Lines > CurrentStateMaxLines {
		findings = append(findings, Finding{Severity: SeverityError, Code: "current-state-too-long", Path: path,
			Message: fmt.Sprintf("%d lines, cap %d: keep only now and next; decisions and results go in records", state.Lines, CurrentStateMaxLines)})
	}
	for _, file := range state.FilesInPlay {
		if _, err := os.Stat(filepath.Join(repo.Root, filepath.FromSlash(file))); err != nil {
			findings = append(findings, Finding{Severity: SeverityError, Code: "file-in-play-missing", Path: path,
				Message: fmt.Sprintf("files in play lists %q, which does not exist", file)})
		}
	}
	// D: the anchor proves when current-state was written.
	switch {
	case !state.HasAnchor || state.Anchor == "":
		return append(findings, Finding{Severity: SeverityError, Code: "anchor-missing", Path: path,
			Message: "no `Commit: <hash>` line", Fixable: git})
	case !state.AnchorWellFormed():
		return append(findings, Finding{Severity: SeverityError, Code: "anchor-malformed", Path: path,
			Message: fmt.Sprintf("Commit: %q is not a commit hash", state.Anchor), Fixable: git})
	}
	if !git {
		return findings
	}
	freshness := repo.Freshness(state)
	if !freshness.AnchorResolves {
		findings = append(findings, Finding{Severity: SeverityWarn, Code: "anchor-unresolved", Path: path,
			Message: fmt.Sprintf("Commit: %s is not in this repository's history (squash merge or rebase?); freshness is measured from the last commit that changed current-state.md", state.Anchor)})
	}
	if freshness.Known && freshness.CodeCommits > 0 {
		severity := SeverityWarn
		if options.RequireFresh {
			severity = SeverityError
		}
		findings = append(findings, Finding{Severity: severity, Code: "anchor-stale", Path: path,
			Message: fmt.Sprintf("%d commit(s) changed code since current-state was written (%s); rewrite it", freshness.CodeCommits, short(freshness.Since))})
	}
	return findings
}

// checkRepoLayout (F) warns when context/overview.md doesn't mention a top-level tracked folder.
// Detection only: the fix is a content edit.
func checkRepoLayout(repo *Repo, git bool) []Finding {
	overview, err := os.ReadFile(filepath.Join(repo.Dir, "context", "overview.md"))
	if err != nil || !git {
		return nil
	}
	out, err := repo.Git("ls-files")
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	var missing []string
	for _, file := range strings.Split(out, "\n") {
		top, _, nested := strings.Cut(file, "/")
		if !nested || seen[top] || strings.HasPrefix(top, ".") {
			continue
		}
		seen[top] = true
		if !strings.Contains(string(overview), "`"+top+"/") {
			missing = append(missing, top)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(missing)
	return []Finding{{Severity: SeverityWarn, Code: "layout-drift", Path: ".compass/context/overview.md",
		Message: "repo layout doesn't mention: " + strings.Join(missing, ", ")}}
}

func short(hash string) string {
	if len(hash) > 7 {
		return hash[:7]
	}
	return hash
}

// Fix repairs what is mechanical: it regenerates INDEX.md and stamps a missing or malformed
// anchor with HEAD. It never stamps a stale anchor: that would claim current-state was rewritten.
// It returns one line per change made.
func Fix(repo *Repo) ([]string, error) {
	var changes []string
	changed, err := WriteIndex(repo)
	if err != nil {
		return nil, err
	}
	if changed {
		changes = append(changes, "regenerated .compass/"+IndexPath)
	}
	state, err := repo.ReadCurrentState()
	if err != nil || state == nil {
		return changes, err
	}
	if state.HasAnchor && state.AnchorWellFormed() {
		return changes, nil
	}
	head := repo.Head()
	if head == "" {
		return changes, nil
	}
	path := filepath.Join(repo.Dir, filepath.FromSlash(CurrentStatePath))
	if err := writeFileAtomic(path, []byte(StampAnchor(state.Text, head))); err != nil {
		return changes, err
	}
	return append(changes, "stamped .compass/"+CurrentStatePath+" Commit: "+short(head)), nil
}
