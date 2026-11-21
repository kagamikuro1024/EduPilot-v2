#!/bin/bash
# $1 = p5|p6 ; chạy gate với API_URL trỏ stack qc2s
cd ~/Documents/TA_Agent_qc2
L=docs/regression/evidence-8/logs
export SEED_ADMIN_CMD="docker exec -i qc2s-gw /gateway admin create" MOCK_IMAP_URL=http://127.0.0.1:8791 MOCK_GRAPH_URL=http://127.0.0.1:8790 API_URL=http://127.0.0.1:18231/api/v1 SEED_DEFAULT_PASSWORD='Edupilot#Seed-2026' MAILPIT_URL=http://127.0.0.1:18233 NODE_TLS_REJECT_UNAUTHORIZED=0
export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true
( time bash scripts/gate-$1.sh ) > $L/gate-$1.log 2>&1; echo "rc=$?" >> $L/gate-$1.log
