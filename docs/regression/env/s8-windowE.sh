#!/bin/bash
# SEED-36 rerun sau DEF-Q2-S8-02 (9c1336ce): DB trống, SEED_RNG=1, so sổ điểm lớp 1 THEO account với expected_final_grades.csv
R=$HOME/Documents/TA_Agent_qc2; L=$R/docs/regression/evidence-8/logs; cd $R/docs/regression/env
./qc2s-stack.sh up > $L/winE-up.log 2>&1
cd $R
export SEED_RNG=1 API_URL=http://127.0.0.1:18231/api/v1 MAILPIT_URL=http://127.0.0.1:18233 SEED_DEFAULT_PASSWORD='Edupilot#Seed-2026' NODE_TLS_REJECT_UNAUTHORIZED=0 MOCK_IMAP_URL=http://127.0.0.1:8791 MOCK_GRAPH_URL=http://127.0.0.1:8790 SEED_ADMIN_CMD="docker exec -i qc2s-gw /gateway admin create"
s=$(date +%s); node scripts/seed.mjs > $L/winE-seed.log 2>&1; echo "seed RNG=1: rc=$? $(( $(date +%s)-s )) s" > $L/winE-time.txt
python3 docs/regression/evidence-8/scripts/s8seed-g.py > $L/winE-compare.log 2>&1
echo DONE >> $L/winE-time.txt
