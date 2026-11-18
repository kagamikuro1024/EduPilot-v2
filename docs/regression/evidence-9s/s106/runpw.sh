#!/bin/bash
# runpw.sh <spec> <grep> — Playwright trên ep-qc, khoá /tmp/qc9s-pw.lock
until mkdir /tmp/qc9s-pw.lock 2>/dev/null; do sleep 5; done
cd ~/Documents/TA_Agent_qc5/frontend
cp ~/Documents/TA_Agent_qc9s/docs/regression/evidence-9s/s106/specs/$1 e2e/
export QC_REAL=1 QC_BASE=https://localhost:943 QC_OUT=/tmp/qc9s QC_PG=ep-qc-postgres-1 QC_MAILPIT=http://localhost:8525 QC_RES=$HOME/Documents/TA_Agent_qc9s/docs/regression/evidence-9s/s106/results_106.jsonl QC_PW="$(grep '^SEED_DEFAULT_PASSWORD=' ~/Documents/TA_Agent_qc5/.env.local | cut -d= -f2-)"
export QC_STALE='{"count":4,"editing":0,"reviewed":8,"versions":[0]}' QC_AID=$(python3 -c "import json;print(json.load(open('/tmp/gd_ui_a.json'))['a'])") QC_AID2=$(python3 -c "import json;print(json.load(open('/tmp/gd_ui_a2.json'))['a'])" 2>/dev/null)
timeout ${3:-280} npx playwright test -c playwright.qc-real-flow.config.ts e2e/$1 -g "$2" 2>&1 | grep -E "VERDICT|✓|✘|rror|opts|Error|ROWS|MAIN" | cut -c1-900
rmdir /tmp/qc9s-pw.lock
