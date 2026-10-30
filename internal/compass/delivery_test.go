package compass

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// claudeConfig builds a Claude Code config dir with a healthy user-scope compass install, then
// lets each case break one thing.
func claudeConfig(t *testing.T) (configDir, installPath string) {
	t.Helper()
	configDir = t.TempDir()
	installPath = filepath.Join(configDir, "plugins", "cache", "compass", "compass", "0.3.1")
	write(t, filepath.Join(installPath, "hooks", "hooks.json"), `{"hooks":{}}`)
	write(t, filepath.Join(configDir, "settings.json"), `{"enabledPlugins":{"compass@compass":true}}`)
	write(t, filepath.Join(configDir, "plugins", "installed_plugins.json"),
		`{"version":2,"plugins":{"compass@compass":[{"scope":"user","installPath":"`+installPath+`"}]}}`)
	return configDir, installPath
}

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func hasProblem(report *DeliveryReport, fragment string) bool {
	for _, problem := range report.Problems {
		if strings.Contains(problem, fragment) {
			return true
		}
	}
	return false
}

func TestDeliveryHealthyInstallHasNoProblems(t *testing.T) {
	config, _ := claudeConfig(t)
	report := Delivery(config, []string{t.TempDir()}, true)
	if len(report.Problems) != 0 {
		t.Fatalf("problems on a healthy install: %v", report.Problems)
	}
}

func TestDeliveryDetectsEachBreak(t *testing.T) {
	cases := []struct {
		name     string
		breakIt  func(t *testing.T, config, install, project string)
		fragment string
	}{
		{"dangling settings symlink", func(t *testing.T, config, _, _ string) {
			path := filepath.Join(config, "settings.json")
			os.Remove(path)
			if err := os.Symlink(filepath.Join(config, "gone", "settings.json"), path); err != nil {
				t.Fatal(err)
			}
		}, "is a symlink to"},
		{"invalid settings JSON", func(t *testing.T, config, _, _ string) {
			write(t, filepath.Join(config, "settings.json"), `{"enabledPlugins": {`)
		}, "is not valid JSON"},
		{"plugin disabled", func(t *testing.T, config, _, _ string) {
			write(t, filepath.Join(config, "settings.json"), `{"enabledPlugins":{"compass@compass":false}}`)
		}, "not enabled"},
		{"plugin never enabled", func(t *testing.T, config, _, _ string) {
			write(t, filepath.Join(config, "settings.json"), `{}`)
		}, "not enabled"},
		{"disabled in project local settings", func(t *testing.T, _, _, project string) {
			write(t, filepath.Join(project, ".claude", "settings.local.json"), `{"enabledPlugins":{"compass@compass":false}}`)
		}, "not enabled"},
		{"plugin not installed", func(t *testing.T, config, _, _ string) {
			write(t, filepath.Join(config, "plugins", "installed_plugins.json"), `{"version":2,"plugins":{}}`)
		}, "not installed"},
		{"install directory missing", func(t *testing.T, _, install, _ string) {
			os.RemoveAll(install)
		}, "install directory"},
		{"only a stale marketplace key enabled", func(t *testing.T, config, _, _ string) {
			write(t, filepath.Join(config, "settings.json"), `{"enabledPlugins":{"compass@compass":false,"compass@old":true}}`)
		}, "not enabled"},
		{"disableAllHooks", func(t *testing.T, config, _, _ string) {
			write(t, filepath.Join(config, "settings.json"), `{"enabledPlugins":{"compass@compass":true},"disableAllHooks":true}`)
		}, "disableAllHooks"},
	}
	for _, name := range v2HookNames {
		cases = append(cases, struct {
			name     string
			breakIt  func(t *testing.T, config, install, project string)
			fragment string
		}{"v2 hook " + name, func(t *testing.T, config, _, _ string) {
			write(t, filepath.Join(config, "settings.json"), `{"enabledPlugins":{"compass@compass":true},"hooks":{"Stop":[{"hooks":[{"type":"command","command":"$HOME/.claude/hooks/`+name+`.py"}]}]}}`)
		}, name})
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			config, install := claudeConfig(t)
			project := t.TempDir()
			c.breakIt(t, config, install, project)
			report := Delivery(config, []string{project}, true)
			if !hasProblem(report, c.fragment) {
				t.Fatalf("want a problem with %q, got problems %v notes %v", c.fragment, report.Problems, report.Notes)
			}
		})
	}
}

func TestDeliveryPrecedenceIsPerKey(t *testing.T) {
	config, _ := claudeConfig(t)
	project := t.TempDir()
	// An old marketplace's key turned off locally doesn't turn off compass@compass.
	write(t, filepath.Join(project, ".claude", "settings.local.json"), `{"enabledPlugins":{"compass@old":false}}`)
	if report := Delivery(config, []string{project}, true); len(report.Problems) != 0 {
		t.Fatalf("false problem: %v", report.Problems)
	}
	// Enabled only in project settings, not in user settings: fine.
	write(t, filepath.Join(config, "settings.json"), `{}`)
	write(t, filepath.Join(project, ".claude", "settings.json"), `{"enabledPlugins":{"compass@compass":true}}`)
	if report := Delivery(config, []string{project}, true); len(report.Problems) != 0 {
		t.Fatalf("false problem with project-scope enablement: %v", report.Problems)
	}
}

