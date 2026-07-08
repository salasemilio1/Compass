#!/usr/bin/env bash
# Unit tests for install.sh's copy-based install (skills/hooks are copied, not symlinked,
# so they keep working after the source checkout is deleted) + a linking smoke test.
# Never touches the real ~/.claude -- COMPASS_HOME is always overridden to a temp dir,
# and COMPASS_TEST_SOURCE=1 sources the script instead of running its network-touching main().
set -uo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$TEST_DIR/.." && pwd)"
INSTALL_SH="$REPO_ROOT/install.sh"
# shellcheck source=./helpers.sh
source "$TEST_DIR/helpers.sh"

CLEANUP_DIRS=()
cleanup() {
  local d
  for d in "${CLEANUP_DIRS[@]:-}"; do
    [ -n "$d" ] && rm -rf "$d"
  done
}
trap cleanup EXIT

new_tmpdir() {
  local d
  d="$(mktemp -d)"
  CLEANUP_DIRS+=("$d")
  echo "$d"
}

echo "-- smoke test: install steps land under an isolated COMPASS_HOME"
{
  home="$(new_tmpdir)"

  out="$(COMPASS_HOME="$home" COMPASS_TEST_SOURCE=1 bash -c "
    source '$INSTALL_SH'
    link_skills
    link_hook
    register_hook
    register_context_nudge_hook
    register_tool_routing_hook
  " 2>&1)"
  rc=$?
  assert_eq "0" "$rc" "smoke: install steps succeed"

  assert_eq "0" "$([ -L "$home/skills/warm-up" ] && echo 1 || echo 0)" "smoke: warm-up skill is a real copy, not a symlink"
  assert_eq "1" "$([ -f "$home/skills/warm-up/SKILL.md" ] && echo 1 || echo 0)" "smoke: warm-up skill content present"
  assert_eq "0" "$([ -L "$home/hooks/compass-session-start.sh" ] && echo 1 || echo 0)" "smoke: SessionStart hook is a real copy, not a symlink"
  assert_eq "0" "$([ -L "$home/hooks/session-context-nudge.py" ] && echo 1 || echo 0)" "smoke: Stop hook (session-context-nudge.py) is a real copy"
  assert_eq "0" "$([ -L "$home/hooks/tool-routing-guard.py" ] && echo 1 || echo 0)" "smoke: PreToolUse hook (tool-routing-guard.py) is a real copy"
  assert_eq "0" "$([ -L "$home/tools/session-context" ] && echo 1 || echo 0)" "smoke: tools/session-context is a real copy"
  assert_eq "1" "$([ -f "$home/tools/session-context/session_status.py" ] && echo 1 || echo 0)" "smoke: tools/session-context content present"

  assert_file_contains "$home/settings.json" "compass-session-start.sh" "smoke: settings.json registers the SessionStart hook"
  assert_file_contains "$home/settings.json" "$home/hooks/compass-session-start.sh" "smoke: settings.json points at the isolated COMPASS_HOME, not the real one"
  assert_file_contains "$home/settings.json" "session-context-nudge.py" "smoke: settings.json registers the Stop hook"
  assert_file_contains "$home/settings.json" "tool-routing-guard.py" "smoke: settings.json registers the PreToolUse hook"
  assert_file_contains "$home/settings.json" '"matcher": "Bash"' "smoke: PreToolUse hook is scoped to the Bash matcher"

  echo "  note: real \$HOME/.claude was never referenced by this test (COMPASS_HOME overridden throughout)"
}

echo "-- installed copies survive source deletion"
{
  home="$(new_tmpdir)"
  src_copy="$(new_tmpdir)"
  cp -R "$REPO_ROOT/skills/warm-up" "$src_copy/warm-up"

  COMPASS_HOME="$home" COMPASS_TEST_SOURCE=1 bash -c "
    source '$INSTALL_SH'
    install_path '$src_copy/warm-up' '$home/skills/warm-up'
  " >/dev/null 2>&1

  rm -rf "$src_copy"
  assert_eq "1" "$([ -f "$home/skills/warm-up/SKILL.md" ] && echo 1 || echo 0)" "installed copy still readable after source is deleted"
}

echo "-- re-running install.sh updates an already-installed copy"
{
  home="$(new_tmpdir)"
  src_a="$(new_tmpdir)"
  src_b="$(new_tmpdir)"
  printf 'version A\n' > "$src_a/file.txt"
  printf 'version B\n' > "$src_b/file.txt"

  COMPASS_HOME="$home" COMPASS_TEST_SOURCE=1 bash -c "
    source '$INSTALL_SH'
    install_path '$src_a/file.txt' '$home/file.txt'
  " >/dev/null 2>&1
  assert_file_contains "$home/file.txt" "version A" "first install: content matches source A"

  COMPASS_HOME="$home" COMPASS_TEST_SOURCE=1 bash -c "
    source '$INSTALL_SH'
    install_path '$src_b/file.txt' '$home/file.txt'
  " >/dev/null 2>&1
  assert_file_contains "$home/file.txt" "version B" "re-install: content updated to source B"
}

echo "-- register_event_hook: without python3, fails loudly instead of silently skipping"
{
  home="$(new_tmpdir)"
  out="$(COMPASS_HOME="$home" COMPASS_TEST_SOURCE=1 bash -c "
    source '$INSTALL_SH'
    command_exists() { [ \"\$1\" != 'python3' ]; }
    register_context_nudge_hook
  " 2>&1)"
  rc=$?
  assert_eq "1" "$rc" "no python3: register_event_hook exits 1 (not a silent skip)"
  assert_file_contains <(echo "$out") "python3 not found" "no python3: error names the missing dependency"
  assert_file_contains <(echo "$out") "add this block" "no python3: still prints manual fallback instructions"
}

report_and_exit
