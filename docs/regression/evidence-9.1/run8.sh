#!/bin/bash
# SmkS lượt 8 (spec ở docs/regression/specs-qc2/qc2-r91, config qc2g): bash run8.sh <spec-không-đuôi> [lớp] [grep] ; log -> logs/<spec>.rerun7${QC_TAG}.log ; giữ khoá qua pwlock.sh
source ~/Documents/TA_Agent_qc2/docs/regression/env/ep-r-lib.sh
cd $REPO/frontend; rlclear
( while true; do rlclear; sleep 3; done ) & RLP=$!
trap "kill $RLP 2>/dev/null" EXIT
[ -n "$2" ] && export QC_CLASS=$2
L=$REPO/docs/regression/evidence-9.1/logs/$1.rerun7${QC_TAG}.log
QC2G_DIR=../docs/regression/specs-qc2/qc2-r91 QC_REAL=1 npx playwright test -c ../docs/regression/specs-qc2/qc2g.config.ts $1.spec.ts --workers=1 --timeout=${QC_TO:-120000} --reporter=list --output=$REPO/docs/regression/evidence-9.1/pw-out ${3:+-g "$3"} 2>&1 | grep -v -e NO_COLOR -e trace-warnings > $L
grep -E "^\s+(✓|✘)" $L | cut -c1-170; grep -E -A6 "^\s+Error:|TimeoutError" $L | cut -c1-220 | head -${QC_ERRN:-60}
