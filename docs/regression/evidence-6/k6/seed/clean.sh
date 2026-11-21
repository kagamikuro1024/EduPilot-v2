#!/bin/bash
cd ~/Documents/TA_Agent_qc3/docs/regression/evidence-6/k6
./qc3s-stack.sh up > seed/up3.log 2>&1
date +%s > seed/t0c
EP_PORT_MOCK_GRAPH=8790 ./qc3s-stack.sh seed > seed/seed3.log 2>&1
date +%s > seed/t1c
cp ~/Documents/TA_Agent_qc3p/benchmarks/reports/seed-t0.json seed/seed-t0-clean.json
