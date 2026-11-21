#!/bin/bash
export NODE_TLS_REJECT_UNAUTHORIZED=0 QC_REAL=1
R=~/Documents/TA_Agent_qc2/docs/regression; D=$R/evidence-defrerun/R5; cd $R/evidence-5/ui
docker exec ep-r-redis-1 sh -c "(redis-cli --scan --pattern 'ep:rl:auth*'; redis-cli --scan --pattern 'ep:rl:ip:*') | xargs -r redis-cli del" > /dev/null
env ONLY=staff /tmp/qc2-pw.sh timeout 480 node smkP_m2.cjs > $D/s4x-staff2.log 2>&1; echo "rc=$?" >> $D/s4x-staff2.log
