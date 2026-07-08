#!/usr/bin/env bash
# Install compass skills and hooks for Claude Code.
# Safe to re-run — each step is idempotent and independent.
# One failure does not stop the rest.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Override for testing/isolation -- defaults to the real global Claude Code config dir.
COMPASS_HOME="${COMPASS_HOME:-$HOME/.claude}"

DEV_MODE=0

# --- logging ------------------------------------------------------------------

if [ -t 1 ]; then
  _reset="$(printf '\033[0m')"
  _blue="$(printf '\033[34m')"
  _yellow="$(printf '\033[33m')"
  _red="$(printf '\033[31m')"
  _green="$(printf '\033[32m')"
else
  _reset=""; _blue=""; _yellow=""; _red=""; _green=""
fi

log_info()    { printf '%s==>%s %s\n' "$_blue" "$_reset" "$*"; }
log_warn()    { printf '%swarn:%s %s\n' "$_yellow" "$_reset" "$*" >&2; }
log_error()   { printf '%serror:%s %s\n' "$_red" "$_reset" "$*" >&2; }
log_success() { printf '%s ok%s  %s\n' "$_green" "$_reset" "$*"; }

# --- helpers ------------------------------------------------------------------

command_exists() { command -v "$1" >/dev/null 2>&1; }

# Tally of failed create_symlink calls, reported as a loud summary at the end of install
# instead of being swallowed -- a broken symlink must not look like a successful install.
LINK_FAILURES=0
track_symlink() { create_symlink "$1" "$2" || LINK_FAILURES=$((LINK_FAILURES + 1)); }

# Symlink source -> target, idempotently. Backs up anything already in the way.
create_symlink() {
  local source_path="$1"
  local target_path="$2"

  if [ ! -e "$source_path" ]; then
    log_error "source missing: $source_path"
    return 1
  fi

  mkdir -p "$(dirname "$target_path")"

  if [ -L "$target_path" ] && [ "$(readlink "$target_path")" = "$source_path" ]; then
    log_success "already linked: $(basename "$target_path")"
    return 0
  fi

  if [ -e "$target_path" ] || [ -L "$target_path" ]; then
    local backup="${target_path}.backup-$(date +%Y%m%d%H%M%S)"
    mv "$target_path" "$backup"
    log_warn "backed up existing file: $target_path -> $backup"
  fi

  if ! ln -s "$source_path" "$target_path"; then
    log_error "failed to link: $target_path"
    return 1
  fi
  log_success "linked: $(basename "$target_path")"
}

# --- install source guard ------------------------------------------------------
#
# Global skills/hook symlinks point at whatever checkout this script runs from. If that's
# an arbitrary dev clone (not the canonical $DOTFILES/compass), a destructive git operation
# there (reset --hard, branch delete, force-push) breaks compass globally, not just locally.
# Refuse by default; --dev is the explicit escape hatch for iterating on compass itself.

check_install_source() {
  local resolved_script_dir
  resolved_script_dir="$(cd "$SCRIPT_DIR" && pwd -P)"

  if [ -n "${DOTFILES:-}" ] && [ -d "$DOTFILES/compass" ]; then
    local canonical
    canonical="$(cd "$DOTFILES/compass" && pwd -P)"
    if [ "$resolved_script_dir" = "$canonical" ]; then
      return 0
    fi
  fi

  if [ "$DEV_MODE" -eq 1 ]; then
    log_warn "dev install: symlinking global skills from a non-canonical checkout:"
    log_warn "  $resolved_script_dir"
    log_warn "this is not \$DOTFILES/compass -- a destructive git operation here (reset --hard,"
    log_warn "branch delete, force-push) will break compass globally until reinstalled from canonical"
    return 0
  fi

  log_error "refusing to install from a non-canonical path: $resolved_script_dir"
  log_error "expected \$DOTFILES/compass (\$DOTFILES=${DOTFILES:-<unset>})"
  log_error "if you're intentionally developing compass here, re-run with: install.sh --dev"
  return 1
}

