#!/bin/bash
# usage: k6gw.sh up <LAT> | down — gateway phụ :18096 cho k6 (cùng mạng/DB/Redis với ep-r, nới hạn mức như benchmarks/load/k6-stack.yml)
if [ "$1" = down ]; then docker rm -f ep-r-gateway-qc3k6 >/dev/null 2>&1; exit 0; fi
LAT=${2:-300-300}; docker rm -f ep-r-gateway-qc3k6 >/dev/null 2>&1
docker inspect ep-r-gateway-1 --format '{{range .Config.Env}}{{println .}}{{end}}' | grep -vE '^(RATE_LIMIT_IP_PER_MIN|RATE_LIMIT_USER_PER_MIN|AUTH_LOGIN_IP_PER_MIN|AUTH_REGISTER_IP_PER_HOUR|AUTH_TOKEN_IP_PER_MIN|LLM_MAX_CONCURRENCY|LLM_DEFAULT_RPM|LLM_DEFAULT_TPM|FAKE_LLM_LATENCY|DISPLAY_RATE_PER_MIN|PATH=|$)' > /tmp/k6gw.env
cat >> /tmp/k6gw.env <<EOF
RATE_LIMIT_IP_PER_MIN=300000
RATE_LIMIT_USER_PER_MIN=100000
AUTH_LOGIN_IP_PER_MIN=1000
AUTH_REGISTER_IP_PER_HOUR=1000
AUTH_TOKEN_IP_PER_MIN=1000
LLM_MAX_CONCURRENCY=100
LLM_DEFAULT_RPM=100000
LLM_DEFAULT_TPM=100000000
FAKE_LLM_LATENCY=$LAT
DISPLAY_RATE_PER_MIN=120
EOF
IMG=$(docker inspect ep-r-gateway-1 --format '{{.Config.Image}}'); NET=$(docker inspect ep-r-gateway-1 --format '{{range $k,$v := .NetworkSettings.Networks}}{{$k}}{{end}}')
docker run -d --name ep-r-gateway-qc3k6 --network $NET --env-file /tmp/k6gw.env -p 18096:8080 $IMG >/dev/null; rm -f /tmp/k6gw.env
for i in $(seq 1 40); do curl -sk http://localhost:18096/api/v1/me -o /dev/null -w '%{http_code}' 2>/dev/null | grep -q 401 && echo ready && exit 0; sleep 2; done; echo notready
