package hook

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/salasemilio1/Compass/internal/compass"
)

func gitRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	run := func(args ...string) {
		command := exec.Command("git", append([]string{"-C", root}, args...)...)
		command.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@x", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@x")
		if out, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	write := func(path, text string) {
		os.MkdirAll(filepath.Dir(filepath.Join(root, path)), 0o755)
		os.WriteFile(filepath.Join(root, path), []byte(text), 0o644)
	}
	run("init", "-q", "-b", "main")
	write("a.go", "package a\n")
	run("add", "-A")
	run("commit", "-q", "-m", "first")
	out, _ := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	write(".compass/context/current-state.md", "# current state\nCommit: "+strings.TrimSpace(string(out))+"\n\n## Next steps\n1. finish\n")
	run("add", "-A")
	run("commit", "-q", "-m", "state")
	// A later code commit makes current-state stale.
	write("b.go", "package b\n")
	run("add", "-A")
	run("commit", "-q", "-m", "more code")
	return root
}

func decode(t *testing.T, data []byte) output {
	t.Helper()
	var out output
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("not JSON: %s", data)
	}
	return out
}

func TestSessionStartInjectsTheBrief(t *testing.T) {
	root := gitRepo(t)
	pythonDir := t.TempDir()
	os.WriteFile(filepath.Join(pythonDir, "compass"), []byte("#!/home/u/.local/share/uv/tools/compass-cli/bin/python3\n"), 0o755)

	data, err := SessionStart(Input{Cwd: filepath.Join(root), Source: "compact"}, pythonDir, "/plugins/compass")
	if err != nil {
		t.Fatal(err)
	}
	out := decode(t, data)
	if out.HookSpecificOutput == nil || out.HookSpecificOutput.HookEventName != "SessionStart" {
		t.Fatalf("output: %s", data)
	}
	context := out.HookSpecificOutput.AdditionalContext
	for _, want := range []string{"1. finish", "STALE: 1 code commit(s)", "rewrite it now", "old Python compass-cli", `"/plugins/compass/bin/compass"`} {
		if !strings.Contains(context, want) {
			t.Errorf("context lacks %q:\n%s", want, context)
		}
	}
	if len(context) > SessionStartMaxChars {
		t.Errorf("context is %d chars", len(context))
	}
}

// briefRepo builds a compass project whose current-state is current and committed on a clean
// tree; change then moves it into the condition under test.
func briefRepo(t *testing.T, change func(root string, run func(...string))) string {
	t.Helper()
	root := t.TempDir()
	run := func(args ...string) {
		command := exec.Command("git", append([]string{"-C", root}, args...)...)
		command.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@x", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@x")
		if out, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n"), 0o644)
	run("init", "-q", "-b", "main")
	run("add", "-A")
	run("commit", "-q", "-m", "first")
	out, _ := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	os.MkdirAll(filepath.Join(root, ".compass", "context"), 0o755)
	os.WriteFile(filepath.Join(root, ".compass", "context", "current-state.md"), []byte("# current state\nCommit: "+
		strings.TrimSpace(string(out))+"\n\n## Objective\nShip the gate.\n\n## Just done\n- JUST-DONE-DETAIL\n\n## Next steps\n1. finish\n2. polish\n"), 0o644)
	run("add", "-A")
	run("commit", "-q", "-m", "state")
	if change != nil {
		change(root, run)
	}
	return root
}

