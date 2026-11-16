#!/bin/bash
# Dựng stack LLM riêng (smk3-pg + smk3-redis + gateway tuỳ chọn) trên mạng ep-r_default để chạy ca đổi tuyến/ngân sách/redis dừng... KHÔNG chạm ep-r.
# usage: mini.sh up | gw NAME PORT [K=V ...] | rmgw NAME | down
IMG=edupilot-gateway-test; NET=ep-r_default; ENVF=/tmp/qc2i-gw.env.full
DB='postgres://edupilot:smk3pw@smk3-pg:5432/edupilot?sslmode=disable'; RD='redis://smk3-redis:6379/0'
case $1 in
up)
  docker rm -f smk3-pg smk3-redis >/dev/null 2>&1
  docker run -d --name smk3-pg --network $NET -e POSTGRES_PASSWORD=smk3pw -e POSTGRES_USER=edupilot -e POSTGRES_DB=edupilot -p 6144:5432 pgvector/pgvector:pg18 >/dev/null
  docker run -d --name smk3-redis --network $NET -p 6145:6379 redis:8 >/dev/null
  for i in $(seq 1 40); do docker exec smk3-pg pg_isready -U edupilot -d edupilot >/dev/null 2>&1 && break; sleep 1; done; sleep 2
  docker run --rm --network $NET --env-file $ENVF -e DATABASE_URL=$DB -e PGBOUNCER_URL=$DB -e REDIS_URL=$RD $IMG migrate up 2>&1 | tail -2
  docker run --rm --network $NET --env-file $ENVF -e DATABASE_URL=$DB -e PGBOUNCER_URL=$DB -e REDIS_URL=$RD -e ADMIN_PASSWORD='Zx9#Quartz-Lantern-2026' -e BCRYPT_COST=4 $IMG admin create --email admin@smk3.test --name 'Smk3 Admin' 2>&1 | tail -1
  ;;
gw)
  N=$2; P=$3; shift 3; E=(); for kv in "$@"; do E+=(-e "$kv"); done
  docker rm -f $N >/dev/null 2>&1
  docker run -d --name $N --network $NET -p $P:8080 --read-only --tmpfs /tmp -v /Users/kuro/Documents/TA_Agent_qc2/seed/demo:/seed/demo:ro --add-host host.docker.internal:host-gateway \
    --env-file $ENVF -e DATABASE_URL=$DB -e PGBOUNCER_URL=$DB -e REDIS_URL=$RD -e BCRYPT_COST=4 -e GUIDANCE_ENABLED=false -e RATE_LIMIT_IP_PER_MIN=1000000 -e RATE_LIMIT_USER_PER_MIN=1000000 -e AUTH_LOGIN_IP_PER_MIN=100000 "${E[@]}" $IMG >/dev/null
  for i in $(seq 1 40); do curl -fs http://localhost:$P/api/v1/readyz >/dev/null 2>&1 && { echo "$N ready"; exit 0; }; sleep 1; done; echo "$N NOT ready"; docker logs --tail 5 $N
  ;;
rmgw) docker rm -f $2 >/dev/null 2>&1;;
down) docker rm -f smk3-gw1 smk3-gw2 smk3-pg smk3-redis >/dev/null 2>&1; echo down;;
esac
