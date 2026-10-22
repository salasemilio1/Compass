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
	docs, findings, err := LoadDocs(repo.Dir)
	if err != nil {
		return nil, err
	}
	add := func(severity Severity, code, path, format string, args ...any) {
		findings = append(findings, Finding{Severity: severity, Code: code, Path: path, Message: fmt.Sprintf(format, args...)})
	}

	// A, B1: INDEX.md is generated, so a stale one is the only way it can be wrong.
	generated, err := GenerateIndex(repo, docs)
	if err != nil {
		return nil, err
	}
	if current, err := os.ReadFile(filepath.Join(repo.Dir, IndexPath)); err != nil || string(current) != generated {
		// A warning: the index is generated, pre-commit rewrites it, and a merge of two branches
		// that both added records leaves it stale through no one's fault.
		findings = append(findings, Finding{Severity: SeverityWarn, Code: "index-stale", Path: ".compass/" + IndexPath,
			Message: "does not match the records (regenerate: compass index)", Fixable: true})
	}

	// Parallel branches each take the next decision number, so a merge can hold two 0013s. That is
	// legal (records are never renamed); only a bare `supersedes: 0013` becomes ambiguous.
	numbers := map[string][]string{}
	for _, doc := range docs {
		if number := doc.Number(); number != "" {
			numbers[number] = append(numbers[number], doc.ID)
		}
	}
	for number, ids := range numbers {
		if len(ids) > 1 {
			sort.Strings(ids)
			add(SeverityWarn, "decision-number-duplicate", ".compass/decisions/", "number %s is used by %s (parallel branches?); refer to them by full ID", number, strings.Join(ids, " and "))
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
			case len(numbers[strings.TrimSpace(target)]) > 1:
				add(SeverityError, "supersedes-ambiguous", path, "supersedes %q, which names %s: use the full ID", target, strings.Join(numbers[strings.TrimSpace(target)], " and "))
			case old == nil && doc.LegacySupersedes:
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
	for _, candidate := range []string{path, ".compass/" + path, ".compass/" + filepath.ToSlash(filepath.Dir(doc.Path)) + "/" + path} {
		if repo.exists(candidate) {
			return ""
		}
	}
	return fmt.Sprintf("ref %q: no such path (relative to the repo root, .compass/, or this record)", ref)
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
			Message: "no current-state.md: a fresh agent has no front door (write one: objective, next steps)"}}
	}
	var findings []Finding
	if state.Lines > CurrentStateMaxLines {
		findings = append(findings, Finding{Severity: SeverityError, Code: "current-state-too-long", Path: path,
			Message: fmt.Sprintf("%d lines, cap %d: keep only now and next; decisions and results go in records", state.Lines, CurrentStateMaxLines)})
	}
	if !state.HasNextSteps {
		findings = append(findings, Finding{Severity: SeverityWarn, Code: "next-steps-missing", Path: path,
			Message: "no `## Next steps` with content: the next agent won't know what to do first"})
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
	// A real commit outside HEAD's history is a rebase or squash merge: routine when the fallback
	// measures freshness, so it warns only when freshness is unknown.
	if !freshness.AnchorResolves {
		problem, fixable := "", false
		switch repo.LookupCommit(state.Anchor) {
		case CommitMissing:
			problem, fixable = "names no commit (an invented hash: set it to `git rev-parse HEAD`)", true
		case CommitMissingShallow:
			problem = "is not in this shallow clone (fetch the full history: `git fetch --unshallow`)"
		case CommitAmbiguous:
			problem = "matches several commits (write the full hash)"
		default:
			if !freshness.Known {
				problem = "is not in HEAD's history (squash merge or rebase)"
			}
		}
		if problem != "" {
			message := fmt.Sprintf("Commit: %s %s", state.Anchor, problem)
			severity := SeverityWarn
			switch {
			case freshness.Known:
				message += fmt.Sprintf("; freshness is measured from the last commit that changed current-state.md (%s)", short(freshness.Since))
			case freshness.StateDirty:
				message += "; current-state.md has uncommitted changes, so its last commit can't stand in: freshness is unknown"
			default:
				message += "; no commit of current-state.md can stand in: freshness is unknown"
			}
			if !freshness.Known && options.RequireFresh {
				severity = SeverityError
			}
			findings = append(findings, Finding{Severity: severity, Code: "anchor-unresolved", Path: path,
				Message: message, Fixable: fixable})
		}
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

func short(hash string) string {
	if len(hash) > 7 {
		return hash[:7]
	}
	return hash
}

// Fix repairs what is mechanical: it regenerates INDEX.md and stamps HEAD into an anchor that is
// missing, malformed, or names no commit at all (agents have been seen inventing hashes). It never
// restamps a real but stale anchor: that would claim current-state was rewritten. An ambiguous
// short hash, or one a shallow clone may simply lack, is left alone for the same reason.
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
	if state.HasAnchor && state.AnchorWellFormed() && (!repo.IsGit() || repo.LookupCommit(state.Anchor) != CommitMissing) {
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
