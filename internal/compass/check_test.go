package compass

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestCheckPassesACleanProject(t *testing.T) {
	f := newFixture(t)
	f.write("src/main.go", "package main\n")
	head := f.commit("code")
	f.state(head, "src/main.go")
	f.record("goals/2026-01-01-ship.md", "claim: ship")
	f.record("decisions/0001-go.md", "claim: use Go\nrefs:\n  - commit:"+head[:7]+"\n  - src/main.go\n  - https://example.com/x\n  - goals/2026-01-01-ship.md\n  - ../goals/2026-01-01-ship.md")
	f.clean()
	f.commit("state")
	if findings := f.check(CheckOptions{RequireFresh: true}); len(findings) != 0 {
		t.Errorf("want no findings, got %v", findings)
	}
}

func TestCheckFlagsEachRecordRule(t *testing.T) {
	f := newFixture(t)
	head := f.commit("init")
	f.state(head)
	f.record("decisions/0001-a.md", "date: 2026-01-01")
	f.record("decisions/0002-b.md", "claim: "+strings.Repeat("x", ClaimMaxChars+1))
	f.record("decisions/0003-c.md", "claim: c\nstatus: superseded")
	f.record("decisions/0004-d.md", "claim: d\nsupersedes: [0099]")
	f.record("decisions/0005-e.md", "claim: e\nrefs: [missing/file.go, commit:deadbeef]")
	f.record("decisions/0006-f.md", "claim: f\nscope: team")
	f.write(".compass/goals/long.md", "---\nclaim: long\n---\n"+strings.Repeat("line\n", RecordMaxLines))
	f.clean()
	got := map[string]int{}
	for _, finding := range f.check(CheckOptions{}) {
		got[finding.Path+" "+finding.Code]++
	}
	want := map[string]int{
		".compass/decisions/0001-a.md claim-missing":       1,
		".compass/decisions/0002-b.md claim-too-long":      1,
		".compass/decisions/0003-c.md status-handwritten":  1,
		".compass/decisions/0004-d.md supersedes-dangling": 1,
		".compass/decisions/0005-e.md ref-unresolved":      2,
		".compass/decisions/0006-f.md scope-invalid":       1,
		".compass/goals/long.md record-too-long":           1,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v\nwant %v", got, want)
	}
}

func TestCheckFindsSupersessionCycles(t *testing.T) {
	f := newFixture(t)
	f.state(f.commit("init"))
	f.record("decisions/0001-a.md", "claim: a\nsupersedes: [0002]")
	f.record("decisions/0002-b.md", "claim: b\nsupersedes: [0001]")
	f.clean()
	if got := codes(f.check(CheckOptions{})); !reflect.DeepEqual(got, []string{"error:supersedes-cycle", "error:supersedes-cycle"}) {
		t.Errorf("got %v", got)
	}
}

func TestCheckFlagsAStaleIndexAndFixRegeneratesIt(t *testing.T) {
	f := newFixture(t)
	f.state(f.commit("init"))
	f.record("decisions/0001-a.md", "claim: a")
	f.clean()
	f.record("decisions/0002-b.md", "claim: b")
	if !hasCode(f.check(CheckOptions{}), "index-stale") {
		t.Fatal("a record added after indexing must make the index stale")
	}
	if _, err := Fix(f.repo); err != nil {
		t.Fatal(err)
	}
	if findings := f.check(CheckOptions{}); len(findings) != 0 {
		t.Errorf("after fix: %v", findings)
	}
}

func TestCheckCurrentState(t *testing.T) {
	f := newFixture(t)
	head := f.commit("init")
	f.clean()

	f.write(".compass/context/current-state.md", "# current state\n")
	if got := codes(f.check(CheckOptions{})); !reflect.DeepEqual(got, []string{"error:anchor-missing"}) {
		t.Errorf("no anchor: %v", got)
	}
	f.write(".compass/context/current-state.md", "Commit: HEAD\n")
	if got := codes(f.check(CheckOptions{})); !reflect.DeepEqual(got, []string{"error:anchor-malformed"}) {
		t.Errorf("malformed anchor: %v", got)
	}
	f.state(head, "gone.go", "*.md", "https://x.y/z")
	if got := codes(f.check(CheckOptions{})); !reflect.DeepEqual(got, []string{"error:file-in-play-missing"}) {
		t.Errorf("missing file in play (globs and URLs skipped): %v", got)
	}
	f.write(".compass/context/current-state.md", "Commit: "+head+"\n"+strings.Repeat("line\n", CurrentStateMaxLines))
	if got := codes(f.check(CheckOptions{})); !reflect.DeepEqual(got, []string{"error:current-state-too-long"}) {
		t.Errorf("over the cap: %v", got)
	}
}

