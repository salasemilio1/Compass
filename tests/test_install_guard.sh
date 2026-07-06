#!/usr/bin/env bash
# Unit tests for install.sh's canonical-source guard (gap C) + a linking smoke test.
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

echo "-- canonical source: \$DOTFILES/compass matches"
{
  dotfiles="$(new_tmpdir)"
  ln -s "$REPO_ROOT" "$dotfiles/compass"
  out="$(COMPASS_TEST_SOURCE=1 DOTFILES="$dotfiles" bash -c "source '$INSTALL_SH'; check_install_source" 2>&1)"
  rc=$?
  assert_eq "0" "$rc" "canonical DOTFILES match: check_install_source exits 0"
  assert_file_not_contains <(echo "$out") "refusing" "canonical DOTFILES match: no refusal printed"
}

echo "-- non-canonical, no --dev: refuses"
{
  out="$(COMPASS_TEST_SOURCE=1 bash -c "unset DOTFILES; source '$INSTALL_SH'; check_install_source" 2>&1)"
  rc=$?
  assert_eq "1" "$rc" "non-canonical without --dev: check_install_source exits 1"
  assert_file_contains <(echo "$out") "refusing to install" "non-canonical without --dev: prints refusal"
  assert_file_contains <(echo "$out") "install\.sh --dev" "non-canonical without --dev: mentions the escape hatch"
}

echo "-- non-canonical with --dev (DEV_MODE=1): proceeds with a warning"
{
  out="$(COMPASS_TEST_SOURCE=1 bash -c "unset DOTFILES; source '$INSTALL_SH'; DEV_MODE=1; check_install_source" 2>&1)"
  rc=$?
  assert_eq "0" "$rc" "non-canonical with --dev: check_install_source exits 0"
  assert_file_contains <(echo "$out") "dev install" "non-canonical with --dev: prints a dev-install warning"
}

echo "-- smoke test: linking steps land under an isolated COMPASS_HOME"
{
  home="$(new_tmpdir)"
  dotfiles="$(new_tmpdir)"
  ln -s "$REPO_ROOT" "$dotfiles/compass"

  out="$(COMPASS_HOME="$home" DOTFILES="$dotfiles" COMPASS_TEST_SOURCE=1 bash -c "
    source '$INSTALL_SH'
    check_install_source || exit 1
    link_skills
    link_hook
    register_hook
  " 2>&1)"
  rc=$?
  assert_eq "0" "$rc" "smoke: guarded linking steps succeed"

  assert_eq "1" "$([ -L "$home/skills/warm-up" ] && echo 1 || echo 0)" "smoke: warm-up skill symlinked"
  assert_eq "$REPO_ROOT/skills/warm-up" "$(readlink "$home/skills/warm-up" 2>/dev/null || echo MISSING)" "smoke: warm-up symlink points at this checkout"
  assert_eq "1" "$([ -L "$home/hooks/compass-session-start.sh" ] && echo 1 || echo 0)" "smoke: SessionStart hook symlinked"

  if command -v jq >/dev/null 2>&1; then
    assert_file_contains "$home/settings.json" "compass-session-start.sh" "smoke: settings.json registers the hook (jq path)"
    assert_file_contains "$home/settings.json" "$home/hooks/compass-session-start.sh" "smoke: settings.json points at the isolated COMPASS_HOME, not the real one"
  else
    assert_file_contains <(echo "$out") "jq not found" "smoke: no jq -- falls back to manual instructions instead of failing"
  fi

  echo "  note: real \$HOME/.claude was never referenced by this test (COMPASS_HOME overridden throughout)"
}

report_and_exit
