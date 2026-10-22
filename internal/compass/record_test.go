package compass

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecordWritesADecisionAndRegeneratesTheIndex(t *testing.T) {
	f := newFixture(t)
	f.write("src/x.go", "package x\n")
	f.record("decisions/0006-old.md", "claim: old")
	path, err := Record(f.repo, RecordRequest{
		Kind: "decision", Claim: "Use Go: one binary", Date: "2026-10-21", Global: true,
		Supersedes: []string{"0006"}, Premises: []string{"goal:portable", "startup < 50 ms"}, Refs: []string{"src/x.go"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if path != ".compass/decisions/0007-use-go-one-binary.md" {
		t.Fatalf("path = %s", path)
	}
	want := `---
claim: "Use Go: one binary"
date: 2026-10-21
scope: global
supersedes:
  - 0006-old
premises:
  - goal:portable
  - startup < 50 ms
refs:
  - src/x.go
---
# Use Go: one binary

## Context

## Decision

## Alternatives

## Consequences
`
	if got := f.read(path); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	if !strings.Contains(f.read(".compass/INDEX.md"), "0006-old → 0007-use-go-one-binary") {
		t.Errorf("index not regenerated:\n%s", f.read(".compass/INDEX.md"))
	}
	doc, err := ParseDoc(f.repo.Dir, "decisions/0007-use-go-one-binary.md")
	if err != nil || doc.Claim != "Use Go: one binary" || doc.Supersedes[0] != "0006-old" || !doc.Global() {
		t.Errorf("does not parse back: %+v %v", doc, err)
	}
}

// Quoting only where needed keeps records readable; numbers are quoted for strict YAML readers.
func TestFormatValueQuotesOnlyWhereNeeded(t *testing.T) {
	for value, want := range map[string]string{"goal:portable": "goal:portable", "0006": `"0006"`, "yes": `"yes"`, "a: b": `"a: b"`} {
		if got := formatValue(value); got != want {
			t.Errorf("formatValue(%q) = %s, want %s", value, got, want)
		}
	}
}

func TestRecordRefusesBadRequestsWithoutWriting(t *testing.T) {
	f := newFixture(t)
	f.record("experiments/2026-10-21-run.md", "claim: run")
	cases := []RecordRequest{
		{Kind: "learning", Claim: "x"},
		{Kind: "decision", Claim: "  "},
		{Kind: "decision", Claim: strings.Repeat("x", ClaimMaxChars+1)},
		{Kind: "decision", Claim: "x", Supersedes: []string{"0042"}},
		{Kind: "decision", Claim: "x", Refs: []string{"no/such/file"}},
		{Kind: "experiment", Claim: "run", Date: "2026-10-21"},
		{Kind: "goal", Claim: "!!!"},
	}
	for _, request := range cases {
		if request.Date == "" {
			request.Date = "2026-10-21"
		}
		_, err := Record(f.repo, request)
		if _, refused := err.(*RecordError); !refused {
			t.Errorf("%+v: want a RecordError, got %v", request, err)
		}
	}
	for _, dir := range []string{"decisions", "goals"} {
		if entries, _ := os.ReadDir(filepath.Join(f.repo.Dir, dir)); len(entries) != 0 {
			t.Errorf("%s/ was written to", dir)
		}
	}
}

func TestSlugify(t *testing.T) {
	for in, want := range map[string]string{
		"Use Go: one static binary, shipped as a plugin": "use-go-one-static-binary-shipped",
		"  Ünïcode & symbols!! ":                         "n-code-symbols",
		"":                                               "",
	} {
		if got := Slugify(in, 6); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

// Regression (dogfooding): a dated context record made the next decision 2027, and an explicit
// --slug was cut to six words.
func TestRecordNumbersOnlyFromDecisions(t *testing.T) {
	f := newFixture(t)
	f.record("decisions/0011-x.md", "claim: x")
	f.record("context/2026-10-15-retro.md", "claim: retro")
	path, err := Record(f.repo, RecordRequest{Kind: "decision", Claim: "c", Date: "2026-10-21", Slug: "one two three four five six seven"})
	if err != nil || path != ".compass/decisions/0012-one-two-three-four-five-six-seven.md" {
		t.Errorf("path %q, err %v", path, err)
	}
}

func TestCheckFlagsDuplicateDecisionNumbers(t *testing.T) {
	f := newFixture(t)
	f.state(f.commit("init"))
	f.record("decisions/0012-a.md", "claim: a")
	f.record("decisions/0012-b.md", "claim: b")
	f.record("decisions/0013-c.md", "claim: c\nsupersedes: [0012-a]")
	f.clean()
	// Parallel branches legitimately produce this; it warns, it doesn't block.
	if got := codes(f.check(CheckOptions{})); len(got) != 1 || got[0] != "warn:decision-number-duplicate" {
		t.Errorf("got %v", got)
	}
	f.record("decisions/0014-d.md", "claim: d\nsupersedes: [0012]")
	f.clean()
	if got := codes(f.check(CheckOptions{})); !hasCode(f.check(CheckOptions{}), "supersedes-ambiguous") {
		t.Errorf("a bare duplicate number in supersedes must be an error: %v", got)
	}
}
