#!/bin/bash
# SmkS lượt 7: bash run7.sh <spec-không-đuôi, vd qc-uiux91-rw4-real> [lớp, mặc định 761987] [grep] ; log -> logs/<spec>.rerun7${QC_TAG}.log ; giữ khoá /tmp/qc2-pw.lock (pwlock.sh)
source ~/Documents/TA_Agent_qc2/docs/regression/env/ep-r-lib.sh
cd $REPO/frontend; rlclear
f=$(ls e2e/qc2-r91/$1.spec.ts 2>/dev/null || ls e2e/$1.spec.ts)
[ -n "$2" ] && export QC_CLASS=$2
QC_REAL=1 npx playwright test -c playwright.qc-real.config.ts "$f" --workers=1 --timeout=${QC_TO:-120000} --reporter=list --project=desktop ${3:+-g "$3"} 2>&1 | grep -v -e NO_COLOR -e trace-warnings > $REPO/docs/regression/evidence-9.1/logs/$1.rerun7${QC_TAG}.log
grep -E "^\s+(✓|✘)" $REPO/docs/regression/evidence-9.1/logs/$1.rerun7${QC_TAG}.log | cut -c1-170; grep -E -A6 "^\s+Error:|TimeoutError" $REPO/docs/regression/evidence-9.1/logs/$1.rerun7${QC_TAG}.log | cut -c1-220 | head -${QC_ERRN:-60}
