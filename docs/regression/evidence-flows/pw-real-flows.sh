#!/bin/bash
# qc2 25/11: spec stack-only (KHÔNG gọi LLM) của F1 / F2 / F15 trên ep-r, tuần tự, workers=1 (ca dò mật khẩu chạm giới hạn đăng nhập sai theo IP)
cd "$(dirname "$0")/../../../frontend"
mkdir /tmp/qc2-pw.lock 2>/dev/null || { echo "pw bận"; exit 9; }
trap 'rmdir /tmp/qc2-pw.lock' EXIT
export QC_REAL=1 QC_BASE=https://localhost:1143 QC_PW='Edupilot#Seed-2026' QC_PG=ep-r-postgres-1 QC_REDIS=ep-r-redis-1 QC_MAILPIT=http://localhost:8725 QC_OUT=/tmp/qc2-real-shots NODE_TLS_REJECT_UNAUTHORIZED=0
for s in qc-real-f1-account qc-real-f2-class qc-real-f15-llm; do echo "##### $s"; node_modules/.bin/playwright test -c playwright.qc-real-flow.config.ts --workers=1 "$s" 2>&1 | tail -40; done
