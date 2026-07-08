#!/usr/bin/env bash
# Deterministic checker+fixer for a .compass/ directory.
#
# Closes gaps A (INDEX.md archival), B1 (ADR supersession), D (current-state.md
# Commit: anchor) fully; E (AGENTS.md spec-version drift) is detection-only and
# never blocks — its remediation is a manual content edit.
#
# Modes:
#   compass-check.sh [--repo-root DIR] [.compass path]        check-only, exits 1 on a real violation
#   compass-check.sh --fix [--repo-root DIR] [.compass path]  mutates to correct A/B1/D
#
# Every decision here is mechanical (position in a list, an explicit metadata field, a
# regex-shaped line, a version-number comparison) -- no semantic judgment is exercised.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MAX_ENTRIES=15

# --- logging ------------------------------------------------------------------

if [ -t 1 ]; then
  _reset="$(printf '\033[0m')"
  _yellow="$(printf '\033[33m')"
  _red="$(printf '\033[31m')"
  _green="$(printf '\033[32m')"
else
  _reset=""; _yellow=""; _red=""; _green=""
fi

log_warn()    { printf '%swarn:%s %s\n' "$_yellow" "$_reset" "$*" >&2; }
log_error()   { printf '%serror:%s %s\n' "$_red" "$_reset" "$*" >&2; }
log_success() { printf '%s ok%s  %s\n' "$_green" "$_reset" "$*"; }

usage() {
  cat <<'EOF'
Usage: compass-check.sh [--fix] [--repo-root DIR] [COMPASS_DIR]

  COMPASS_DIR    path to the .compass/ directory to check (default: ./.compass)
  --repo-root    git repo root, used to resolve HEAD for the Commit: stamp (default: .)
  --fix          mutate to correct violations instead of just reporting them
EOF
}

# --- A: INDEX.md archival ------------------------------------------------------

