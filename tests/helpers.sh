# Sourced by tests/test_*.sh. Plain-bash assertions, no framework dependency.

TESTS_RUN=0
TESTS_FAILED=0

assert_eq() {
  local expected="$1" actual="$2" msg="${3:-}"
  TESTS_RUN=$((TESTS_RUN + 1))
  if [ "$expected" != "$actual" ]; then
    TESTS_FAILED=$((TESTS_FAILED + 1))
    echo "  FAIL: $msg -- expected [$expected] got [$actual]"
  else
    echo "  ok: $msg"
  fi
}

assert_exit_code() {
  local expected="$1" msg="$2"; shift 2
  TESTS_RUN=$((TESTS_RUN + 1))
  local out
  out="$("$@" 2>&1)"
  local actual=$?
  if [ "$actual" != "$expected" ]; then
    TESTS_FAILED=$((TESTS_FAILED + 1))
    echo "  FAIL: $msg -- expected exit $expected got $actual"
    echo "$out" | sed 's/^/    /'
  else
    echo "  ok: $msg"
  fi
}

assert_file_contains() {
  local file="$1" pattern="$2" msg="$3"
  TESTS_RUN=$((TESTS_RUN + 1))
  if grep -qE -- "$pattern" "$file" 2>/dev/null; then
    echo "  ok: $msg"
  else
    TESTS_FAILED=$((TESTS_FAILED + 1))
    echo "  FAIL: $msg -- pattern [$pattern] not found in $file"
  fi
}

assert_file_not_contains() {
  local file="$1" pattern="$2" msg="$3"
  TESTS_RUN=$((TESTS_RUN + 1))
  if grep -qE -- "$pattern" "$file" 2>/dev/null; then
    TESTS_FAILED=$((TESTS_FAILED + 1))
    echo "  FAIL: $msg -- pattern [$pattern] unexpectedly found in $file"
  else
    echo "  ok: $msg"
  fi
}

report_and_exit() {
  echo "  -- $TESTS_RUN checks, $TESTS_FAILED failed"
  [ "$TESTS_FAILED" -eq 0 ]
}
