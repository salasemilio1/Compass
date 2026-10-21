// Command compass is the CLI and Claude Code hook handler for the .compass/ agent-memory format.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/salasemilio1/Compass/internal/compass"
	"github.com/salasemilio1/Compass/internal/hook"
)

const usage = `compass — project memory a fresh agent can resume from

Usage:
  compass resume [--json] [--max-chars N]   print the resume brief: current-state, live global decisions, uncommitted work
  compass check [--fix] [--require-fresh] [--json]
                                            validate .compass/ (exit 1 on errors)
  compass index                             regenerate .compass/INDEX.md
  compass record <decision|goal|experiment> --claim TEXT [--slug S] [--global]
                 [--supersedes ID]... [--premise P]... [--ref R]... [--body-file F|-] [--json]
                                            write a dated record and regenerate the index
  compass init [--branch-local] [--no-git-hook]
                                            scaffold .compass/, the AGENTS.md block and the git hook (idempotent)
  compass doctor [--json]                   find install problems, such as an old compass shadowing this one
  compass hook <session-start|stop|pre-compact|pre-commit>
                                            hook handlers (Claude Code reads JSON on stdin)
  compass version

Exit status: 0 ok, 1 findings or failure, 2 usage error.
`

// usageError is a malformed command line; it exits 2.
type usageError struct{ message string }

func (e *usageError) Error() string { return e.message }

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	command, rest := args[0], args[1:]
	var err error
	switch command {
	case "resume":
		err = runResume(rest, stdout)
	case "check":
		err = runCheck(rest, stdout)
	case "index":
		err = runIndex(rest, stdout)
	case "record":
		err = runRecord(rest, stdin, stdout)
	case "init":
		err = runInit(rest, stdout)
	case "doctor":
		err = runDoctor(rest, stdout)
	case "hook":
		return runHook(rest, stdin, stdout, stderr)
	case "version", "--version", "-v":
		fmt.Fprintf(stdout, "compass %s (format %d)\n", compass.Version, compass.FormatVersion)
		return 0
	case "help", "--help", "-h":
		fmt.Fprint(stdout, usage)
		return 0
	default:
		err = &usageError{fmt.Sprintf("unknown command %q", command)}
	}
	var usageErr *usageError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &usageErr):
		fmt.Fprintf(stderr, "compass: %s\n\n%s", err, usage)
		return 2
	case errors.Is(err, failure{}):
		return 1
	default:
		fmt.Fprintf(stderr, "compass: %s\n", err)
		return 1
	}
}

// failure exits 1 after the command already reported why on stdout.
type failure struct{}

func (failure) Error() string { return "failed" }

func newFlags(name string) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	return flags
}

func parse(flags *flag.FlagSet, args []string) error {
	if err := flags.Parse(args); err != nil {
		return &usageError{err.Error()}
	}
	if flags.NArg() > 0 {
		return &usageError{fmt.Sprintf("%s: unexpected argument %q", flags.Name(), flags.Arg(0))}
	}
	return nil
}

func findRepo() (*compass.Repo, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return compass.Find(cwd)
}

