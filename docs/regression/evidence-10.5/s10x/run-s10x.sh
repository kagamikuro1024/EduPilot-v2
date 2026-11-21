#!/bin/bash
# S10X: D01-07 (a, c, b) rồi D04-09/D04-01 (p1, 3 cấu hình, fin); mỗi lượt qua khoá /tmp/qc2-pw.sh (≤ 8 phút).
cd "$(dirname "$0")"; export SEED_DEFAULT_PASSWORD='Edupilot#Seed-2026'
rl(){ docker exec ep-r-redis-1 sh -c "(redis-cli --scan --pattern 'ep:rl:auth*'; redis-cli --scan --pattern 'ep:rl:ip:*') | xargs -r redis-cli del" >/dev/null; }
for s in "d0107.py a" "d0107.py c" "d0107.py b" "d0409.py p1" "d0409.py always_yes" "d0409.py always_no" "d0409.py error" "d0409.py fin"; do
  rl; echo "== $s $(date +%T)" >> run-s10x.log; /tmp/qc2-pw.sh timeout 470 python3 $s >> run-s10x.log 2>&1; echo "rc=$?" >> run-s10x.log
done
echo ALLDONE >> run-s10x.log
