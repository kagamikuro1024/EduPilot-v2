#!/bin/bash
# S4X: DEF-R5-10 phần giao diện (TC-PE03-33): Drawer GV của bài đã công bố sau khi sửa test ẩn; E/A lấy từ p0333 (R5/s4x-p0333.log dòng EXAMID)
export NODE_TLS_REJECT_UNAUTHORIZED=0 QC_REAL=1
R=~/Documents/TA_Agent_qc2/docs/regression; D=$R/evidence-defrerun/R5
read -r _ E A < <(grep '^EXAMID' $D/s4x-p0333.log | tail -1)
docker exec ep-r-redis-1 sh -c "(redis-cli --scan --pattern 'ep:rl:auth*'; redis-cli --scan --pattern 'ep:rl:ip:*') | xargs -r redis-cli del" > /dev/null
cd $R/evidence-5/ui; python3 $R/evidence-5/scripts/s20d_fixture.py > $D/s4x-fixture-stale.log 2>&1
env ONLY=stale E=$E A=$A /tmp/qc2-pw.sh timeout 300 node smkP_m.cjs > $D/s4x-stale-ui.log 2>&1; echo "rc=$?" >> $D/s4x-stale-ui.log
