#!/bin/bash
export NODE_TLS_REJECT_UNAUTHORIZED=0 QC_REAL=1 DOCKER_HOST=unix://$HOME/.colima/default/docker.sock
R=~/Documents/TA_Agent_qc2/docs/regression; D=$R/evidence-defrerun/R5
cat > /tmp/s4x-inner4.sh <<EOF
clrl(){ docker exec ep-r-redis-1 sh -c "(redis-cli --scan --pattern 'ep:rl:auth*'; redis-cli --scan --pattern 'ep:rl:ip:*') | xargs -r redis-cli del" > /dev/null 2>&1; }
cd $R/evidence-5/ui
(while sleep 1; do clrl; done) & L=\$!
ONLY=s27 U=r5s34 timeout 240 node smkP_m.cjs > $D/s4x-s27-fin3.log 2>&1; echo rc=\$? >> $D/s4x-s27-fin3.log
kill \$L
EOF
/tmp/qc2-pw.sh bash /tmp/s4x-inner4.sh
echo S27D-DONE >> $D/s4x-all.log
