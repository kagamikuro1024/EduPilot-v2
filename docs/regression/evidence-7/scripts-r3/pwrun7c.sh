#!/bin/bash
# usage: pwrun7c.sh <spec-file-filter> [grep]   (holds /tmp/qc3-pw-lock)
spec=$1; g=$2
for i in $(seq 1 240); do mkdir /tmp/qc3-pw-lock 2>/dev/null && break; sleep 15; done
[ -d /tmp/qc3-pw-lock ] || exit 9
echo "$(date +%H:%M:%S) qc3S7c: PW LOCK $spec $g" >> /tmp/qc3-pw-lock.log
cd ~/Documents/TA_Agent_qc2/frontend
if [ -n "$g" ]; then NODE_PATH=$PWD/node_modules timeout 1500 node_modules/.bin/playwright test -c /tmp/qc3e/pw.config.ts "$spec" -g "$g" 2>&1; else NODE_PATH=$PWD/node_modules timeout 1500 node_modules/.bin/playwright test -c /tmp/qc3e/pw.config.ts "$spec" 2>&1; fi
rc=$?
rmdir /tmp/qc3-pw-lock; echo "$(date +%H:%M:%S) qc3S7c: PW UNLOCK rc=$rc" >> /tmp/qc3-pw-lock.log
exit $rc
