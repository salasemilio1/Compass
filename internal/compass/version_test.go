package compass

import (
	"encoding/json"
	"os"
	"testing"
)

// Claude Code caches a plugin by version, so a release that changes the binary without bumping
// both manifests would never reach installed users.
func TestManifestVersionsMatchTheBinary(t *testing.T) {
	var plugin struct{ Version string }
	var marketplace struct {
		Plugins []struct{ Name, Version, Source string }
	}
	for path, target := range map[string]any{
		"../../plugin/.claude-plugin/plugin.json": &plugin,
		"../../.claude-plugin/marketplace.json":   &marketplace,
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, target); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
	}
	if plugin.Version != Version {
		t.Errorf("plugin.json version %q, binary %q", plugin.Version, Version)
	}
	if len(marketplace.Plugins) != 1 || marketplace.Plugins[0].Version != Version || marketplace.Plugins[0].Source != "./plugin" {
		t.Errorf("marketplace.json: %+v, binary %q", marketplace.Plugins, Version)
	}
}
