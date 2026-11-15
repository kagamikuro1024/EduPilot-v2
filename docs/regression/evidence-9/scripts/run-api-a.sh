#!/bin/bash
# nhóm không chấm LLM: chạy tuần tự, log vào logs/
cd $HOME/Documents/TA_Agent_qc2; S=docs/regression/evidence-9/scripts; L=docs/regression/evidence-9/logs
export GW=https://localhost:1143 NODE_TLS_REJECT_UNAUTHORIZED=0 PSQL="docker exec -i ep-r-postgres-1 psql -U edupilot -d edupilot -tAc" MP=http://127.0.0.1:8725 MOCK_IMAP_URL=http://127.0.0.1:8791 MOCK_GRAPH_URL=http://127.0.0.1:8790
export QC_API=https://localhost:1143/api/v1 QC_PG=ep-r-postgres-1
for n in 02 03 04 05; do python3 $S/p7-api-$n.py > $L/api-$n.log 2>&1; echo "rc=$?" >> $L/api-$n.log; done
python3 $S/p7-api-11.py > $L/api-11.log 2>&1; echo "rc=$?" >> $L/api-11.log
python3 $S/p7-imap-live.py > $L/imap-live.log 2>&1; echo "rc=$?" >> $L/imap-live.log
echo DONE > $L/api-a.done
