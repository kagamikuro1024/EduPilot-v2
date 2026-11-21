#!/bin/bash
# chạy tuần tự các kịch bản exam-submit.js trên qc3s (cổng 18241); log từng lần ở pe/out
cd ~/Documents/TA_Agent_qc3/docs/regression/evidence-6/k6/pe
for spec in "$@"; do
  sc=${spec%%:*}; run=${spec#*:}; [ "$run" = "$spec" ] && run=""
  tag=$sc${run:+-$run}; echo "$(date +%H:%M:%S) bắt đầu $tag uptime: $(uptime | sed 's/.*load/load/')" >> out/run.log
  k6 run exam-submit.js -e BASE=http://127.0.0.1:18241 -e SCENARIO=$sc ${run:+-e RUN=$run} > out/$tag.txt 2>&1; echo "$(date +%H:%M:%S) xong $tag rc=$?" >> out/run.log
  sleep 20
done