func TestCheckFixStampsOnlyAMissingAnchor(t *testing.T) {
	f := newFixture(t)
	head := f.commit("init")
	f.clean()
	f.write(".compass/context/current-state.md", "# current state\n\nDate: 2026-10-21\n\n## Next\n")
	if _, err := Fix(f.repo); err != nil {
		t.Fatal(err)
	}
	if got := f.read(".compass/context/current-state.md"); got != "# current state\n\nDate: 2026-10-21\nCommit: "+head+"\n\n## Next\n" {
		t.Errorf("stamped:\n%s", got)
	}

	// An invented hash, which names no commit, is restamped.
	f.write(".compass/context/current-state.md", "Date: 2026-10-21\nCommit: caf5e589e5f8a3c8a47b8e8e8e8e8e8e8e8e8e8\n")
	if _, err := Fix(f.repo); err != nil {
		t.Fatal(err)
	}
	if got := f.read(".compass/context/current-state.md"); got != "Date: 2026-10-21\nCommit: "+head+"\n" {
		t.Errorf("invented hash not restamped: %q", got)
	}
	f.write(".compass/context/current-state.md", "# current state\n\nDate: 2026-10-21\nCommit: "+head+"\n\n## Next\n")

	// A real but stale anchor is not restamped: that would claim current-state was rewritten.
	f.commit("state")
	f.write("code.go", "package x\n")
	f.commit("code")
	before := f.read(".compass/context/current-state.md")
	if _, err := Fix(f.repo); err != nil {
		t.Fatal(err)
	}
	if f.read(".compass/context/current-state.md") != before {
		t.Error("Fix rewrote a well-formed stale anchor")
	}
}

func TestFreshness(t *testing.T) {
	f := newFixture(t)
	f.clean()
	base := f.commit("init")
	f.state(base)
	f.commit("state at init")

	fresh := func() Freshness {
		state, err := f.repo.ReadCurrentState()
		if err != nil {
			t.Fatal(err)
		}
		return f.repo.Freshness(state)
	}
	if got := fresh(); !got.Known || got.CodeCommits != 0 || !got.AnchorResolves {
		t.Fatalf("right after writing: %+v", got)
	}

	f.record("decisions/0001-x.md", "claim: x")
	f.clean()
	f.commit("compass-only commit")
	if got := fresh(); got.CodeCommits != 0 {
		t.Errorf("a commit touching only .compass/ must not count: %+v", got)
	}

	f.write("a.go", "package a\n")
	f.commit("code 1")
	f.write("b.go", "package b\n")
	f.commit("code 2")
	if got := fresh(); got.CodeCommits != 2 {
		t.Errorf("two code commits: %+v", got)
	}

	// Rewritten but uncommitted, stamped with HEAD: fresh.
	f.state(f.git("rev-parse", "HEAD"))
	if got := fresh(); got.CodeCommits != 0 {
		t.Errorf("dirty and stamped at HEAD: %+v", got)
	}

	// Committed together with code, stamped at the parent: that commit doesn't count.
	parent := f.git("rev-parse", "HEAD")
	f.write("c.go", "package c\n")
	f.state(parent)
	f.commit("code and state together")
	if got := fresh(); got.CodeCommits != 0 {
		t.Errorf("code and state in one commit: %+v", got)
	}

	// An anchor missing from history (squash merge) falls back to the last state commit.
	f.state("1234567")
	f.commit("state with a foreign anchor")
	if got := fresh(); got.AnchorResolves || !got.Known || got.CodeCommits != 0 {
		t.Errorf("unresolvable anchor: %+v", got)
	}
	f.write("d.go", "package d\n")
	f.commit("code 3")
	findings := f.check(CheckOptions{RequireFresh: true})
	if got := codes(findings); !reflect.DeepEqual(got, []string{"error:anchor-stale", "warn:anchor-unresolved"}) {
		t.Errorf("findings: %v", got)
	}
	if got := codes(f.check(CheckOptions{})); !reflect.DeepEqual(got, []string{"warn:anchor-unresolved", "warn:anchor-stale"}) {
		t.Errorf("without --require-fresh staleness is a warning: %v", got)
	}
}

