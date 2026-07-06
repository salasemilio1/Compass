#!/usr/bin/env bash
# Unit tests for scripts/compass-check.sh -- one test per logical branch, plus a smoke test.
set -uo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$TEST_DIR/.." && pwd)"
CHECK="$REPO_ROOT/scripts/compass-check.sh"
# shellcheck source=./helpers.sh
source "$TEST_DIR/helpers.sh"

WORK=""
cleanup() { [ -n "$WORK" ] && rm -rf "$WORK"; }
trap cleanup EXIT

new_fixture() {
  WORK="$(mktemp -d)"
  mkdir -p "$WORK/.compass/context" "$WORK/.compass/decisions" "$WORK/.compass/logs" "$WORK/.compass/experiments"
  git -C "$WORK" init -q
  git -C "$WORK" config user.email test@test.com
  git -C "$WORK" config user.name test
  cat > "$WORK/.compass/decisions/0001-first.md" <<'EOF'
# 0001 -- first
Date: 2026-01-01

## Decision
placeholder
EOF
  cat > "$WORK/.compass/context/current-state.md" <<'EOF'
# current state
Date: 2026-01-01

placeholder
EOF
  build_index_with_logs "$WORK/.compass/INDEX.md" 0
  ( cd "$WORK" && git add -A && git commit -q -m init )
  # base fixture is "clean" by default -- give it a valid Commit: anchor so tests
  # targeting other checks (A, B1, E) don't trip the D check incidentally.
  printf 'Commit: %s\n' "$(git -C "$WORK" rev-parse HEAD)" >> "$WORK/.compass/context/current-state.md"
  echo "$WORK"
}

build_index_with_logs() {
  local file="$1" count="$2"
  {
    echo "# compass index -- fixture"
    echo
    echo "## decisions"
    echo "- [0001-first.md](decisions/0001-first.md) -- first decision"
    echo
    echo "## logs"
    echo "<!-- newest first -->"
    local i
    for ((i = 1; i <= count; i++)); do
      printf -- '- [entry-%03d.md](logs/entry-%03d.md) -- log entry %d\n' "$i" "$i" "$i"
    done
  } > "$file"
}

echo "-- A: index bloat"
{
  d="$(new_fixture)"
  build_index_with_logs "$d/.compass/INDEX.md" 5
  assert_exit_code 0 "under cap: check-only passes" "$CHECK" --repo-root "$d" .compass
  rm -rf "$d"

  d="$(new_fixture)"
  build_index_with_logs "$d/.compass/INDEX.md" 20
  assert_exit_code 1 "over cap: check-only fails" "$CHECK" --repo-root "$d" .compass
  "$CHECK" --fix --repo-root "$d" .compass >/dev/null 2>&1
  kept=$(grep -c '^- \[entry-' "$d/.compass/INDEX.md")
  assert_eq "15" "$kept" "over cap --fix: INDEX.md keeps exactly 15"
  assert_file_contains "$d/.compass/INDEX.md" '^- \[entry-001\.md\]' "over cap --fix: newest entry kept"
  assert_file_not_contains "$d/.compass/INDEX.md" '^- \[entry-020\.md\]' "over cap --fix: oldest entry removed from INDEX"
  assert_file_contains "$d/.compass/logs/archive.md" 'entry-020' "over cap --fix: oldest entry archived"
  archived=$(grep -c '^- \[entry-' "$d/.compass/logs/archive.md")
  assert_eq "5" "$archived" "over cap --fix: archived exactly the 5 overflow entries"
  assert_exit_code 0 "over cap --fix: check-only now passes" "$CHECK" --repo-root "$d" .compass
  "$CHECK" --fix --repo-root "$d" .compass >/dev/null 2>&1
  archived_again=$(grep -c '^- \[entry-' "$d/.compass/logs/archive.md")
  assert_eq "5" "$archived_again" "re-running --fix is idempotent (no duplicate archiving)"
  rm -rf "$d"
}

echo "-- B1: ADR supersession"
{
  d="$(new_fixture)"
  assert_exit_code 0 "no Supersedes line: check-only passes" "$CHECK" --repo-root "$d" .compass
  rm -rf "$d"

  d="$(new_fixture)"
  cat > "$d/.compass/decisions/0004-second.md" <<'EOF'
# 0004 -- second
Date: 2026-02-01
Supersedes: 0001

## Decision
placeholder
EOF
  assert_exit_code 1 "missing marker: check-only fails" "$CHECK" --repo-root "$d" .compass
  "$CHECK" --fix --repo-root "$d" .compass >/dev/null 2>&1
  assert_file_contains "$d/.compass/INDEX.md" '\[SUPERSEDED by 0004\] \[0001-first\.md\]' "missing marker --fix: rewrites INDEX row"
  assert_exit_code 0 "missing marker --fix: check-only now passes" "$CHECK" --repo-root "$d" .compass

  before="$(cat "$d/.compass/INDEX.md")"
  "$CHECK" --fix --repo-root "$d" .compass >/dev/null 2>&1
  after="$(cat "$d/.compass/INDEX.md")"
  assert_eq "$before" "$after" "marker already correct: --fix does not double-prefix"
  rm -rf "$d"
}

