#!/bin/bash
# @real specs trên qc2seedt2 (cổng 4643): các đối số sau là bộ lọc/thông số playwright
cd "$(dirname "$0")/../../../../frontend"
mkdir /tmp/qc2-pw.lock 2>/dev/null || { echo "pw bận"; exit 9; }
trap 'rmdir /tmp/qc2-pw.lock' EXIT
export QC_REAL=1 QC_BASE=https://localhost:4643 QC_PW='Edupilot#Seed-2026' QC_PG=qc2seedt2-postgres-1 QC_REDIS=qc2seedt2-redis-1 QC_MAILPIT=http://localhost:12225 QC_OUT=/tmp/qc2-real-shots NODE_TLS_REJECT_UNAUTHORIZED=0
node_modules/.bin/playwright test -c playwright.qc-real-flow.config.ts --workers=2 "$@"
