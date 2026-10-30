package compass

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// DeliveryReport says whether Claude Code will load the compass plugin and its hooks. Claude Code
// fails quietly here: an unreadable settings.json means no hooks and no plugins, and nothing in the
// session says so (ebay-flipper, 2026-10-28: a dangling symlink turned compass off for 5 sessions).
// The SessionStart hook can't report its own absence, so resume, pre-commit and doctor do.
type DeliveryReport struct {
	ConfigDir string `json:"config_dir"`
	// PluginVersion is the highest version among the compass plugin installs that apply here.
	PluginVersion string   `json:"plugin_version,omitempty"`
	Problems      []string `json:"problems"`
	Notes         []string `json:"notes"`
}

// v2 registered these hooks in user settings; v3 cut them (decision 0015) and nothing removes them.
var v2HookNames = []string{"tool-routing-guard", "session-context-nudge", "compass-session-start"}

// ClaudeConfigDir is where Claude Code keeps user settings and plugins.
func ClaudeConfigDir(getenv func(string) string) string {
	if dir := getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return dir
	}
	if home := getenv("HOME"); home != "" {
		return filepath.Join(home, ".claude")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude")
}

type settingsFile struct {
	path   string
	values map[string]json.RawMessage
}

// Delivery reads, never writes, Claude Code's settings and plugin registry. projectDirs are the
// directories Claude Code may have been launched from (cwd and the repo root); inClaudeCode is
// CLAUDECODE=1. Missing plugin pieces are problems only where there is evidence Claude Code is meant
// to run compass: inClaudeCode, a compass@ key in settings, or an installed compass entry. A Codex
// or terminal-only user gets notes.
func Delivery(configDir string, projectDirs []string, inClaudeCode bool) *DeliveryReport {
	report := &DeliveryReport{ConfigDir: configDir}
	if _, err := os.Stat(configDir); err != nil && !inClaudeCode {
		report.Notes = append(report.Notes, fmt.Sprintf("no Claude Code config at %s (only needed for Claude Code)", configDir))
		return report
	}

	userPath := filepath.Join(configDir, "settings.json")
	user, problem := readSettings(userPath)
	userReadable := problem == ""
	if problem != "" {
		report.Problems = append(report.Problems, problem+": Claude Code loads no hooks and no plugins, compass included")
	}
	// Precedence, lowest first: user, then each project's shared and local settings.
	var files []settingsFile
	if user != nil {
		files = append(files, settingsFile{userPath, user})
	}
	for _, dir := range uniqueDirs(projectDirs) {
		for _, name := range []string{"settings.json", "settings.local.json"} {
			path := filepath.Join(dir, ".claude", name)
			values, problem := readSettings(path)
			if problem != "" {
				report.Notes = append(report.Notes, problem+" (Claude Code skips it)")
			}
			if values != nil {
				files = append(files, settingsFile{path, values})
			}
		}
	}

	enabled := map[string]bool{}
	hooksOffIn := "" // the file whose disableAllHooks wins, when it is true
	for _, file := range files {
		var hooksOff bool
		if raw, ok := file.values["disableAllHooks"]; ok && json.Unmarshal(raw, &hooksOff) == nil {
			hooksOffIn = map[bool]string{true: file.path, false: ""}[hooksOff]
		}
		if raw, ok := file.values["hooks"]; ok {
			for _, name := range v2HookNames {
				if bytes.Contains(raw, []byte(name)) {
					report.Problems = append(report.Problems, fmt.Sprintf("%s still registers v2's %s hook; delete that entry from its \"hooks\" (v3's plugin replaces it)", file.path, name))
				}
			}
		}
		var plugins map[string]json.RawMessage
		if raw, ok := file.values["enabledPlugins"]; ok && json.Unmarshal(raw, &plugins) == nil {
			for key, raw := range plugins {
				if !strings.HasPrefix(key, "compass@") {
					continue
				}
				var on bool
				if json.Unmarshal(raw, &on) != nil {
					report.Notes = append(report.Notes, fmt.Sprintf("%s: enabledPlugins[%q] is not true or false", file.path, key))
					continue
				}
				enabled[key] = on
			}
		}
	}

	if hooksOffIn != "" {
		report.Problems = append(report.Problems, fmt.Sprintf("%s sets disableAllHooks: true, so compass's hooks never run; remove it", hooksOffIn))
	}

	installed, known, note := installedCompass(configDir, projectDirs)
	if note != "" {
		report.Notes = append(report.Notes, note)
	}
	if !known {
		// A registry shape compass can't read says nothing about the install; don't guess.
		return report
	}
	intent := inClaudeCode || len(enabled) > 0 || len(installed) > 0
	flag := func(message string) {
		if intent {
			report.Problems = append(report.Problems, message)
		} else {
			report.Notes = append(report.Notes, message+" (only needed for Claude Code)")
		}
	}
	if len(installed) == 0 {
		flag("the compass plugin is not installed: claude plugin marketplace add salasemilio1/Compass && claude plugin install compass@compass")
	}
	// Claude Code may load hooks from elsewhere (a directory marketplace), so a missing hooks.json is
	// only a note; a missing install directory is a broken install.
	installedOn := false
	for _, entry := range installed {
		// Directory marketplaces may record a sha or nothing; only real versions count.
		if _, ok := CompareVersions(entry.version, "0.0"); ok {
			if cmp, _ := CompareVersions(entry.version, report.PluginVersion); report.PluginVersion == "" || cmp > 0 {
				report.PluginVersion = entry.version
			}
		}
		if _, err := os.Stat(entry.path); err != nil {
			report.Problems = append(report.Problems, fmt.Sprintf("the compass plugin's install directory %s is missing; reinstall it: claude plugin install compass@compass", entry.path))
		} else if _, err := os.Stat(filepath.Join(entry.path, "hooks", "hooks.json")); err != nil {
			report.Notes = append(report.Notes, fmt.Sprintf("the compass plugin at %s has no hooks/hooks.json", entry.path))
		}
		// Only the installed key counts: a stale compass@<old marketplace> true enables nothing.
		installedOn = installedOn || enabled[entry.key]
	}
	// With user settings unreadable, enablement is unknowable and already reported above.
	if userReadable && len(installed) > 0 && !installedOn {
		flag("the compass plugin is installed but not enabled: claude plugin enable compass@compass")
	}
	return report
}

