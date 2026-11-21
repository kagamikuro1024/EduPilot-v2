#!/bin/bash
# usage: pwrun.sh <file-filter> [grep]  — giữ khoá Playwright chung /tmp/qc3-pw-lock (mỗi lúc một lượt)
for i in $(seq 1 120); do mkdir /tmp/qc3-pw-lock 2>/dev/null && break; sleep 10; done
[ -d /tmp/qc3-pw-lock ] || exit 9
cd ~/Documents/TA_Agent_qc2/frontend
NODE_PATH=$PWD/node_modules SPECDIR=$HOME/Documents/TA_Agent_qc3/docs/regression/evidence-7/scripts-r3/specs-d timeout 900 node_modules/.bin/playwright test -c $HOME/Documents/TA_Agent_qc3/docs/regression/evidence-7/scripts-r3/specs-d/pw.config.ts "$@" 2>&1
rc=$?; rmdir /tmp/qc3-pw-lock; exit $rc
