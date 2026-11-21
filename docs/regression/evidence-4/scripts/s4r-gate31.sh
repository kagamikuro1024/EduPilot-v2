#!/bin/bash
# TC-GATEP2-05: account.spec.ts trên gate build (cổng 33410/33412), qua khoá chung
cd ~/Documents/TA_Agent_qc2/frontend
for i in $(seq 1 1800); do mkdir /tmp/qc2-pw.lock 2>/dev/null && break; sleep 1; done
echo "$(date +%T) S4R: PW LOCK gate31" >> /tmp/qc2-global-changes.log
trap 'rmdir /tmp/qc2-pw.lock 2>/dev/null; echo "$(date +%T) S4R: PW UNLOCK gate31" >> /tmp/qc2-global-changes.log' EXIT
E2E_PORT=33410 E2E_API_PORT=33412 timeout 470 npx playwright test data-layer.spec.ts -g "auto refresh" --workers=1 --output=../docs/regression/evidence-4/pw-out-gate > ../docs/regression/evidence-4/logs/s4r-gate31.log 2>&1
echo "rc=$?" >> ../docs/regression/evidence-4/logs/s4r-gate31.log
