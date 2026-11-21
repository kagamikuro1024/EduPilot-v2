#!/bin/bash
# usage: pwrun.sh <file-filter> [-g grep]  — giữ khoá Playwright chung /tmp/qc3-pw-lock (mỗi lúc một lượt); ≥2500 MB RAM rảnh
for i in $(seq 1 120); do mkdir /tmp/qc3-pw-lock 2>/dev/null && break; sleep 10; done
[ -d /tmp/qc3-pw-lock ] || exit 9
echo "$(date +%H:%M:%S) qc3S7e: PW LOCK $*" >> /tmp/qc2-global-changes.log
D=$HOME/Documents/TA_Agent_qc3/docs/regression/evidence-7/scripts-r7
cd ~/Documents/TA_Agent_qc2/frontend
NODE_PATH=$PWD/node_modules SPECDIR=$D/specs NODE_TLS_REJECT_UNAUTHORIZED=0 timeout 1200 node_modules/.bin/playwright test -c $D/pw.config.ts "$@" 2>&1
rc=$?; rmdir /tmp/qc3-pw-lock; echo "$(date +%H:%M:%S) qc3S7e: PW UNLOCK rc=$rc" >> /tmp/qc2-global-changes.log; exit $rc
