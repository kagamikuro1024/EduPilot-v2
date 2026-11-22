#!/bin/bash
# Lượt đo lại seed (P-11-76): project qc2seedt2, offset 4200, cây /Users/kuro/checkouts/qc2seedt2 (>= f473c2ff).
# $1 = prebuild (dựng + lên, KHÔNG seed) | measure (dev.mjs với SEED_ON_EMPTY_DB=true) | checks | second (seed lần hai)
cd /Users/kuro/checkouts/qc2seedt2
L=$HOME/Documents/TA_Agent_qc2/docs/regression/evidence-2/fresh/seedt2
mkdir -p $L
export_env=(env -i HOME="$HOME" PATH="$PATH" DOCKER_HOST=unix://$HOME/.colima/default/docker.sock COMPOSE_PROJECT_NAME=qc2seedt2 EP_PORT_OFFSET=4200 COMPOSE_PROFILES=ingest)
ls_json() { docker compose ls -a --format json | python3 -c "import sys,json; print([(p['Name'],p['Status']) for p in json.load(sys.stdin)])"; }
ts() { while IFS= read -r l; do echo "$(date +%H:%M:%S) $l"; done; }
case "$1" in
prebuild) "${export_env[@]}" node scripts/dev.mjs 2>&1 | ts > $L/prebuild.log; echo "prebuild exit=${PIPESTATUS[0]}" >> $L/prebuild.log ;;
measure)
  ls_json > $L/compose-ls-before.txt
  { echo "sha: $(git log -1 --format=%h)"; echo "load: $(uptime | sed 's/.*load/load/')"; colima ssh -- free -m | sed -n 2p; docker ps --format '{{.Names}}'; } > $L/pre.txt
  date +%s > $L/start.txt
  "${export_env[@]}" SEED_ON_EMPTY_DB=true node scripts/dev.mjs 2>&1 | ts > $L/dev.log
  echo "exit=${PIPESTATUS[0]}" >> $L/dev.log
  date +%s > $L/end.txt
  { echo "load sau: $(uptime | sed 's/.*load/load/')"; } >> $L/pre.txt
  ls_json > $L/compose-ls-after.txt; echo DONE >> $L/dev.log ;;
checks)
  U="API_URL=https://localhost:4643/api/v1 MAILPIT_URL=http://localhost:12225 MOCK_IMAP_URL=http://localhost:12291 MOCK_GRAPH_URL=http://localhost:12290 NODE_TLS_REJECT_UNAUTHORIZED=0 COMPOSE_PROJECT_NAME=qc2seedt2"
  for c in practice assignments grade crm docs; do
    echo "== check-$c-seed" ; env -i HOME="$HOME" PATH="$PATH" DOCKER_HOST=unix://$HOME/.colima/default/docker.sock $U timeout 600 node scripts/check-$c-seed.mjs 2>&1 | tail -6; echo "rc=${PIPESTATUS[0]}"
  done > $L/checks-${2:-1}.log 2>&1 ;;
second)
  env -i HOME="$HOME" PATH="$PATH" DOCKER_HOST=unix://$HOME/.colima/default/docker.sock API_URL=https://localhost:4643/api/v1 MAILPIT_URL=http://localhost:12225 MOCK_IMAP_URL=http://localhost:12291 MOCK_GRAPH_URL=http://localhost:12290 NODE_TLS_REJECT_UNAUTHORIZED=0 COMPOSE_PROJECT_NAME=qc2seedt2 node scripts/seed.mjs 2>&1 | ts > $L/second.log; echo "exit=${PIPESTATUS[0]}" >> $L/second.log ;;
esac
