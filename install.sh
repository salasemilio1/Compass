#!/usr/bin/env bash
# Install compass skills and hooks for Claude Code.
# Safe to re-run — each step is idempotent and independent.
# One failure does not stop the rest.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

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

  ln -s "$source_path" "$target_path"
  log_success "linked: $(basename "$target_path")"
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

  return $failed
}

# --- skills -------------------------------------------------------------------

SKILLS=(warm-up wrap-up init-compass)

link_skills() {
  local skills_src="$SCRIPT_DIR/skills"
  local skills_dst="$HOME/.claude/skills"
  mkdir -p "$skills_dst"

  for skill in "${SKILLS[@]}"; do
    create_symlink "$skills_src/$skill" "$skills_dst/$skill" || true
  done
}

# --- hook ---------------------------------------------------------------------

link_hook() {
  create_symlink \
    "$SCRIPT_DIR/hooks/compass-session-start.sh" \
    "$HOME/.claude/hooks/compass-session-start.sh" || true
}

# --- settings.json ------------------------------------------------------------

HOOK_COMMAND='$HOME/.claude/hooks/compass-session-start.sh'

settings_has_hook() {
  local settings_file="$HOME/.claude/settings.json"
  [ -f "$settings_file" ] && grep -q "compass-session-start.sh" "$settings_file"
}

register_hook() {
  local settings_file="$HOME/.claude/settings.json"

  if settings_has_hook; then
    log_success "SessionStart hook already registered"
    return 0
  fi

  if ! command_exists python3; then
    log_warn "python3 not found — skipping automatic settings.json update"
    log_warn "add this block to ~/.claude/settings.json manually:"
    printf '    "hooks": {\n'
    printf '      "SessionStart": [{"hooks": [{"type": "command", "command": "%s"}]}]\n' "$HOOK_COMMAND"
    printf '    }\n'
    return 0
  fi

  mkdir -p "$(dirname "$settings_file")"

  python3 - "$settings_file" "$HOOK_COMMAND" << 'PYEOF'
import json, sys, os

settings_file = sys.argv[1]
hook_command  = sys.argv[2]

data = {}
if os.path.exists(settings_file):
    with open(settings_file) as f:
        data = json.load(f)

data.setdefault("hooks", {}).setdefault("SessionStart", [])
data["hooks"]["SessionStart"].append(
    {"hooks": [{"type": "command", "command": hook_command}]}
)

tmp = settings_file + ".tmp"
with open(tmp, "w") as f:
    json.dump(data, f, indent=2)
    f.write("\n")
os.replace(tmp, settings_file)
PYEOF

  if [ $? -eq 0 ]; then
    log_success "registered SessionStart hook in $settings_file"
  else
    log_error "failed to update $settings_file — add the hook manually"
  fi
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

# --- main ---------------------------------------------------------------------

main() {
  log_info "compass install (from: $SCRIPT_DIR)"
  echo

  log_info "prerequisites"
  if ! check_prerequisites; then
    echo
    log_error "required tools missing — install them and re-run"
    exit 1
  fi
  echo

  log_info "skills  (~/.claude/skills/)"
  link_skills
  echo

  log_info "hook  (~/.claude/hooks/)"
  link_hook
  echo

  log_info "SessionStart hook  (~/.claude/settings.json)"
  register_hook
  echo

  log_info "context-mode MCP (token efficiency)"
  install_context_mode
  echo

  log_success "compass install complete"
  echo
  log_info "next steps:"
  log_info "  open any project in Claude Code"
  log_info "  run /init-compass to set it up"
  log_info "  see README.md for the full guide"
}

main "$@"