// readSettings returns the parsed file, or a problem sentence when the file exists (or a symlink
// to it does) but can't be used. A missing file is neither.
func readSettings(path string) (map[string]json.RawMessage, string) {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ""
	}
	if err != nil {
		return nil, fmt.Sprintf("%s can't be read (%v)", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
			target, _ := os.Readlink(path)
			return nil, fmt.Sprintf("%s is a symlink to %s, which doesn't exist; replace it with a real file or fix the link", path, target)
		}
	}
	values, err := parseSettings(path)
	if err != nil {
		// A write in progress (claude plugin install) can read as truncated JSON; look once more.
		time.Sleep(100 * time.Millisecond)
		values, err = parseSettings(path)
	}
	if err != nil {
		return nil, fmt.Sprintf("%s is not valid JSON (%v)", path, err)
	}
	return values, ""
}

func parseSettings(path string) (map[string]json.RawMessage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	data = bytes.TrimPrefix(data, []byte("\xEF\xBB\xBF")) // Claude Code accepts a UTF-8 BOM
	var values map[string]json.RawMessage
	if len(bytes.TrimSpace(data)) == 0 {
		return map[string]json.RawMessage{}, nil
	}
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, err
	}
	if values == nil {
		values = map[string]json.RawMessage{}
	}
	return values, nil
}

type installEntry struct{ key, path, version string }

// installedCompass lists the compass plugin entries that apply here: user scope, or project/local
// scope for one of projectDirs. known is false when the registry exists but can't be read or has a
// shape compass doesn't know (Claude Code migrates its formats); the note says so.
func installedCompass(configDir string, projectDirs []string) (entries []installEntry, known bool, note string) {
	path := filepath.Join(configDir, "plugins", "installed_plugins.json")
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, true, ""
	}
	if err != nil {
		return nil, false, fmt.Sprintf("%s can't be read (%v); plugin install not checked", path, err)
	}
	var registry struct {
		Plugins map[string][]struct {
			Scope       string `json:"scope"`
			ProjectPath string `json:"projectPath"`
			InstallPath string `json:"installPath"`
			Version     string `json:"version"`
		} `json:"plugins"`
	}
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, false, fmt.Sprintf("%s has a shape compass doesn't know (%v); plugin install not checked", path, err)
	}
	dirs := map[string]bool{}
	for _, dir := range uniqueDirs(projectDirs) {
		dirs[dir] = true
	}
	for key, list := range registry.Plugins {
		if !strings.HasPrefix(key, "compass@") {
			continue
		}
		for _, entry := range list {
			if entry.InstallPath == "" {
				continue
			}
			if entry.Scope == "project" || entry.Scope == "local" {
				if !dirs[cleanDir(entry.ProjectPath)] {
					continue
				}
			}
			entries = append(entries, installEntry{key, entry.InstallPath, entry.Version})
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].path < entries[j].path })
	return entries, true, ""
}

func cleanDir(dir string) string {
	if dir == "" {
		return ""
	}
	if real, err := filepath.EvalSymlinks(dir); err == nil {
		dir = real
	}
	return filepath.Clean(dir)
}

func uniqueDirs(dirs []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, dir := range dirs {
		dir = cleanDir(dir)
		if dir != "" && !seen[dir] {
			seen[dir] = true
			out = append(out, dir)
		}
	}
	return out
}
