// Package compass is the only code that reads or writes a .compass/ directory. The CLI and the
// hook handler are thin adapters over it.
package compass

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// RecordCategories are the v3 categories whose files are records: each needs a claim, is capped
// in size, and takes part in supersession. context/ also holds current-state.md, which is not a
// record and is handled on its own.
var RecordCategories = []string{"decisions", "goals", "experiments", "context"}

// Doc is one markdown file under a record category.
type Doc struct {
	// Path is relative to the .compass/ directory, with forward slashes ("decisions/0010-x.md").
	Path     string
	Category string
	// ID is the filename without ".md".
	ID         string
	Claim      string
	Date       string
	Scope      string
	Supersedes []string
	Premises   []string
	Refs       []string
	// Fields holds every top-level frontmatter key that was present, so checks can tell a
	// missing key from an empty one and spot keys that must not be written by hand.
	Fields map[string]bool
	Lines  int
	// SupersededBy is derived from other docs' Supersedes lists, never read from the file.
	SupersededBy []string
	// BodyEmpty is EmptyBody of the record's body.
	BodyEmpty bool
	// LegacySupersedes means Supersedes came from a v2 body line, where a 4-digit number may be a
	// year in prose; a target that doesn't resolve is then ignored rather than reported.
	LegacySupersedes bool
}

// Live reports whether no other record supersedes this one.
func (d *Doc) Live() bool { return len(d.SupersededBy) == 0 }

// Global reports whether the record applies to the whole project, so resume delivers it.
func (d *Doc) Global() bool { return d.Scope == "global" }

// Number returns the NNNN prefix of a decision's ID, or "" for any other record (a dated
// "2026-10-15-x" context file must not read as decision 2026).
func (d *Doc) Number() string {
	if d.Category != "decisions" || datePrefix.MatchString(d.ID) {
		return ""
	}
	if m := decisionNumber.FindString(d.ID); m != "" {
		return strings.TrimSuffix(m, "-")
	}
	return ""
}

var (
	decisionNumber = regexp.MustCompile(`^[0-9]{4}-`)
	// v2 docs carried these as body lines; they are still read so a v2 repo works unmigrated.
	legacySupersedes = regexp.MustCompile(`^(?:\*\*)?Supersedes:(?:\*\*)?\s+(.+?)\s*$`)
	legacyDate       = regexp.MustCompile(`^(?:\*\*)?Date:(?:\*\*)?\s+([0-9]{4}-[0-9]{2}-[0-9]{2})`)
	legacyNumber     = regexp.MustCompile(`\b[0-9]{4}\b`)
	datePrefix       = regexp.MustCompile(`^([0-9]{4}-[0-9]{2}-[0-9]{2})-`)
)

// ParseDoc reads one record file. path is relative to compassDir.
func ParseDoc(compassDir, path string) (*Doc, error) {
	data, err := os.ReadFile(filepath.Join(compassDir, filepath.FromSlash(path)))
	if err != nil {
		return nil, err
	}
	doc := &Doc{
		Path:     path,
		Category: strings.SplitN(path, "/", 2)[0],
		ID:       strings.TrimSuffix(filepath.Base(path), ".md"),
		Fields:   map[string]bool{},
	}
	text := string(data)
	doc.Lines = strings.Count(text, "\n")
	if !strings.HasSuffix(text, "\n") && text != "" {
		doc.Lines++
	}

	front, body, err := splitFrontmatter(text)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	fields, err := parseFrontmatter(front)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	for key, values := range fields {
		doc.Fields[key] = true
		switch key {
		case "claim":
			doc.Claim = strings.Join(values, " ")
		case "date":
			doc.Date = strings.Join(values, " ")
		case "scope":
			doc.Scope = strings.Join(values, " ")
		case "supersedes":
			doc.Supersedes = values
		case "premises":
			doc.Premises = values
		case "refs":
			doc.Refs = values
		}
	}
	// "description" is what Claude-memory-style frontmatter calls the one-line claim; accepting
	// it lets a repo written that way work without rewriting every record.
	if doc.Claim == "" && fields["description"] != nil {
		doc.Claim = strings.Join(fields["description"], " ")
	}

	doc.BodyEmpty = EmptyBody(body)
	scanner := bufio.NewScanner(strings.NewReader(body))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if doc.Date == "" {
			if m := legacyDate.FindStringSubmatch(line); m != nil {
				doc.Date = m[1]
			}
		}
		if !doc.Fields["supersedes"] {
			if m := legacySupersedes.FindStringSubmatch(line); m != nil {
				// Prose follows the legacy line ("Supersedes: 0006 (the install half)", "none"),
				// so only decision numbers are taken from it.
				doc.Supersedes = append(doc.Supersedes, legacyNumber.FindAllString(m[1], -1)...)
				doc.LegacySupersedes = true
			}
		}
	}
	if doc.Date == "" {
		if m := datePrefix.FindStringSubmatch(doc.ID); m != nil {
			doc.Date = m[1]
		}
	}
	return doc, nil
}

// EmptyBody reports whether a record body holds nothing but headings and HTML comments, such as
// `compass record`'s skeleton left unfilled: a claim with no reasons behind it.
func EmptyBody(body string) bool {
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		comment := strings.HasPrefix(trimmed, "<!--") && strings.HasSuffix(trimmed, "-->")
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") && !comment {
			return false
		}
	}
	return true
}

