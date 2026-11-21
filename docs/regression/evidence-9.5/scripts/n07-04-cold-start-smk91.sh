#!/bin/bash
# TC-N07-04 (SmkS4, 2026-11-21): stack smk91 (13 container, profile ingest): `compose down` (giữ image + volume) → `up -d` → thời gian tới GET /api/v1/readyz = 200; 3 lần lạnh, trung vị. Cần /tmp/smk91.override.yml (cp từ docs/regression/env/smk91.override.yml).
source ~/Documents/TA_Agent_qc2/docs/regression/env/smk91-env.sh; cp $REPO/docs/regression/env/smk91.override.yml /tmp/smk91.override.yml
cd $REPO
echo "máy: $(colima ssh -- free -m | sed -n 2p | tr -s ' ') | image gateway: $(docker images --format '{{.Repository}}:{{.Tag}}' | grep -c edupilot-gateway-test)"
for i in 1 2 3; do
  $SMK_C down >/dev/null 2>&1; sleep 3
  t0=$(date +%s.%N); $SMK_C up -d --no-build >/dev/null 2>&1
  for k in $(seq 1 600); do c=$(curl -sk -o /tmp/smk-readyz.json -w '%{http_code}' --max-time 2 https://localhost:1343/api/v1/readyz); [ "$c" = 200 ] && break; sleep 0.25; done
  t1=$(date +%s.%N); echo "lần $i: $(printf '%.1f' $(echo "$t1-$t0" | bc)) s tới readyz=$c $(cat /tmp/smk-readyz.json) | container Up: $(docker ps --format '{{.Names}}' | grep -c '^smk91')"
  sleep 20
done
