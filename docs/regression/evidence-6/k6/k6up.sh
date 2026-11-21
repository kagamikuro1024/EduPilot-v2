#!/bin/bash
# usage: k6up.sh <LAT>|reset — dựng lại qc3s-gw/qc3s-worker (DB qc3_seed) với hạn mức nới + FAKE_LLM_LATENCY=<LAT> cho k6 exam-submit.js (reset = trả env mặc định của stack qc3s)
set -u
REPO=$HOME/Documents/TA_Agent_qc3p; NET=ep-r_default
GWENV=$HOME/.config/edupilot-team/qc3s-gw.env; WKENV=$HOME/.config/edupilot-team/qc3s-wk.env
K6G=$HOME/.config/edupilot-team/qc3s-gw.k6.env; K6W=$HOME/.config/edupilot-team/qc3s-wk.k6.env
docker rm -f qc3s-gw qc3s-worker >/dev/null 2>&1
if [ "$1" = reset ]; then G=$GWENV; W=$WKENV; else
  X="RATE_LIMIT_IP_PER_MIN=300000\nRATE_LIMIT_USER_PER_MIN=100000\nAUTH_LOGIN_IP_PER_MIN=100000\nAUTH_REGISTER_IP_PER_HOUR=1000\nAUTH_TOKEN_IP_PER_MIN=100000\nLLM_MAX_CONCURRENCY=100\nLLM_DEFAULT_RPM=100000\nLLM_DEFAULT_TPM=100000000\nFAKE_LLM_LATENCY=$1\nDISPLAY_RATE_PER_MIN=120"
  grep -vE '^(RATE_LIMIT_IP_PER_MIN|RATE_LIMIT_USER_PER_MIN|AUTH_LOGIN_IP_PER_MIN|AUTH_REGISTER_IP_PER_HOUR|AUTH_TOKEN_IP_PER_MIN|LLM_MAX_CONCURRENCY|LLM_DEFAULT_RPM|LLM_DEFAULT_TPM|FAKE_LLM_LATENCY|DISPLAY_RATE_PER_MIN)=' $GWENV > $K6G; printf "$X\n" >> $K6G
  grep -vE '^(LLM_MAX_CONCURRENCY|LLM_DEFAULT_RPM|LLM_DEFAULT_TPM|FAKE_LLM_LATENCY)=' $WKENV > $K6W; printf "LLM_MAX_CONCURRENCY=100\nLLM_DEFAULT_RPM=100000\nLLM_DEFAULT_TPM=100000000\nFAKE_LLM_LATENCY=$1\n" >> $K6W; G=$K6G; W=$K6W; fi
docker run -d --name qc3s-gw --network $NET --env-file $G --memory 512m -v $REPO/seed/demo:/seed/demo:ro -p 127.0.0.1:18241:8080 edupilot-gateway-test >/dev/null
docker run -d --name qc3s-worker --network $NET --env-file $W --memory 512m -v $REPO/seed/demo:/seed/demo:ro edupilot-worker-test >/dev/null
docker network connect ep-r_judge_net qc3s-worker 2>/dev/null; docker network connect ep-r_judge_net qc3s-gw 2>/dev/null
for i in $(seq 1 40); do curl -s http://127.0.0.1:18241/api/v1/me -o /dev/null -w '%{http_code}' 2>/dev/null | grep -q 401 && echo ready && exit 0; sleep 2; done; echo notready
