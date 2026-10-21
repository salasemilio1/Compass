#!/bin/sh
# Smoke test of the built plugin binaries, through the launcher, on a throwaway repository, with
# no Go toolchain on PATH. Run with `make smoke` (which builds first).
set -eu

repo_root=$(cd "$(dirname "$0")/.." && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

fail() { echo "FAIL: $*" >&2; exit 1; }
pass() { echo "ok   $*"; }

# A PATH with only system tools and compass: proves the plugin needs no Go.
mkdir -p "$work/bin"
ln -s "$repo_root/plugin/bin/compass" "$work/bin/compass"
export PATH="$work/bin:/usr/bin:/bin"
export GIT_CONFIG_GLOBAL=/dev/null GIT_AUTHOR_NAME=t GIT_AUTHOR_EMAIL=t@x GIT_COMMITTER_NAME=t GIT_COMMITTER_EMAIL=t@x
command -v go >/dev/null 2>&1 && fail "go is on the test PATH"
compass version | grep -q '(format 3)' || fail "version through a symlinked launcher"
pass "launcher runs through a symlink with no Go on PATH"

project=$work/project
mkdir "$project" && cd "$project"
git init -q -b main
echo 'package main' > main.go
git add -A && git commit -q -m "code"

compass init > "$work/init.out"
grep -q "installed git pre-commit hook" "$work/init.out" || fail "init did not install the git hook: $(cat "$work/init.out")"
grep -q "compass resume" AGENTS.md || fail "AGENTS.md block"
if compass check > "$work/check.out"; then fail "check passed with the template's placeholder file in play"; fi
grep -q "file-in-play-missing" "$work/check.out" || fail "expected file-in-play-missing: $(cat "$work/check.out")"
pass "init scaffolds; check rejects the unfilled template"

head=$(git rev-parse HEAD)
cat > .compass/context/current-state.md <<EOF
# current state

Date: 2026-10-21
Commit: $head

## Objective
Smoke-test compass.

## Files in play
- \`main.go\` — the program

## Next steps
1. Add a flag.
EOF
path=$(compass record decision --claim "CLI flags use the standard library" --global --premise "no dependencies")
[ "$path" = ".compass/decisions/0001-cli-flags-use-the-standard-library.md" ] || fail "record printed $path"
compass check > /dev/null || fail "check after filling current-state: $(compass check)"
git add -A && git commit -q -m "compass" || fail "commit through the pre-commit hook"
pass "record + check + commit through the pre-commit hook"

echo 'package main // more' > main.go
git commit -qam "more code"
compass check | grep -q "warn: .*anchor-stale" || fail "stale anchor should warn"
if compass check --require-fresh > /dev/null; then fail "--require-fresh passed on a stale anchor"; fi
compass hook stop < /dev/null | grep -q '"decision":"block"' || fail "stop hook did not block on a stale anchor"
echo '{"stop_hook_active":true}' | compass hook stop | grep -q . && fail "stop hook blocked twice"
pass "staleness: warns, fails with --require-fresh, Stop blocks once"

out=$(echo "{\"cwd\":\"$project\",\"source\":\"startup\"}" | compass hook session-start)
echo "$out" | grep -q '"hookEventName":"SessionStart"' || fail "session-start output: $out"
echo "$out" | grep -q 'CLI flags use the standard library' || fail "session-start lacks the global claim"
compass resume | grep -q "Add a flag" || fail "resume lacks next steps"
pass "session-start and resume deliver state and global claims"

# A broken commit is refused by the pre-commit hook.
printf -- '---\ndate: 2026-10-21\n---\n' > .compass/decisions/0002-no-claim.md
git add -A
if git commit -q -m "bad record" 2> "$work/commit.err"; then fail "pre-commit let a claimless record through"; fi
grep -q "claim-missing" "$work/commit.err" || fail "pre-commit message: $(cat "$work/commit.err")"
rm .compass/decisions/0002-no-claim.md && git add -A
pass "pre-commit blocks a record without a claim"

# The old Python compass-cli first on PATH: commits are not blocked, doctor names it.
mkdir "$work/pybin"
printf '#!/usr/bin/env python3\nimport sys; sys.exit(2)\n' > "$work/pybin/compass"
chmod +x "$work/pybin/compass"
PATH="$work/pybin:$PATH" git commit -q --allow-empty -m "with python shadow" 2> "$work/shadow.err" || fail "the Python shadow blocked a commit"
grep -q "no Go compass on PATH" "$work/shadow.err" || fail "no shadow warning: $(cat "$work/shadow.err")"
if PATH="$work/pybin:$PATH" "$repo_root/plugin/bin/compass" doctor > "$work/doctor.out"; then fail "doctor passed with a shadow"; fi
grep -q "uv tool uninstall compass-cli" "$work/doctor.out" || fail "doctor output: $(cat "$work/doctor.out")"
pass "a shadowing Python compass-cli: commits proceed, doctor flags it"

# Hooks fail open when the platform binary is missing.
mkdir -p "$work/broken/bin" "$work/broken/libexec"
cp "$repo_root/plugin/bin/compass" "$work/broken/bin/compass"
"$work/broken/bin/compass" hook session-start < /dev/null 2> /dev/null || fail "launcher: a hook call must exit 0 without a binary"
if "$work/broken/bin/compass" check 2> /dev/null; then fail "launcher: a CLI call must fail without a binary"; fi
pass "launcher fails open for hooks, closed for the CLI"

echo "smoke: all passed"
