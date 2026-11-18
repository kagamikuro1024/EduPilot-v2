#!/bin/bash
# smkt-gw.sh up <tên> <cổng host> [ENV=VAL…] | down <tên>  — gateway phụ trên mạng ep-r_default (cùng DB/Redis ep-r), không vào Caddy.
# env lấy từ ep-r-gateway-1 vào /tmp/smkT/gw.env (0600, chứa bí mật, KHÔNG commit)
case "$1" in
up)
  n=$2; p=$3; shift 3; extra=(); for e in "$@"; do extra+=(-e "$e"); done
  umask 077; docker inspect ep-r-gateway-1 --format '{{range .Config.Env}}{{println .}}{{end}}' | grep -vE '^(INSTANCE_ID|HTTP_ADDR)=' | grep -E '=' > /tmp/smkT/gw.env
  docker rm -f $n >/dev/null 2>&1
  docker run -d --name $n --network ep-r_default --env-file /tmp/smkT/gw.env -e INSTANCE_ID=$n "${extra[@]}" -p 127.0.0.1:$p:8080 -v $HOME/Documents/TA_Agent_qc2/seed/demo:/seed/demo:ro ${GWIMG:-edupilot-gateway-test}
  for i in $(seq 8); do curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:$p/healthz | grep -q 200 && echo ready && break; sleep 2; done
  ;;
down) docker rm -f $2 ;;
esac