# --- prerequisites ------------------------------------------------------------

check_prerequisites() {
  local failed=0

  if command_exists git; then
    log_success "git: $(git --version)"
  else
    log_error "git is required but not installed"
    failed=1
  fi

  if command_exists claude; then
    local claude_version
    claude_version="$(claude --version 2>/dev/null || echo 'version unknown')"
    log_success "claude: $claude_version"
  else
    log_error "Claude Code CLI is required. Install it at: https://claude.ai/code"
    failed=1
  fi

  if command_exists python3; then
    log_success "python3: $(python3 --version 2>&1)"
  else
    log_error "python3 is required (used to register hooks and by the session-context hook itself)"
    failed=1
  fi

  return $failed
}

# --- skills -------------------------------------------------------------------

SKILLS=(warm-up wrap-up init-compass)

link_skills() {
  local skills_src="$SCRIPT_DIR/skills"
  local skills_dst="$COMPASS_HOME/skills"
  mkdir -p "$skills_dst"

  for skill in "${SKILLS[@]}"; do
    track_symlink "$skills_src/$skill" "$skills_dst/$skill"
  done
}

# --- hook ---------------------------------------------------------------------

link_hook() {
  track_symlink "$SCRIPT_DIR/hooks/compass-session-start.sh" "$COMPASS_HOME/hooks/compass-session-start.sh"
  track_symlink "$SCRIPT_DIR/hooks/session-context-nudge.py" "$COMPASS_HOME/hooks/session-context-nudge.py"
  track_symlink "$SCRIPT_DIR/hooks/tool-routing-guard.py" "$COMPASS_HOME/hooks/tool-routing-guard.py"
  track_symlink "$SCRIPT_DIR/tools/session-context" "$COMPASS_HOME/tools/session-context"
}

# --- settings.json ------------------------------------------------------------

settings_has_hook() {
  local marker="$1"
  local settings_file="$COMPASS_HOME/settings.json"
  [ -f "$settings_file" ] && grep -q "$marker" "$settings_file"
}

print_manual_hook_block() {
  local event="$1" hook_command="$2" matcher="$3"
  printf '    "hooks": {\n'
  if [ -n "$matcher" ]; then
    printf '      "%s": [{"matcher": "%s", "hooks": [{"type": "command", "command": "%s"}]}]\n' "$event" "$matcher" "$hook_command"
  else
    printf '      "%s": [{"hooks": [{"type": "command", "command": "%s"}]}]\n' "$event" "$hook_command"
  fi
  printf '    }\n'
}

register_event_hook() {
  local event="$1"
  local hook_command="$2"
  local marker="$3"
  local matcher="${4:-}"
  local settings_file="$COMPASS_HOME/settings.json"

  if settings_has_hook "$marker"; then
    log_success "$event hook already registered"
    return 0
  fi

  if ! command_exists python3; then
    log_error "python3 not found — cannot register the $event hook automatically"
    log_error "add this block to $settings_file manually:"
    print_manual_hook_block "$event" "$hook_command" "$matcher"
    return 1
  fi

  mkdir -p "$(dirname "$settings_file")"

  if python3 "$SCRIPT_DIR/scripts/register_hook.py" "$settings_file" "$event" "$hook_command" "$matcher"; then
    log_success "added $event hook to $settings_file"
    return 0
  fi

  log_error "failed to update $settings_file — add this block manually:"
  print_manual_hook_block "$event" "$hook_command" "$matcher"
  return 1
}

register_hook() {
  register_event_hook "SessionStart" "$COMPASS_HOME/hooks/compass-session-start.sh" "compass-session-start.sh"
}

register_context_nudge_hook() {
  register_event_hook "Stop" "$COMPASS_HOME/hooks/session-context-nudge.py" "session-context-nudge.py"
}

