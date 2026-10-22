package compass

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestResumeBrief(t *testing.T) {
	f := newFixture(t)
	f.write("a.go", "package a\n")
	head := f.commit("first")
	f.state(head, "a.go")
	f.record("decisions/0001-old.md", "claim: old global\nscope: global")
	f.record("decisions/0002-new.md", "claim: new global\nscope: global\nsupersedes: [0001]")
	f.record("decisions/0003-local.md", "claim: local only")
	f.clean()
	f.commit("state")
	f.write("b.go", "package b\n")
	f.commit("add b")
	f.write("a.go", "package a // edited\n")
	f.write("new.txt", "untracked\n")

	brief, err := Resume(f.repo)
	if err != nil {
		t.Fatal(err)
	}
	text := brief.Text(0)
	for _, want := range []string{
		"# compass resume — " + f.repo.Name() + " (branch main)",
		"Freshness: STALE: 1 code commit(s)",
		"## Objective\nShip it.",
		"## Commits since current-state was written\n",
		"add b",
		"- 0002-new — new global (.compass/decisions/0002-new.md)",
		" M a.go", "?? new.txt",
		"1 file changed",
		"Map of all 3 records: .compass/INDEX.md",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("brief lacks %q:\n%s", want, text)
		}
	}
	// Line 2 tells the agent what it can skip, so it doesn't spend its first turns re-reading.
	if lines := strings.Split(text, "\n"); len(lines) < 2 ||
		!strings.HasPrefix(lines[1], "This brief already contains current-state.md") ||
		!strings.Contains(lines[1], "INDEX.md is for lookup, not for reading at start") {
		t.Errorf("brief line 2 doesn't say what to skip:\n%s", text)
	}
	for _, unwanted := range []string{"old global", "local only"} {
		if strings.Contains(text, unwanted) {
			t.Errorf("brief delivers %q, which is superseded or not global", unwanted)
		}
	}
}

func TestResumeWorksOutsideGit(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".compass", "context"), 0o755)
	os.WriteFile(filepath.Join(root, ".compass", "context", "current-state.md"), []byte("# state\nNext: x\n"), 0o644)
	brief, err := Resume(Open(root))
	if err != nil {
		t.Fatal(err)
	}
	text := brief.Text(0)
	if !strings.Contains(text, "Next: x") || strings.Contains(text, "git status") {
		t.Errorf("brief:\n%s", text)
	}
}

// The SessionStart cap must hold even for a bloated current-state, keeping its head and tail.
func TestBriefTextRespectsMaxChars(t *testing.T) {
	var state strings.Builder
	state.WriteString("# current state\n## Objective\nfirst line\n")
	for i := 0; i < 2000; i++ {
		state.WriteString("- filler line about past work, the kind that bloats current-state\n")
	}
	state.WriteString("## Next steps\n1. the last line\n")
	brief := &Brief{Project: "p", Branch: "main", Head: "abc x", CurrentState: state.String(), Claims: []string{"a — b (x)"}}
	text := brief.Text(9500)
	if len(text) > 9500 {
		t.Fatalf("text is %d bytes", len(text))
	}
	for _, want := range []string{"first line", "1. the last line", "lines cut", "## Live global records (decisions and goals with scope: global)"} {
		if !strings.Contains(text, want) {
			t.Errorf("lacks %q", want)
		}
	}
}

