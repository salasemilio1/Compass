package compass

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Version is the binary's version; the plugin manifests carry the same string and a test keeps
// them in step. FormatVersion is the .compass/ format it reads and writes.
const (
	Version       = "0.3.0"
	FormatVersion = 3
)

const currentStateTemplate = `# current state

Date: %s
Commit: %s

## Objective
<what this work is for, in one or two lines>

## Just done
- <what landed, pointing at commits rather than restating them>

## Files in play
- ` + "`<path>`" + ` — <why it matters right now>

## Next steps
1. <the next concrete action>
`

// gitPreCommit uses the Go compass on PATH, else the Claude Code plugin's copy, so a plugin-only
// install is still checked. An older Python compass-cli first on PATH would fail every commit with
// "no such command", hence the version probe.
const gitPreCommit = `#!/bin/sh
# Installed by ` + "`compass init`" + `: regenerates .compass/INDEX.md and runs ` + "`compass check`" + ` on what is staged.
compass=
if compass version 2>/dev/null | grep -q '(format 3)'; then
  compass=compass
else
  for candidate in "${CLAUDE_CONFIG_DIR:-$HOME/.claude}"/plugins/cache/compass/compass/*/bin/compass; do
    [ -x "$candidate" ] && [ ! -e "${candidate%/bin/compass}/.orphaned_at" ] && compass=$candidate
  done
fi
if [ -z "$compass" ]; then
  echo "compass: no Go compass on PATH or in the Claude Code plugin; skipping .compass/ checks (see: compass doctor)" >&2
  exit 0
fi
exec "$compass" hook pre-commit
`

// InitOptions configures `compass init`.
type InitOptions struct {
	// BranchLocal gitignores all of .compass/ (a guest in someone else's repo); otherwise only
	// .compass/scratch/ is ignored.
	BranchLocal bool
	NoGitHook   bool
	Today       time.Time
}

// Init scaffolds .compass/ into root, or brings an existing one up to date. It never overwrites a
// file the user owns: it creates what is missing and refreshes only compass's managed block.
// It returns one line per action taken or skipped.
func Init(root string, options InitOptions) ([]string, error) {
	repo := Open(root)
	var actions []string
	for _, category := range RecordCategories {
		if err := os.MkdirAll(filepath.Join(repo.Dir, category), 0o755); err != nil {
			return actions, err
		}
	}
	actions = append(actions, "ensured .compass/{"+strings.Join(RecordCategories, ",")+"}/")

	statePath := filepath.Join(repo.Dir, filepath.FromSlash(CurrentStatePath))
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		head := repo.Head()
		text := fmt.Sprintf(currentStateTemplate, options.Today.Format("2006-01-02"), head)
		if head == "" {
			text = strings.Replace(text, "Commit: \n", "", 1)
		}
		if err := writeFileAtomic(statePath, []byte(text)); err != nil {
			return actions, err
		}
		actions = append(actions, "wrote .compass/"+CurrentStatePath+" (a template: fill it in)")
	}

	if _, err := WriteIndex(repo); err != nil {
		return actions, err
	}
	actions = append(actions, "generated .compass/"+IndexPath)

	ignore := ".compass/scratch/"
	if options.BranchLocal {
		ignore = ".compass/"
	}
	if added, err := ensureLine(filepath.Join(root, ".gitignore"), ignore); err != nil {
		return actions, err
	} else if added {
		actions = append(actions, "added "+ignore+" to .gitignore")
	}

	if changed, err := UpsertAgentsBlock(root); err != nil {
		return actions, err
	} else if changed {
		actions = append(actions, "wrote the compass block in AGENTS.md")
	}
	if data, err := os.ReadFile(filepath.Join(root, "AGENTS.md")); err == nil && legacyAgentsMarker.Match(data) {
		actions = append(actions, "note: AGENTS.md still has v2 compass instructions (the compass-spec-version marker and text after it); remove them by hand")
	}
	if changed, err := EnsureClaudePointer(root); err != nil {
		return actions, err
	} else if changed {
		actions = append(actions, "added @AGENTS.md to CLAUDE.md")
	}

	if !options.NoGitHook && repo.IsGit() {
		action, err := installGitHook(repo)
		if err != nil {
			return actions, err
		}
		actions = append(actions, action)
	}
	return actions, nil
}

func installGitHook(repo *Repo) (string, error) {
	path, err := repo.Git("rev-parse", "--git-path", "hooks/pre-commit")
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(repo.Root, path)
	}
	if data, err := os.ReadFile(path); err == nil {
		switch {
		case string(data) == gitPreCommit:
			return "git pre-commit hook already runs compass", nil
		case !strings.HasPrefix(string(data), "#!/bin/sh\n# Installed by `compass init`"):
			if strings.Contains(string(data), "compass hook pre-commit") {
				return "git pre-commit hook already runs compass", nil
			}
			return "skipped git pre-commit hook: one exists; add `compass hook pre-commit` to it", nil
		}
		// compass's own hook from an older version: refresh it below.
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(gitPreCommit), 0o755); err != nil {
		return "", err
	}
	return "installed git pre-commit hook (" + path + ")", nil
}

// ensureLine appends line to the file unless an identical line is already there.
func ensureLine(path, line string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	for _, existing := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(existing) == line {
			return false, nil
		}
	}
	text := string(data)
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	return true, writeFileAtomic(path, []byte(text+line+"\n"))
}

// PreCommit is the git pre-commit step. It checks .compass/ as staged, not as in the working
// tree, so an unstaged fix can't hide a broken staged record and an untracked record can't land in
// the index: it exports the staged files, regenerates INDEX.md from them, stages that, and checks
// the export. A gitignored .compass/ (branch-local mode) is checked in the working tree instead.
// It returns the findings; the caller fails the commit on errors.
func PreCommit(repo *Repo) ([]Finding, error) {
	index := ".compass/" + IndexPath
	if _, err := repo.Git("check-ignore", "-q", index); err == nil {
		if _, err := WriteIndex(repo); err != nil {
			return nil, err
		}
		return Check(repo, CheckOptions{})
	}
	files, err := repo.Git("ls-files", "-z", "--cached", "--", ".compass")
	if err != nil || strings.Trim(files, "\x00") == "" {
		return nil, err
	}
	export, err := os.MkdirTemp("", "compass-staged-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(export)
	command := exec.Command("git", "-C", repo.Root, "checkout-index", "--stdin", "-z", "--prefix="+export+string(filepath.Separator))
	command.Stdin = strings.NewReader(files)
	if out, err := command.CombinedOutput(); err != nil {
		return nil, &GitError{Args: []string{"checkout-index"}, Stderr: string(out), Err: err}
	}
	staged := &Repo{Root: repo.Root, Dir: filepath.Join(export, ".compass"), gitTimeout: repo.gitTimeout, deadline: repo.deadline}
	if changed, err := WriteIndex(staged); err != nil {
		return nil, err
	} else if changed {
		data, err := os.ReadFile(filepath.Join(staged.Dir, IndexPath))
		if err != nil {
			return nil, err
		}
		if err := writeFileAtomic(filepath.Join(repo.Dir, IndexPath), data); err != nil {
			return nil, err
		}
		if _, err := repo.Git("add", "--", index); err != nil {
			return nil, err
		}
	}
	return Check(staged, CheckOptions{})
}
