#!/bin/bash
# chạy tuần tự: mỗi tham số = tệp.js:SCENARIO:RUN ; lấy mẫu docker stats + load mỗi 10 s ra out/stats.log
cd ~/Documents/TA_Agent_qc3/docs/regression/evidence-6/k6/pe
( while true; do echo "$(date +%H:%M:%S) $(uptime | sed 's/.*load averages*: //') | $(docker stats --no-stream --format '{{.Name}} {{.CPUPerc}}' qc3s-gw qc3s-worker ep-r-postgres-1 2>/dev/null | tr '\n' ' ')" >> out/stats.log; sleep 10; done ) &
SP=$!
for spec in "$@"; do
  IFS=: read -r js sc run <<< "$spec"; tag=${js%.js}-$sc${run:+-$run}
  echo "$(date +%H:%M:%S) bắt đầu $tag" >> out/run.log
  k6 run $js -e BASE=http://127.0.0.1:18241 -e SCENARIO=$sc ${run:+-e RUN=$run} > out/$tag.txt 2>&1; rc=$?; echo "$(date +%H:%M:%S) xong $tag rc=$rc" >> out/run.log
  sleep 20
done
kill $SP
