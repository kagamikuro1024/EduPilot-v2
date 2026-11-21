#!/bin/bash
# DEF-R4-04 chạy lại: DB tạm trong ep-r-postgres-1 (schema của ep-r gồm migration 00029 + 60.000 user mô phỏng), EXPLAIN câu ListUsersAdmin; xoá DB tạm
set -u; P="docker exec -i ep-r-postgres-1"; D=qc2i_s4idx; OUT=docs/regression/evidence-4/logs/s4n-p20628-explain.txt
$P psql -U edupilot -d postgres -qc "drop database if exists $D with (force)" -c "create database $D"
$P pg_dump -U edupilot -d edupilot -s | $P psql -U edupilot -d $D -q >/dev/null 2>&1
Q="$P psql -U edupilot -d $D -At"
$Q -c "insert into users (email,full_name,role,status,created_at,password_hash,email_verified_at) select 'qc2s4-bulk'||g||'@example.test','Bulk User '||g,(array['STUDENT','TEACHER','TA'])[1+g%3]::user_role,'ACTIVE',now()-(g||' seconds')::interval,'x',now() from generate_series(1,60000) g" -c "analyze users" >/dev/null
{ echo "users: $($Q -c "select count(*) from users")"; echo "indexes: $($Q -c "select indexname from pg_indexes where tablename='users'" | tr '\n' ',')"
for c in "mặc định|true|true|true|" "lọc role|role='TEACHER'::user_role|true|true|" "lọc status|true|status='ACTIVE'::user_status|true|" "con trỏ|true|true|true|and (created_at,id) < (now()-interval '3 hours','00000000-0000-7000-8000-000000000000'::uuid)"; do
 IFS='|' read -r lab r s n cur <<<"$c"; echo "### $lab"; $Q -c "explain (analyze, format text) select id,email,full_name,role,status,last_login_at,version,created_at from users where ($r) and ($s) and ($n) $cur order by created_at desc, id desc limit 21"; done; } > $OUT 2>&1
$P psql -U edupilot -d postgres -qc "drop database if exists $D with (force)"
cat $OUT | grep -E "^###|Scan|Sort|Execution|users:|indexes:"
