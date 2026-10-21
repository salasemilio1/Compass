package hook

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
