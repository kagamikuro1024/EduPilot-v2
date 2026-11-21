#!/bin/bash
export SEED_DEFAULT_PASSWORD="$(grep '^SEED_DEFAULT_PASSWORD=' ~/Documents/TA_Agent_qc5/.env.local | cut -d= -f2-)"
until mkdir /tmp/qc-global.lock 2>/dev/null; do sleep 5; done; echo "QC3 TC-02-18 docling dừng" > /tmp/qc-global.lock/owner
cd ~/Documents/TA_Agent_qc9s/docs/regression/evidence-9s/s10
echo ok > /Users/kuro/.config/edupilot-team/qc3-work/stub10_mode
(python3 stub_s10.py 9761 >/dev/null 2>&1 & echo $! > /Users/kuro/.config/edupilot-team/qc3-work/stub10.pid); sleep 1
docker stop ep-qc-docling-1 >/dev/null
python3 perf/run0218.py 2>&1 | tail -12 | cut -c1-900
docker start ep-qc-docling-1 >/dev/null; kill $(cat /Users/kuro/.config/edupilot-team/qc3-work/stub10.pid) 2>/dev/null
rm -rf /tmp/qc-global.lock
