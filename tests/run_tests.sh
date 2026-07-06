#!/usr/bin/env bash
# Discovers and runs every tests/test_*.sh, plain bash, no framework.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
overall_fail=0

for t in "$SCRIPT_DIR"/test_*.sh; do
  [ -e "$t" ] || continue
  echo "=== $(basename "$t") ==="
  if bash "$t"; then
    echo "PASS: $(basename "$t")"
  else
    echo "FAIL: $(basename "$t")"
    overall_fail=1
  fi
  echo
done

exit $overall_fail