func TestDeliveryValidSymlinkedSettingsIsFine(t *testing.T) {
	config, _ := claudeConfig(t)
	real := filepath.Join(t.TempDir(), "dotfiles-settings.json")
	data, _ := os.ReadFile(filepath.Join(config, "settings.json"))
	write(t, real, string(data))
	os.Remove(filepath.Join(config, "settings.json"))
	if err := os.Symlink(real, filepath.Join(config, "settings.json")); err != nil {
		t.Fatal(err)
	}
	if report := Delivery(config, nil, true); len(report.Problems) != 0 {
		t.Fatalf("false problem on a valid symlink: %v", report.Problems)
	}
}

func TestDeliveryProjectScopeInstallCountsOnlyForItsProject(t *testing.T) {
	config, install := claudeConfig(t)
	project, other := t.TempDir(), t.TempDir()
	registry := `{"version":2,"plugins":{"compass@compass":[{"scope":"project","projectPath":"` + project + `","installPath":"` + install + `"}]}}`
	write(t, filepath.Join(config, "plugins", "installed_plugins.json"), registry)
	if report := Delivery(config, []string{project}, true); len(report.Problems) != 0 {
		t.Fatalf("project install in its own project: %v", report.Problems)
	}
	if report := Delivery(config, []string{other}, true); !hasProblem(report, "not installed") {
		t.Fatalf("another project's install counted here: %v", report.Problems)
	}
	// A stale entry for another project with a missing path isn't this project's problem.
	stale := `{"version":2,"plugins":{"compass@compass":[{"scope":"user","installPath":"` + install + `"},{"scope":"local","projectPath":"` + other + `","installPath":"/nowhere"}]}}`
	write(t, filepath.Join(config, "plugins", "installed_plugins.json"), stale)
	if report := Delivery(config, []string{project}, true); len(report.Problems) != 0 {
		t.Fatalf("another project's stale entry raised: %v", report.Problems)
	}
}

func TestDeliveryWithoutClaudeCodeIntentOnlyNotes(t *testing.T) {
	// A Codex or terminal-only user: no compass key, no install, not in Claude Code.
	config := t.TempDir()
	write(t, filepath.Join(config, "settings.json"), `{}`)
	report := Delivery(config, nil, false)
	if len(report.Problems) != 0 {
		t.Fatalf("problems without intent: %v", report.Problems)
	}
	if len(report.Notes) == 0 {
		t.Fatal("want a note that the plugin isn't installed")
	}
	// No config dir at all.
	if report := Delivery(filepath.Join(config, "missing"), nil, false); len(report.Problems) != 0 {
		t.Fatalf("problems with no config dir: %v", report.Problems)
	}
}

func TestDeliveryUnknownShapesAreNotes(t *testing.T) {
	config, _ := claudeConfig(t)
	write(t, filepath.Join(config, "plugins", "installed_plugins.json"), `["a list, not an object"]`)
	write(t, filepath.Join(config, "settings.json"), `{"enabledPlugins":{"compass@compass":"yes"}}`)
	project := t.TempDir()
	write(t, filepath.Join(project, ".claude", "settings.json"), `not json`)
	report := Delivery(config, []string{project}, false)
	if len(report.Problems) != 0 {
		t.Fatalf("unknown shapes raised problems: %v", report.Problems)
	}
	if len(report.Notes) < 3 {
		t.Fatalf("want notes for each unknown shape, got %v", report.Notes)
	}
}

func TestLefthookRunsCompass(t *testing.T) {
	cases := map[string]bool{
		"pre-commit:\n  commands:\n    compass:\n      run: compass hook pre-commit\n":                                               true,
		"pre-commit:\n  commands:\n    compass:\n      # run: compass hook pre-commit\n":                                             false,
		"commit-msg:\n  commands:\n    compass:\n      run: compass hook pre-commit\n":                                               false,
		"pre-push:\n  commands:\n    x:\n      run: go test\npre-commit:\n  commands:\n    c:\n      run: compass hook pre-commit\n": true,
	}
	for text, want := range cases {
		if got := lefthookRunsCompass(text); got != want {
			t.Errorf("lefthookRunsCompass(%q) = %v, want %v", text, got, want)
		}
	}
}

