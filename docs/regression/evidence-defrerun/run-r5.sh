#!/bin/bash
# Chạy lại DEF-R5-* trên ep-r (HEAD qc2). Mỗi bước qua khoá Playwright.
export NODE_TLS_REJECT_UNAUTHORIZED=0 QC_REAL=1
R=~/Documents/TA_Agent_qc2/docs/regression; D=$R/evidence-defrerun/R5; U=$R/evidence-5/ui; S=$R/evidence-5/scripts
W=/tmp/qc2-pw.sh
step(){ n=$1; shift; echo "== $n $(date +%T)" >> $D/all.log; ( "$@" ) > $D/$n.log 2>&1; echo "rc=$?" >> $D/$n.log; cat $D/$n.log >> $D/all.log; }
cd $U
sleep 30
step s27 env ONLY=s27 $W timeout 480 node smkP_m.cjs
sleep 30
step s18 env ONLY=s18 $W timeout 480 node smkP_m.cjs
sleep 30
step stale env ONLY=stale $W timeout 480 node smkP_m.cjs
sleep 30
step res env ONLY=res $W timeout 480 node smkP_m2.cjs
sleep 30
step staff env ONLY=staff $W timeout 480 node smkP_m2.cjs
cd $S
step p0333 $W timeout 480 python3 smkP_s5api.py p0333
sleep 20
step p0804 $W timeout 480 python3 smkP_s5api.py p0804
echo ALLDONE >> $D/all.log
