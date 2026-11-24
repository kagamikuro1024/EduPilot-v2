#!/bin/bash
# @real specs trên ep-r: $1 = qc-real-audit | qc-real-insights ; workers=2
cd "$(dirname "$0")/../../../../frontend"
mkdir /tmp/qc2-pw.lock 2>/dev/null || { echo "pw bận"; exit 9; }
trap 'rmdir /tmp/qc2-pw.lock' EXIT
export QC_REAL=1 QC_BASE=https://localhost:1143 QC_PW='Edupilot#Seed-2026' QC_PG=ep-r-postgres-1 QC_REDIS=ep-r-redis-1 QC_MAILPIT=http://localhost:8725 QC_OUT=/tmp/qc2-real-shots NODE_TLS_REJECT_UNAUTHORIZED=0
node_modules/.bin/playwright test -c playwright.qc-real-flow.config.ts --workers=2 "$1"