register_tool_routing_hook() {
  register_event_hook "PreToolUse" "$COMPASS_HOME/hooks/tool-routing-guard.py" "tool-routing-guard.py" "Bash"
}

# --- context-mode MCP ---------------------------------------------------------

install_context_mode() {
  if claude mcp list 2>/dev/null | grep -q "context-mode"; then
    log_success "context-mode MCP already installed"
    return 0
  fi

  if ! command_exists node; then
    log_warn "node not found — skipping context-mode install (optional)"
    log_warn "to install later: claude mcp add context-mode npx @anthropic-ai/mcp-server-context-mode"
    return 0
  fi

  log_info "installing context-mode MCP"
  if claude mcp add context-mode npx @anthropic-ai/mcp-server-context-mode 2>/dev/null; then
    log_success "context-mode MCP installed"
  else
    log_warn "context-mode install failed — run manually:"
    log_warn "  claude mcp add context-mode npx @anthropic-ai/mcp-server-context-mode"
  fi
}

# --- graphify (optional navigation layer) --------------------------------------

install_graphify() {
  if command_exists graphify; then
    log_success "graphify already installed"
    return 0
  fi

  local installer=""
  if command_exists uv; then
    installer="uv tool install graphify"
  elif command_exists pipx; then
    installer="pipx install graphify"
  elif command_exists pip; then
    installer="pip install graphify"
  else
    log_warn "no uv/pipx/pip found — skipping graphify install (optional)"
    log_warn "to install later: uv tool install graphify && graphify install"
    return 0
  fi

  log_info "installing graphify ($installer)"
  if $installer >/dev/null 2>&1 && command_exists graphify; then
    log_success "graphify installed"
  else
    log_warn "graphify install failed — run manually:"
    log_warn "  $installer && graphify install"
    return 0
  fi

  if graphify install >/dev/null 2>&1; then
    log_success "graphify skill registered"
  else
    log_warn "graphify skill registration failed — run manually: graphify install"
  fi
}

# --- main ---------------------------------------------------------------------

main() {
  while [ $# -gt 0 ]; do
    case "$1" in
      --dev) DEV_MODE=1; shift ;;
      *) shift ;;
    esac
  done

  log_info "compass install (from: $SCRIPT_DIR)"
  echo

  log_info "prerequisites"
  if ! check_prerequisites; then
    echo
    log_error "required tools missing — install them and re-run"
    exit 1
  fi
  echo

  log_info "install source"
  if ! check_install_source; then
    exit 1
  fi
  echo

  log_info "skills  ($COMPASS_HOME/skills/)"
  link_skills
  echo

  log_info "hook  ($COMPASS_HOME/hooks/)"
  link_hook
  echo

  log_info "SessionStart hook  ($COMPASS_HOME/settings.json)"
  register_hook
  echo

  log_info "Stop hook: context-usage nudge  ($COMPASS_HOME/settings.json)"
  register_context_nudge_hook
  echo

  log_info "PreToolUse hook: deterministic tool routing  ($COMPASS_HOME/settings.json)"
  register_tool_routing_hook
  echo

  log_info "context-mode MCP (token efficiency)"
  install_context_mode
  echo

  log_info "graphify (optional navigation layer)"
  install_graphify
  echo

  if [ "$LINK_FAILURES" -gt 0 ]; then
    log_error "$LINK_FAILURES symlink(s) failed -- compass is not fully installed"
    exit 1
  fi

  log_success "compass install complete"
  echo
  log_info "next steps:"
  log_info "  open any project in Claude Code"
  log_info "  run /init-compass to set it up"
  log_info "  see README.md for the full guide"
}

# Allows tests to `source` this file for its functions (check_install_source, etc.)
# without running the full install (which does real network/MCP side effects).
if [ "${COMPASS_TEST_SOURCE:-0}" != "1" ]; then
  main "$@"
fi
