#!/bin/bash
# D04-01 làm lại: env LLM_DECISION_FAKE_MODE cho worker (always_yes / always_no / error), mỗi lượt qua khoá.
cd "$(dirname "$0")"; export SEED_DEFAULT_PASSWORD='Edupilot#Seed-2026'
for s in "d0409.py always_yes" "d0409.py always_no" "d0409.py error" "d0409.py fin"; do
  docker exec ep-r-redis-1 sh -c "(redis-cli --scan --pattern 'ep:rl:auth*'; redis-cli --scan --pattern 'ep:rl:ip:*') | xargs -r redis-cli del" >/dev/null
  echo "== $s $(date +%T)" >> run-s10x3.log; /tmp/qc2-pw.sh timeout 470 python3 $s >> run-s10x3.log 2>&1; echo "rc=$?" >> run-s10x3.log
done; echo ALLDONE >> run-s10x3.log
