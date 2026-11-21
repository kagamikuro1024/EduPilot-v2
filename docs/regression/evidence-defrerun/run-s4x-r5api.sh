#!/bin/bash
# S4X: DEF-R5-10 (TC-PE03-33 API) + DEF-R5-11 (TC-PE08-04) trên ep-r, qua khoá chung. Log R5/s4x-*.log
export NODE_TLS_REJECT_UNAUTHORIZED=0 QC_REAL=1 DOCKER_HOST=unix://$HOME/.colima/default/docker.sock
R=~/Documents/TA_Agent_qc2/docs/regression; D=$R/evidence-defrerun/R5; S=$R/evidence-5/scripts
W=env
step(){ n=$1; shift; echo "== $n $(date +%T)" >> $D/s4x-all.log; ( "$@" ) > $D/s4x-$n.log 2>&1; echo "rc=$?" >> $D/s4x-$n.log; cat $D/s4x-$n.log >> $D/s4x-all.log; }
clrl(){ docker exec ep-r-redis-1 sh -c "(redis-cli --scan --pattern 'ep:rl:auth*'; redis-cli --scan --pattern 'ep:rl:ip:*') | xargs -r redis-cli del" > /dev/null; }
mkdir -p /tmp/qc2L /tmp/qc2-c5b
cd $S
clrl; step p0333 $W timeout 480 python3 smkP_s5api.py p0333
clrl; step p0804 $W timeout 480 python3 smkP_s5api.py p0804
echo API-ALLDONE >> $D/s4x-all.log
