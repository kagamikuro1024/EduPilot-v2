#!/bin/bash
# Cửa sổ cuối: DB trống qc2s, seed --if-empty (đo giây: P212-14 ≤ 360 s theo P-11-76; SEED-21), 7 check-*-seed (TC-N02-02), seed lần 2 + đếm bảng (TC-N02-03)
R=$HOME/Documents/TA_Agent_qc2; L=$R/docs/regression/evidence-8/logs; cd $R/docs/regression/env
./qc2s-stack.sh up > $L/winD-up.log 2>&1
cd $R
export API_URL=http://127.0.0.1:18231/api/v1 MAILPIT_URL=http://127.0.0.1:18233 SEED_DEFAULT_PASSWORD='Edupilot#Seed-2026' NODE_TLS_REJECT_UNAUTHORIZED=0 MOCK_IMAP_URL=http://127.0.0.1:8791 MOCK_GRAPH_URL=http://127.0.0.1:8790 SEED_ADMIN_CMD="docker exec -i qc2s-gw /gateway admin create"
unset SEED_RNG
cnt(){ docker exec -i ep-r-postgres-1 psql -U edupilot -d qc2i_seed -At -F ' ' -c "select table_name, (xpath('/row/c/text()', query_to_xml('select count(*) as c from public.'||quote_ident(table_name), false, true, '')))[1]::text from information_schema.tables where table_schema='public' and table_type='BASE TABLE' order by 1" > $1; }
s=$(date +%s); node scripts/seed.mjs --if-empty > $L/winD-seed1.log 2>&1; rc=$?; e=$(date +%s)
echo "seed --if-empty DB trống: rc=$rc $((e-s)) s (đồng hồ treo tường; không có stack/agent nặng khác ngoài ep-r nền)" > $L/winD-time.txt
for f in scripts/check-{assignments,chat,crm,docs,exam,grade,knowledge}-seed.mjs; do
  case $f in *exam*) for m in bank scores demo; do node $f $m --no-rerun > $L/winD-$(basename $f .mjs)-$m.log 2>&1; echo "$f $m $?"; done ;; *) node $f --no-rerun > $L/winD-$(basename $f .mjs).log 2>&1; echo "$f $?" ;; esac
done > $L/winD-checks-rc.txt
cnt $L/winD-n1.txt
s=$(date +%s); node scripts/seed.mjs > $L/winD-seed2.log 2>&1; echo "seed lần 2: rc=$? $(( $(date +%s)-s )) s" >> $L/winD-time.txt
cnt $L/winD-n2.txt; diff $L/winD-n1.txt $L/winD-n2.txt > $L/winD-diff.txt; echo "diff rc=$? dòng=$(wc -l < $L/winD-diff.txt)" >> $L/winD-time.txt
P(){ docker exec ep-r-postgres-1 psql -U edupilot -d qc2i_seed -tAc "$1"; }
{ echo "email trùng: $(P "select count(*) from (select lower(email) from users group by 1 having count(*)>1) x")"; echo "mã lớp trùng: $(P "select count(*) from (select class_code from courses group by 1 having count(*)>1) x")"; echo "tài liệu (course,title) trùng: $(P "select count(*) from (select course_id,title from documents group by 1,2 having count(*)>1) x")"; } > $L/winD-dups.txt
echo DONE >> $L/winD-time.txt
