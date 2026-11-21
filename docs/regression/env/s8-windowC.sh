#!/bin/bash
# SEED-19 (kill giữa chừng + resume) + default RNG: DB mới, seed RNG=1; kill khi thấy [15/20]; resume.
L=$HOME/Documents/TA_Agent_qc2/docs/regression/evidence-8/logs; cd $HOME/Documents/TA_Agent_qc2/docs/regression/env
./qc2s-stack.sh up > $L/winC-up.log 2>&1
cd ~/Documents/TA_Agent_qc2
export MOCK_IMAP_URL=http://127.0.0.1:8791 MOCK_GRAPH_URL=http://127.0.0.1:8790 NODE_TLS_REJECT_UNAUTHORIZED=0 API_URL=http://127.0.0.1:18231/api/v1 MAILPIT_URL=http://127.0.0.1:18233 SEED_DEFAULT_PASSWORD='Edupilot#Seed-2026'
export SEED_ADMIN_CMD="docker exec -i qc2s-gw /gateway admin create"
node scripts/seed.mjs > $L/winC-seed1.log 2>&1 &
SP=$!
for i in $(seq 900); do grep -q '^\[15/20\]' $L/winC-seed1.log && break; sleep 1; done
sleep 3
kill $SP 2>/dev/null; for c in $(pgrep -P $SP); do kill $c 2>/dev/null; done; wait $SP 2>/dev/null
echo "killed at $(grep -E '^\[[0-9]+/20\]' $L/winC-seed1.log | tail -1)" > $L/winC-kill.txt
( time node scripts/seed.mjs ) > $L/winC-seed2.log 2>&1; echo "rc=$?" >> $L/winC-seed2.log
echo DONE >> $L/winC-seed2.log