echo "-- D: current-state.md Commit: anchor"
{
  d="$(new_fixture)"
  sed -i.bak '/^Commit: /d' "$d/.compass/context/current-state.md" && rm -f "$d/.compass/context/current-state.md.bak"
  assert_exit_code 1 "missing Commit: line: check-only fails" "$CHECK" --repo-root "$d" .compass
  head_hash="$(git -C "$d" rev-parse HEAD)"
  "$CHECK" --fix --repo-root "$d" .compass >/dev/null 2>&1
  assert_file_contains "$d/.compass/context/current-state.md" "^Commit: $head_hash$" "missing --fix: stamps valid HEAD hash"
  assert_exit_code 0 "missing --fix: check-only now passes" "$CHECK" --repo-root "$d" .compass
  rm -rf "$d"

  d="$(new_fixture)"
  sed -i.bak 's/^Commit: .*/Commit: not-a-hash/' "$d/.compass/context/current-state.md" && rm -f "$d/.compass/context/current-state.md.bak"
  assert_exit_code 1 "malformed Commit: line: check-only fails" "$CHECK" --repo-root "$d" .compass
  head_hash="$(git -C "$d" rev-parse HEAD)"
  "$CHECK" --fix --repo-root "$d" .compass >/dev/null 2>&1
  assert_file_not_contains "$d/.compass/context/current-state.md" 'not-a-hash' "malformed --fix: removes bad line"
  assert_file_contains "$d/.compass/context/current-state.md" "^Commit: $head_hash$" "malformed --fix: replaces with valid hash"
  rm -rf "$d"

  d="$(new_fixture)"
  sed -i.bak 's/^Commit: .*/Commit: abc1234/' "$d/.compass/context/current-state.md" && rm -f "$d/.compass/context/current-state.md.bak"
  assert_exit_code 0 "well-formed Commit: line: check-only passes" "$CHECK" --repo-root "$d" .compass
  "$CHECK" --fix --repo-root "$d" .compass >/dev/null 2>&1
  assert_file_contains "$d/.compass/context/current-state.md" '^Commit: abc1234$' "valid --fix: leaves well-formed hash untouched"
  rm -rf "$d"
}

echo "-- E: AGENTS.md spec-version (warn-only)"
{
  d="$(new_fixture)"
  printf '<!-- compass-spec-version: 1 -->\n' > "$d/AGENTS.md"
  out="$("$CHECK" --repo-root "$d" .compass 2>&1)"
  assert_exit_code 0 "matching spec version: still exits 0" "$CHECK" --repo-root "$d" .compass
  if echo "$out" | grep -q 'spec version'; then
    TESTS_RUN=$((TESTS_RUN + 1)); TESTS_FAILED=$((TESTS_FAILED + 1))
    echo "  FAIL: matching spec version should not warn"
  else
    TESTS_RUN=$((TESTS_RUN + 1)); echo "  ok: matching spec version does not warn"
  fi
  rm -rf "$d"

  d="$(new_fixture)"
  printf '<!-- compass-spec-version: 999 -->\n' > "$d/AGENTS.md"
  out="$("$CHECK" --repo-root "$d" .compass 2>&1)"
  assert_exit_code 0 "mismatched spec version: still exits 0 (never blocks)" "$CHECK" --repo-root "$d" .compass
  assert_file_contains <(echo "$out") "spec version" "mismatched spec version: prints a warning"
  rm -rf "$d"
}

echo "-- smoke test: mixed violations, single --fix pass resolves all"
{
  d="$(new_fixture)"
  build_index_with_logs "$d/.compass/INDEX.md" 20
  cat > "$d/.compass/decisions/0004-second.md" <<'EOF'
# 0004 -- second
Date: 2026-02-01
Supersedes: 0001

## Decision
placeholder
EOF
  sed -i.bak 's/^Commit: .*/Commit: bogus/' "$d/.compass/context/current-state.md" && rm -f "$d/.compass/context/current-state.md.bak"

  assert_exit_code 1 "smoke: fixture starts with violations" "$CHECK" --repo-root "$d" .compass
  "$CHECK" --fix --repo-root "$d" .compass >/dev/null 2>&1
  assert_exit_code 0 "smoke: single --fix pass resolves all blocking violations" "$CHECK" --repo-root "$d" .compass
  rm -rf "$d"
}

report_and_exit