extract_section_lines() {
  local file="$1" heading="$2"
  awk -v h="## $heading" '
    $0 == h { insec = 1; next }
    insec && /^## / { insec = 0 }
    insec && /^- \[/ { print }
  ' "$file"
}

check_index_bloat() {
  local index_file="$1"
  local violation=0 heading count
  for heading in logs experiments; do
    count=$(extract_section_lines "$index_file" "$heading" | wc -l | tr -d ' ')
    if [ "$count" -gt "$MAX_ENTRIES" ]; then
      log_error "INDEX.md: ## $heading has $count entries (cap $MAX_ENTRIES) -- run with --fix to archive overflow"
      violation=1
    fi
  done
  return $violation
}

fix_section_cap() {
  local index_file="$1" heading="$2" archive_file="$3"
  local count
  count=$(extract_section_lines "$index_file" "$heading" | wc -l | tr -d ' ')
  if [ "$count" -le "$MAX_ENTRIES" ]; then
    return 0
  fi

  if [ ! -f "$archive_file" ]; then
    printf '# archived %s entries\n\n' "$heading" > "$archive_file"
  fi
  extract_section_lines "$index_file" "$heading" | tail -n "+$((MAX_ENTRIES + 1))" >> "$archive_file"

  local tmp
  tmp="$(mktemp)"
  awk -v h="## $heading" -v n="$MAX_ENTRIES" '
    $0 == h { print; insec = 1; kept = 0; next }
    insec && /^## / { insec = 0; print; next }
    insec && /^- \[/ {
      kept++
      if (kept <= n) print
      next
    }
    { print }
  ' "$index_file" > "$tmp"

  # No entry may vanish in the rewrite: what's kept in INDEX.md plus what's newly
  # archived must equal what was there before. Abort loudly rather than overwrite
  # INDEX.md with a partial/corrupt result.
  local kept_count
  kept_count=$(extract_section_lines "$tmp" "$heading" | wc -l | tr -d ' ')
  if [ ! -s "$tmp" ] || [ "$kept_count" -ne "$MAX_ENTRIES" ]; then
    log_error "INDEX.md: rewrite for ## $heading looks corrupt (kept $kept_count, expected $MAX_ENTRIES) -- aborting, no changes made"
    rm -f "$tmp"
    return 1
  fi
  mv "$tmp" "$index_file"

  log_success "INDEX.md: archived $((count - MAX_ENTRIES)) $heading entries to $archive_file"
}

# --- B1: ADR supersession -------------------------------------------------------

superseded_marker() { printf '[SUPERSEDED by %s]' "$1"; }

each_supersession() {
  # prints "<new_id> <old_basename>" for every decisions/*.md with a Supersedes: line
  # whose target actually exists on disk.
  local compass_dir="$1" f supersedes_id new_id old_file
  for f in "$compass_dir"/decisions/*.md; do
    [ -e "$f" ] || continue
    supersedes_id=$(grep -m1 -E '^Supersedes: [0-9]{4}$' "$f" | awk '{print $2}') || true
    [ -z "${supersedes_id:-}" ] && continue
    new_id=$(basename "$f" | grep -oE '^[0-9]{4}')
    old_file=$(find "$compass_dir/decisions" -maxdepth 1 -name "${supersedes_id}-*.md" | head -1)
    [ -z "$old_file" ] && continue
    printf '%s %s\n' "$new_id" "$(basename "$old_file")"
  done
}

check_supersession() {
  local compass_dir="$1" index_file="$compass_dir/INDEX.md"
  local violation=0 new_id old_basename index_line
  while read -r new_id old_basename; do
    [ -z "${new_id:-}" ] && continue
    index_line=$(grep -F "](decisions/$old_basename)" "$index_file" | head -1)
    if [ -n "$index_line" ] && [[ "$index_line" != *"$(superseded_marker "$new_id")"* ]]; then
      log_error "decisions: $old_basename is superseded by $new_id but INDEX.md isn't marked -- run --fix"
      violation=1
    fi
  done < <(each_supersession "$compass_dir")
  return $violation
}

fix_supersession() {
  local compass_dir="$1" index_file="$compass_dir/INDEX.md"
  local new_id old_basename tmp marker
  while read -r new_id old_basename; do
    [ -z "${new_id:-}" ] && continue
    marker="$(superseded_marker "$new_id")"
    tmp="$(mktemp)"
    awk -v link="](decisions/$old_basename)" -v marker="$marker " '
      index($0, link) > 0 && index($0, "[SUPERSEDED by") == 0 {
        sub(/^- /, "- " marker)
      }
      { print }
    ' "$index_file" > "$tmp"
    if [ ! -s "$tmp" ]; then
      log_error "INDEX.md: rewrite while marking $old_basename produced an empty file -- aborting, no changes made"
      rm -f "$tmp"
      continue
    fi
    mv "$tmp" "$index_file"
    log_success "INDEX.md: marked decisions/$old_basename as $marker"
  done < <(each_supersession "$compass_dir")
}

# --- D: current-state.md Commit: anchor -----------------------------------------

check_current_state_commit() {
  local compass_dir="$1" repo_root="$2" file="$compass_dir/context/current-state.md"
  [ -f "$file" ] || return 0

  local stamped
  stamped=$(grep -oE '^Commit: [0-9a-f]{7,40}$' "$file" | awk '{print $2}')
  if [ -z "${stamped:-}" ]; then
    log_error "context/current-state.md: missing or malformed 'Commit: <hash>' line -- run --fix"
    return 1
  fi

  # Shape alone doesn't prove the stamp is real -- confirm it resolves in this repo.
  if ! git -C "$repo_root" cat-file -e "${stamped}^{commit}" 2>/dev/null; then
    log_error "context/current-state.md: Commit: $stamped does not resolve in this repo -- run --fix"
    return 1
  fi
  return 0
}

fix_current_state_commit() {
  local compass_dir="$1" repo_root="$2" file="$compass_dir/context/current-state.md"
  [ -f "$file" ] || return 0

  local existing
  existing=$(grep -oE '^Commit: [0-9a-f]{7,40}$' "$file" | awk '{print $2}')
  if [ -n "${existing:-}" ] && git -C "$repo_root" cat-file -e "${existing}^{commit}" 2>/dev/null; then
    return 0
  fi

  local head_hash
  head_hash=$(git -C "$repo_root" rev-parse HEAD 2>/dev/null) || {
    log_warn "context/current-state.md: not a git repo, skipping Commit: stamp"
    return 0
  }

  local stripped tmp
  stripped="$(mktemp)"
  grep -vE '^Commit: ' "$file" > "$stripped"

  if grep -qE '^Date: ' "$stripped"; then
    tmp="$(mktemp)"
    awk -v hash="Commit: $head_hash" '{ print } /^Date: / { print hash }' "$stripped" > "$tmp"
    mv "$tmp" "$file"
  else
    tmp="$(mktemp)"
    { printf 'Commit: %s\n' "$head_hash"; cat "$stripped"; } > "$tmp"
    mv "$tmp" "$file"
  fi
  rm -f "$stripped"
  log_success "context/current-state.md: stamped Commit: $head_hash"
}

# --- E: AGENTS.md spec-version drift (warn-only, never blocks) ------------------

current_spec_version() {
  local spec_file="$SCRIPT_DIR/../SPEC.md"
  [ -f "$spec_file" ] || return 1
  grep -oE '\*\*Spec version:\*\* [0-9]+' "$spec_file" | grep -oE '[0-9]+$'
}

check_spec_version() {
  local repo_root="$1" agents_file="$repo_root/AGENTS.md"
  [ -f "$agents_file" ] || return 0

  local stamped
  stamped=$(grep -oE '<!-- compass-spec-version: [0-9]+ -->' "$agents_file" | grep -oE '[0-9]+')
  if [ -z "${stamped:-}" ]; then
    log_warn "AGENTS.md: no compass-spec-version stamp found (informational only)"
    return 0
  fi

  local current
  current=$(current_spec_version) || return 0
  [ -z "${current:-}" ] && return 0

  if [ "$stamped" != "$current" ]; then
    log_warn "AGENTS.md: stamped spec version $stamped, compass SPEC.md is at $current -- review for drift (non-blocking)"
  fi
  return 0
}

# --- main -----------------------------------------------------------------------

main() {
  local fix=0 compass_dir=".compass" repo_root="."

  while [ $# -gt 0 ]; do
    case "$1" in
      --fix) fix=1; shift ;;
      --repo-root) repo_root="$2"; shift 2 ;;
      -h|--help) usage; exit 0 ;;
      *) compass_dir="$1"; shift ;;
    esac
  done

  repo_root="$(cd "$repo_root" && pwd)"
  case "$compass_dir" in
    /*) : ;;
    *) compass_dir="$repo_root/$compass_dir" ;;
  esac

  if [ ! -d "$compass_dir" ]; then
    log_error "no .compass/ directory at: $compass_dir"
    exit 1
  fi

  local index_file="$compass_dir/INDEX.md"
  local exit_code=0

  if [ "$fix" -eq 1 ]; then
    fix_section_cap "$index_file" "logs" "$compass_dir/logs/archive.md"
    fix_section_cap "$index_file" "experiments" "$compass_dir/experiments/archive.md"
    fix_supersession "$compass_dir"
    fix_current_state_commit "$compass_dir" "$repo_root"
  else
    check_index_bloat "$index_file" || exit_code=1
    check_supersession "$compass_dir" || exit_code=1
    check_current_state_commit "$compass_dir" "$repo_root" || exit_code=1
  fi

  check_spec_version "$repo_root"

  exit $exit_code
}

main "$@"
