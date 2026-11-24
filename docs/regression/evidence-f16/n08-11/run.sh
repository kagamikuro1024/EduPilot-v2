#!/bin/bash
# TC-N08-11: LLM thật (OpenCode Go, deepseek-v4.1-flash) trên ep-r. Tuần tự.
cd "$(dirname "$0")/../../../.."
export BASE=https://localhost:1143 INSECURE_TLS=1 OUT=$PWD/docs/regression/evidence-f16/n08-11/out
export AUDIT_PSQL="docker exec -i ep-r-postgres-1 psql -U edupilot -d edupilot -tA -v ON_ERROR_STOP=1"
for c in misc crisis threads chat; do echo "=== $c $(date +%T)"; python3 benchmarks/real-llm/real_llm.py $c || echo "rc=$? $c"; done
echo "=== done $(date +%T)"
