#!/bin/bash
# usage: . api.sh ; tok admin ; api METHOD path [json] [extra curl args]
B=https://localhost:943/api/v1
tok(){ curl -sk -m 20 -X POST $B/auth/login -H 'Content-Type: application/json' -d "{\"email\":\"$1@edupilot.local\",\"password\":\"Edupilot#Seed-2026\"}" | python3 -c "import sys,json;print(json.load(sys.stdin)['access_token'])"; }
api(){ local m=$1 p=$2 d=$3; shift 3; curl -sk -m 60 -X $m "$B$p" -H "Authorization: Bearer $T" -H 'Content-Type: application/json' ${d:+-d "$d"} "$@"; }
PSQL(){ docker exec -i ep-qc-postgres-1 psql -U edupilot -d edupilot -At -c "$1"; }
