package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// inRepo runs the test from a fresh git repository with one commit.
func inRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, args := range [][]string{{"init", "-q", "-b", "main"}, {"commit", "-q", "--allow-empty", "-m", "first"}} {
		command := exec.Command("git", append([]string{"-C", root}, args...)...)
		command.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@x",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@x")
		if out, err := command.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
	}
	t.Chdir(root)
	return root
}

func call(stdin string, args ...string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := run(args, strings.NewReader(stdin), &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestExitCodes(t *testing.T) {
	inRepo(t)
	cases := []struct {
		args []string
		code int
	}{
		{nil, 2},
		{[]string{"nope"}, 2},
		{[]string{"check", "--bogus"}, 2},
		{[]string{"check", "extra"}, 2},
		{[]string{"check"}, 1}, // no .compass/ yet
		{[]string{"version"}, 0},
		{[]string{"hook", "stop"}, 0},
		{[]string{"hook", "pre-commit"}, 0}, // no .compass/: nothing to check
		{[]string{"record"}, 2},
	}
	for _, c := range cases {
		if code, _, stderr := call("", c.args...); code != c.code {
			t.Errorf("compass %v: exit %d, want %d (%s)", c.args, code, c.code, stderr)
		}
	}
}

func TestInitRecordCheckResumeFlow(t *testing.T) {
	inRepo(t)
	if code, out, stderr := call("", "init", "--no-git-hook"); code != 0 {
		t.Fatalf("init: %d %s %s", code, out, stderr)
	}
	// The template's placeholder file in play fails check until the agent fills it in.
	if code, out, _ := call("", "check"); code != 1 || !strings.Contains(out, "file-in-play-missing") {
		t.Errorf("check after init: %d\n%s", code, out)
	}
	os.WriteFile(".compass/context/current-state.md", []byte("# state\nCommit: "+head(t)+"\n## Next steps\n1. go\n"), 0o644)

	code, out, stderr := call("the body\n", "record", "decision", "--claim", "We ship Go", "--global", "--premise", "p1", "--body-file", "-")
	if code != 0 || strings.TrimSpace(out) != ".compass/decisions/0001-we-ship-go.md" {
		t.Fatalf("record: %d %q %s", code, out, stderr)
	}
	if code, _, stderr := call("", "record", "decision", "--claim", "x", "--supersedes", "0099"); code != 2 || !strings.Contains(stderr, "not a record") {
		t.Errorf("bad supersedes: %d %s", code, stderr)
	}

	code, out, _ = call("", "check", "--json")
	var result struct {
		Findings []struct{ Code, Severity string }
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil || code != 0 {
		t.Fatalf("check --json: %d %v\n%s", code, err, out)
	}

	code, out, _ = call("", "resume")
	if code != 0 || !strings.Contains(out, "0001-we-ship-go — We ship Go") || !strings.Contains(out, "1. go") {
		t.Errorf("resume: %d\n%s", code, out)
	}
	code, out, _ = call("", "resume", "--json")
	var brief struct {
		Claims []string `json:"global_claims"`
	}
	if err := json.Unmarshal([]byte(out), &brief); err != nil || code != 0 || len(brief.Claims) != 1 {
		t.Errorf("resume --json: %d %v %s", code, err, out)
	}
}

func head(t *testing.T) string {
	out, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

func TestHookSessionStartPrintsJSON(t *testing.T) {
	root := inRepo(t)
	call("", "init", "--no-git-hook")
	code, out, _ := call(`{"cwd":"`+root+`","source":"startup"}`, "hook", "session-start")
	var decoded map[string]map[string]string
	if code != 0 || json.Unmarshal([]byte(out), &decoded) != nil || !strings.Contains(decoded["hookSpecificOutput"]["additionalContext"], "compass resume") {
		t.Errorf("%d %s", code, out)
	}
}