func TestSessionStartGatesTheFullBrief(t *testing.T) {
	write := func(root, path, text string) {
		os.MkdirAll(filepath.Dir(filepath.Join(root, path)), 0o755)
		os.WriteFile(filepath.Join(root, path), []byte(text), 0o644)
	}
	pythonDir := t.TempDir()
	os.WriteFile(filepath.Join(pythonDir, "compass"), []byte("#!/home/u/.local/share/uv/tools/compass-cli/bin/python3\n"), 0o755)
	tests := []struct {
		name     string
		source   string
		change   func(root string, run func(...string))
		pathList string
		full     bool
	}{
		{name: "fresh start, clean and current", source: "startup", full: false},
		{name: "after /clear, clean and current", source: "clear", full: false},
		{name: "after compaction", source: "compact", full: true},
		{name: "resumed session", source: "resume", full: true},
		{name: "uncommitted edit", source: "startup", full: true, change: func(root string, run func(...string)) {
			write(root, "a.go", "package a // edited\n")
		}},
		{name: "untracked file", source: "startup", full: true, change: func(root string, run func(...string)) {
			write(root, "new.go", "package a\n")
		}},
		{name: "stale", source: "startup", full: true, change: func(root string, run func(...string)) {
			write(root, "b.go", "package b\n")
			run("add", "-A")
			run("commit", "-q", "-m", "more code")
		}},
		{name: "Python compass-cli shadows the binary", source: "startup", full: true, pathList: pythonDir},
		{name: "freshness unknown", source: "startup", full: true, change: func(root string, run func(...string)) {
			// Branch-local on a fresh history: current-state is ignored, never committed, and its
			// anchor names no commit.
			write(root, ".gitignore", ".compass/\n")
			run("checkout", "-q", "--orphan", "fresh")
			run("rm", "-r", "-q", "--cached", ".compass")
			run("add", "-A")
			run("commit", "-q", "-m", "branch-local")
			write(root, ".compass/context/current-state.md", "# current state\nCommit: deadbeef\n\n## Objective\nShip the gate.\n\n## Next steps\n1. finish\n")
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := briefRepo(t, test.change)
			data, err := SessionStart(Input{Cwd: root, Source: test.source}, test.pathList, "")
			if err != nil {
				t.Fatal(err)
			}
			context := decode(t, data).HookSpecificOutput.AdditionalContext
			if test.full {
				for _, want := range []string{"already loaded", "## Current state", "## Live global records"} {
					if !strings.Contains(context, want) {
						t.Errorf("full brief lacks %q:\n%s", want, context)
					}
				}
				return
			}
			for _, want := range []string{"Objective: Ship the gate.", "Next step: 1. finish", "run `compass resume`"} {
				if !strings.Contains(strings.ToLower(context), strings.ToLower(want)) {
					t.Errorf("stub lacks %q:\n%s", want, context)
				}
			}
			for _, unwanted := range []string{"already loaded", "JUST-DONE-DETAIL", "2. polish"} {
				if strings.Contains(context, unwanted) {
					t.Errorf("stub has %q:\n%s", unwanted, context)
				}
			}
			if len(context) > SessionStartStubMaxChars {
				t.Errorf("stub is %d chars, want at most %d", len(context), SessionStartStubMaxChars)
			}
		})
	}
}

// A git status that failed or timed out may hide uncommitted work, so it gets the full brief.
func TestNeedsFullBriefWhenStatusUnknown(t *testing.T) {
	current := compass.Brief{Freshness: "current: no code commits since it was written"}
	if needsFullBrief("startup", &current, "") {
		t.Error("clean and current should get the stub")
	}
	unknown := current
	unknown.StatusUnknown = true
	if !needsFullBrief("startup", &unknown, "") {
		t.Error("unknown status should get the full brief")
	}
}

func TestSessionStartIsSilentWithoutCompass(t *testing.T) {
	data, err := SessionStart(Input{Cwd: t.TempDir()}, "", "")
	if err != nil || data != nil {
		t.Errorf("want nothing, got %s %v", data, err)
	}
}

func TestStopBlocksOnceWhenStale(t *testing.T) {
	root := gitRepo(t)
	data, err := Stop(Input{Cwd: root})
	if err != nil {
		t.Fatal(err)
	}
	if out := decode(t, data); out.Decision != "block" || !strings.Contains(out.Reason, "1 commit(s) changed code") {
		t.Errorf("output: %s", data)
	}
	if data, _ := Stop(Input{Cwd: root, StopHookActive: true}); data != nil {
		t.Errorf("blocked twice: %s", data)
	}
	if data, _ := Stop(Input{Cwd: t.TempDir()}); data != nil {
		t.Errorf("blocked outside a compass project: %s", data)
	}
}

func TestPreCompactBlocksOnlyManualCompactionOncePerSession(t *testing.T) {
	root := gitRepo(t)
	stateDir := t.TempDir()
	if data, _ := PreCompact(Input{Cwd: root, Trigger: "auto", SessionID: "s1"}, stateDir); data != nil {
		t.Errorf("blocked auto compaction: %s", data)
	}
	data, _ := PreCompact(Input{Cwd: root, Trigger: "manual", SessionID: "s1"}, stateDir)
	if out := decode(t, data); out.Decision != "block" {
		t.Errorf("manual compaction not blocked: %s", data)
	}
	if data, _ := PreCompact(Input{Cwd: root, Trigger: "manual", SessionID: "s1"}, stateDir); data != nil {
		t.Errorf("blocked twice in one session: %s", data)
	}
}

func TestReadInputToleratesGarbage(t *testing.T) {
	input := ReadInput(strings.NewReader("not json"))
	if input.Cwd == "" {
		t.Error("cwd should fall back to the process's")
	}
}
