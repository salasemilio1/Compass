package compass

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func parseText(t *testing.T, path, text string) *Doc {
	t.Helper()
	dir := t.TempDir()
	full := filepath.Join(dir, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, err := ParseDoc(dir, path)
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestParseDocReadsTheFrontmatterSubset(t *testing.T) {
	doc := parseText(t, "decisions/0012-use-go.md", `---
claim: "Go: one static binary"
date: 2026-10-21
scope: global
supersedes: [0006, decisions/0007-x.md]
premises:
  - goal:portable-memory
  - "startup under 50 ms, # not 300"
refs:
  - commit:abc1234
metadata:
  type: decision
---
# body
`)
	want := &Doc{
		Path: "decisions/0012-use-go.md", Category: "decisions", ID: "0012-use-go",
		Claim: "Go: one static binary", Date: "2026-10-21", Scope: "global",
		Supersedes: []string{"0006", "decisions/0007-x.md"},
		Premises:   []string{"goal:portable-memory", "startup under 50 ms, # not 300"},
		Refs:       []string{"commit:abc1234"},
		Fields: map[string]bool{"claim": true, "date": true, "scope": true, "supersedes": true,
			"premises": true, "refs": true, "metadata": true},
		Lines: 14,
		// The body is a lone heading: nothing written under the claim.
		BodyEmpty: true,
	}
	if !reflect.DeepEqual(doc, want) {
		t.Errorf("got  %+v\nwant %+v", doc, want)
	}
	if doc.Number() != "0012" || !doc.Global() || !doc.Live() {
		t.Errorf("Number=%q Global=%v Live=%v", doc.Number(), doc.Global(), doc.Live())
	}
}

// A v2 decision has no frontmatter: date and supersession come from body lines, and only
// decision numbers count from a prose Supersedes: line.
func TestParseDocReadsLegacyBodyLines(t *testing.T) {
	doc := parseText(t, "decisions/0010-go.md", "# 0010 — go\n\nDate: 2026-10-21\nSupersedes: 0006 (the install half), not none\n")
	if doc.Date != "2026-10-21" || !reflect.DeepEqual(doc.Supersedes, []string{"0006"}) || doc.Claim != "" {
		t.Errorf("date=%q supersedes=%v claim=%q", doc.Date, doc.Supersedes, doc.Claim)
	}
	none := parseText(t, "decisions/0009-x.md", "**Date:** 2026-07-01\n**Supersedes:** none\n")
	if none.Date != "2026-07-01" || len(none.Supersedes) != 0 {
		t.Errorf("date=%q supersedes=%v", none.Date, none.Supersedes)
	}
}

// Claude-memory-style frontmatter calls the one-line claim "description".
func TestParseDocAcceptsDescriptionAsClaim(t *testing.T) {
	doc := parseText(t, "decisions/0001-x.md", "---\nname: 0001-x\ndescription: releases are cut by hand\nmetadata:\n  type: decision\n---\n")
	if doc.Claim != "releases are cut by hand" {
		t.Errorf("claim = %q", doc.Claim)
	}
}

func TestParseDocDatesFromFilename(t *testing.T) {
	doc := parseText(t, "experiments/2026-09-01-baseline.md", "---\nclaim: x\n---\n")
	if doc.Date != "2026-09-01" {
		t.Errorf("date = %q", doc.Date)
	}
}

func TestParseDocRejectsUnclosedFrontmatter(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "goals"), 0o755)
	os.WriteFile(filepath.Join(dir, "goals", "x.md"), []byte("---\nclaim: x\n"), 0o644)
	if _, err := ParseDoc(dir, "goals/x.md"); err == nil {
		t.Fatal("expected an error for frontmatter that never closes")
	}
}

// formatValue's output must read back as the same value.
func TestFormatValueRoundTrips(t *testing.T) {
	for _, value := range []string{"plain", "a: b", "# heading", "x #y", `"quoted"`, "[list]", " padded ", "it's"} {
		fields, err := parseFrontmatter("k: " + formatValue(value))
		if err != nil {
			t.Fatalf("%q: %v", value, err)
		}
		if got := fields["k"]; len(got) != 1 || got[0] != value {
			t.Errorf("%q round-tripped as %q", value, got)
		}
	}
}

func TestLoadDocsDerivesSupersession(t *testing.T) {
	f := newFixture(t)
	f.record("decisions/0001-old.md", "claim: old")
	f.record("decisions/0002-new.md", "claim: new\nsupersedes: [0001]")
	f.record("goals/2026-01-01-dashboard.md", "claim: a dashboard")
	f.record("goals/2026-06-01-flipper.md", "claim: a flipper\nsupersedes: [2026-01-01-dashboard]")
	f.write(".compass/context/current-state.md", "# not a record\n")
	f.write(".compass/decisions/archive.md", "# not a record\n")
	f.write(".compass/learnings/x.md", "# not a record category\n")
	docs, _, err := LoadDocs(f.repo.Dir)
	if err != nil {
		t.Fatal(err)
	}
	status := map[string][]string{}
	for _, doc := range docs {
		status[doc.ID] = doc.SupersededBy
	}
	want := map[string][]string{
		"0001-old": {"0002-new"}, "0002-new": nil,
		"2026-01-01-dashboard": {"2026-06-01-flipper"}, "2026-06-01-flipper": nil,
	}
	if !reflect.DeepEqual(status, want) {
		t.Errorf("got %v, want %v", status, want)
	}
}
