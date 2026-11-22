#!/bin/bash
# Dựng lại ep-r từ worktree qc2 (giữ volume). Dùng: nohup bash docs/regression/env/rebuild.sh > /tmp/qc2-rebuild.out 2>&1 &
set -u
R=$HOME/Documents/TA_Agent_qc2; cd $R
source $R/docs/regression/env/ep-r-env.sh
LOCK=/tmp/qc2-lock-global; LOG=/tmp/qc2-global-changes.log
for i in $(seq 1 80); do mkdir $LOCK 2>/dev/null && break; sleep 15; done
echo qc2 > $LOCK/owner
echo "$(date +%T) qc2: LOCK rebuild ep-r gateway/worker/frontend/migrate trên HEAD $(git rev-parse --short HEAD); ≤15 phút" >> $LOG
for sv in gateway worker frontend migrate; do fr=$(colima ssh -- free -m | awk '/^Mem/{print $7}'); echo "build $sv free=$fr"; [ "${fr:-0}" -ge 2500 ] || { echo 'RAM < 2500, dừng'; exit 3; }; $EPR_C build $sv 2>&1 | tail -3; done
$EPR_C up -d --force-recreate migrate 2>&1 | tail -3
$EPR_C up -d 2>&1 | tail -15
for i in $(seq 1 60); do [ "$(curl -ks -o /dev/null -w '%{http_code}' https://localhost:1143/api/v1/readyz)" = 200 ] && break; sleep 5; done
echo "$(date +%T) qc2: UNLOCK rebuild xong readyz=$(curl -ks -o /dev/null -w '%{http_code}' https://localhost:1143/api/v1/readyz)" >> $LOG
rm -f $LOCK/owner; rmdir $LOCK
echo DONE