// splitFrontmatter separates a leading "---" block from the body. A file without one has an
// empty frontmatter.
func splitFrontmatter(text string) (front, body string, err error) {
	if !strings.HasPrefix(text, "---\n") && !strings.HasPrefix(text, "---\r\n") {
		return "", text, nil
	}
	rest := text[strings.Index(text, "\n")+1:]
	for offset := 0; offset < len(rest); {
		end := strings.Index(rest[offset:], "\n")
		var line string
		if end < 0 {
			line = rest[offset:]
			end = len(rest) - offset
		} else {
			line = rest[offset : offset+end]
		}
		if strings.TrimRight(line, "\r") == "---" {
			next := offset + end + 1
			if next > len(rest) {
				next = len(rest)
			}
			return rest[:offset], rest[next:], nil
		}
		offset += end + 1
	}
	return "", "", fmt.Errorf("frontmatter opened with --- but never closed")
}

// parseFrontmatter reads the YAML subset compass writes: "key: value", "key: [a, b]", and a key
// followed by "  - item" lines. Indented non-list lines (a nested map such as Claude memory's
// "metadata:") are skipped. Values are strings; a double-quoted value is unquoted.
func parseFrontmatter(front string) (map[string][]string, error) {
	fields := map[string][]string{}
	var current string
	for number, raw := range strings.Split(front, "\n") {
		line := strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indented := line[0] == ' ' || line[0] == '\t'
		if indented {
			if current != "" && strings.HasPrefix(trimmed, "- ") {
				value, err := unquote(strings.TrimSpace(trimmed[2:]))
				if err != nil {
					return nil, fmt.Errorf("frontmatter line %d: %w", number+1, err)
				}
				fields[current] = append(fields[current], value)
			}
			continue
		}
		key, value, ok := strings.Cut(trimmed, ":")
		if !ok {
			return nil, fmt.Errorf("frontmatter line %d: expected \"key: value\", got %q", number+1, trimmed)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		current = key
		fields[key] = []string{}
		switch {
		case value == "":
		case value == ">" || value == "|" || strings.HasPrefix(value, ">-") || strings.HasPrefix(value, "|-"):
			return nil, fmt.Errorf("frontmatter line %d: block scalars (%s) are not supported; write the value on one line", number+1, value)
		case strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]"):
			fields[key] = splitList(value[1 : len(value)-1])
		default:
			unquoted, err := unquote(value)
			if err != nil {
				return nil, fmt.Errorf("frontmatter line %d: %w", number+1, err)
			}
			fields[key] = []string{unquoted}
		}
	}
	return fields, nil
}

func unquote(value string) (string, error) {
	if strings.HasPrefix(value, `"`) {
		unquoted, err := strconv.Unquote(value)
		if err != nil {
			return "", fmt.Errorf("bad quoted value %s", value)
		}
		return unquoted, nil
	}
	if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") && len(value) >= 2 {
		return strings.ReplaceAll(value[1:len(value)-1], "''", "'"), nil
	}
	return value, nil
}

func splitList(text string) []string {
	var out []string
	for _, item := range strings.Split(text, ",") {
		item = strings.Trim(strings.TrimSpace(item), `"'`)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

// yamlScalar matches values a strict YAML reader would turn into a number, bool or null.
var yamlScalar = regexp.MustCompile(`(?i)^([-+]?[0-9][0-9_.eE+-]*|true|false|yes|no|on|off|null|~)$`)

// formatValue quotes a frontmatter value when compass's reader would misparse it bare, or when a
// strict YAML reader would read it as something other than a string ("0006" as the number 6).
func formatValue(value string) string {
	if value == "" || yamlScalar.MatchString(value) || strings.ContainsAny(value[:1], `"'[#&*!|>%@`+"`") ||
		strings.Contains(value, ": ") || strings.Contains(value, " #") ||
		strings.ContainsAny(value, "\n\r\t") || value != strings.TrimSpace(value) {
		return strconv.Quote(value)
	}
	return value
}

// LoadDocs parses every record under the record categories and derives SupersededBy. It skips
// context/current-state.md and archive.md files, which are not records. A record that doesn't
// parse is reported as a finding and left out, so one bad file never hides the rest; only an
// unreadable directory is an error.
func LoadDocs(compassDir string) ([]*Doc, []Finding, error) {
	var docs []*Doc
	var problems []Finding
	for _, category := range RecordCategories {
		entries, err := os.ReadDir(filepath.Join(compassDir, category))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, nil, err
		}
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, ".md") || name == "archive.md" {
				continue
			}
			path := category + "/" + name
			if path == CurrentStatePath {
				continue
			}
			doc, err := ParseDoc(compassDir, path)
			if err != nil {
				problems = append(problems, Finding{Severity: SeverityError, Code: "record-unparseable",
					Path: ".compass/" + path, Message: strings.TrimPrefix(err.Error(), path+": ")})
				continue
			}
			docs = append(docs, doc)
		}
	}
	sort.Slice(docs, func(i, j int) bool { return docs[i].Path < docs[j].Path })
	for _, doc := range docs {
		for _, target := range doc.Supersedes {
			// v2 prose and repeated --supersedes can name one record several times.
			if old := Resolve(docs, target); old != nil && old != doc && !slices.Contains(old.SupersededBy, doc.ID) {
				old.SupersededBy = append(old.SupersededBy, doc.ID)
			}
		}
	}
	return docs, problems, nil
}

// Resolve finds the record a supersedes: entry names: an ID ("0006-copy-not-symlink"), a
// decision number ("0006"), or a path relative to .compass/ ("decisions/0006-x.md").
func Resolve(docs []*Doc, target string) *Doc {
	target = strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(target), "./"), ".md")
	for _, doc := range docs {
		if doc.ID == target || strings.TrimSuffix(doc.Path, ".md") == target {
			return doc
		}
	}
	for _, doc := range docs {
		if doc.Category == "decisions" && doc.Number() == target {
			return doc
		}
	}
	return nil
}
