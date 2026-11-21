#!/bin/bash
# đếm dòng mọi bảng của DB qc3_seed + thư Mailpit qc3s; in "bảng|số"
cd ~/Documents/TA_Agent_qc3/docs/regression/evidence-6/k6
./qc3s-stack.sh psql "select table_name||'|'||(xpath('/row/c/text()', query_to_xml(format('select count(*) as c from %I.%I', table_schema, table_name), false, true, '')))[1]::text from information_schema.tables where table_schema='public' and table_type='BASE TABLE' order by 1"
echo "mailpit|$(curl -s 'http://127.0.0.1:18243/api/v1/messages?limit=1' | python3 -c 'import sys,json;print(json.load(sys.stdin)["total"])')"
