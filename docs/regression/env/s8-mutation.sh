#!/bin/bash
# TC-P6-SEED-25: đột biến trong checkout ovnfs qc2mut (không đụng worktree); gate-p6 phải đỏ
L=~/Documents/TA_Agent_qc2/docs/regression/evidence-8/logs
cd /Users/kuro/checkouts/qc2mut || exit 1
export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true
export API_URL=http://127.0.0.1:18231/api/v1 SEED_DEFAULT_PASSWORD='Edupilot#Seed-2026' NODE_TLS_REJECT_UNAUTHORIZED=0
# đột biến 1: float64 trong internal/grade
printf 'package grade\n\nvar _ = float64(1)\n' > backend-go/internal/grade/qc2_mut.go
( time bash scripts/gate-p6.sh ) > $L/seed25-mut1-float64.log 2>&1; echo "rc=$?" >> $L/seed25-mut1-float64.log
rm -f backend-go/internal/grade/qc2_mut.go
# đột biến 2: làm tròn nửa-chẵn thay vì nửa-lên (engine.go dòng 375)
sed -i 's#r.HPRaw.DivRound(step, DivPrecision).Round(0).Mul(step)#r.HPRaw.DivRound(step, DivPrecision).RoundBank(0).Mul(step)#' backend-go/internal/grade/engine.go
grep -n RoundBank backend-go/internal/grade/engine.go > $L/seed25-mut2-diff.txt
( time bash scripts/gate-p6.sh ) > $L/seed25-mut2-halfeven.log 2>&1; echo "rc=$?" >> $L/seed25-mut2-halfeven.log
git checkout -q backend-go/internal/grade/engine.go 2>/dev/null; echo DONE >> $L/seed25-mut2-halfeven.log
