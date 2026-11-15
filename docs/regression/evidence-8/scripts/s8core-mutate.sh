#!/bin/bash
# s8core: TC-P6-02-05..09, 18, 21 — đột biến M1..M7 trên BẢN SAO (git worktree /tmp/qc-mut-s8core), test của dev phải ĐỎ. Không đụng worktree chính.
W=/tmp/qc-mut-s8core; G=$W/backend-go/internal/grade; L=$HOME/Documents/TA_Agent_qc2/docs/regression/evidence-8/logs/s8core-mutations.log
: > $L
cd $W/backend-go
run() { # run <tên> <regex test> ; in tên test đỏ (cả ca con)
  out=$(go test -count=1 -run "$2" ./internal/grade 2>&1); rc=$?
  echo "=== $1 rc=$rc" >> $L
  echo "$out" | grep -E '^\s*--- FAIL' | sed 's/ (.*//' | head -40 >> $L
  echo "$out" | grep -E '^(ok|FAIL|panic)' >> $L
}
reset() { git -C $W checkout -q -- backend-go; rm -f $G/qc_mut.go; }
T='TestEngineExpected30|TestRoundHalfUpTable|TestEngineCases|TestNoIntermediateRounding|TestFinalExactlyThreeNotFail|TestNoFloatInGrade|TestForbiddenImports|TestFailConditions'
reset; echo "=== BASE (không đột biến)" >> $L; run base "$T"
reset; sed -i.bak 's/r.HPRaw.DivRound(step, DivPrecision).Round(0).Mul(step)/r.HPRaw.DivRound(step, DivPrecision).RoundBank(0).Mul(step)/' $G/engine.go; grep -c RoundBank $G/engine.go >> $L; run "M1 nửa-chẵn" "$T"
reset; sed -i.bak 's/decimal.Min(in.BonusPoints, s.Rules.Bonus.Cap)/decimal.Min(in.BonusPoints, s.Rules.Bonus.Cap.Mul(decimal.RequireFromString("1.5")))/' $G/engine.go; run "M2 trần cộng 1,0→1,5" "$T"
reset; sed -i.bak 's/in.Absent-p.Free)))/in.Absent-p.Free-1)))/' $G/engine.go; run "M3 miễn 2→3 buổi" "$T"
reset; sed -i.bak 's/r.QT = decimal.Min(MaxScale, decimal.Max(decimal.Zero, r.QTBeforeClamp))/r.QT = decimal.Min(MaxScale, decimal.Max(decimal.Zero, r.QTBeforeClamp)).Round(1)/' $G/engine.go; run "M4 làm tròn trung gian QT" "$T"
reset; sed -i.bak 's/r.CK.LessThan(f.Value)/r.CK.LessThanOrEqual(f.Value)/' $G/engine.go; run "M5 liệt CK<=3,0" "$T"
reset; printf 'package grade\n\nvar _ = float64(1)\n' > $G/qc_mut.go; run "M6 float64 trong grade" "TestNoFloatInGrade"
echo "--- lệnh grep cổng P6 (M6):" >> $L; grep -rnE 'float(32|64)|math\.(Round|Floor|Ceil)|ParseFloat|RoundBank|StringFixedBank' $G --include=*.go | grep -v _test.go >> $L; echo "grep rc=$? (0 = có dòng = cổng đỏ)" >> $L
reset; printf 'package grade\n\nimport _ "github.com/edupilot/backend-go/internal/activity"\n' > $G/qc_mut.go; run "M7 import activity" "TestForbiddenImports"
reset; echo "=== sau khôi phục:" >> $L; run restored "$T"
git -C $W status --short | head -3 >> $L; echo DONE >> $L
