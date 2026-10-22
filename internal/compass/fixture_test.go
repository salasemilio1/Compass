package compass

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fixture is a throwaway git repository with a .compass/ directory.
type fixture struct {
	t    *testing.T
	repo *Repo
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	root := t.TempDir()
	f := &fixture{t: t, repo: Open(root)}
	f.git("init", "-q", "-b", "main")
	f.git("config", "user.email", "test@example.com")
	f.git("config", "user.name", "test")
	f.git("config", "commit.gpgsign", "false")
	if err := os.MkdirAll(f.repo.Dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *fixture) git(args ...string) string {
	f.t.Helper()
	command := exec.Command("git", append([]string{"-C", f.repo.Root}, args...)...)
	command.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	out, err := command.CombinedOutput()
	if err != nil {
		f.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// write creates path (relative to the repo root) with content.
func (f *fixture) write(path, content string) {
	f.t.Helper()
	full := filepath.Join(f.repo.Root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) read(path string) string {
	f.t.Helper()
	data, err := os.ReadFile(filepath.Join(f.repo.Root, filepath.FromSlash(path)))
	if err != nil {
		f.t.Fatal(err)
	}
	return string(data)
}

func (f *fixture) commit(message string) string {
	f.t.Helper()
	f.git("add", "-A")
	f.git("commit", "-q", "--allow-empty", "-m", message)
	return f.git("rev-parse", "HEAD")
}

// state writes a valid current-state.md anchored at anchor.
func (f *fixture) state(anchor string) {
	f.t.Helper()
	f.write(".compass/context/current-state.md", "# current state\n\nDate: 2026-10-21\nCommit: "+anchor+
		"\n\n## Objective\nShip it.\n\n## Next steps\n1. Next.\n")
}

func (f *fixture) record(path, frontmatter string) {
	f.t.Helper()
	f.write(".compass/"+path, "---\n"+frontmatter+"\n---\n# body\nwhy it holds\n")
}

// codes returns the finding codes, for compact assertions.
func codes(findings []Finding) []string {
	var out []string
	for _, finding := range findings {
		out = append(out, string(finding.Severity)+":"+finding.Code)
	}
	return out
}

func hasCode(findings []Finding, code string) bool {
	for _, finding := range findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}

func (f *fixture) check(options CheckOptions) []Finding {
	f.t.Helper()
	findings, err := Check(f.repo, options)
	if err != nil {
		f.t.Fatal(err)
	}
	return findings
}

// clean brings the fixture to a passing state: index regenerated, AGENTS.md block present.
func (f *fixture) clean() {
	f.t.Helper()
	if _, err := WriteIndex(f.repo); err != nil {
		f.t.Fatal(err)
	}
	if _, err := UpsertAgentsBlock(f.repo.Root); err != nil {
		f.t.Fatal(err)
	}
}
