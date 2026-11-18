#!/bin/bash
# Stack TẠM riêng cho ca cần DB trống/seed sạch và ca pnpm dev (sprint 1/2/5/8). KHÔNG phải project edupilot.
# Dùng: nohup bash docs/regression/env/ep-sd-up.sh > /tmp/qc2-sd-up.out 2>&1 &    Dọn: cd repo && docker compose -p ep-sd --env-file .env.local -f docker-compose.local.yml down   (KHÔNG -v nếu không cần; volume ep-sd_* tên riêng)
cd ~/Documents/TA_Agent_qc2 || exit 1
export COMPOSE_PROJECT_NAME=ep-sd EP_PORT_OFFSET=1500 SEED_ON_EMPTY_DB=true COMPOSE_PROFILES=ingest
node scripts/dev.mjs up