func TestCheckWarnsOnAgentsBlockAndLayoutDrift(t *testing.T) {
	f := newFixture(t)
	f.write("cmd/x.go", "package x\n")
	f.write("docs/a.md", "a\n")
	f.write(".compass/context/overview.md", "---\nclaim: overview\n---\n`cmd/` holds commands\n")
	f.state(f.commit("init"))
	f.clean()
	f.write("AGENTS.md", "<!-- compass-spec-version: 2 -->\n# old\n")
	got := f.check(CheckOptions{})
	if strings.Join(codes(got), ",") != "warn:layout-drift,warn:agents-block-drift" {
		t.Fatalf("got %v", got)
	}
	if !strings.Contains(got[0].Message, "docs") || strings.Contains(got[0].Message, "cmd") {
		t.Errorf("layout message: %s", got[0].Message)
	}
}

func TestGenerateIndexIsDeterministicAndListsOnlyLiveRecords(t *testing.T) {
	f := newFixture(t)
	f.record("decisions/0001-old.md", "claim: old way")
	f.record("decisions/0002-new.md", "claim: new way\nsupersedes: [0001]\nscope: global")
	f.record("decisions/0003-local.md", "claim: local rule")
	f.record("experiments/2026-01-02-b.md", "claim: run b")
	f.record("experiments/2026-03-04-a.md", "claim: run a")
	f.write(".compass/learnings/x.md", "x\n")
	docs, _, err := LoadDocs(f.repo.Dir)
	if err != nil {
		t.Fatal(err)
	}
	first, _ := GenerateIndex(f.repo, docs)
	second, _ := GenerateIndex(f.repo, docs)
	if first != second {
		t.Fatal("index output differs between runs")
	}
	want := indexHeader + `# compass index

Start with ` + "`compass resume`" + `: current-state, live global claims, uncommitted work.
Records below are live; search all of .compass/ with ` + "`rg -i <term> .compass/`" + `.

## global (delivered by ` + "`compass resume`" + `)
- [0002-new](decisions/0002-new.md) — new way

## decisions
- [0003-local](decisions/0003-local.md) — local rule

## experiments
- [2026-03-04-a](experiments/2026-03-04-a.md) — run a
- [2026-01-02-b](experiments/2026-01-02-b.md) — run b

## superseded (1, not delivered)
0001-old → 0002-new

## other folders (not records, not indexed)
` + "`learnings/` (1)" + `
`
	if first != want {
		t.Errorf("got:\n%s\nwant:\n%s", first, want)
	}
}

func TestGenerateIndexStaysUnderTheTokenCap(t *testing.T) {
	f := newFixture(t)
	for i := 1; i <= 400; i++ {
		f.record(fmt.Sprintf("decisions/%04d-d.md", i), "claim: "+strings.Repeat("word ", 30))
		f.record(fmt.Sprintf("experiments/2026-01-01-e%03d.md", i), "claim: "+strings.Repeat("word ", 30))
	}
	docs, _, err := LoadDocs(f.repo.Dir)
	if err != nil {
		t.Fatal(err)
	}
	text, err := GenerateIndex(f.repo, docs)
	if err != nil {
		t.Fatal(err)
	}
	if tokens := EstimateTokens(text); tokens > IndexMaxTokens {
		t.Errorf("index is %d tokens, cap %d", tokens, IndexMaxTokens)
	}
	if !strings.Contains(text, "more, not listed") || !strings.Contains(text, "[0400-d]") {
		t.Errorf("want newest decisions first and an overflow pointer:\n%s", text)
	}
}
