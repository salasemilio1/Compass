#!/bin/sh
# End-to-end test of a fresh install: a clean HOME and Claude Code config, this checkout added as a
# plugin marketplace, the plugin installed with the real `claude` CLI, and a headless session whose
# SessionStart hook must deliver the resume brief. Then the failure that hid compass for 5 sessions
# on ebay-flipper (settings.json a dangling symlink): the brief must be gone and compass must say why.
# No API key: hook events fire before authentication. Run with `make e2e`.
set -eu

repo_root=$(cd "$(dirname "$0")/.." && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

fail() { echo "FAIL: $*" >&2; exit 1; }
pass() { echo "ok   $*"; }

command -v claude > /dev/null || fail "the claude CLI is not on PATH (npm install -g @anthropic-ai/claude-code)"
claude_bin=$(command -v claude)

# Clean slate: a new HOME and config dir, and nothing inherited from a Claude Code session running
# this script (a nested session, an API key, a logged-in Keychain profile).
for name in $(env | sed -n 's/^\(CLAUDE_CODE_[A-Za-z0-9_]*\)=.*/\1/p'); do unset "$name"; done
unset CLAUDECODE ANTHROPIC_API_KEY ANTHROPIC_AUTH_TOKEN CLAUDE_PLUGIN_ROOT CLAUDE_PLUGIN_DATA || true
export HOME="$work/home"
export CLAUDE_CONFIG_DIR="$HOME/.claude"
mkdir -p "$HOME" "$work/bin"
ln -s "$repo_root/plugin/bin/compass" "$work/bin/compass"
export PATH="$work/bin:$PATH"
export GIT_CONFIG_GLOBAL=/dev/null GIT_AUTHOR_NAME=t GIT_AUTHOR_EMAIL=t@x GIT_COMMITTER_NAME=t GIT_COMMITTER_EMAIL=t@x

# A project with a filled current-state, so the brief has something to deliver.
project=$work/project
mkdir "$project" && cd "$project"
git init -q -b main
echo 'package main' > main.go
git add -A && git commit -q -m "code"
compass init > /dev/null
cat > .compass/context/current-state.md <<EOF
# current state

Date: 2026-10-29
Commit: $(git rev-parse HEAD)

## Objective
End-to-end test of compass delivery.

## Next steps
1. The e2e marker step.
EOF
git add -A && git commit -q -m "compass"

# session OUT: one headless turn with hook events; prints SessionStart hook outputs to OUT.
session() {
  (cd "$project" && "$claude_bin" -p "hi" --output-format stream-json --verbose --include-hook-events --max-turns 1 < /dev/null > "$1.raw" 2>&1) || true
  python3 -I - "$1.raw" > "$1" <<'PY'
import json, sys
for line in open(sys.argv[1], errors="replace"):
    try:
        event = json.loads(line)
    except ValueError:
        continue
    if event.get("subtype") == "hook_response" and event.get("hook_event") == "SessionStart":
        print(event.get("output", "") + event.get("stdout", ""))
PY
}

install() {
  claude plugin marketplace add "$1" > "$work/install.out" 2>&1 || fail "marketplace add: $(cat "$work/install.out")"
  claude plugin install compass@compass > "$work/install.out" 2>&1 || fail "plugin install: $(cat "$work/install.out")"
}

# 1. A fresh install delivers the brief.
install "$repo_root"
session "$work/healthy"
grep -q "e2e marker step" "$work/healthy" || fail "no compass brief from the SessionStart hook; raw output: $(head -c 2000 "$work/healthy.raw")"
pass "fresh install: the SessionStart hook delivers the brief"

out=$(compass doctor 2>&1) || true
echo "$out" | grep -q "^problem:" && fail "doctor reports problems on a healthy install: $out"
echo "$out" | grep -q "Claude Code config: $CLAUDE_CONFIG_DIR" || fail "doctor doesn't name the config dir: $out"
pass "doctor is clean on a healthy install"

# 2. settings.json a dangling symlink: no brief, and compass says why.
mv "$CLAUDE_CONFIG_DIR/settings.json" "$work/settings.json.saved"
ln -s "$work/gone/settings.json" "$CLAUDE_CONFIG_DIR/settings.json"
session "$work/broken"
grep -q "e2e marker step" "$work/broken" && fail "the brief arrived with settings.json dangling: the failure this test reproduces didn't happen"
compass doctor > "$work/doctor.out" 2>&1 || true
grep -q "^problem: .*settings.json is a symlink to" "$work/doctor.out" || fail "doctor doesn't name the dangling symlink: $(cat "$work/doctor.out")"
(cd "$project" && CLAUDECODE=1 compass resume > /dev/null 2> "$work/resume.err")
grep -q "compass: warning: .*is a symlink to" "$work/resume.err" || fail "resume in Claude Code didn't warn: $(cat "$work/resume.err")"
pass "dangling settings: no brief; doctor and resume name the cause"
rm "$CLAUDE_CONFIG_DIR/settings.json" && mv "$work/settings.json.saved" "$CLAUDE_CONFIG_DIR/settings.json"

# 3. An older Go compass ahead of the plugin's on PATH: the brief says so and names the plugin's.
mkdir -p "$work/old-bin"
printf '#!/bin/sh\n# compass\necho "compass 0.0.1 (format 3)"\n' > "$work/old-bin/compass" && chmod +x "$work/old-bin/compass"
PATH="$work/old-bin:$PATH" session "$work/shadowed"
grep -q "version 0.0.1, older than the plugin's" "$work/shadowed" || fail "no older-compass note; hook output: $(cat "$work/shadowed")"
pass "an older compass first on PATH: the brief names it and the plugin's binary"

# 4. The test can fail: a plugin whose hooks.json lacks SessionStart must deliver nothing.
rm -rf "$CLAUDE_CONFIG_DIR" && mkdir -p "$CLAUDE_CONFIG_DIR"
copy=$work/plugin-copy
mkdir "$copy" && cp -R "$repo_root/.claude-plugin" "$repo_root/plugin" "$copy/"
python3 -I - "$copy/plugin/hooks/hooks.json" <<'PY'
import json, sys
path = sys.argv[1]
data = json.load(open(path))
data["hooks"].pop("SessionStart")
json.dump(data, open(path, "w"))
PY
install "$copy"
session "$work/nohook"
grep -q "e2e marker step" "$work/nohook" && fail "a plugin without SessionStart still delivered: the check can't fail"
pass "control: a plugin without SessionStart delivers nothing"
