#!/bin/bash
# S4X: toàn bộ lượt còn lại, tuần tự (mỗi lượt qua khoá /tmp/qc2-pw.sh). Chạy SAU «ep-r lên lại»: nohup bash run-s4x-post.sh > R5/s4x-post.out 2>&1 &
export NODE_TLS_REJECT_UNAUTHORIZED=0 QC_REAL=1 DOCKER_HOST=unix://$HOME/.colima/default/docker.sock
R=~/Documents/TA_Agent_qc2/docs/regression; D=$R/evidence-defrerun/R5; U=$R/evidence-5/ui; S=$R/evidence-5/scripts; E4=$R/evidence-4
W=/tmp/qc2-pw.sh
mkdir -p /tmp/qc2L /tmp/qc2-c5 /tmp/qc2-c5b $R/evidence-defrerun/R3-01
clrl(){ docker exec ep-r-redis-1 sh -c "(redis-cli --scan --pattern 'ep:rl:auth*'; redis-cli --scan --pattern 'ep:rl:ip:*') | xargs -r redis-cli del" > /dev/null; }
step(){ n=$1; shift; echo "== $n $(date +%T)" >> $D/s4x-all.log; ( "$@" ) > $D/s4x-$n.log 2>&1; echo "rc=$?" >> $D/s4x-$n.log; cat $D/s4x-$n.log >> $D/s4x-all.log; }
cd $E4
bash scripts/s4n-pw.sh s4x2b desktop smkt3-p20231.spec.ts
bash scripts/s4n-pw.sh s4x5 desktop r4ui-audit.spec.ts -g "· (student|ta)$"
bash scripts/s4n-pw.sh s4x6 desktop r4ui-audit.spec.ts -g "· (teacher|admin)$"
cd $R/evidence-3/ui; clrl; step r301 $W timeout 400 node smkP_p10544.cjs
cd $U
for p in "s27 smkP_m.cjs" "s18 smkP_m.cjs" "res smkP_m2.cjs" "staff smkP_m2.cjs"; do
  set -- $p
  if [ "$1" = s27 ] || [ "$1" = s18 ]; then python3 $S/s20d_fixture.py > $D/s4x-fixture-$1.log 2>&1; fi
  clrl; step $1 env ONLY=$1 $W timeout 480 node $2
done
read -r _ E A < <(grep '^EXAMID' $D/s4x-p0333.log | tail -1)
python3 $S/s20d_fixture.py > $D/s4x-fixture-stale.log 2>&1; clrl; step stale-ui env ONLY=stale E=$E A=$A $W timeout 300 node smkP_m.cjs
bash $R/evidence-defrerun/R9.5-17/run.sh > $R/evidence-defrerun/R9.5-17/run.out 2>&1
echo POST-ALLDONE >> $D/s4x-all.log
