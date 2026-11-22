#!/bin/bash
# Đo giờ seed theo P-11-76: project qc2fresh (volume trống, docling riêng), MỘT lần pnpm dev tự seed; mốc giờ từng dòng
cd /Users/kuro/checkouts/qc2fresh
L=$HOME/Documents/TA_Agent_qc2/docs/regression/evidence-2/fresh
D() { env -i HOME="$HOME" PATH="$PATH" DOCKER_HOST=unix://$HOME/.colima/default/docker.sock COMPOSE_PROJECT_NAME=qc2fresh EP_PORT_OFFSET=3000 COMPOSE_PROFILES=ingest "$@"; }
D node scripts/dev.mjs down -v > $L/timing-down.log 2>&1
echo "load trước đo: $(uptime | sed 's/.*load/load/')  free: $(colima ssh -- free -m | sed -n 2p)  container đang chạy: $(docker ps -q | wc -l)" > $L/timing-pre.txt
docker ps --format '{{.Names}}' >> $L/timing-pre.txt
date +%s > $L/timing-start.txt
env -i HOME="$HOME" PATH="$PATH" DOCKER_HOST=unix://$HOME/.colima/default/docker.sock COMPOSE_PROJECT_NAME=qc2fresh EP_PORT_OFFSET=3000 COMPOSE_PROFILES=ingest SEED_ON_EMPTY_DB=true node scripts/dev.mjs 2>&1 | while IFS= read -r l; do echo "$(date +%H:%M:%S) $l"; done > $L/timing-dev.log
echo "exit=${PIPESTATUS[0]}" >> $L/timing-dev.log
date +%s > $L/timing-end.txt
echo "load sau đo: $(uptime | sed 's/.*load/load/')" >> $L/timing-pre.txt
echo DONE >> $L/timing-dev.log
