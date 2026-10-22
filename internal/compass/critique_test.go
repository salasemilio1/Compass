package compass

import (
	"fmt"
	"os"
	"os/exec"
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
	// The anchor still names a real commit and the fallback is known: routine, so no warning.
	if findings := f.check(CheckOptions{}); hasCode(findings, "anchor-unresolved") {
		t.Errorf("squashed-away anchor warned: %v", codes(findings))
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

// Regression tests for the second fresh-context review (2026-10-21).

// A ref to an untracked file passes on disk but would fail in a fresh clone.
func TestPreCommitChecksPathsAgainstTheStagedTree(t *testing.T) {
	f := newFixture(t)
	f.state(f.commit("init"))
	f.clean()
	f.commit("compass")
	f.write("local-only.txt", "not staged\n")
	f.record("decisions/0001-x.md", "claim: x\nrefs: [local-only.txt]")
	f.git("add", ".compass/")
	findings, err := PreCommit(f.repo)
	if err != nil {
		t.Fatal(err)
	}
	if !hasCode(findings, "ref-unresolved") {
		t.Errorf("a ref to an unstaged file passed: %v", findings)
	}
	f.git("add", "local-only.txt")
	if findings, err := PreCommit(f.repo); err != nil || HasErrors(findings) {
		t.Errorf("staged ref still fails: %v %v", findings, err)
	}
}

// The staged index matches the staged records, the working-tree index the working-tree ones.
func TestPreCommitKeepsEachIndexWithItsRecords(t *testing.T) {
	f := newFixture(t)
	f.state(f.commit("init"))
	f.clean()
	f.commit("compass")
	f.record("decisions/0001-staged.md", "claim: staged")
	f.git("add", ".compass/decisions/0001-staged.md")
	f.record("decisions/0002-unstaged.md", "claim: unstaged")
	if findings, err := PreCommit(f.repo); err != nil || HasErrors(findings) {
		t.Fatalf("%v %v", findings, err)
	}
	staged := f.git("show", ":.compass/INDEX.md")
	working := f.read(".compass/INDEX.md")
	if !strings.Contains(staged, "0001-staged") || strings.Contains(staged, "0002-unstaged") {
		t.Errorf("staged index:\n%s", staged)
	}
	if !strings.Contains(working, "0002-unstaged") {
		t.Errorf("working index lacks the unstaged record:\n%s", working)
	}
	f.git("commit", "-q", "-m", "partial")
	if hasCode(f.check(CheckOptions{}), "index-stale") {
		t.Error("working tree reports a stale index after a partial commit")
	}
}

func TestInitMarksTheIndexForUnionMerges(t *testing.T) {
	f := newFixture(t)
	f.commit("init")
	if _, err := Init(f.repo.Root, InitOptions{NoGitHook: true}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(f.read(".gitattributes"), ".compass/INDEX.md merge=union") {
		t.Error("no merge=union line")
	}
}

// A rebase rewrites the anchored commit; the old hash survives in the object store, so no warning.
func TestCheckIsSilentOnARebasedAnchor(t *testing.T) {
	f := newFixture(t)
	f.clean()
	f.commit("init")
	f.git("checkout", "-q", "-b", "feature")
	f.write("feature.go", "package f\n")
	anchor := f.commit("feature code")
	f.state(anchor)
	f.commit("state on the branch")
	f.git("checkout", "-q", "main")
	f.write("main.go", "package m\n")
	f.commit("main code")
	f.git("checkout", "-q", "feature")
	f.git("rebase", "-q", "main")

	state, _ := f.repo.ReadCurrentState()
	if got := f.repo.Freshness(state); got.AnchorResolves || !got.Known {
		t.Fatalf("want a rebased-away anchor with known fallback freshness: %+v", got)
	}
	if findings := f.check(CheckOptions{}); hasCode(findings, "anchor-unresolved") {
		t.Errorf("rebased anchor warned: %v", codes(findings))
	}
}

// A real anchor outside HEAD's history with current-state.md dirty: the fallback is skipped, so
// freshness is unknown, the message says why, and --require-fresh makes it an error.
func TestCheckWarnsOnAnOrphanedAnchorWithDirtyState(t *testing.T) {
	f := newFixture(t)
	f.clean()
	f.commit("init")
	f.git("checkout", "-q", "-b", "feature")
	anchor := f.commit("feature code")
	f.git("checkout", "-q", "main")
	f.state(anchor)

	findings := f.check(CheckOptions{})
	finding := findingWithCode(t, findings, "anchor-unresolved")
	if finding.Severity != SeverityWarn || finding.Fixable ||
		!strings.Contains(finding.Message, "uncommitted changes") || strings.Contains(finding.Message, "no commit of") {
		t.Errorf("dirty fallback: %+v", finding)
	}
	if finding := findingWithCode(t, f.check(CheckOptions{RequireFresh: true}), "anchor-unresolved"); finding.Severity != SeverityError {
		t.Errorf("unknown freshness under --require-fresh must be an error: %+v", finding)
	}
}

// The invented-hash message claims a fallback only when there is one.
func TestInventedAnchorMessageClaimsFallbackOnlyWhenKnown(t *testing.T) {
	f := newFixture(t)
	f.clean()
	f.commit("init")
	f.state("caf5e589e5f8a3c8a47b8e8e8e8e8e8e8e8e8e8")
	finding := findingWithCode(t, f.check(CheckOptions{}), "anchor-unresolved")
	if !finding.Fixable || !strings.Contains(finding.Message, "names no commit") || strings.Contains(finding.Message, "measured from") {
		t.Errorf("dirty, so unknown: %+v", finding)
	}
	f.commit("state")
	finding = findingWithCode(t, f.check(CheckOptions{}), "anchor-unresolved")
	if !strings.Contains(finding.Message, "measured from the last commit") {
		t.Errorf("committed, so known: %+v", finding)
	}
}

// In a shallow clone a real anchor may simply be absent: not invented, not fixable, not restamped.
func TestShallowCloneLeavesAMissingAnchorAlone(t *testing.T) {
	source := newFixture(t)
	source.clean()
	old := source.commit("init")
	source.write("a.go", "package a\n")
	source.commit("code")
	source.state(old)
	source.commit("state")

	clone := filepath.Join(t.TempDir(), "clone")
	source.git("clone", "-q", "--depth", "1", "file://"+source.repo.Root, clone)
	f := &fixture{t: t, repo: Open(clone)}
	finding := findingWithCode(t, f.check(CheckOptions{}), "anchor-unresolved")
	if finding.Fixable || !strings.Contains(finding.Message, "shallow clone") || strings.Contains(finding.Message, "names no commit") {
		t.Errorf("shallow: %+v", finding)
	}
	before := f.read(".compass/context/current-state.md")
	if _, err := Fix(f.repo); err != nil {
		t.Fatal(err)
	}
	if f.read(".compass/context/current-state.md") != before {
		t.Error("Fix restamped an anchor the shallow clone lacks")
	}
}

// A short hash matching several commits is real: reported as ambiguous and never restamped.
func TestAmbiguousAnchorIsNotCalledInvented(t *testing.T) {
	f := newFixture(t)
	// Enough deterministic commits (fixed author, date and messages) to hold a 7-hex collision.
	var stream strings.Builder
	for i := 1; i <= 45179; i++ {
		message := fmt.Sprint(i)
		fmt.Fprintf(&stream, "commit refs/heads/bulk\ncommitter t <t@example.com> 1700000000 +0000\ndata %d\n%s\n", len(message), message)
	}
	command := exec.Command("git", "-C", f.repo.Root, "fast-import", "--quiet")
	command.Stdin = strings.NewReader(stream.String())
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("fast-import: %v\n%s", err, out)
	}
	seen := map[string]bool{}
	prefix := ""
	for _, hash := range strings.Fields(f.git("rev-list", "bulk")) {
		if seen[hash[:7]] {
			prefix = hash[:7]
			break
		}
		seen[hash[:7]] = true
	}
	if prefix == "" {
		t.Fatal("no 7-hex collision among the bulk commits")
	}

	f.clean()
	f.commit("init")
	f.state(prefix)
	f.commit("state")
	finding := findingWithCode(t, f.check(CheckOptions{}), "anchor-unresolved")
	if finding.Fixable || !strings.Contains(finding.Message, "matches several commits") {
		t.Errorf("ambiguous: %+v", finding)
	}
	before := f.read(".compass/context/current-state.md")
	if _, err := Fix(f.repo); err != nil {
		t.Fatal(err)
	}
	if f.read(".compass/context/current-state.md") != before {
		t.Error("Fix overwrote an ambiguous anchor")
	}
}

func findingWithCode(t *testing.T, findings []Finding, code string) Finding {
	t.Helper()
	for _, finding := range findings {
		if finding.Code == code {
			return finding
		}
	}
	t.Fatalf("no %s in %v", code, codes(findings))
	return Finding{}
}
