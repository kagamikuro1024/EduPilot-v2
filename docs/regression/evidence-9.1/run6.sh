#!/bin/bash
# SmkR lượt 6, MỘT spec: bash run6.sh <tên-không-đuôi, ví dụ qc-uiux91-rs19-real> [tên log] [grep]
source ~/Documents/TA_Agent_qc2/docs/regression/env/ep-r-lib.sh
cd $REPO/frontend; rlclear
f=$(ls e2e/qc2-r91/$1.spec.ts 2>/dev/null || ls e2e/$1.spec.ts)
QC_REAL=1 npx playwright test -c playwright.qc-real.config.ts "$f" --workers=1 --timeout=300000 --reporter=list --project=desktop ${3:+-g "$3"} 2>&1 | grep -v -e NO_COLOR -e trace-warnings > $REPO/docs/regression/evidence-9.1/logs/${2:-$1}.rerun6.log
tail -4 $REPO/docs/regression/evidence-9.1/logs/${2:-$1}.rerun6.log | cut -c1-200
