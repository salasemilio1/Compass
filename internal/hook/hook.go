// Package hook translates Claude Code hook events into compass operations. Every handler fails
// open: on any internal error it writes nothing and the caller exits 0, because a crashing hook
// must never block the agent.
package hook

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/salasemilio1/Compass/internal/compass"
)

// gitBudget bounds all of one hook's git calls together, inside the 10 s timeout hooks.json gives
// Claude Code; past it the brief degrades (git sections empty) instead of being killed.
const gitBudget = 6 * time.Second

func findRepo(cwd string) (*compass.Repo, error) {
	repo, err := compass.Find(cwd)
	if err == nil {
		repo.SetDeadline(time.Now().Add(gitBudget))
	}
	return repo, err
}

// SessionStartMaxChars keeps the brief under Claude Code's 10,000-character cap on SessionStart
// context; past it the model sees only a 2,000-character preview.
const SessionStartMaxChars = 9500

// Input is the subset of Claude Code's hook payload compass reads.
type Input struct {
	SessionID      string `json:"session_id"`
	Cwd            string `json:"cwd"`
	Source         string `json:"source"`
	Trigger        string `json:"trigger"`
	StopHookActive bool   `json:"stop_hook_active"`
}

// ReadInput decodes the hook payload; an empty or malformed one yields the process's cwd.
func ReadInput(stdin io.Reader) Input {
	var input Input
	data, _ := io.ReadAll(io.LimitReader(stdin, 1<<20))
	_ = json.Unmarshal(data, &input)
	if input.Cwd == "" {
		input.Cwd, _ = os.Getwd()
	}
	return input
}

type hookSpecificOutput struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext"`
}

type output struct {
	Decision           string              `json:"decision,omitempty"`
	Reason             string              `json:"reason,omitempty"`
	HookSpecificOutput *hookSpecificOutput `json:"hookSpecificOutput,omitempty"`
}

// SessionStartStubMaxChars bounds the short brief, preface included. It was 500 before the stub
// carried SearchRecordsLine (~125 chars); the line is added room, not taken from the objective.
const SessionStartStubMaxChars = 640

// SessionStart injects the resume brief: the full one when the agent is likely mid-task or the
// state can't be trusted (after compaction or a resumed session, with uncommitted changes, or
// when current-state is stale or its freshness unknown), else a stub that points to
// `compass resume`, so a session that may not need the rest doesn't pay for it. pluginRoot, when
// set, is used to name the right binary if a Python compass-cli shadows it on PATH.
func SessionStart(input Input, pathList, pluginRoot string) ([]byte, error) {
	repo, err := findRepo(input.Cwd)
	if err != nil {
		return nil, nil
	}
	brief, err := compass.Resume(repo)
	if err != nil {
		return nil, err
	}
	shadow := compass.ShadowingPython(pathList)
	full := needsFullBrief(input.Source, brief, shadow)
	var preface strings.Builder
	if full {
		preface.WriteString("compass: this is the resume brief (`compass resume`), already loaded; don't re-run it.\n")
	} else {
		preface.WriteString("compass: current-state is current and the tree is clean; this is its short form.\n")
	}
	if strings.HasPrefix(brief.Freshness, "STALE") && input.Source == "compact" {
		preface.WriteString("Context was just compacted and current-state.md is stale: rewrite it now, before continuing.\n")
	}
	if shadow != "" {
		binary := "compass"
		if pluginRoot != "" {
			binary = `"` + filepath.Join(pluginRoot, "bin", "compass") + `"`
		}
		fmt.Fprintf(&preface, "Note: `compass` on PATH is the old Python compass-cli (%s); run %s instead, and tell the user to remove it (uv tool uninstall compass-cli).\n", shadow, binary)
	}
	var text string
	if full {
		text = preface.String() + "\n" + brief.Text(SessionStartMaxChars-preface.Len()-1)
	} else {
		text = preface.String() + brief.Stub(max(SessionStartStubMaxChars-preface.Len(), 0))
	}
	return json.Marshal(output{HookSpecificOutput: &hookSpecificOutput{HookEventName: "SessionStart", AdditionalContext: text}})
}

// needsFullBrief reports whether the agent should get the whole brief: it is likely mid-task
// (compaction, a resumed session, uncommitted or unknown work), current-state can't be trusted, or
// a Python compass-cli shadows the binary, whose note would crowd the stub.
func needsFullBrief(source string, brief *compass.Brief, shadow string) bool {
	return source == "compact" || source == "resume" || len(brief.Status) > 0 || brief.StatusUnknown ||
		!brief.Current() || shadow != ""
}

// staleness returns a non-empty reason when commits changed code after current-state was written.
func staleness(cwd string) string {
	repo, err := findRepo(cwd)
	if err != nil {
		return ""
	}
	state, err := repo.ReadCurrentState()
	if err != nil || state == nil {
		return ""
	}
	freshness := repo.Freshness(state)
	if !freshness.Known || freshness.CodeCommits == 0 {
		return ""
	}
	return fmt.Sprintf("compass: %d commit(s) changed code since .compass/context/current-state.md was written. "+
		"Git and the user's current prompt win over it. Rewrite it whole (objective and next steps; at most 80 lines), "+
		"set its `Commit:` line to the output of `git rev-parse HEAD`, and commit it, so the next agent can resume.", freshness.CodeCommits)
}

// Stop blocks ending the turn once when current-state is stale. stop_hook_active means this turn
// already continued because of a stop hook, so it never blocks twice.
func Stop(input Input) ([]byte, error) {
	if input.StopHookActive {
		return nil, nil
	}
	reason := staleness(input.Cwd)
	if reason == "" {
		return nil, nil
	}
	return json.Marshal(output{Decision: "block", Reason: reason})
}

// PreCompact blocks a manual /compact once per session when current-state is stale; the reason is
// shown to the user, who can ask for the update or run /compact again. It never blocks automatic
// compaction: that would leave a full context with nowhere to go. After any compaction, the
// SessionStart brief (source "compact") asks for the rewrite instead.
func PreCompact(input Input, stateDir string) ([]byte, error) {
	if input.Trigger != "manual" {
		return nil, nil
	}
	reason := staleness(input.Cwd)
	if reason == "" {
		return nil, nil
	}
	if stateDir != "" && input.SessionID != "" {
		marker := filepath.Join(stateDir, "precompact-"+safeName(input.SessionID))
		if _, err := os.Stat(marker); err == nil {
			return nil, nil
		}
		if err := os.MkdirAll(stateDir, 0o755); err == nil {
			_ = os.WriteFile(marker, nil, 0o644)
		}
	}
	return json.Marshal(output{Decision: "block", Reason: reason + " (Run /compact again to compact anyway.)"})
}

func safeName(text string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' {
			return r
		}
		return '_'
	}, text)
}
