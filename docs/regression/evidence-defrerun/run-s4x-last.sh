#!/bin/bash
# S4X: lượt cuối trong MỘT lần giữ khoá (≤ 8 phút): s27 (R5-06), staff (R5-09), stale UI (R5-10); xoá giới hạn đăng nhập NGAY TRONG khoá.
export NODE_TLS_REJECT_UNAUTHORIZED=0 QC_REAL=1 DOCKER_HOST=unix://$HOME/.colima/default/docker.sock
R=~/Documents/TA_Agent_qc2/docs/regression; D=$R/evidence-defrerun/R5; S=$R/evidence-5/scripts; cd $R/evidence-5/ui
read -r _ E A < <(grep '^EXAMID' $D/s4x-p0333.log | tail -1)
cat > /tmp/s4x-inner.sh <<EOF
clrl(){ docker exec ep-r-redis-1 sh -c "(redis-cli --scan --pattern 'ep:rl:auth*'; redis-cli --scan --pattern 'ep:rl:ip:*') | xargs -r redis-cli del" > /dev/null; }
cd $R/evidence-5/ui
python3 $S/s20d_fixture.py > $D/s4x-fixture-last.log 2>&1
clrl; ONLY=s27 U=r5s31 timeout 150 node smkP_m.cjs > $D/s4x-s27-last.log 2>&1; echo rc=\$? >> $D/s4x-s27-last.log
clrl; ONLY=staff timeout 150 node smkP_m2.cjs > $D/s4x-staff-last.log 2>&1; echo rc=\$? >> $D/s4x-staff-last.log
clrl; ONLY=stale E=$E A=$A timeout 100 node smkP_m.cjs > $D/s4x-stale-last.log 2>&1; echo rc=\$? >> $D/s4x-stale-last.log
EOF
/tmp/qc2-pw.sh bash /tmp/s4x-inner.sh
echo LAST-DONE >> $D/s4x-all.log
