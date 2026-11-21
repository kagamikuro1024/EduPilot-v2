#!/bin/bash
export NODE_TLS_REJECT_UNAUTHORIZED=0 QC_REAL=1
R=~/Documents/TA_Agent_qc2/docs/regression; D=$R/evidence-defrerun/R5; cd $R/evidence-5/ui
for u in r5s31 r5s32; do
  docker exec ep-r-redis-1 sh -c "(redis-cli --scan --pattern 'ep:rl:auth*'; redis-cli --scan --pattern 'ep:rl:ip:*') | xargs -r redis-cli del" > /dev/null
  env ONLY=s27 U=$u /tmp/qc2-pw.sh timeout 480 node smkP_m.cjs > $D/s4x-s27-$u.log 2>&1; echo "rc=$?" >> $D/s4x-s27-$u.log
  grep -q '^P TC-PE06-27' $D/s4x-s27-$u.log && break
done
