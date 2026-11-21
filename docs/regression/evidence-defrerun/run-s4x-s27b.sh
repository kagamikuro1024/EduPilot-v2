#!/bin/bash
export NODE_TLS_REJECT_UNAUTHORIZED=0 QC_REAL=1 DOCKER_HOST=unix://$HOME/.colima/default/docker.sock
R=~/Documents/TA_Agent_qc2/docs/regression; D=$R/evidence-defrerun/R5
cat > /tmp/s4x-inner3.sh <<EOF
docker exec ep-r-redis-1 sh -c "(redis-cli --scan --pattern 'ep:rl:auth*'; redis-cli --scan --pattern 'ep:rl:ip:*') | xargs -r redis-cli del" > /dev/null
cd $R/evidence-5/ui
ONLY=s27 U=r5s32 timeout 200 node smkP_m.cjs > $D/s4x-s27-fin.log 2>&1; echo rc=\$? >> $D/s4x-s27-fin.log
EOF
/tmp/qc2-pw.sh bash /tmp/s4x-inner3.sh
echo S27-DONE >> $D/s4x-all.log