func writeJSON(stdout io.Writer, value any) error {
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func runResume(args []string, stdout io.Writer) error {
	flags := newFlags("resume")
	asJSON := flags.Bool("json", false, "")
	maxChars := flags.Int("max-chars", 0, "")
	if err := parse(flags, args); err != nil {
		return err
	}
	repo, err := findRepo()
	if err != nil {
		return err
	}
	brief, err := compass.Resume(repo)
	if err != nil {
		return err
	}
	if *asJSON {
		return writeJSON(stdout, brief)
	}
	_, err = fmt.Fprint(stdout, brief.Text(*maxChars))
	return err
}

func runCheck(args []string, stdout io.Writer) error {
	flags := newFlags("check")
	fix := flags.Bool("fix", false, "")
	requireFresh := flags.Bool("require-fresh", false, "")
	asJSON := flags.Bool("json", false, "")
	if err := parse(flags, args); err != nil {
		return err
	}
	repo, err := findRepo()
	if err != nil {
		return err
	}
	var changes []string
	if *fix {
		if changes, err = compass.Fix(repo); err != nil {
			return err
		}
	}
	findings, err := compass.Check(repo, compass.CheckOptions{RequireFresh: *requireFresh})
	if err != nil {
		return err
	}
	if *asJSON {
		if err := writeJSON(stdout, struct {
			Fixed    []string          `json:"fixed"`
			Findings []compass.Finding `json:"findings"`
		}{nonNil(changes), nonNilFindings(findings)}); err != nil {
			return err
		}
	} else {
		for _, change := range changes {
			fmt.Fprintln(stdout, "fixed: "+change)
		}
		for _, finding := range findings {
			fmt.Fprintln(stdout, finding.String())
		}
		if len(findings) == 0 {
			fmt.Fprintln(stdout, "ok")
		}
	}
	if compass.HasErrors(findings) {
		return failure{}
	}
	return nil
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func nonNilFindings(values []compass.Finding) []compass.Finding {
	if values == nil {
		return []compass.Finding{}
	}
	return values
}

func runIndex(args []string, stdout io.Writer) error {
	flags := newFlags("index")
	if err := parse(flags, args); err != nil {
		return err
	}
	repo, err := findRepo()
	if err != nil {
		return err
	}
	changed, err := compass.WriteIndex(repo)
	if err != nil {
		return err
	}
	if changed {
		fmt.Fprintln(stdout, "regenerated .compass/"+compass.IndexPath)
	}
	return nil
}

// listFlag collects a repeatable string flag.
type listFlag []string

func (l *listFlag) String() string     { return strings.Join(*l, ",") }
func (l *listFlag) Set(v string) error { *l = append(*l, v); return nil }

func runRecord(args []string, stdin io.Reader, stdout io.Writer) error {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return &usageError{"record: first argument is the kind: decision, goal or experiment"}
	}
	request := compass.RecordRequest{Kind: args[0], Date: time.Now().Format("2006-01-02")}
	flags := newFlags("record")
	flags.StringVar(&request.Claim, "claim", "", "")
	flags.StringVar(&request.Slug, "slug", "", "")
	flags.BoolVar(&request.Global, "global", false, "")
	var supersedes, premises, refs listFlag
	flags.Var(&supersedes, "supersedes", "")
	flags.Var(&premises, "premise", "")
	flags.Var(&refs, "ref", "")
	bodyFile := flags.String("body-file", "", "")
	asJSON := flags.Bool("json", false, "")
	if err := parse(flags, args[1:]); err != nil {
		return err
	}
	request.Supersedes, request.Premises, request.Refs = supersedes, premises, refs
	switch *bodyFile {
	case "":
	case "-":
		body, err := io.ReadAll(stdin)
		if err != nil {
			return err
		}
		request.Body = string(body)
	default:
		body, err := os.ReadFile(*bodyFile)
		if err != nil {
			return err
		}
		request.Body = string(body)
	}
	repo, err := findRepo()
	if err != nil {
		return err
	}
	path, err := compass.Record(repo, request)
	var refused *compass.RecordError
	if errors.As(err, &refused) {
		return &usageError{"record: " + refused.Reason}
	}
	if err != nil {
		return err
	}
	if *asJSON {
		return writeJSON(stdout, map[string]string{"path": path})
	}
	_, err = fmt.Fprintln(stdout, path)
	return err
}

func runInit(args []string, stdout io.Writer) error {
	flags := newFlags("init")
	var options compass.InitOptions
	flags.BoolVar(&options.BranchLocal, "branch-local", false, "")
	flags.BoolVar(&options.NoGitHook, "no-git-hook", false, "")
	if err := parse(flags, args); err != nil {
		return err
	}
	options.Today = time.Now()
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	actions, err := compass.Init(root, options)
	for _, action := range actions {
		fmt.Fprintln(stdout, action)
	}
	return err
}

func runDoctor(args []string, stdout io.Writer) error {
	flags := newFlags("doctor")
	asJSON := flags.Bool("json", false, "")
	if err := parse(flags, args); err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	report := compass.Doctor(cwd, os.Getenv("PATH"))
	if *asJSON {
		if err := writeJSON(stdout, report); err != nil {
			return err
		}
	} else {
		fmt.Fprintf(stdout, "compass %s at %s\n", report.Version, report.Executable)
		for _, entry := range report.OnPath {
			fmt.Fprintf(stdout, "on PATH: %s (%s)\n", entry.Path, entry.Kind)
		}
		for _, note := range report.Notes {
			fmt.Fprintln(stdout, "note: "+note)
		}
		for _, problem := range report.Problems {
			fmt.Fprintln(stdout, "problem: "+problem)
		}
		if len(report.Problems) == 0 {
			fmt.Fprintln(stdout, "ok")
		}
	}
	if len(report.Problems) > 0 {
		return failure{}
	}
	return nil
}

// runHook dispatches hook events. Claude Code events always exit 0 (a failing hook must not block
// the agent); the git pre-commit event exits 1 on check errors, which is its job.
func runHook(args []string, stdin io.Reader, stdout, stderr io.Writer) (code int) {
	if len(args) != 1 {
		fmt.Fprint(stderr, "compass hook: expected one event: session-start, stop, pre-compact, pre-commit\n")
		return 2
	}
	if args[0] == "pre-commit" {
		repo, err := findRepo()
		if errors.Is(err, compass.ErrNoCompass) {
			return 0
		}
		if err != nil {
			fmt.Fprintf(stderr, "compass: %s\n", err)
			return 1
		}
		findings, err := compass.PreCommit(repo)
		if err != nil {
			fmt.Fprintf(stderr, "compass: %s\n", err)
			return 1
		}
		for _, finding := range findings {
			fmt.Fprintln(stderr, "compass: "+finding.String())
		}
		if compass.HasErrors(findings) {
			fmt.Fprintln(stderr, "compass: commit blocked by .compass/ errors above")
			return 1
		}
		return 0
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			fmt.Fprintf(stderr, "compass hook %s: internal error: %v\n", args[0], recovered)
			code = 0
		}
	}()
	input := hook.ReadInput(stdin)
	var out []byte
	var err error
	switch args[0] {
	case "session-start":
		out, err = hook.SessionStart(input, os.Getenv("PATH"), os.Getenv("CLAUDE_PLUGIN_ROOT"))
	case "stop":
		out, err = hook.Stop(input)
	case "pre-compact":
		out, err = hook.PreCompact(input, os.Getenv("CLAUDE_PLUGIN_DATA"))
	default:
		fmt.Fprintf(stderr, "compass hook: unknown event %q\n", args[0])
		return 0
	}
	if err != nil {
		fmt.Fprintf(stderr, "compass hook %s: %s\n", args[0], err)
		return 0
	}
	if len(out) > 0 {
		stdout.Write(append(out, '\n'))
	}
	return 0
}
