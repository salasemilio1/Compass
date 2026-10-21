package compass

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// RecordKinds maps `compass record <kind>` to the category it writes into.
var RecordKinds = map[string]string{
	"decision":   "decisions",
	"goal":       "goals",
	"experiment": "experiments",
}

var bodySkeletons = map[string]string{
	"decision":   "## Context\n\n## Decision\n\n## Alternatives\n\n## Consequences\n",
	"goal":       "## Goal\n\n## Why\n",
	"experiment": "## Setup\n\n## Results\n\n## Conclusion\n",
}

// RecordRequest is one `compass record` call.
type RecordRequest struct {
	Kind       string
	Claim      string
	Slug       string
	Date       string
	Global     bool
	Supersedes []string
	Premises   []string
	Refs       []string
	// Body replaces the skeleton when non-empty.
	Body string
}

// RecordError is a request compass refuses before writing anything.
type RecordError struct{ Reason string }

func (e *RecordError) Error() string { return e.Reason }

var slugUnsafe = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify turns a claim into a short kebab-case filename part.
func Slugify(text string) string {
	words := strings.Fields(slugUnsafe.ReplaceAllString(strings.ToLower(text), " "))
	if len(words) > 6 {
		words = words[:6]
	}
	return strings.Join(words, "-")
}

// Record writes a new dated record, regenerates INDEX.md, and returns the record's path relative
// to the project root. It validates everything first, so a refused request writes nothing.
func Record(repo *Repo, request RecordRequest) (string, error) {
	category, ok := RecordKinds[request.Kind]
	if !ok {
		return "", &RecordError{fmt.Sprintf("unknown kind %q (decision, goal, experiment)", request.Kind)}
	}
	claim := oneLine(request.Claim)
	if claim == "" {
		return "", &RecordError{"--claim is required: one line stating what holds"}
	}
	if len([]rune(claim)) > ClaimMaxChars {
		return "", &RecordError{fmt.Sprintf("claim is %d characters, cap %d", len([]rune(claim)), ClaimMaxChars)}
	}
	slug := Slugify(request.Slug)
	if slug == "" {
		slug = Slugify(claim)
	}
	if slug == "" {
		return "", &RecordError{"cannot derive a filename from the claim; pass --slug"}
	}
	docs, err := LoadDocs(repo.Dir)
	if err != nil {
		return "", err
	}
	for _, target := range request.Supersedes {
		if Resolve(docs, target) == nil {
			return "", &RecordError{fmt.Sprintf("--supersedes %q is not a record", target)}
		}
	}

	name := request.Date + "-" + slug + ".md"
	if category == "decisions" {
		name = fmt.Sprintf("%04d-%s.md", nextDecisionNumber(docs), slug)
	}
	relative := category + "/" + name
	probe := &Doc{Path: relative}
	for _, ref := range request.Refs {
		if message := checkRef(repo, probe, ref, repo.IsGit()); message != "" {
			return "", &RecordError{message}
		}
	}

	var text strings.Builder
	text.WriteString("---\n")
	fmt.Fprintf(&text, "claim: %s\n", formatValue(claim))
	fmt.Fprintf(&text, "date: %s\n", request.Date)
	if request.Global {
		text.WriteString("scope: global\n")
	}
	writeList(&text, "supersedes", request.Supersedes)
	writeList(&text, "premises", request.Premises)
	writeList(&text, "refs", request.Refs)
	text.WriteString("---\n")
	fmt.Fprintf(&text, "# %s\n\n", claim)
	body := request.Body
	if strings.TrimSpace(body) == "" {
		body = bodySkeletons[request.Kind]
	}
	text.WriteString(strings.TrimRight(body, "\n") + "\n")

	path := filepath.Join(repo.Dir, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	// O_EXCL: two agents recording at once must not overwrite each other's record.
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, os.ErrExist) {
		return "", &RecordError{fmt.Sprintf("%s already exists; pass a different --slug", relative)}
	}
	if err != nil {
		return "", err
	}
	if _, err := file.WriteString(text.String()); err != nil {
		file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	if _, err := WriteIndex(repo); err != nil {
		return "", fmt.Errorf("recorded %s but regenerating the index failed: %w", relative, err)
	}
	return ".compass/" + relative, nil
}

func writeList(text *strings.Builder, key string, values []string) {
	if len(values) == 0 {
		return
	}
	text.WriteString(key + ":\n")
	for _, value := range values {
		fmt.Fprintf(text, "  - %s\n", formatValue(oneLine(value)))
	}
}

func nextDecisionNumber(docs []*Doc) int {
	highest := 0
	for _, doc := range docs {
		var number int
		if _, err := fmt.Sscanf(doc.Number(), "%04d", &number); err == nil && number > highest {
			highest = number
		}
	}
	return highest + 1
}
