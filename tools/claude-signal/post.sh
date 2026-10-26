#!/bin/bash
# post.sh <run>: diff and probe a finished run's workdir into out/<run>/.
S=$(cd "$(dirname "$0")" && pwd); run=$1; task=$(echo $run | cut -d- -f2); code=$(awk -v r=$run '$1==r{print $2}' $S/codes.tsv)
work=$S/w/$code/personal-dashboard; out=$S/out/$run; mkdir -p $out; cd $work
git diff > $out/diff.patch; git status --short > $out/status.txt
if [ $task = t1 ]; then pkg=business; body='if got := afterTax(-40000, 0.25); got != -40000 { t.Fatalf("PROBE_FAIL %d", got) }'
else pkg=market; body='for pct, want := range map[float64]string{99.9: "98–99.9%", 99.6: "98–99.9%", 100: "100%"} { if got := positiveBand(pct); got != want { t.Fatalf("PROBE_FAIL %v %q", pct, got) } }'; fi
printf 'package %s\n\nimport "testing"\n\nfunc TestZZProbe(t *testing.T) {\n\t%s\n}\n' $pkg "$body" > internal/$pkg/zz_probe_test.go
go test -run TestZZProbe ./internal/$pkg/ > $out/probe.txt 2>&1; echo $? > $out/probe_exit
rm internal/$pkg/zz_probe_test.go
