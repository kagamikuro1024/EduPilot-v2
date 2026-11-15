#!/bin/bash
# grading-batch.sh <tên> <script...> : lấy khoá toàn cục (owner=SmkD), GRADING→QC fake, chạy tuần tự; mỗi script chạy lại nếu route bị người khác đổi giữa chừng; cuối cùng trả route + nhả khoá nếu còn là của mình.
source /tmp/ep-r-lib.sh
cd $HOME/Documents/TA_Agent_qc2; S=docs/regression/evidence-9/scripts; L=docs/regression/evidence-9/logs
export GW=https://localhost:1143 NODE_TLS_REJECT_UNAUTHORIZED=0 MOCK_IMAP_URL=http://127.0.0.1:8791 MOCK_GRAPH_URL=http://127.0.0.1:8790 MP=http://127.0.0.1:8725
FAKE=01a1df2d-a8ee-70d6-91cd-be284dbb4097; REAL=01a1df2d-a8d7-76a5-9cd8-5dbdb8646c78
acquire(){ for i in $(seq 1 180); do if mkdir /tmp/qc2-lock-global 2>/dev/null; then echo SmkD > /tmp/qc2-lock-global/owner; return 0; fi; sleep 10; done; return 1; }
mine(){ [ "$(cat /tmp/qc2-lock-global/owner 2>/dev/null)" = SmkD ]; }
prov(){ AT=$(tok admin); curl -sk $B/admin/llm/routes -H "authorization: Bearer $AT" | jq -r '.items[]|select(.task=="GRADING")|.chain[0].provider_name'; }
setroute(){ AT=$(tok admin); V=$(curl -sk $B/admin/llm/routes -H "authorization: Bearer $AT" | jq '.items[]|select(.task=="GRADING").version'); curl -sk -X PUT $B/admin/llm/routes -H "authorization: Bearer $AT" -H 'content-type: application/json' -d "{\"task\":\"GRADING\",\"chain\":[\"$1\"],\"params\":{},\"version\":$V}" | jq -c '.chain[0].provider_name'; }
name=$1; shift
for s in "$@"; do
  for try in 1 2 3; do
    mine || { acquire || { echo "no lock" >> $L/$s.log; continue; }; echo "$(date +%T) SmkD: GRADING→QC fake (batch $name/$s)" >> /tmp/qc2-global-changes.log; setroute $FAKE >/dev/null; }
    [ "$(prov)" = "QC fake" ] || { setroute $FAKE >/dev/null; }
    python3 $S/$s.py > $L/$s.log 2>&1; echo "rc=$?" >> $L/$s.log
    if mine && [ "$(prov)" = "QC fake" ]; then break; fi
    echo "route/lock bị đổi giữa chừng, chạy lại ($try)" >> $L/$s.log.retry; sleep 5
  done
done
if mine; then setroute $REAL >/dev/null; echo "$(date +%T) SmkD: GRADING→QC 9router (hoàn lại), nhả khoá" >> /tmp/qc2-global-changes.log; rmdir /tmp/qc2-lock-global 2>/dev/null || rm -rf /tmp/qc2-lock-global; fi
