#!/bin/bash
# s4n-pw.sh <tên-lượt> <project> <spec...> : một lượt Playwright ≤ 8 phút qua khoá chung /tmp/qc2-pw.lock (hỏi khoá 2 s/lần), log evidence-4/logs/s4n-pw-<tên>.log
name=$1; proj=$2; shift 2
cd ~/Documents/TA_Agent_qc2/frontend
for i in $(seq 1 600); do mkdir /tmp/qc2-pw.lock 2>/dev/null && break; sleep 2; done
echo "$(date +%T) SmkT4N: PW LOCK $name" >> /tmp/qc2-global-changes.log
trap 'rmdir /tmp/qc2-pw.lock 2>/dev/null; echo "$(date +%T) SmkT4N: PW UNLOCK $name" >> /tmp/qc2-global-changes.log' EXIT
QC2G_DIR=x QC_REAL=1 NODE_TLS_REJECT_UNAUTHORIZED=0 timeout 480 npx playwright test -c ../docs/regression/specs-qc2/qc2-r4.config.ts "$@" --project=$proj --output=../docs/regression/evidence-4/pw-out > ../docs/regression/evidence-4/logs/s4n-pw-$name.log 2>&1
echo "done rc=$?" >> ../docs/regression/evidence-4/logs/s4n-pw-$name.log
