#!/bin/bash
# Chạy tuần tự trên stack riêng smk91 (seed sạch): log -> logs/qc-uiux91-r<w>-real.rerun5.log ; $1.. = tên spec rút gọn (w4, r91sv...)
source ~/Documents/TA_Agent_qc2/docs/regression/env/ep-r-lib.sh
export QC_BASE=https://localhost:1343
cd $REPO/frontend
for w in "$@"; do
  f=$(ls e2e/qc2-r91/qc-uiux91-r${w}-real.spec.ts 2>/dev/null || ls e2e/qc-uiux91-${w}-real.spec.ts)
  QC_REAL=1 npx playwright test -c playwright.qc-real.config.ts "$f" --workers=1 --timeout=300000 --reporter=list > $REPO/docs/regression/evidence-9.1/logs/qc-uiux91-r${w}-real.rerun5.log 2>&1
  echo "$(date +%T) $w exit=$?" >> /tmp/qc2-s91-run5.log
done
