package compass

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CurrentStatePath is current-state.md's path relative to .compass/.
const CurrentStatePath = "context/current-state.md"

// ErrNoCompass means no .compass/ directory was found at or above the start directory.
var ErrNoCompass = errors.New("no .compass/ directory here or in any parent (run `compass init`)")

// Repo locates a project: Root is the directory holding .compass/, Dir is .compass/ itself.
type Repo struct {
	Root string
	Dir  string
	// gitTimeout bounds each git call, so nothing hangs on a wedged repository.
	gitTimeout time.Duration
	// deadline, when set, bounds all git calls together: a hook makes a dozen of them under one
	// timeout from Claude Code.
	deadline time.Time
	// staged, when set, is the set of paths in git's index: pre-commit checks that paths exist in
	// what is being committed, not in the working tree.
	staged map[string]bool
}

// exists reports whether a path relative to the project root exists: in the staged tree during
// pre-commit, on disk otherwise. A directory exists when any staged file is under it.
func (r *Repo) exists(path string) bool {
	path = filepath.ToSlash(filepath.Clean(path))
	if r.staged == nil {
		_, err := os.Stat(filepath.Join(r.Root, filepath.FromSlash(path)))
		return err == nil
	}
	if r.staged[path] {
		return true
	}
	for staged := range r.staged {
		if strings.HasPrefix(staged, path+"/") {
			return true
		}
	}
	return false
}

// SetDeadline bounds every later git call by t; calls past it fail fast and the caller degrades.
func (r *Repo) SetDeadline(t time.Time) { r.deadline = t }

// Find walks up from start to the nearest directory containing .compass/, stopping at a git
// root so a nested repository never picks up an outer project's memory.
func Find(start string) (*Repo, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return nil, err
	}
	for {
		if info, err := os.Stat(filepath.Join(dir, ".compass")); err == nil && info.IsDir() {
			return Open(dir), nil
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return nil, ErrNoCompass
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, ErrNoCompass
		}
		dir = parent
	}
}

// Open returns the Repo rooted at root without checking that .compass/ exists.
func Open(root string) *Repo {
	return &Repo{Root: root, Dir: filepath.Join(root, ".compass"), gitTimeout: 5 * time.Second}
}

// Name is the project's directory name.
func (r *Repo) Name() string { return filepath.Base(r.Root) }

// GitError is a failed git invocation, with git's own stderr.
type GitError struct {
	Args   []string
	Stderr string
	Err    error
}

func (e *GitError) Error() string {
	message := strings.TrimSpace(e.Stderr)
	if message == "" {
		message = e.Err.Error()
	}
	return fmt.Sprintf("git %s: %s", strings.Join(e.Args, " "), message)
}

func (e *GitError) Unwrap() error { return e.Err }

// Git runs git in the project root and returns trimmed stdout.
func (r *Repo) Git(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.gitTimeout)
	defer cancel()
	if !r.deadline.IsZero() {
		var cancelDeadline context.CancelFunc
		ctx, cancelDeadline = context.WithDeadline(ctx, r.deadline)
		defer cancelDeadline()
	}
	command := exec.CommandContext(ctx, "git", append([]string{"-C", r.Root}, args...)...)
	// English messages: LookupCommit tells an ambiguous hash from a missing one by git's stderr.
	command.Env = append(os.Environ(), "LC_ALL=C")
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return "", &GitError{Args: args, Stderr: stderr.String(), Err: err}
	}
	return strings.TrimRight(stdout.String(), "\n"), nil
}

// IsGit reports whether the project root is inside a git work tree.
func (r *Repo) IsGit() bool {
	out, err := r.Git("rev-parse", "--is-inside-work-tree")
	return err == nil && out == "true"
}

// CommitExists reports whether ref names a commit in this repository.
func (r *Repo) CommitExists(ref string) bool {
	_, err := r.Git("cat-file", "-e", ref+"^{commit}")
	return err == nil
}

// CommitLookup is what git says about a hash compass was given.
type CommitLookup int

const (
	// CommitFound means the hash names exactly one commit.
	CommitFound CommitLookup = iota
	// CommitAmbiguous means a short hash matches several commits: real, but unusable as given.
	CommitAmbiguous
	// CommitMissing means no commit matches in a full clone: an invented or foreign hash.
	CommitMissing
	// CommitMissingShallow means no commit matches, but this is a shallow clone that may lack it.
	CommitMissingShallow
)

// LookupCommit resolves hash, separating the ways it can fail to name a commit.
func (r *Repo) LookupCommit(hash string) CommitLookup {
	_, err := r.Git("rev-parse", "--verify", hash+"^{commit}")
	if err == nil {
		return CommitFound
	}
	var gitErr *GitError
	if errors.As(err, &gitErr) && strings.Contains(gitErr.Stderr, "is ambiguous") {
		return CommitAmbiguous
	}
	if shallow, _ := r.Git("rev-parse", "--is-shallow-repository"); shallow == "true" {
		return CommitMissingShallow
	}
	return CommitMissing
}

// Head returns the full hash of HEAD, or "" in a repository with no commits.
func (r *Repo) Head() string {
	out, err := r.Git("rev-parse", "--verify", "-q", "HEAD")
	if err != nil {
		return ""
	}
	return out
}
