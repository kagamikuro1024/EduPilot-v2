# Thư viện QC trên ep-r:  source docs/regression/env/ep-r-lib.sh
export B=https://localhost:1143/api/v1 PW='Edupilot#Seed-2026'
export QC_BASE=https://localhost:1143 QC_PW="$PW" QC_PG=ep-r-postgres-1 QC_REDIS=ep-r-redis-1 QC_MAILPIT=http://localhost:8725 EP_PORT_MOCK_GRAPH=8790 EP_PORT_MOCK_IMAP=8791 COMPOSE_PROJECT_NAME=ep-r
export NODE_TLS_REJECT_UNAUTHORIZED=0
export REPO=$HOME/Documents/TA_Agent_qc2
tok(){ curl -sk -X POST $B/auth/login -H 'content-type: application/json' -H "Idempotency-Key: $(uuidgen)" -d "{\"email\":\"$1@edupilot.local\",\"password\":\"$PW\"}" | jq -r .access_token; }
psqlr(){ docker exec -i ep-r-postgres-1 psql -U edupilot -d edupilot -At -F '|' "$@"; }
rlclear(){ docker exec ep-r-redis-1 sh -c "redis-cli --scan --pattern 'ep:rl:auth*' | xargs -r redis-cli del" >/dev/null; }
