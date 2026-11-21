#!/bin/bash
# Sau run-s10x.sh: chạy lại D01-07 a, b (đã sửa), rồi audit 10.7 (run-r107.sh tự lấy khoá từng pha).
cd "$(dirname "$0")"; export SEED_DEFAULT_PASSWORD='Edupilot#Seed-2026'
while ! grep -q ALLDONE run-s10x.log; do sleep 15; done
rl(){ docker exec ep-r-redis-1 sh -c "(redis-cli --scan --pattern 'ep:rl:auth*'; redis-cli --scan --pattern 'ep:rl:ip:*') | xargs -r redis-cli del" >/dev/null; }
for s in "d0107.py a" "d0107.py b"; do rl; echo "== $s $(date +%T)" >> run-s10x2.log; /tmp/qc2-pw.sh timeout 470 python3 $s >> run-s10x2.log 2>&1; echo "rc=$?" >> run-s10x2.log; done
R=~/Documents/TA_Agent_qc2/docs/regression; rm -rf $R/evidence-10.7/audit/rr-defrerun; rm -f $R/evidence-defrerun/R10.7/*.log; rl
cd $R/evidence-defrerun && bash run-r107.sh
echo ALLDONE >> ~/Documents/TA_Agent_qc2/docs/regression/evidence-10.5/s10x/run-s10x2.log
