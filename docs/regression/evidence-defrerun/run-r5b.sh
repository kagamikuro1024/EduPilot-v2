#!/bin/bash
# S4R: phần UI của DEF-R5-06/07/08/09/10 (ep-r), mỗi bước qua khoá Playwright
export NODE_TLS_REJECT_UNAUTHORIZED=0 QC_REAL=1
R=~/Documents/TA_Agent_qc2/docs/regression; D=$R/evidence-defrerun/R5; U=$R/evidence-5/ui; S=$R/evidence-5/scripts
W=/tmp/qc2-pw.sh
step(){ n=$1; shift; echo "== $n $(date +%T)" >> $D/s4r-all.log; ( "$@" ) > $D/s4r-$n.log 2>&1; echo "rc=$?" >> $D/s4r-$n.log; cat $D/s4r-$n.log >> $D/s4r-all.log; }
mkdir -p /tmp/qc2L /tmp/qc2-c5b
cd $U
for p in "s27 smkP_m.cjs" "s18 smkP_m.cjs" "stale smkP_m.cjs" "res smkP_m2.cjs" "staff smkP_m2.cjs"; do
  set -- $p
  if [ "$1" = s27 ] || [ "$1" = s18 ]; then python3 $S/s20d_fixture.py > $D/s4r-fixture-$1.log 2>&1; fi
  step $1 env ONLY=$1 $W timeout 480 node $2
  sleep 20
done
echo ALLDONE >> $D/s4r-all.log
