#!/bin/bash
# TC-P212-17: gateway APP_ENV=production từ chối join_code đặt tay (422); APP_ENV=test chấp nhận. DB riêng qc2i_s4prod, redis riêng; xoá sau khi đo.
set -u; NET=ep-r_default; PGC=ep-r-postgres-1; DB=qc2i_s4prod; OUT=~/Documents/TA_Agent_qc2/docs/regression/evidence-4/logs/s4n-p21217.txt
ENVF=$(mktemp); trap 'docker rm -f s4n-gwprod s4n-gwtest s4n-redis >/dev/null 2>&1; docker exec $PGC psql -U edupilot -d postgres -qc "drop database if exists $DB with (force)"; rm -f $ENVF' EXIT
docker inspect ep-r-gateway-1 --format '{{range .Config.Env}}{{println .}}{{end}}' | grep -v -E '^(APP_ENV|DATABASE_URL|PGBOUNCER_URL|REDIS_URL|LLM_DEMO_FILE|LLM_PROVIDER|TEAMS_ENABLED|GRAPH_[A-Z_]*|PATH|SMTP_[A-Z_]*|IMAP_[A-Z_]*|MAIL_[A-Z_]*|APP_PUBLIC_URL)=' | grep . > $ENVF
URL="postgres://edupilot:edupilot-dev@postgres:5432/$DB?sslmode=disable"
docker exec $PGC psql -U edupilot -d postgres -qc "drop database if exists $DB with (force)" -c "create database $DB"
docker run -d --name s4n-redis --network $NET redis:8 >/dev/null
docker run --rm --network $NET --env-file $ENVF -e DATABASE_URL=$URL -e PGBOUNCER_URL=$URL edupilot-gateway:latest migrate up | tail -1
run() { n=$1; port=$2; env=$3; docker run -d --name $n --network $NET --env-file $ENVF -e APP_ENV=$env -e DATABASE_URL=$URL -e PGBOUNCER_URL=$URL -e REDIS_URL=redis://s4n-redis:6379/0 -e APP_PUBLIC_URL=https://localhost -e TEAMS_ENABLED=false -e LLM_PROVIDER=fake -e SMTP_HOST=localhost -e SMTP_PORT=1025 -e SMTP_TLS=none -p 127.0.0.1:$port:8080 edupilot-gateway:latest >/dev/null; }
run s4n-gwprod 18291 production; run s4n-gwtest 18292 test
for p in 18291 18292; do for i in $(seq 30); do [ "$(curl -s -o /dev/null -w %{http_code} http://127.0.0.1:$p/api/v1/healthz)" = 200 ] && break; sleep 2; done; done
{ echo "gwprod logs: $(docker logs s4n-gwprod 2>&1 | tail -2 | cut -c1-300)"
PW='S4nProd!Pass-2026x'
for c in s4n-gwprod s4n-gwtest; do ADMIN_PASSWORD=$PW docker run --rm --network $NET --env-file $ENVF -e APP_ENV=test -e DATABASE_URL=$URL -e PGBOUNCER_URL=$URL -e ADMIN_PASSWORD edupilot-gateway:latest admin create --email adm@s4n.test --name Adm >/dev/null 2>&1; break; done
for pair in "production 18291" "test 18292"; do set -- $pair; B=http://127.0.0.1:$2/api/v1
 T=$(curl -s -X POST $B/auth/login -H 'content-type: application/json' -d "{\"email\":\"adm@s4n.test\",\"password\":\"$PW\"}" | python3 -c 'import sys,json; print(json.load(sys.stdin).get("access_token",""))')
 [ -n "$T" ] || echo "$1 login lỗi"
 for body in '{"subject_code":"INT1006","class_code":"S4P%s","name":"x","semester":"2026-2027-HK1","join_code":"ABCDEFG"}' '{"subject_code":"INT1006","class_code":"S4Q%s","name":"x","semester":"2026-2027-HK1"}'; do
  b=$(printf "$body" $1); echo "APP_ENV=$1 body có join_code=$(echo "$b" | grep -c join_code): $(curl -s -w ' HTTP %{http_code}' -X POST $B/admin/courses -H "authorization: Bearer $T" -H "Idempotency-Key: $(uuidgen)" -H 'content-type: application/json' -d "$b" | cut -c1-260)"; done
done
echo "DB courses: $(docker exec $PGC psql -U edupilot -d $DB -At -F, -c "select class_code,join_code from courses order by class_code" | tr '\n' ' ')"; } > $OUT 2>&1
cat $OUT
