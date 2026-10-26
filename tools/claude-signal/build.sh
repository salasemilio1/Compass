#!/bin/bash
# Builds the signal-test fixtures from a clone of ebay-flipper (the original is never touched):
#   base/      committed HEAD, v2 .compass/ migrated to v3, two buried decisions planted, history squashed
#   arm-new/   base + compass init with the fixed binary (search line in CLAUDE.md and the brief)
#   arm-old/   base + compass init with the merged binary (search line only in AGENTS.md)
#   arm-none/  base without .compass/ or any compass instruction
set -euo pipefail
S=$(cd "$(dirname "$0")" && pwd)
COMPASS=/Users/emilio.kungfu/Dev/compass
rm -rf "$S/base" "$S/arm-"* "$S/plugin-"*

git -C "$COMPASS" archive a595c9a plugin | (mkdir -p "$S/plugin-old" && tar -x -C "$S/plugin-old")
git -C "$COMPASS" archive 202e1d0 plugin | (mkdir -p "$S/plugin-new" && tar -x -C "$S/plugin-new")

git clone -q --no-hardlinks /Users/emilio.kungfu/Dev/Misc/personal-dashboard "$S/base"
cd "$S/base"
git remote remove origin

# Remove the code-side carriers of each why, so the record is the only one.
sed -i '' 's|		return profit // a loss carries no tax|		return profit|' internal/business/forecast.go
python3 - <<'EOF'
import pathlib
p = pathlib.Path("internal/business/forecast_test.go")
s = p.read_text()
old = """		if m.ProfitCents < 0 && m.TaxCents != 0 {
			t.Fatalf("month %d taxed a loss: %d", m.Index, m.TaxCents)
		}
"""
assert old in s
p.write_text(s.replace(old, "").replace("func TestALossPaysNoTaxAndNoSalary", "func TestALossPaysNoSalary"))
EOF
grep -q 'a loss carries no tax' internal/business/forecast.go && { echo "comment still present"; exit 1; }

# v2 -> v3 records, as a user would migrate them.
python3 -I "$S/migrate.py" .compass overview.md agent-capability-map.md agent-plugin-boundaries.md \
  business-model-and-payout.md dashboard-ia-map.md local-testing-env.md pi-physical-setup.md

# Plant: t1 in the numbering gap at 0043, t3 replacing an unrelated 0075 of the right age.
rm .compass/decisions/0075-*.md
cat > .compass/decisions/0043-a-loss-month-owes-no-tax.md <<'EOF'
---
claim: "A loss month owes no tax and books no refund: the forecast's monthly tax is floored at zero"
date: 2026-09-22
---
# 0043: a loss month owes no tax and books no refund

## Context
The business forecast computes tax month by month from that month's profit (`afterTax`). A month
can lose money: a heavy overhead month, or a month of buying with few sales.

## Decision
A month with zero or negative profit pays zero tax. There is no negative tax, credit or refund in a
loss month; the loss is carried as-is.

## Alternatives
- **Symmetric tax** (a loss month shows negative tax, e.g. a credit of rate × loss): rejected. Tax
  is filed annually, not monthly, so a monthly credit is cash that nobody pays out. The salary rule
  pays from after-tax profit, so a credit would fund salary from money that doesn't exist.
EOF
cat > .compass/decisions/0075-the-top-seller-band-is-exactly-100.md <<'EOF'
---
claim: "The top \"Seller positive\" market band is exactly 100%; 99.x% sellers stay in the 98–99.9% band"
date: 2026-10-08
---
# 0075: the top "Seller positive" band is exactly 100%

## Context
Market features bucket a seller's feedback percentage into bands (`positiveBand`) that the market
pages and price features group by.

## Decision
The top band is exactly 100%. Anything from 98% up to 99.9% shares the middle band.

## Alternatives
- **A "99.5% and up" top band**, matching how buyers read eBay's feedback badge: rejected. Measured
  sell-through and return rates show sellers at 99.x% behave like 98.x% sellers; only a perfect
  record separates. A 99.5% cut would mix the two populations and blur the signal.
EOF

sed -i '' '/^This repo uses the compass protocol. Read `.compass\/INDEX.md` first.$/d' AGENTS.md CLAUDE.md
cat > .compass/context/current-state.md <<'EOF'
# current state

Date: 2026-10-21
Commit: PLACEHOLDER

## Objective
Keep the pipeline and the Workshop pages correct while the operator uses them daily.

## Next steps
1. Triage the operator's reports as they come in; fix real bugs, explain intended behaviour.
EOF

# One snapshot commit: commit messages can't carry the why.
git checkout -q --orphan snapshot
git add -A
GIT_AUTHOR_DATE=2026-10-21T12:00:00 GIT_COMMITTER_DATE=2026-10-21T12:00:00 git commit -q -m "snapshot"
git branch -D main master 2>/dev/null || true
git branch -m main

make_arm() { # name plugin-dir|""
  cp -R "$S/base" "$S/arm-$1"
  cd "$S/arm-$1"
  if [ -n "$2" ]; then
    "$S/plugin-$2/plugin/bin/compass" init >/dev/null
    sed -i '' "s/^Commit: PLACEHOLDER$/Commit: $(git rev-parse HEAD)/" .compass/context/current-state.md
    "$S/plugin-$2/plugin/bin/compass" index >/dev/null
    git add -A && GIT_AUTHOR_DATE=2026-10-21T12:05:00 GIT_COMMITTER_DATE=2026-10-21T12:05:00 git commit -q --no-verify -m "memory"
    "$S/plugin-$2/plugin/bin/compass" check 2>&1 | grep -E "^(error|ok)" | head -5 || true
  else
    rm -rf .compass
    git add -A && GIT_AUTHOR_DATE=2026-10-21T12:05:00 GIT_COMMITTER_DATE=2026-10-21T12:05:00 git commit -q -m "memory"
  fi
}
make_arm new new
make_arm old old
make_arm none ""
echo built
