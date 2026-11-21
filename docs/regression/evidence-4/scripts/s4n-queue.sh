#!/bin/bash
# hàng đợi lượt Playwright của SmkT4N (nghỉ ≥ 30 s giữa hai lượt)
cd ~/Documents/TA_Agent_qc2/docs/regression/evidence-4/scripts
for job in "s4b desktop ui-s4b" "s4c desktop ui-s4c" "s4d desktop ui-s4d"; do bash s4n-pw.sh $job; sleep 35; done
echo queue-done > ../logs/s4n-queue.done
