package compass

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// PathEntry is one `compass` executable found on PATH.
type PathEntry struct {
	Path string `json:"path"`
	// Kind is "go" (this compass), "python" (the v2 compass-cli) or "other".
	Kind    string `json:"kind"`
	Version string `json:"version,omitempty"`
}

// FindOnPath lists every executable named compass on PATH, in lookup order.
func FindOnPath(pathList string) []PathEntry {
	var entries []PathEntry
	seen := map[string]bool{}
	for _, dir := range filepath.SplitList(pathList) {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, "compass")
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
			continue
		}
		real, err := filepath.EvalSymlinks(candidate)
		if err != nil || seen[real] {
			continue
		}
		seen[real] = true
		entries = append(entries, PathEntry{Path: candidate, Kind: classify(candidate)})
	}
	return entries
}

// classify reads only the file's first bytes, so the SessionStart hook can afford it: a python
// shebang is the v2 compass-cli; anything else is resolved by running it (doctor only).
func classify(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return "other"
	}
	defer file.Close()
	head := make([]byte, 256)
	n, _ := file.Read(head)
	head = head[:n]
	if bytes.HasPrefix(head, []byte("#!")) {
		firstLine, _, _ := bytes.Cut(head, []byte("\n"))
		if bytes.Contains(firstLine, []byte("python")) {
			return "python"
		}
	}
	return "unknown"
}

// ShadowingPython returns the path of a Python compass-cli found on PATH before any other
// compass, or "". Cheap: no process is started.
func ShadowingPython(pathList string) string {
	entries := FindOnPath(pathList)
	if len(entries) > 0 && entries[0].Kind == "python" {
		return entries[0].Path
	}
	return ""
}

// DoctorReport is the result of `compass doctor`.
type DoctorReport struct {
	Version    string      `json:"version"`
	Executable string      `json:"executable"`
	OnPath     []PathEntry `json:"on_path"`
	// ClaudeConfigDir is the Claude Code config dir the delivery checks read.
	ClaudeConfigDir string   `json:"claude_config_dir"`
	Problems        []string `json:"problems"`
	Notes           []string `json:"notes"`
}

// Doctor inspects the installation and, when start is inside a compass project, the project.
// getenv supplies PATH, CLAUDECODE, CLAUDE_CONFIG_DIR and HOME, so tests never read the real ones.
func Doctor(start string, getenv func(string) string) *DoctorReport {
	pathList := getenv("PATH")
	report := &DoctorReport{Version: Version}
	report.Executable, _ = os.Executable()
	report.OnPath = FindOnPath(pathList)
	for i := range report.OnPath {
		entry := &report.OnPath[i]
		if entry.Kind != "unknown" {
			continue
		}
		entry.Kind = "other"
		if out, err := runQuiet(entry.Path, "version"); err == nil && strings.Contains(out, fmt.Sprintf("(format %d)", FormatVersion)) {
			entry.Kind = "go"
			entry.Version = strings.TrimSpace(out)
		}
	}
	switch {
	case len(report.OnPath) == 0:
		report.Problems = append(report.Problems, "no `compass` on PATH: agents outside Claude Code can't run it (the git hook falls back to the Claude Code plugin's copy). Install a release binary or `go install` it")
	case report.OnPath[0].Kind == "python":
		report.Problems = append(report.Problems, fmt.Sprintf("%s is the Python compass-cli (v2) and shadows this one; remove it: uv tool uninstall compass-cli", report.OnPath[0].Path))
	case report.OnPath[0].Kind != "go":
		report.Problems = append(report.Problems, fmt.Sprintf("%s is first on PATH and is not this compass", report.OnPath[0].Path))
	}
	for _, entry := range report.OnPath[min(1, len(report.OnPath)):] {
		if entry.Kind == "python" {
			report.Notes = append(report.Notes, fmt.Sprintf("%s is the Python compass-cli (v2), shadowed; remove it: uv tool uninstall compass-cli", entry.Path))
		}
	}
	if _, err := exec.LookPath("git"); err != nil {
		report.Problems = append(report.Problems, "git is not on PATH: anchors, freshness and the resume brief's git sections need it")
	}

	repo, err := Find(start)
	dirs := []string{start}
	if err == nil {
		dirs = append(dirs, repo.Root)
	}
	delivery := Delivery(ClaudeConfigDir(getenv), dirs, getenv("CLAUDECODE") == "1")
	report.ClaudeConfigDir = delivery.ConfigDir
	report.Problems = append(report.Problems, delivery.Problems...)
	report.Notes = append(report.Notes, delivery.Notes...)
	if err != nil {
		report.Notes = append(report.Notes, "not inside a compass project")
		return report
	}
	report.Notes = append(report.Notes, "project: "+repo.Root)
	findings, err := Check(repo, CheckOptions{})
	switch {
	case err != nil:
		report.Problems = append(report.Problems, "compass check failed to run: "+err.Error())
	case HasErrors(findings):
		report.Problems = append(report.Problems, "compass check reports errors (run it for details)")
	}
	if repo.IsGit() && !PreCommitRunsCompass(repo) {
		report.Notes = append(report.Notes, "the git pre-commit hook doesn't run compass (compass init installs it when none exists)")
	}
	return report
}

// PreCommitRunsCompass reports whether a commit runs `compass hook pre-commit`: from the git hook
// itself, or from a lefthook config (lefthook's own git hook script doesn't name compass).
func PreCommitRunsCompass(repo *Repo) bool {
	if path, err := repo.Git("rev-parse", "--git-path", "hooks/pre-commit"); err == nil {
		if !filepath.IsAbs(path) {
			path = filepath.Join(repo.Root, path)
		}
		if data, err := os.ReadFile(path); err == nil && runsCompassPreCommit(string(data)) {
			return true
		}
	}
	for _, name := range []string{"lefthook.yml", "lefthook.yaml", ".lefthook.yml", ".lefthook.yaml"} {
		if data, err := os.ReadFile(filepath.Join(repo.Root, name)); err == nil && lefthookRunsCompass(string(data)) {
			return true
		}
	}
	return false
}

// runsCompassPreCommit looks for the command on a line that isn't a shell or YAML comment.
func runsCompassPreCommit(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		if trimmed := strings.TrimSpace(line); !strings.HasPrefix(trimmed, "#") && strings.Contains(trimmed, "compass hook pre-commit") {
			return true
		}
	}
	return false
}

// lefthookRunsCompass finds the command inside the top-level pre-commit: section only, so a
// commit-msg or pre-push entry doesn't count.
func lefthookRunsCompass(text string) bool {
	var section []string
	inPreCommit := false
	for _, line := range strings.Split(text, "\n") {
		if line != "" && line[0] != ' ' && line[0] != '\t' && !strings.HasPrefix(line, "#") {
			inPreCommit = strings.HasPrefix(line, "pre-commit:")
			continue
		}
		if inPreCommit {
			section = append(section, line)
		}
	}
	return runsCompassPreCommit(strings.Join(section, "\n"))
}

func runQuiet(path string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, args...).Output()
	return string(out), err
}
