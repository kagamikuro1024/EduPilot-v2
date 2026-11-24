#!/bin/bash
# Dựng lại gateway rồi worker ep-r (từng service, giữ volume, không migrate). nohup bash docs/regression/env/rebuild-gw-worker.sh > /tmp/qc2-rb2.out 2>&1 &
set -u
R=$HOME/Documents/TA_Agent_qc2; cd $R
unset $(env | grep -oE '^(EP_PORT_[A-Z_]+|COMPOSE_[A-Z_]+|BLOB_[A-Z_]+)')
source $R/docs/regression/env/ep-r-env.sh
LOCK=/tmp/qc2-lock-global; LOG=/tmp/qc2-global-changes.log
for i in $(seq 1 80); do mkdir $LOCK 2>/dev/null && break; sleep 15; done
echo qc2 > $LOCK/owner
echo "$(date +%T) qc2: LOCK rebuild ep-r gateway+worker @$(git rev-parse --short HEAD) (+AI_SUMMARY_TIMEOUT=60s); ≤15 phút" >> $LOG
for sv in gateway worker; do
  fr=$(colima ssh -- free -m | awk '/^Mem/{print $7}'); echo "build $sv free=$fr"; [ "${fr:-0}" -ge 2500 ] || { echo 'RAM < 2500, dừng'; break; }
  $EPR_C build $sv 2>&1 | tail -3; $EPR_C up -d --no-deps $sv 2>&1 | tail -3
  for i in $(seq 1 60); do [ "$(docker inspect -f '{{.State.Health.Status}}' ep-r-$sv-1 2>/dev/null)" = healthy ] && break; sleep 5; done
  echo "$sv: $(docker inspect -f '{{.State.Health.Status}}' ep-r-$sv-1)"
done
echo "$(date +%T) qc2: UNLOCK rebuild ep-r xong readyz=$(curl -ks -o /dev/null -w '%{http_code}' https://localhost:1143/api/v1/readyz)" >> $LOG
rm -f $LOCK/owner; rmdir $LOCK
echo DONE