// A long but well-formed current-state is cut from the least useful sections, not the middle:
// every next step and the objective survive, and line 2 stops claiming the brief holds it all.
func TestBriefTextCutsLowPrioritySectionsFirst(t *testing.T) {
	var state strings.Builder
	state.WriteString("# current state\n\nDate: 2026-10-21\nCommit: abc1234\n\n## Objective\nShip the rewrite.\n\n### Just done\n")
	for i := 0; i < 250; i++ {
		fmt.Fprintf(&state, "- landed change %d, described at some length so it takes room\n", i)
	}
	state.WriteString("\n## Known issues\n")
	for i := 0; i < 39; i++ {
		fmt.Fprintf(&state, "- issue %d\n", i)
	}
	state.WriteString("\n## Files in play\n")
	for i := 0; i < 30; i++ {
		fmt.Fprintf(&state, "- `file%d.go` — in play\n", i)
	}
	state.WriteString("\n## Next steps\n")
	for i := 1; i <= 10; i++ {
		fmt.Fprintf(&state, "%d. step number %d\n", i, i)
	}
	if lines := strings.Count(state.String(), "\n"); lines != 344 {
		t.Fatalf("fixture is %d lines, want 344", lines)
	}
	brief := &Brief{Project: "p", Branch: "main", CurrentState: strings.TrimRight(state.String(), "\n")}
	text := brief.Text(9500)
	if len(text) > 9500 {
		t.Fatalf("text is %d bytes", len(text))
	}
	for i := 1; i <= 10; i++ {
		if !strings.Contains(text, fmt.Sprintf("%d. step number %d\n", i, i)) {
			t.Errorf("next step %d dropped", i)
		}
	}
	for _, want := range []string{"Commit: abc1234", "## Objective\nShip the rewrite.", "- landed change 0,", "- `file29.go`", "- issue 38", "lines cut"} {
		if !strings.Contains(text, want) {
			t.Errorf("lacks %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "landed change 249") {
		t.Error("Just done should be trimmed before anything else")
	}
	line2 := strings.Split(text, "\n")[1]
	if !strings.Contains(line2, "truncated") || strings.Contains(line2, "don't re-read them") {
		t.Errorf("line 2 doesn't qualify the skip: %s", line2)
	}
	if full := brief.Text(0); !strings.Contains(strings.Split(full, "\n")[1], "don't re-read them") {
		t.Error("uncut brief lost its line 2")
	}
}

// Without recognisable headings there is nothing to rank, so the cut keeps head and tail.
func TestBriefTextFallsBackToCutMiddle(t *testing.T) {
	var state strings.Builder
	state.WriteString("first line\n")
	for i := 0; i < 2000; i++ {
		state.WriteString("free-form notes with no headings at all, line after line\n")
	}
	state.WriteString("the last line")
	brief := &Brief{Project: "p", CurrentState: state.String()}
	text := brief.Text(9500)
	if len(text) > 9500 {
		t.Fatalf("text is %d bytes", len(text))
	}
	for _, want := range []string{"first line", "the last line", "… current-state cut here to fit"} {
		if !strings.Contains(text, want) {
			t.Errorf("lacks %q", want)
		}
	}
}

// Whatever the shape of current-state, the brief stays under the cap and keeps every next step
// and the records that follow current-state.
func TestBriefTextKeepsNextStepsUnderTheCap(t *testing.T) {
	var topics strings.Builder
	for i := 0; i < 300; i++ {
		fmt.Fprintf(&topics, "## Topic %d\nnote %d\n", i, i)
	}
	huge := strings.Repeat("x", 5000)
	for name, test := range map[string]struct {
		state string
		max   int
	}{
		"many small sections": {"# s\n## Objective\nobj\n" + topics.String() + "## Next steps\n1. first\n2. second", 4000},
		"huge preamble line":  {"# s\n" + huge + "\n## Objective\nobj\n## Next steps\n1. first\n2. second", 3000},
		"huge heading line":   {"# s\n## Just done " + huge + "\n- a\n## Next steps:\n1. first\n2. second", 3000},
		"huge objective":      {"# s\n## Objective\n" + strings.Repeat("objective line\n", 400) + "## Next steps (in order)\n1. first\n2. second", 3000},
	} {
		brief := &Brief{Project: "p", Branch: "main", CurrentState: test.state, Claims: []string{"0001-x — y (z)"}}
		text := brief.Text(test.max)
		if len(text) > test.max {
			t.Errorf("%s: %d bytes, cap %d", name, len(text), test.max)
		}
		for _, want := range []string{"1. first\n2. second\n", "- 0001-x — y (z)", "Map of all"} {
			if !strings.Contains(text, want) {
				t.Errorf("%s: lacks %q:\n%s", name, want, text)
			}
		}
	}
}

// A cut marker costs bytes; a small overshoot must come out of Just done, not the objective or
// the next steps.
func TestCutStateSectionsSmallOvershoot(t *testing.T) {
	text := "# s\n## Objective\nShip the rewrite.\n## Just done\n" +
		strings.Repeat("- landed a change described in about sixty bytes of text\n", 5) +
		"## Next steps\n1. first\n2. second"
	max := len(text) - 5
	got := cutStateSections(text, max)
	if len(got) > max {
		t.Errorf("%d bytes, cap %d", len(got), max)
	}
	for _, want := range []string{"## Objective\nShip the rewrite.\n", "## Next steps\n1. first\n2. second", "## Just done\n- landed"} {
		if !strings.Contains(got, want) {
			t.Errorf("lacks %q:\n%s", want, got)
		}
	}
}

// A heading inside a code fence is content: it must not shield its section as next steps.
func TestCutStateSectionsIgnoresFencedHeadings(t *testing.T) {
	text := "# s\n## Objective\nobj\n## Just done\n```\n## Next steps\n```\n" +
		strings.Repeat("- filler that should be cut first\n", 100) + "## Next steps\n1. real"
	got := cutStateSections(text, 500)
	if len(got) > 500 || !strings.Contains(got, "## Objective\nobj\n") || !strings.HasSuffix(got, "## Next steps\n1. real") {
		t.Errorf("cut:\n%s", got)
	}
}

// A fence typo must not hide the headings after it: an unclosed fence is ignored, and a fence
// closes only with its own marker.
func TestCutStateSectionsSurvivesBadFences(t *testing.T) {
	filler := strings.Repeat("- filler that should be cut first\n", 100)
	for name, text := range map[string]string{
		"unclosed":   "# s\n## Objective\nobj\n## Just done\n```go\n" + filler + "## Next steps\n1. real",
		"mismatched": "# s\n## Objective\nobj\n## Just done\n```\n~~~\n```\n" + filler + "## Next steps\n1. real",
	} {
		got := cutStateSections(text, 500)
		if len(got) > 500 || !strings.Contains(got, "## Objective\nobj\n") || !strings.HasSuffix(got, "## Next steps\n1. real") {
			t.Errorf("%s:\n%s", name, got)
		}
	}
}

// An earlier next-steps section (an archive) must not crowd out the live list at the end.
func TestCutStateSectionsProtectsTheLastNextSteps(t *testing.T) {
	text := "# s\n## Objective\nobj\n## Next steps (done last week)\n" +
		strings.Repeat("1. an old step that is finished now\n", 100) + "## Next steps\n1. real\n2. also real"
	got := cutStateSections(text, 500)
	if len(got) > 500 || !strings.Contains(got, "## Objective\nobj\n") || !strings.HasSuffix(got, "## Next steps\n1. real\n2. also real") {
		t.Errorf("cut:\n%s", got)
	}
}

// "## Objectives" is the objective, kept over an unrecognised section.
func TestCutStateSectionsKnowsObjectives(t *testing.T) {
	objective := "## Objectives\n- ship the rewrite of the command line tool\n- retire v2 cleanly and on time\n- keep every repo working\n"
	text := "# s\n" + objective + "## Notes\n" + strings.Repeat("- a note that matters less than the objective\n", 5) + "## Next steps\n1. go"
	got := cutStateSections(text, len(text)-30)
	if len(got) > len(text)-30 || !strings.Contains(got, objective) || !strings.HasSuffix(got, "## Next steps\n1. go") {
		t.Errorf("cut:\n%s", got)
	}
}

// Section titles match case-insensitively at H2 or H3, as check does.
func TestCutStateSectionsHeadingVariants(t *testing.T) {
	text := "## OBJECTIVE\nkeep me\n### just done\n" + strings.Repeat("- old work\n", 100) + "## Next Steps\n1. keep me too"
	got := cutStateSections(text, 300)
	if len(got) > 300 || !strings.Contains(got, "## OBJECTIVE\nkeep me") || !strings.HasSuffix(got, "## Next Steps\n1. keep me too") ||
		!strings.Contains(got, "### just done\n- old work") {
		t.Errorf("cut:\n%s", got)
	}
}

func TestInitScaffoldsAndIsIdempotent(t *testing.T) {
	f := newFixture(t)
	f.write("AGENTS.md", "# Mine\n\nKeep this.\n")
	f.write(".gitignore", "node_modules/")
	head := f.commit("init")
	today := time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC)

	if _, err := Init(f.repo.Root, InitOptions{Today: today}); err != nil {
		t.Fatal(err)
	}
	agents := f.read("AGENTS.md")
	if !strings.HasPrefix(agents, "# Mine\n\nKeep this.\n\n"+agentsBegin) || !strings.Contains(agents, "compass resume") {
		t.Errorf("AGENTS.md:\n%s", agents)
	}
	if f.read("CLAUDE.md") != "# CLAUDE.md\n\n"+claudePointer+"\n" {
		t.Errorf("CLAUDE.md: %q", f.read("CLAUDE.md"))
	}
	if f.read(".gitignore") != "node_modules/\n.compass/scratch/\n" {
		t.Errorf(".gitignore: %q", f.read(".gitignore"))
	}
	if !strings.Contains(f.read(".compass/context/current-state.md"), "Commit: "+head) {
		t.Error("current-state not anchored at HEAD")
	}
	hookPath := filepath.Join(f.repo.Root, ".git", "hooks", "pre-commit")
	if info, err := os.Stat(hookPath); err != nil || info.Mode()&0o100 == 0 {
		t.Errorf("git hook not installed executable: %v", err)
	}

	// A user's edits survive a second run; only the managed block is refreshed.
	f.write(".compass/context/current-state.md", "Commit: "+head+"\n# mine\n")
	f.write("AGENTS.md", strings.Replace(f.read("AGENTS.md"), "Comment the why", "EDITED", 1)+"\ntrailer\n")
	if _, err := Init(f.repo.Root, InitOptions{Today: today}); err != nil {
		t.Fatal(err)
	}
	if f.read(".compass/context/current-state.md") != "Commit: "+head+"\n# mine\n" {
		t.Error("init overwrote current-state.md")
	}
	agents = f.read("AGENTS.md")
	if strings.Contains(agents, "EDITED") || !strings.HasSuffix(agents, "\ntrailer\n") || strings.Count(agents, agentsBegin) != 1 {
		t.Errorf("block not refreshed in place:\n%s", agents)
	}
	f.write(".git/hooks/pre-commit", "#!/bin/sh\n# Installed by `compass init`: an older version\n")
	if _, err := Init(f.repo.Root, InitOptions{Today: today}); err != nil {
		t.Fatal(err)
	}
	if f.read(".git/hooks/pre-commit") != gitPreCommit {
		t.Error("compass's own older git hook was not refreshed")
	}
	if strings.Count(f.read(".gitignore"), ".compass/scratch/") != 1 {
		t.Error(".gitignore line duplicated")
	}
	if findings := f.check(CheckOptions{}); HasErrors(findings) {
		t.Errorf("after init: %v", findings)
	}
}

func TestInitLeavesAnExistingGitHookAlone(t *testing.T) {
	f := newFixture(t)
	f.commit("init")
	f.write(".git/hooks/pre-commit", "#!/bin/sh\nlefthook run pre-commit\n")
	actions, err := Init(f.repo.Root, InitOptions{Today: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if f.read(".git/hooks/pre-commit") != "#!/bin/sh\nlefthook run pre-commit\n" {
		t.Error("existing hook overwritten")
	}
	if !strings.Contains(strings.Join(actions, "\n"), "add `compass hook pre-commit` to it") {
		t.Errorf("actions: %v", actions)
	}
}

func TestPreCommitStagesTheRegeneratedIndex(t *testing.T) {
	f := newFixture(t)
	f.state(f.commit("init"))
	f.clean()
	f.commit("compass")
	f.record("decisions/0001-x.md", "claim: x")
	f.git("add", ".compass/decisions/0001-x.md")
	findings, err := PreCommit(f.repo)
	if err != nil || HasErrors(findings) {
		t.Fatalf("%v %v", findings, err)
	}
	if staged := f.git("diff", "--cached", "--name-only"); !strings.Contains(staged, ".compass/INDEX.md") {
		t.Errorf("INDEX.md not staged: %q", staged)
	}
}
