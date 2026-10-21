package compass

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Regression tests for the first fresh-context review (2026-10-21).

// A squash merge leaves the branch's anchor in the object store but outside HEAD's history.
func TestFreshnessIgnoresAnAnchorOutsideHeadHistory(t *testing.T) {
	f := newFixture(t)
	f.clean()
	f.commit("init")
	f.git("checkout", "-q", "-b", "feature")
	f.write("feature.go", "package f\n")
	branchHead := f.commit("feature code")
	f.state(branchHead)
	f.commit("state on the branch")
	f.git("checkout", "-q", "main")
	for i := 0; i < 3; i++ {
		f.write(fmt.Sprintf("main%d.go", i), "package m\n")
		f.commit("main code")
	}
	f.git("merge", "-q", "--squash", "feature")
	f.commit("squash feature")

	state, _ := f.repo.ReadCurrentState()
	got := f.repo.Freshness(state)
	if got.AnchorResolves || !got.Known || got.CodeCommits != 0 {
		t.Errorf("want unresolved anchor, measured from the squash commit, 0 stale: %+v", got)
	}
}

func TestIndexIsTheSameInEveryCheckout(t *testing.T) {
	var outputs []string
	for _, name := range []string{"a", "b-worktree"} {
		root := filepath.Join(t.TempDir(), name)
		repo := Open(root)
		os.MkdirAll(filepath.Join(repo.Dir, "goals"), 0o755)
		os.WriteFile(filepath.Join(repo.Dir, "goals", "g.md"), []byte("---\nclaim: g\n---\n"), 0o644)
		docs, _, _ := LoadDocs(repo.Dir)
		text, err := GenerateIndex(repo, docs)
		if err != nil {
			t.Fatal(err)
		}
		outputs = append(outputs, text)
	}
	if outputs[0] != outputs[1] {
		t.Errorf("index depends on the directory name:\n%s\n---\n%s", outputs[0], outputs[1])
	}
}

// Every category keeps its heading and pointer, and the cap holds even with a large tail.
func TestIndexKeepsEverySectionUnderTheCap(t *testing.T) {
	f := newFixture(t)
	long := strings.Repeat("word ", 39)
	for i := 1; i <= 300; i++ {
		f.record(fmt.Sprintf("decisions/%04d-d.md", i), "claim: "+long+"\nscope: global")
		f.record(fmt.Sprintf("decisions/%04d-s.md", i+1000), fmt.Sprintf("claim: s\nsupersedes: [%04d-d]", i))
	}
	for _, category := range []string{"goals", "experiments", "context"} {
		f.record(category+"/2026-01-01-x.md", "claim: "+long)
	}
	for i := 0; i < 200; i++ {
		os.MkdirAll(filepath.Join(f.repo.Dir, fmt.Sprintf("legacy-folder-%03d", i)), 0o755)
	}
	docs, _, _ := LoadDocs(f.repo.Dir)
	text, err := GenerateIndex(f.repo, docs)
	if err != nil {
		t.Fatal(err)
	}
	if tokens := EstimateTokens(text); tokens > IndexMaxTokens {
		t.Errorf("%d tokens, cap %d", tokens, IndexMaxTokens)
	}
	for _, heading := range []string{"## decisions", "## goals", "## experiments", "## context", "## superseded (300"} {
		if !strings.Contains(text, heading) {
			t.Errorf("lacks %q", heading)
		}
	}
}

func TestOneBadRecordDegradesInsteadOfFailing(t *testing.T) {
	f := newFixture(t)
	f.state(f.commit("init"))
	f.record("decisions/0001-good.md", "claim: good\nscope: global")
	f.write(".compass/decisions/0002-bad.md", "---\nclaim: \"Use X\" over Y\n---\n")
	f.write(".compass/decisions/0003-folded.md", "---\nclaim: >\n  folded\n---\n")
	f.clean()
	got := codes(f.check(CheckOptions{}))
	if strings.Join(got, ",") != "error:record-unparseable,error:record-unparseable" {
		t.Errorf("findings: %v", got)
	}
	brief, err := Resume(f.repo)
	if err != nil {
		t.Fatal(err)
	}
	text := brief.Text(0)
	if !strings.Contains(text, "0001-good — good") || !strings.Contains(text, "2 record(s) don't parse") {
		t.Errorf("brief:\n%s", text)
	}
}

func TestLegacyYearsAreNotDanglingSupersedes(t *testing.T) {
	f := newFixture(t)
	f.state(f.commit("init"))
	f.write(".compass/decisions/0001-x.md", "---\nclaim: x\n---\nSupersedes: the 2024 approach\n")
	f.clean()
	if findings := f.check(CheckOptions{}); len(findings) != 0 {
		t.Errorf("findings: %v", findings)
	}
}

func TestPreCommitChecksWhatIsStaged(t *testing.T) {
	f := newFixture(t)
	f.state(f.commit("init"))
	f.clean()
	f.commit("compass")

	// Staged: broken. Working tree: fixed. The commit must be blocked.
	f.write(".compass/decisions/0001-x.md", "---\ndate: 2026-10-21\n---\n")
	f.git("add", "-A")
	f.record("decisions/0001-x.md", "claim: fixed in the working tree only")
	findings, err := PreCommit(f.repo)
	if err != nil {
		t.Fatal(err)
	}
	if !hasCode(findings, "claim-missing") {
		t.Errorf("staged broken record passed: %v", findings)
	}

	// An untracked record must not reach the committed index.
	f.git("add", "-A")
	f.record("decisions/0002-untracked.md", "claim: not staged")
	if findings, err := PreCommit(f.repo); err != nil || HasErrors(findings) {
		t.Fatalf("%v %v", findings, err)
	}
	staged := f.git("show", ":.compass/INDEX.md")
	if !strings.Contains(staged, "0001-x") || strings.Contains(staged, "0002-untracked") {
		t.Errorf("staged index:\n%s", staged)
	}
}
