#!/bin/bash
# Lượt 6 (SmkR): chạy tuần tự trên ep-r hoặc stack khác (QC_BASE do môi trường đặt); log -> logs/qc-uiux91-r<w>-real.rerun6.log
source ~/Documents/TA_Agent_qc2/docs/regression/env/ep-r-lib.sh
[ -n "$QC_BASE_OVERRIDE" ] && export QC_BASE=$QC_BASE_OVERRIDE
cd $REPO/frontend
for w in "$@"; do
  f=$(ls e2e/qc2-r91/qc-uiux91-r${w}-real.spec.ts 2>/dev/null || ls e2e/qc-uiux91-${w}-real.spec.ts) || { echo "KHÔNG THẤY SPEC $w" >> /tmp/smkr-91-run.log; continue; }
  rlclear
  QC_REAL=1 npx playwright test -c playwright.qc-real.config.ts "$f" --workers=1 --timeout=300000 --reporter=list > $REPO/docs/regression/evidence-9.1/logs/qc-uiux91-r${w}-real.rerun6.log 2>&1
  echo "$(date +%T) $w exit=$?" >> /tmp/smkr-91-run.log
done
