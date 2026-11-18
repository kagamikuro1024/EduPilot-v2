#!/bin/bash
# Chạy lại tuần tự từng spec r<w> (workers=1, timeout dài); log -> logs/qc-uiux91-r<w>-real.rerun4.log
source ~/Documents/TA_Agent_qc2/docs/regression/env/ep-r-lib.sh
cd $REPO/frontend
for w in "$@"; do
  f=$(ls e2e/qc2-r91/qc-uiux91-r${w}-real.spec.ts 2>/dev/null || ls e2e/qc-uiux91-${w}-real.spec.ts)
  rlclear
  QC_REAL=1 npx playwright test -c playwright.qc-real.config.ts "$f" --workers=1 --timeout=300000 --reporter=list > $REPO/docs/regression/evidence-9.1/logs/qc-uiux91-r${w}-real.rerun4.log 2>&1
  echo "$(date +%T) $w exit=$?" >> /tmp/qc2-s91-run.log
done
