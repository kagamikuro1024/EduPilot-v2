#!/bin/bash
# gate-p5/p6 chống stack qc2seedt2 (https://localhost:4643)
cd ~/Documents/TA_Agent_qc2
L=docs/regression/evidence-8/logs
export API_URL=https://localhost:4643/api/v1 SEED_DEFAULT_PASSWORD='Edupilot#Seed-2026' MAILPIT_URL=http://localhost:12225 NODE_TLS_REJECT_UNAUTHORIZED=0 MOCK_IMAP_URL=http://localhost:12291 MOCK_GRAPH_URL=http://localhost:12290
export SEED_ADMIN_CMD="docker exec -i qc2seedt2-gateway-1 /gateway admin create"
export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true
( time bash scripts/gate-$1.sh ) > $L/gate-$1-seedt2.log 2>&1; echo "rc=$?" >> $L/gate-$1-seedt2.log
