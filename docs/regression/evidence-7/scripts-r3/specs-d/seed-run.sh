#!/bin/bash
# chạy scripts/seed.mjs của bản checkout ~/checkouts/qc3s7seed lên stack seed phụ (gateway :18095, DB qc3seed, mailpit :18026); in log ra stdout
cd ~/checkouts/qc3s7seed
export NODE_TLS_REJECT_UNAUTHORIZED=0 APP_ENV=test API_URL=http://127.0.0.1:18095/api/v1 MAILPIT_URL=http://127.0.0.1:18026 SEED_DEFAULT_PASSWORD='Edupilot#Seed-2026'
export MOCK_IMAP_URL=http://127.0.0.1:18091 MOCK_GRAPH_URL=http://127.0.0.1:8790
export SEED_ADMIN_CMD="docker exec -i -e APP_ENV=test ep-r-gateway-s7seed /gateway admin create"
exec node scripts/seed.mjs "$@"
