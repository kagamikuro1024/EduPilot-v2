#!/bin/bash
# $1 = prebuild (không seed) | measure (SEED_ON_EMPTY_DB=true, mốc giờ từng dòng)
cd /Users/kuro/checkouts/qc2fresh
L=$HOME/Documents/TA_Agent_qc2/docs/regression/evidence-2/fresh
BASE=(env -i HOME="$HOME" PATH="$PATH" DOCKER_HOST=unix://$HOME/.colima/default/docker.sock COMPOSE_PROJECT_NAME=qc2seedt EP_PORT_OFFSET=4100 COMPOSE_PROFILES=ingest)
if [ "$1" = prebuild ]; then
  "${BASE[@]}" node scripts/dev.mjs > $L/seedt-prebuild.log 2>&1; echo "prebuild exit=$?" >> $L/seedt-prebuild.log
else
  docker compose ls -a --format json | python3 -c "import sys,json; print([(p['Name'],p['Status']) for p in json.load(sys.stdin)])" > $L/compose-ls-before-measure.txt
  echo "load trước đo: $(uptime | sed 's/.*load/load/')  free: $(colima ssh -- free -m | sed -n 2p)" > $L/timing-pre.txt; docker ps --format '{{.Names}}' >> $L/timing-pre.txt
  date +%s > $L/timing-start.txt
  "${BASE[@]}" SEED_ON_EMPTY_DB=true node scripts/dev.mjs 2>&1 | while IFS= read -r l; do echo "$(date +%H:%M:%S) $l"; done > $L/timing-dev.log
  echo "exit=${PIPESTATUS[0]}" >> $L/timing-dev.log
  date +%s > $L/timing-end.txt
  echo "load sau đo: $(uptime | sed 's/.*load/load/')" >> $L/timing-pre.txt
  docker compose ls -a --format json | python3 -c "import sys,json; print([(p['Name'],p['Status']) for p in json.load(sys.stdin)])" > $L/compose-ls-after-measure.txt
  echo DONE >> $L/timing-dev.log
fi
