#!/bin/bash
until mkdir /tmp/qc9s-pw.lock 2>/dev/null; do sleep 5; done
cd "$(dirname "$0")"
export CID=01a1ef3e-8372-7b15-9e9e-902d23312b7d AID=$(cat /tmp/qc_lh_aid.txt) QC_PW="$(grep '^SEED_DEFAULT_PASSWORD=' ~/Documents/TA_Agent_qc5/.env.local | cut -d= -f2-)" RUNS=${RUNS:-3} PATHS="${LHPATH:-/practice/$AID}" OUTF=$PWD/lh-1217-id.json DIAGFILE=${DIAGFILE:-}
timeout 900 node lh_practice.mjs 2>&1 | grep -vE "^\s*at " | tail -5
pkill -f "remote-debugging-port=9333" 2>/dev/null; rmdir /tmp/qc9s-pw.lock
