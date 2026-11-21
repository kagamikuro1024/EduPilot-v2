#!/bin/bash
export NODE_TLS_REJECT_UNAUTHORIZED=0 QC_REAL=1 DOCKER_HOST=unix://$HOME/.colima/default/docker.sock
R=~/Documents/TA_Agent_qc2/docs/regression; D=$R/evidence-defrerun/R5
XM=$(python3 -c "import json;print(json.load(open('/tmp/qc2L/uix2.json'))['xmcq'])")
cat > /tmp/s4x-inner2.sh <<EOF
clrl(){ docker exec ep-r-redis-1 sh -c "(redis-cli --scan --pattern 'ep:rl:auth*'; redis-cli --scan --pattern 'ep:rl:ip:*') | xargs -r redis-cli del" > /dev/null; }
cd $R/evidence-5/ui
clrl; node s4x_dbg.cjs > $D/s4x-dbg2.log 2>&1
clrl; ONLY=staff EMPTY=$XM timeout 150 node smkP_m2.cjs > $D/s4x-staff-fin.log 2>&1; echo rc=\$? >> $D/s4x-staff-fin.log
EOF
/tmp/qc2-pw.sh bash /tmp/s4x-inner2.sh
echo FIN-DONE >> $D/s4x-all.log
