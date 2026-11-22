#!/bin/bash
# pnpm dev (scripts/dev.mjs) trên project qc2fresh, volume trống, seed tự chạy (SEED_ON_EMPTY_DB=true); mỗi dòng có mốc giờ
cd /Users/kuro/checkouts/qc2fresh
L=$HOME/Documents/TA_Agent_qc2/docs/regression/evidence-2/fresh
date +%s > $L/dev-start.txt
env -i HOME="$HOME" PATH="$PATH" DOCKER_HOST=unix://$HOME/.colima/default/docker.sock COMPOSE_PROJECT_NAME=qc2fresh EP_PORT_OFFSET=3000 COMPOSE_PROFILES=ingest SEED_ON_EMPTY_DB=true node scripts/dev.mjs 2>&1 | while IFS= read -r l; do echo "$(date +%H:%M:%S) $l"; done > $L/pnpm-dev-3.log
echo "exit=${PIPESTATUS[0]}" >> $L/pnpm-dev-3.log
date +%s > $L/dev-end.txt
