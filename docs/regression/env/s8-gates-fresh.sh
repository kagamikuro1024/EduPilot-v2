#!/bin/bash
# gate-p5/p6 chống stack qc2fresh (https://localhost:3443)
cd ~/Documents/TA_Agent_qc2
L=docs/regression/evidence-8/logs
export API_URL=https://localhost:3443/api/v1 SEED_DEFAULT_PASSWORD='Edupilot#Seed-2026' MAILPIT_URL=http://localhost:11025 NODE_TLS_REJECT_UNAUTHORIZED=0 MOCK_IMAP_URL=http://localhost:11091 MOCK_GRAPH_URL=http://localhost:11090
export SEED_ADMIN_CMD="docker exec -i qc2fresh-gateway-1 /gateway admin create"
export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true
( time bash scripts/gate-$1.sh ) > $L/gate-$1-fresh.log 2>&1; echo "rc=$?" >> $L/gate-$1-fresh.log
