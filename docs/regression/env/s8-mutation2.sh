#!/bin/bash
L=~/Documents/TA_Agent_qc2/docs/regression/evidence-8/logs
cd /Users/kuro/checkouts/qc2mut || exit 1
export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true API_URL=http://127.0.0.1:18231/api/v1 SEED_DEFAULT_PASSWORD='Edupilot#Seed-2026' NODE_TLS_REJECT_UNAUTHORIZED=0
( time bash scripts/gate-p6.sh ) > $L/seed25-mut2-halfeven.log 2>&1; echo "rc=$?" >> $L/seed25-mut2-halfeven.log
git checkout -q backend-go/internal/grade/engine.go; echo DONE >> $L/seed25-mut2-halfeven.log
