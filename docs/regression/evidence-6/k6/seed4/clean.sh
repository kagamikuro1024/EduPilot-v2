#!/bin/bash
cd ~/Documents/TA_Agent_qc3/docs/regression/evidence-6/k6
./qc3s-stack.sh up > seed4/up.log 2>&1
date +%s > seed4/t0
EP_PORT_MOCK_GRAPH=8790 ./qc3s-stack.sh seed > seed4/seed.log 2>&1
date +%s > seed4/t1
cp ~/Documents/TA_Agent_qc3p/benchmarks/reports/seed-t0.json seed4/seed-t0.json
(cd ~/Documents/TA_Agent_qc3p && git checkout benchmarks/reports/seed-t0.json 2>/dev/null)
echo "qc3s seed xong $(date +%H:%M:%S) trong $(( $(cat seed4/t1) - $(cat seed4/t0) )) s" > /tmp/qc3s-ready
