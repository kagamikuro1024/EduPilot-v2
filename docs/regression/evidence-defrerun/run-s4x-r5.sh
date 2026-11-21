#!/bin/bash
# S4X: chạy lại DEF-R5-06/07/08/09/10 (UI) + R3-01 trên ep-r; mỗi bước qua khoá Playwright /tmp/qc2-pw.sh. Log: evidence-defrerun/R5/s4x-*.log
export NODE_TLS_REJECT_UNAUTHORIZED=0 QC_REAL=1
R=~/Documents/TA_Agent_qc2/docs/regression; D=$R/evidence-defrerun/R5; U=$R/evidence-5/ui; S=$R/evidence-5/scripts
W=/tmp/qc2-pw.sh
step(){ n=$1; shift; echo "== $n $(date +%T)" >> $D/s4x-all.log; ( "$@" ) > $D/s4x-$n.log 2>&1; echo "rc=$?" >> $D/s4x-$n.log; cat $D/s4x-$n.log >> $D/s4x-all.log; }
clrl(){ docker exec ep-r-redis-1 sh -c "(redis-cli --scan --pattern 'ep:rl:auth*'; redis-cli --scan --pattern 'ep:rl:ip:*') | xargs -r redis-cli del" > /dev/null; }
mkdir -p /tmp/qc2L /tmp/qc2-c5b $R/evidence-defrerun/R3-01
cd $R/evidence-3/ui; clrl; step r301 $W timeout 400 node smkP_p10544.cjs
cd $U
for p in "s27 smkP_m.cjs" "s18 smkP_m.cjs" "stale smkP_m.cjs" "res smkP_m2.cjs" "staff smkP_m2.cjs"; do
  set -- $p
  if [ "$1" = s27 ] || [ "$1" = s18 ]; then python3 $S/s20d_fixture.py > $D/s4x-fixture-$1.log 2>&1; fi
  clrl; step $1 env ONLY=$1 $W timeout 480 node $2
done
echo ALLDONE >> $D/s4x-all.log
