#!/bin/bash
# Chạy lại audit 10.7 (qc-ui-audit-real.spec.ts) trên ep-r; pha rr-defrerun; mỗi giai đoạn qua khoá Playwright.
R=~/Documents/TA_Agent_qc2/docs/regression; D=$R/evidence-defrerun/R10.7; PH=rr-defrerun
export NODE_TLS_REJECT_UNAUTHORIZED=0 QC_REAL=1 QC_BASE=https://localhost:1143 QC_PW='Edupilot#Seed-2026' QC_PHASE=$PH QC_OUT=$R/evidence-10.7/audit
cd ~/Documents/TA_Agent_qc2/frontend
run(){ n=$1; shift; echo "== $n $(date +%T)" >> $D/all.log; ( env "$@" /tmp/qc2-pw.sh timeout 480 npx playwright test -c playwright.qc-audit.config.ts --output=$D/pw-out ) > $D/$n.log 2>&1; echo "rc=$?" >> $D/$n.log; tail -6 $D/$n.log >> $D/all.log; sleep 30; }
for r in student ta teacher admin; do run data-$r QC_STATE=data QC_ROLE=$r; done
for r in student ta teacher; do run empty-$r QC_STATE=empty QC_CLASS=QCEMPTYr107 QC_ROLE=$r; done
run empty-admin QC_STATE=empty QC_ROLE=admin
run loading QC_STATE=loading
run error QC_STATE=error
cd $R/evidence-10.7 && python3 scripts/audit_report.py $PH > $D/report.log 2>&1; echo rc=$? >> $D/report.log
echo ALLDONE >> $D/all.log
