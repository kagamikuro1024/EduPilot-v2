#!/bin/bash
# qc2 25/11: chạy MỘT spec stack-only trên ep-r (hoặc qc2seedt2 nếu đặt T2=1); $@ = đối số playwright
cd "$(dirname "$0")/../../../frontend"
mkdir /tmp/qc2-pw.lock 2>/dev/null || { echo "pw bận"; exit 9; }
trap 'rmdir /tmp/qc2-pw.lock' EXIT
if [ -n "$T2" ]; then export QC_BASE=https://localhost:4643 QC_PG=qc2seedt2-postgres-1 QC_REDIS=qc2seedt2-redis-1 QC_MAILPIT=http://localhost:12225
else export QC_BASE=https://localhost:1143 QC_PG=ep-r-postgres-1 QC_REDIS=ep-r-redis-1 QC_MAILPIT=http://localhost:8725; fi
export QC_REAL=1 QC_PW='Edupilot#Seed-2026' QC_OUT=/tmp/qc2-real-shots NODE_TLS_REJECT_UNAUTHORIZED=0
echo "load: $(uptime)"
node_modules/.bin/playwright test -c playwright.qc-real-flow.config.ts --workers=1 "$@" 2>&1 | grep -vE "^\(node|^\(Use"
