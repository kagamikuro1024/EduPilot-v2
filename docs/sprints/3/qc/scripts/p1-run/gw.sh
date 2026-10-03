#!/bin/bash
# usage: gw.sh restart [ENV=VAL ...]
cd /tmp/qcp1; . ./env.sh
pkill -x gw-test; pkill -x gw-prod
for i in $(seq 1 40); do lsof -ti:8080 -ti:8081 >/dev/null 2>&1 || break; sleep 0.5; done
for kv in "$@"; do [ "${kv#UNSET_}" != "$kv" ] && unset ${kv#UNSET_} || export "$kv"; done
nohup ./gw-test serve > gw1.log 2>&1 &
HTTP_ADDR=:8081 nohup ./gw-test serve > gw2.log 2>&1 &
for i in $(seq 1 20); do curl -s localhost:8080/api/v1/readyz >/dev/null && curl -s localhost:8081/api/v1/readyz >/dev/null && break; sleep 0.5; done
curl -s localhost:8080/api/v1/readyz; echo
