package compass

import (
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
	for _, want := range []string{"first line", "1. the last line", "current-state cut here", "## Live global decisions"} {
		if !strings.Contains(text, want) {
			t.Errorf("lacks %q", want)
		}
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
	if f.read("CLAUDE.md") != "# CLAUDE.md\n\n@AGENTS.md\n" {
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