func TestDeliveryBreakPassFalsePositives(t *testing.T) {
	t.Run("settings.json with a UTF-8 BOM", func(t *testing.T) {
		config, _ := claudeConfig(t)
		write(t, filepath.Join(config, "settings.json"), "\xEF\xBB\xBF"+`{"enabledPlugins":{"compass@compass":true}}`)
		if report := Delivery(config, nil, true); len(report.Problems) != 0 {
			t.Fatalf("BOM: %v", report.Problems)
		}
	})
	t.Run("registry in an older shape", func(t *testing.T) {
		config, install := claudeConfig(t)
		write(t, filepath.Join(config, "plugins", "installed_plugins.json"), `{"version":1,"plugins":{"compass@compass":{"installPath":"`+install+`"}}}`)
		report := Delivery(config, nil, true)
		if len(report.Problems) != 0 {
			t.Fatalf("unknown registry shape raised: %v", report.Problems)
		}
	})
	t.Run("disableAllHooks overridden by project local settings", func(t *testing.T) {
		config, _ := claudeConfig(t)
		project := t.TempDir()
		write(t, filepath.Join(config, "settings.json"), `{"enabledPlugins":{"compass@compass":true},"disableAllHooks":true}`)
		write(t, filepath.Join(project, ".claude", "settings.local.json"), `{"disableAllHooks":false}`)
		if report := Delivery(config, []string{project}, true); len(report.Problems) != 0 {
			t.Fatalf("overridden disableAllHooks: %v", report.Problems)
		}
	})
	t.Run("hooks.json missing under installPath is only a note", func(t *testing.T) {
		config, install := claudeConfig(t)
		os.Remove(filepath.Join(install, "hooks", "hooks.json"))
		report := Delivery(config, nil, true)
		if len(report.Problems) != 0 || len(report.Notes) == 0 {
			t.Fatalf("problems %v notes %v", report.Problems, report.Notes)
		}
	})
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		cmp  int
		ok   bool
	}{
		{"0.3.0", "0.3.1", -1, true}, {"0.3.1", "0.3.10", -1, true}, {"0.3.10", "0.3.9", 1, true},
		{"0.3.1", "0.3.1", 0, true}, {"v0.3.1", "0.3.1", 0, true}, {"0.3", "0.3.0", 0, true},
		{"f8c5e7a1dcb7", "0.3.1", 0, false}, {"", "0.3.1", 0, false}, {"0.3.x", "0.3.1", 0, false},
		{"20261030", "0.3.1", 0, false}, {"1234567", "0.3.1", 0, false}, {"0.+3.1", "0.3.1", 0, false},
		{"0.3.2-rc1", "0.3.1", 0, false}, {"0.3.1.4", "0.3.1", 0, false},
	}
	for _, c := range cases {
		if cmp, ok := CompareVersions(c.a, c.b); cmp != c.cmp || ok != c.ok {
			t.Errorf("CompareVersions(%q, %q) = %d, %v; want %d, %v", c.a, c.b, cmp, ok, c.cmp, c.ok)
		}
	}
}

func TestDeliveryPluginVersionIsTheHighestRealOne(t *testing.T) {
	config, install := claudeConfig(t)
	registry := `{"version":2,"plugins":{"compass@compass":[{"scope":"user","installPath":"` + install + `","version":"0.3.1"},` +
		`{"scope":"user","installPath":"` + install + `","version":"0.3.10"},{"scope":"user","installPath":"` + install + `","version":"f8c5e7a1dcb7"},` +
		`{"scope":"user","installPath":"` + install + `","version":"20261030"}]}}`
	write(t, filepath.Join(config, "plugins", "installed_plugins.json"), registry)
	if got := Delivery(config, nil, true).PluginVersion; got != "0.3.10" {
		t.Fatalf("PluginVersion = %q, want 0.3.10", got)
	}
}

func TestDoctorFlagsAnOlderGoCompassFirstOnPath(t *testing.T) {
	config, install := claudeConfig(t)
	write(t, filepath.Join(config, "plugins", "installed_plugins.json"),
		`{"version":2,"plugins":{"compass@compass":[{"scope":"user","installPath":"`+install+`","version":"99.0.0"}]}}`)
	for version, want := range map[string]bool{"0.0.1": true, "99.0.0": false, "99.1.0": false} {
		bin := t.TempDir()
		write(t, filepath.Join(bin, "compass"), "#!/bin/sh\n# compass\necho 'compass "+version+" (format 3)'\n")
		os.Chmod(filepath.Join(bin, "compass"), 0o755)
		env := map[string]string{"PATH": bin + string(os.PathListSeparator) + "/usr/bin:/bin", "CLAUDE_CONFIG_DIR": config, "HOME": t.TempDir()}
		report := Doctor(t.TempDir(), func(key string) string { return env[key] })
		got := false
		for _, problem := range report.Problems {
			got = got || strings.Contains(problem, "is compass "+version+", older than")
		}
		if got != want {
			t.Errorf("version %s on PATH: flagged %v, want %v (%v)", version, got, want, report.Problems)
		}
	}
}

func TestCappedBufferKeepsOnlyItsLimit(t *testing.T) {
	buffer := &cappedBuffer{limit: 8}
	for i := 0; i < 1000; i++ {
		if n, err := buffer.Write([]byte("compass 0.3.1 (format 3)\n")); err != nil || n != 25 {
			t.Fatalf("Write = %d, %v; want 25, nil (a short write would fail the child's pipe)", n, err)
		}
	}
	if buffer.Len() != 8 || buffer.String() != "compass " {
		t.Fatalf("kept %d bytes %q, want the first 8", buffer.Len(), buffer.String())
	}
}
