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
	// Hermetic: never read the developer's Claude Code config, nor warn because this runs inside it.
	t.Setenv("CLAUDECODE", "")
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
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
	// The unfilled template warns that next steps are missing, but doesn't fail check.
	if code, out, _ := call("", "check"); code != 0 || !strings.Contains(out, "next-steps-missing") {
		t.Errorf("check after init: %d\n%s", code, out)
	}
	os.WriteFile(".compass/context/current-state.md", []byte("# state\nCommit: "+head(t)+"\n## Next steps\n1. go\n"), 0o644)

	code, out, stderr := call("the body\n", "record", "decision", "--claim", "We ship Go", "--global", "--premise", "p1", "--body-file", "-")
	if code != 0 || strings.TrimSpace(out) != ".compass/decisions/0001-we-ship-go.md" {
		t.Fatalf("record: %d %q %s", code, out, stderr)
	}
	if strings.Contains(stderr, "only headings") {
		t.Errorf("record with a body warned it was empty: %s", stderr)
	}
	// Without a body, or with headings only, the agent is told to write the why before committing.
	for _, body := range []string{"", "## Context\n## Decision\n"} {
		code, out, stderr := call(body, "record", "goal", "--claim", "Ship it "+strings.Repeat("x", len(body)), "--body-file", "-")
		if code != 0 || !strings.Contains(stderr, strings.TrimSpace(out)+" has only headings") {
			t.Errorf("record with body %q: %d %q %q", body, code, out, stderr)
		}
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

// Superseding retires a claim, so record names it; dropping --global also stops delivery, so
// record says that too, and keeping --global keeps quiet about it.
func TestRecordWarnsWhatASupersedeRetires(t *testing.T) {
	inRepo(t)
	call("", "init", "--no-git-hook")
	call("why\n", "record", "decision", "--claim", "Amounts are cents", "--global", "--body-file", "-")
	code, _, stderr := call("why\n", "record", "decision", "--claim", "Amounts are cents; floats only inside percentages", "--supersedes", "0001", "--body-file", "-")
	if code != 0 || !strings.Contains(stderr, `0001-amounts-are-cents is retired, its claim no longer current: "Amounts are cents"`) ||
		!strings.Contains(stderr, "pass --global to keep it") {
		t.Errorf("supersede without --global: %d %q", code, stderr)
	}
	code, _, stderr = call("why\n", "record", "decision", "--claim", "Amounts are int64 cents", "--global", "--supersedes", "0002", "--body-file", "-")
	if code != 0 || !strings.Contains(stderr, `is retired, its claim no longer current: "Amounts are cents; floats only inside percentages"`) || strings.Contains(stderr, "--global") {
		t.Errorf("supersede of a local record: %d %q", code, stderr)
	}
	// 0001 is already retired: superseding it again would fork, so record points at the live one.
	code, _, stderr = call("why\n", "record", "decision", "--claim", "Cents again", "--supersedes", "0001", "--supersedes", "0001", "--body-file", "-")
	if code != 0 || strings.Count(stderr, "0001-amounts-are-cents was already superseded by 0002-") != 1 || strings.Contains(stderr, "is retired") {
		t.Errorf("supersede of a retired record: %d %q", code, stderr)
	}
}

// brokenClaudeConfig points CLAUDE_CONFIG_DIR at a config whose settings.json is a dangling symlink,
// as on ebay-flipper on 2026-10-28.
func brokenClaudeConfig(t *testing.T) {
	t.Helper()
	config := t.TempDir()
	if err := os.Symlink(config+"/gone/settings.json", config+"/settings.json"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_CONFIG_DIR", config)
}

func TestResumeWarnsOnBrokenDeliveryOnlyInClaudeCode(t *testing.T) {
	inRepo(t)
	if code, out, stderr := call("", "init", "--no-git-hook"); code != 0 {
		t.Fatalf("init: %d %s %s", code, out, stderr)
	}
	_, quietOut, _ := call("", "resume")
	brokenClaudeConfig(t)
	code, out, stderr := call("", "resume")
	if code != 0 || strings.Contains(stderr, "warning") || out != quietOut {
		t.Errorf("outside Claude Code: %d warned=%q", code, stderr)
	}
	t.Setenv("CLAUDECODE", "1")
	code, out, stderr = call("", "resume")
	if code != 0 || !strings.Contains(stderr, "compass: warning: ") || !strings.Contains(stderr, "is a symlink to") || !strings.Contains(stderr, "compass doctor") {
		t.Errorf("inside Claude Code: %d stderr=%q", code, stderr)
	}
	if out != quietOut {
		t.Errorf("the warning changed the brief on stdout")
	}
}

func TestPreCommitWarnsButDoesNotBlockOnBrokenDelivery(t *testing.T) {
	inRepo(t)
	if code, out, stderr := call("", "init", "--no-git-hook"); code != 0 {
		t.Fatalf("init: %d %s %s", code, out, stderr)
	}
	brokenClaudeConfig(t)
	t.Setenv("CLAUDECODE", "1")
	code, _, stderr := call("", "hook", "pre-commit")
	if code != 0 || !strings.Contains(stderr, "is a symlink to") {
		t.Errorf("pre-commit: %d %q", code, stderr)
	}
}

func TestDoctorNamesTheDanglingSettingsAndCountsLefthook(t *testing.T) {
	root := inRepo(t)
	if code, out, stderr := call("", "init", "--no-git-hook"); code != 0 {
		t.Fatalf("init: %d %s %s", code, out, stderr)
	}
	if err := os.WriteFile(root+"/lefthook.yml", []byte("pre-commit:\n  commands:\n    compass:\n      run: compass hook pre-commit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	brokenClaudeConfig(t)
	_, out, _ := call("", "doctor")
	if !strings.Contains(out, "problem: ") || !strings.Contains(out, "is a symlink to") || !strings.Contains(out, "Claude Code config: ") {
		t.Errorf("doctor on a dangling settings symlink: %s", out)
	}
	if strings.Contains(out, "pre-commit hook doesn't run compass") {
		t.Errorf("doctor ignored lefthook.yml: %s", out)
	}
}

func TestInitRefreshesTheAgentsBlockWithTheDoctorLine(t *testing.T) {
	root := inRepo(t)
	old := "<!-- compass:begin (managed by `compass init`; edits inside are overwritten) -->\nold text\n<!-- compass:end -->\n"
	if err := os.WriteFile(root+"/AGENTS.md", []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, out, stderr := call("", "init", "--no-git-hook"); code != 0 {
		t.Fatalf("init: %d %s %s", code, out, stderr)
	}
	data, _ := os.ReadFile(root + "/AGENTS.md")
	if !strings.Contains(string(data), "run `compass doctor` and tell the user") || strings.Contains(string(data), "old text") {
		t.Errorf("AGENTS.md block not refreshed: %s", data)
	}
}
