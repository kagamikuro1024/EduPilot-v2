#!/bin/bash
cd "$(dirname "$0")/../../../.."
export BASE=https://localhost:1143 INSECURE_TLS=1 OUT=$PWD/docs/regression/evidence-f16/n08-11/out
export AUDIT_PSQL="docker exec -i ep-r-postgres-1 psql -U edupilot -d edupilot -tA -v ON_ERROR_STOP=1"
echo "=== threads $(date +%T)"; python3 benchmarks/real-llm/real_llm.py threads || echo "rc=$?"
echo "=== done $(date +%T)"
