#!/usr/bin/env bash
# QC US-PG-02 — schema nền 00001, sqlc, blob, outbox (nguồn US.md v1.1 AC1–AC17 + SRS 3.4, 5.0–5.6, 8.1, 8.4, 9.2)
# Hộp đen: chỉ dùng psql / redis-cli / docker / curl / go test (qua `gt`, có chứng minh --- PASS). Không đọc mã dev.
#   bash docs/sprints/2/qc/scripts/pg02.sh            # chạy hết
#   bash docs/sprints/2/qc/scripts/pg02.sh 07 08      # chạy chọn lọc
#   bash docs/sprints/2/qc/scripts/pg02.sh --list
source "$(dirname "$0")/lib.sh"

# ---------- Tiện ích riêng của story 02 ----------
MIG=backend-go/db/migrations/00001_pg_platform.sql

# psql nói rõ SQLSTATE: "ERROR:  23514: new row for relation ..."
qv() { $PSQL -v VERBOSITY=verbose -c "$1" 2>&1; }

# sql_err <mô tả> <SQLSTATE> <chuỗi phải có trong thông báo, '' nếu bỏ qua> <câu SQL (không dấu ; cuối)>
sql_err() {
  local o; o=$(qv "BEGIN; $4; ROLLBACK;")
  printf '== %s\n%s\n' "$1" "$o" >> "$QC_OUT/pg02-sqlstate.log"
  chk_re "$1 → SQLSTATE $2" "$o" "$2:"
  [ -n "$3" ] && chk_re "$1 → thông báo" "$o" "$3"
  return 0
}

# sql_ok <mô tả> <câu INSERT đã có 'returning id'> — phải chèn được, trả uuid v7, rồi ROLLBACK
sql_ok() {
  local o; o=$(qv "BEGIN; $2; ROLLBACK;")
  chk_nre "$1 không được lỗi" "$o" 'ERROR'
  chk_re "$1 → id uuid v7" "$o" '^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
  return 0
}

# cols <bảng> — "tên|kiểu|NN|mặc định" mỗi cột một dòng, đã sắp xếp
cols() {
  $PSQL -c "select a.attname||'|'||format_type(a.atttypid,a.atttypmod)||'|'||(case when a.attnotnull then 'NN' else 'NULL' end)||'|'||coalesce(pg_get_expr(d.adbin,d.adrelid),'-') from pg_attribute a left join pg_attrdef d on d.adrelid=a.attrelid and d.adnum=a.attnum where a.attrelid='public.$1'::regclass and a.attnum>0 and not a.attisdropped" 2>/dev/null | sort
}

# cmp_cols <bảng> <số cột> <bảng chuẩn trên stdin>
cmp_cols() {
  local t=$1 n=$2 want got
  want=$(sort); got=$(cols "$t")
  chk "số cột $t" "$(printf '%s\n' "$got" | grep -c .)" "$n"
  if [ "$got" = "$want" ]; then echo "    ok   $n cột $t khớp SRS (tên, kiểu, nullable, mặc định)"
  else fail_tc "cột $t lệch SRS:"; diff <(printf '%s\n' "$want") <(printf '%s\n' "$got") | sed 's/^/      /'; fi
}

idef() { $PSQL -c "select indexdef from pg_indexes where schemaname='public' and indexname='$1'" 2>/dev/null; }

# Bảng thử `_test_items` (+2 index, do image gateway-test tạo) KHÔNG tính: mọi truy vấn đếm bảng / index / cột `id`
# đều loại `\_test\_%` (US.md 02-AC1 "Làm rõ" #Q-QC-02-3) nên chạy được cả khi DB đã qua `tmode` — không cần DB sạch.

xlen() { local v; v=$($RDS xlen "$1" 2>/dev/null); case "$v" in ''|*ERR*) echo 0;; *) echo "$v";; esac; }
xpend() { local v; v=$($RDS xpending "$1" "$2" 2>/dev/null | head -1); case "$v" in ''|*ERR*) echo 0;; *) echo "$v";; esac; }

mig() {  # mig up|down — service `migrate` có entrypoint ["/gateway","migrate"] + command ["up"] (SRS 8.4), `run` chỉ thay
         # `command` → `$C run --rm migrate down` chạy `/gateway migrate down` (US.md 02-AC6 "Làm rõ" #Q-QC-02-1). rc≠0 là FAIL.
  local o rc
  o=$($C run --rm migrate "$1" 2>&1); rc=$?
  printf '%s\n' "$o" > "$QC_OUT/mig-$1.log"; return $rc
}

newjob() {  # newjob <kind> <steps> <hậu tố khoá idem> → in mã HTTP
  code -X POST -H "$H" -H 'Content-Type: application/json' -H "Idempotency-Key: qc-pg02-$3" \
       -d "{\"steps\":$2,\"kind\":\"$1\"}" "$GW/api/v1/_test/jobs"
}

# Dòng outbox topic lạ đã đi hết đường dead-letter (AC15); dùng chung cho TC-64 và TC-65.
ac15_dead_row() {
  local f=$QC_OUT/ac15-dead-id id i st
  if [ -s "$f" ]; then cat "$f"; return 0; fi
  id=$($PSQL -c "insert into outbox(topic,payload) values('qc.unknown.topic','{\"email\":\"qc-pii-canary@example.com\"}'::jsonb) returning id" 2>/dev/null | head -1)   # QC v2: psql in thêm dòng "INSERT 0 1" sau RETURNING — chỉ lấy dòng đầu
  i=0; while [ $i -lt 90 ]; do
    st=$($PSQL -c "select coalesce(dead_at::text,'') from outbox where id='$id'" 2>/dev/null)
    [ -n "$st" ] && break
    sleep 1; i=$((i+1)); done
  printf '%s' "$id" > "$f"; printf '%s' "$id"
}

# ================= AC1 — 5 bảng + goose_db_version, 3 enum, extension vector =================
tc_pg02_01() {  # AC1 — đúng 6 bảng, không bảng nghiệp vụ nào khác
  local t
  t=$($PSQL -c "select tablename from pg_tables where schemaname='public' and tablename not like '\_test\_%' order by 1" | tr '\n' ' ' | sed 's/ $//')
  chk "danh sách bảng public" "$t" "audit_log goose_db_version idempotency_keys jobs outbox users"
  chk "bảng nghiệp vụ ngoài phạm vi (courses/enrollments/documents/notifications)" \
      "$($PSQL -c "select count(*) from pg_tables where schemaname='public' and tablename in ('courses','enrollments','documents','notifications')")" 0
}
tc_pg02_02() {  # AC1 — 3 enum đúng tên và đúng thứ tự nhãn (SRS 5.0)
  chk "danh sách enum" "$($PSQL -c "select typname from pg_type t join pg_namespace n on n.oid=typnamespace where nspname='public' and typtype='e' order by 1" | tr '\n' ' ' | sed 's/ $//')" "job_status user_role user_status"
  chk "user_role"   "$($PSQL -c "select enum_range(null::user_role)::text")"   '{ADMIN,TEACHER,TA,STUDENT}'
  chk "user_status" "$($PSQL -c "select enum_range(null::user_status)::text")" '{PENDING_VERIFICATION,INVITED,ACTIVE,DISABLED}'
  chk "job_status"  "$($PSQL -c "select enum_range(null::job_status)::text")"  '{QUEUED,RUNNING,SUCCEEDED,FAILED}'
}
tc_pg02_03() {  # AC1 — extension vector
  chk "extension vector" "$($PSQL -c "select extname from pg_extension where extname='vector'")" vector
}

# ================= AC2 — users 16 cột đúng SRS 5.1 =================
tc_pg02_04() {  # AC2 — 16 cột: tên, kiểu, nullable, mặc định
  cmp_cols users 16 <<'EOF'
id|uuid|NN|uuidv7()
email|text|NN|-
password_hash|text|NULL|-
full_name|text|NN|-
role|user_role|NN|-
student_code|text|NULL|-
email_verified_at|timestamp with time zone|NULL|-
failed_logins|integer|NN|0
locked_until|timestamp with time zone|NULL|-
status|user_status|NN|'INVITED'::user_status
ics_token|text|NULL|-
tracking_notice_ack_at|timestamp with time zone|NULL|-
last_login_at|timestamp with time zone|NULL|-
version|integer|NN|1
created_at|timestamp with time zone|NN|now()
updated_at|timestamp with time zone|NN|now()
EOF
}
tc_pg02_05() {  # AC2 — 6 cột dành cho P2/P5/P8 có sẵn
  local l; l=$($PSQL -c "select column_name||':'||data_type||':'||is_nullable from information_schema.columns where table_name='users' and table_schema='public' order by ordinal_position")
  chk "số cột users (information_schema)" "$(printf '%s\n' "$l" | grep -c .)" 16
  chk "6 cột P2/P5/P8" "$(printf '%s\n' "$l" | grep -cE '^(email_verified_at|failed_logins|locked_until|status|ics_token|tracking_notice_ack_at):')" 6
}
tc_pg02_06() { gt ./internal/store 'TestSchema_UsersColumns'; }   # AC2 — test Go của AC

# ================= AC3 — ràng buộc users (mỗi mệnh đề một TC, BEGIN…ROLLBACK) =================
tc_pg02_07() { sql_err "email có chữ hoa 'A@X.com'" 23514 users_email_lower_chk \
  "insert into users(email,full_name,role) values('A@X.com','QC','STUDENT')"; }
tc_pg02_08() { sql_err "email trùng 'qc-dup@x.com' hai lần" 23505 users_email_key \
  "insert into users(email,full_name,role) values('qc-dup@x.com','QC','STUDENT'); insert into users(email,full_name,role) values('qc-dup@x.com','QC2','STUDENT')"; }
tc_pg02_09() { sql_err "role='SUPERUSER' (ngoài enum)" 22P02 '' \
  "insert into users(email,full_name,role) values('qc-role@x.com','QC','SUPERUSER')"; }
tc_pg02_10() { sql_err "failed_logins = -1" 23514 users_failed_logins_chk \
  "insert into users(email,full_name,role,failed_logins) values('qc-fl@x.com','QC','STUDENT',-1)"; }
tc_pg02_11() { sql_err "status='ACTIVE' và password_hash = '' (rỗng)" 23514 users_active_password_chk \
  "insert into users(email,full_name,role,status,password_hash) values('qc-a1@x.com','QC','STUDENT','ACTIVE','')"; }
tc_pg02_12() { sql_err "status='ACTIVE' và password_hash NULL" 23514 users_active_password_chk \
  "insert into users(email,full_name,role,status) values('qc-a2@x.com','QC','STUDENT','ACTIVE')"; }
tc_pg02_13() { sql_err "student_code cho vai TEACHER" 23514 users_student_code_role_chk \
  "insert into users(email,full_name,role,student_code) values('qc-sc@x.com','QC','TEACHER','22010001')"; }
tc_pg02_14() { sql_err "version = 0 (SRS 5.1 CHECK version >= 1)" 23514 '' \
  "insert into users(email,full_name,role,version) values('qc-v0@x.com','QC','STUDENT',0)"; }
tc_pg02_15() { sql_ok "status='INVITED' không mật khẩu là hợp lệ" \
  "insert into users(email,full_name,role,status) values('qc-inv@x.com','QC','STUDENT','INVITED') returning id"; }
tc_pg02_16() { sql_ok "status='ACTIVE' có mật khẩu bcrypt, email chữ thường" \
  "insert into users(email,full_name,role,status,password_hash) values('qc-act@x.com','QC','TEACHER','ACTIVE','\$2b\$12\$abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXY12') returning id"; }
tc_pg02_17() {  # AC3 — student_code với vai STUDENT hợp lệ + mặc định của SRS 5.1
  sql_ok "student_code với role STUDENT" \
    "insert into users(email,full_name,role,student_code) values('qc-stu@x.com','QC','STUDENT','22010002') returning id"
  local d; d=$(qv "BEGIN; insert into users(email,full_name,role) values('qc-def@x.com','QC','STUDENT'); select status||'|'||failed_logins||'|'||version from users where email='qc-def@x.com'; ROLLBACK;")
  chk_re "mặc định status|failed_logins|version" "$d" '^INVITED\|0\|1$'
}
tc_pg02_18() { gt ./internal/store 'TestSchema_UsersConstraints'; }   # AC3 — bảng test SQLSTATE của AC

# ================= AC4 — 4 bảng còn lại, 17 index, uuidv7, các CHECK =================
tc_pg02_19() {  # AC4 — đúng 17 index, đúng tên
  local l; l=$($PSQL -c "select indexname from pg_indexes where schemaname='public' and tablename<>'goose_db_version' and tablename not like '\_test\_%' order by 1" | tr '\n' ' ' | sed 's/ $//')
  chk "số index" "$(printf '%s' "$l" | tr ' ' '\n' | grep -c .)" 17
  chk "danh sách index" "$l" "audit_log_actor_created_idx audit_log_course_created_idx audit_log_entity_idx audit_log_pkey idempotency_keys_created_idx idempotency_keys_pkey idempotency_keys_user_endpoint_key_key jobs_active_idx jobs_owner_created_idx jobs_pkey outbox_pending_idx outbox_pkey outbox_stale_idx users_email_key users_ics_token_key users_pkey users_student_code_idx"
}
tc_pg02_20() {  # AC4 — định nghĩa index: UNIQUE, partial WHERE, DESC (SRS 5.1–5.5)
  local d
  d=$(idef users_email_key);        chk_re 'users_email_key UNIQUE (email)' "$d" 'CREATE UNIQUE INDEX.*\(email\)'; chk_nre 'users_email_key không partial' "$d" 'WHERE'
  d=$(idef users_ics_token_key);    chk_re 'users_ics_token_key UNIQUE partial' "$d" 'CREATE UNIQUE INDEX.*\(ics_token\) WHERE \(ics_token IS NOT NULL\)'
  d=$(idef users_student_code_idx); chk_re 'users_student_code_idx partial' "$d" '\(student_code\) WHERE \(student_code IS NOT NULL\)'; chk_nre 'users_student_code_idx KHÔNG unique (Q17)' "$d" 'UNIQUE'
  d=$(idef audit_log_actor_created_idx);  chk_re 'audit_log_actor_created_idx' "$d" '\(actor_id, created_at DESC, id DESC\) WHERE \(actor_id IS NOT NULL\)'
  d=$(idef audit_log_course_created_idx); chk_re 'audit_log_course_created_idx' "$d" '\(course_id, created_at DESC, id DESC\) WHERE \(course_id IS NOT NULL\)'
  d=$(idef audit_log_entity_idx);         chk_re 'audit_log_entity_idx' "$d" '\(entity, entity_id, created_at DESC\)'
  d=$(idef outbox_pending_idx);     chk_re 'outbox_pending_idx khoá (next_attempt_at, id)' "$d" '\(next_attempt_at, id\) WHERE'
  chk_re 'outbox_pending_idx WHERE dispatched_at IS NULL' "$d" 'dispatched_at IS NULL'
  chk_re 'outbox_pending_idx WHERE dead_at IS NULL' "$d" 'dead_at IS NULL'
  chk_re 'outbox_pending_idx WHERE enqueued_at IS NULL' "$d" 'enqueued_at IS NULL'
  d=$(idef outbox_stale_idx);       chk_re 'outbox_stale_idx khoá (enqueued_at)' "$d" '\(enqueued_at\) WHERE'
  chk_re 'outbox_stale_idx WHERE enqueued_at IS NOT NULL' "$d" 'enqueued_at IS NOT NULL'
  chk_re 'outbox_stale_idx WHERE dispatched_at IS NULL' "$d" 'dispatched_at IS NULL'
  chk_re 'outbox_stale_idx WHERE dead_at IS NULL' "$d" 'dead_at IS NULL'
  d=$(idef jobs_owner_created_idx); chk_re 'jobs_owner_created_idx' "$d" '\(owner_id, created_at DESC, id DESC\)'
  d=$(idef jobs_active_idx);        chk_re 'jobs_active_idx khoá (status, created_at)' "$d" '\(status, created_at\) WHERE'
  chk_re 'jobs_active_idx WHERE QUEUED' "$d" 'QUEUED'; chk_re 'jobs_active_idx WHERE RUNNING' "$d" 'RUNNING'
  d=$(idef idempotency_keys_created_idx); chk_re 'idempotency_keys_created_idx' "$d" '\(created_at\)'
}
tc_pg02_21() {  # AC4 — 5 khoá chính uuid mặc định uuidv7()
  chk "cột id mặc định uuidv7()" "$($PSQL -c "select count(*) from information_schema.columns where table_schema='public' and table_name not like '\_test\_%' and column_name='id' and column_default='uuidv7()'")" 5
  chk "PK là cột id kiểu uuid (5 bảng)" "$($PSQL -c "select count(*) from pg_index i join pg_class c on c.oid=i.indrelid join pg_attribute a on a.attrelid=c.oid and a.attnum=any(i.indkey) where i.indisprimary and c.relname in ('users','audit_log','outbox','jobs','idempotency_keys') and a.attname='id' and format_type(a.atttypid,-1)='uuid'")" 5
}
tc_pg02_22() {  # AC4 — cột audit_log (SRS 5.2), không có updated_at
  cmp_cols audit_log 10 <<'EOF'
id|uuid|NN|uuidv7()
course_id|uuid|NULL|-
actor_id|uuid|NULL|-
entity|text|NN|-
entity_id|text|NN|-
action|text|NN|-
before|jsonb|NULL|-
after|jsonb|NULL|-
trace_id|text|NULL|-
created_at|timestamp with time zone|NN|now()
EOF
  chk "audit_log không có updated_at" "$($PSQL -c "select count(*) from information_schema.columns where table_name='audit_log' and column_name='updated_at'")" 0
}
tc_pg02_23() {  # AC4 — cột outbox (SRS 5.3)
  cmp_cols outbox 11 <<'EOF'
id|uuid|NN|uuidv7()
topic|text|NN|-
payload|jsonb|NN|'{}'::jsonb
created_at|timestamp with time zone|NN|now()
next_attempt_at|timestamp with time zone|NN|now()
enqueued_at|timestamp with time zone|NULL|-
dispatched_at|timestamp with time zone|NULL|-
attempts|integer|NN|0
last_error|text|NULL|-
dead_at|timestamp with time zone|NULL|-
updated_at|timestamp with time zone|NN|now()
EOF
}
tc_pg02_24() {  # AC4 — cột jobs (SRS 5.4)
  cmp_cols jobs 10 <<'EOF'
id|uuid|NN|uuidv7()
kind|text|NN|-
status|job_status|NN|'QUEUED'::job_status
progress|smallint|NN|0
result|jsonb|NULL|-
error|jsonb|NULL|-
owner_id|uuid|NN|-
created_at|timestamp with time zone|NN|now()
updated_at|timestamp with time zone|NN|now()
finished_at|timestamp with time zone|NULL|-
EOF
}
tc_pg02_25() {  # AC4 — cột idempotency_keys (SRS 5.5)
  cmp_cols idempotency_keys 8 <<'EOF'
id|uuid|NN|uuidv7()
user_id|uuid|NN|-
endpoint|text|NN|-
key|text|NN|-
request_hash|text|NN|-
status_code|smallint|NN|-
response|jsonb|NN|-
created_at|timestamp with time zone|NN|now()
EOF
}
tc_pg02_26() {  # AC4 — idempotency_keys_user_endpoint_key_key là UNIQUE (user_id, endpoint, key)
  chk "idempotency_keys_user_endpoint_key_key UNIQUE" "$($PSQL -c "select i.indisunique from pg_index i join pg_class c on c.oid=i.indexrelid where c.relname='idempotency_keys_user_endpoint_key_key'")" t
  chk_re "khoá đúng ba cột" "$(idef idempotency_keys_user_endpoint_key_key)" '\(user_id, endpoint, key\)'
}
tc_pg02_27() {  # AC4 — outbox CHECK (attempts BETWEEN 0 AND 4): biên hai phía
  sql_err "attempts = -1" 23514 '' "insert into outbox(topic,attempts) values('qc.att','-1'::int)"
  sql_err "attempts = 5"  23514 '' "insert into outbox(topic,attempts) values('qc.att',5)"
  sql_ok  "attempts = 0 (biên dưới)" "insert into outbox(topic,attempts) values('qc.att',0) returning id"
  sql_ok  "attempts = 4 (biên trên)" "insert into outbox(topic,attempts) values('qc.att',4) returning id"
}
tc_pg02_28() {  # AC4 — outbox CHECK NOT (dispatched_at IS NOT NULL AND dead_at IS NOT NULL)
  sql_err "dispatched_at và dead_at cùng có giá trị" 23514 '' "insert into outbox(topic,dispatched_at,dead_at) values('qc.both',now(),now())"
  sql_ok  "chỉ dispatched_at" "insert into outbox(topic,dispatched_at) values('qc.one',now()) returning id"
  sql_ok  "chỉ dead_at"       "insert into outbox(topic,dead_at,attempts) values('qc.one',now(),4) returning id"
}
tc_pg02_29() {  # AC4 — outbox_topic_chk ^[a-z][a-z0-9_.]{0,63}$ : biên 64/65, hoa, chấm đầu, số đầu
  local t64 t65; t64="a$(printf 'b%.0s' $(seq 1 63))"; t65="a$(printf 'b%.0s' $(seq 1 64))"
  chk "độ dài topic hợp lệ" "${#t64}" 64
  sql_ok  "topic 64 ký tự (biên trên hợp lệ)" "insert into outbox(topic) values('$t64') returning id"
  sql_err "topic 65 ký tự" 23514 outbox_topic_chk "insert into outbox(topic) values('$t65')"
  sql_err "topic có chữ hoa 'Test.ok'" 23514 outbox_topic_chk "insert into outbox(topic) values('Test.ok')"
  sql_err "topic bắt đầu bằng dấu chấm '.abc'" 23514 outbox_topic_chk "insert into outbox(topic) values('.abc')"
  sql_err "topic bắt đầu bằng số '1abc'" 23514 outbox_topic_chk "insert into outbox(topic) values('1abc')"
  sql_err "topic rỗng" 23514 outbox_topic_chk "insert into outbox(topic) values('')"
  sql_ok  "topic 'test.ok'" "insert into outbox(topic) values('test.ok') returning id"
}
tc_pg02_30() {  # AC4 — jobs CHECK ((status IN ('SUCCEEDED','FAILED')) = (finished_at IS NOT NULL))
  sql_err "SUCCEEDED mà finished_at NULL" 23514 '' "insert into jobs(kind,owner_id,status) values('qc.j','$U1','SUCCEEDED')"
  sql_err "FAILED mà finished_at NULL"    23514 '' "insert into jobs(kind,owner_id,status) values('qc.j','$U1','FAILED')"
  sql_err "QUEUED mà có finished_at"      23514 '' "insert into jobs(kind,owner_id,status,finished_at) values('qc.j','$U1','QUEUED',now())"
  sql_err "RUNNING mà có finished_at"     23514 '' "insert into jobs(kind,owner_id,status,finished_at) values('qc.j','$U1','RUNNING',now())"
  sql_ok  "SUCCEEDED + finished_at" "insert into jobs(kind,owner_id,status,finished_at) values('qc.j','$U1','SUCCEEDED',now()) returning id"
  sql_ok  "QUEUED + finished_at NULL" "insert into jobs(kind,owner_id) values('qc.j','$U1') returning id"
}
tc_pg02_31() {  # AC4 — jobs CHECK (progress BETWEEN 0 AND 100): biên hai phía
  sql_err "progress = -1"  23514 '' "insert into jobs(kind,owner_id,progress) values('qc.p','$U1','-1'::int)"
  sql_err "progress = 101" 23514 '' "insert into jobs(kind,owner_id,progress) values('qc.p','$U1',101)"
  sql_ok  "progress = 0"   "insert into jobs(kind,owner_id,progress) values('qc.p','$U1',0) returning id"
  sql_ok  "progress = 100 (SUCCEEDED)" "insert into jobs(kind,owner_id,progress,status,finished_at) values('qc.p','$U1',100,'SUCCEEDED',now()) returning id"
}
tc_pg02_32() {  # AC4 — trigger set_updated_at: UPDATE đổi updated_at (users, outbox, jobs)
  chk "trigger users_set_updated_at tồn tại" "$($PSQL -c "select count(*) from pg_trigger where tgname='users_set_updated_at' and not tgisinternal")" 1
  chk "hàm set_updated_at tồn tại" "$($PSQL -c "select count(*) from pg_proc p join pg_namespace n on n.oid=p.pronamespace where n.nspname='public' and p.proname='set_updated_at'")" 1
  local o
  o=$(qv "BEGIN; insert into users(email,full_name,role) values('qc-upd@x.com','QC','STUDENT'); update users set full_name='QC2' where email='qc-upd@x.com'; select (updated_at > created_at) from users where email='qc-upd@x.com'; ROLLBACK;")
  chk_re "users.updated_at tăng sau UPDATE" "$o" '^t$'
  o=$(qv "BEGIN; insert into outbox(topic) values('qc.upd'); update outbox set attempts=1 where topic='qc.upd'; select (updated_at > created_at) from outbox where topic='qc.upd'; ROLLBACK;")
  chk_re "outbox.updated_at tăng sau UPDATE" "$o" '^t$'
  o=$(qv "BEGIN; insert into jobs(kind,owner_id) values('qc.upd','$U1'); update jobs set progress=5 where kind='qc.upd'; select (updated_at > created_at) from jobs where kind='qc.upd'; ROLLBACK;")
  chk_re "jobs.updated_at tăng sau UPDATE" "$o" '^t$'
}

# ================= AC5 — audit_log chỉ-thêm =================
tc_pg02_33() {  # AC5 — INSERT vẫn được (dòng này KHÔNG xoá được: entity='qc-test')
  local o n0 n1
  n0=$($PSQL -c "select count(*) from audit_log where entity='qc-test'")
  o=$($PSQL -v VERBOSITY=verbose -c "insert into audit_log(entity,entity_id,action) values('qc-test','1','x')" 2>&1)
  chk_nre "INSERT audit_log không lỗi" "$o" 'ERROR'
  n1=$($PSQL -c "select count(*) from audit_log where entity='qc-test'")
  chk "số dòng entity='qc-test' tăng 1" "$n1" "$((n0+1))"
}
tc_pg02_34() {  # AC5 — UPDATE → 42501 'audit_log is append-only'
  $PSQL -c "insert into audit_log(entity,entity_id,action) values('qc-test','1','x')" >/dev/null 2>&1
  local o; o=$(qv "BEGIN; update audit_log set action='y' where entity='qc-test'; ROLLBACK;")
  chk_re "UPDATE → SQLSTATE 42501" "$o" '42501:'
  chk_re "UPDATE → thông báo" "$o" 'audit_log is append-only'
}
tc_pg02_35() {  # AC5 — DELETE → 42501
  $PSQL -c "insert into audit_log(entity,entity_id,action) values('qc-test','1','x')" >/dev/null 2>&1
  local o; o=$(qv "BEGIN; delete from audit_log where entity='qc-test'; ROLLBACK;")
  chk_re "DELETE → SQLSTATE 42501" "$o" '42501:'
  chk_re "DELETE → thông báo" "$o" 'audit_log is append-only'
}
tc_pg02_36() {  # AC5 — TRUNCATE → 42501
  local o; o=$(qv "BEGIN; truncate audit_log; ROLLBACK;")
  chk_re "TRUNCATE → SQLSTATE 42501" "$o" '42501:'
  chk_re "TRUNCATE → thông báo" "$o" 'audit_log is append-only'
  chk_ge "audit_log vẫn còn dòng sau khi TRUNCATE bị chặn" "$($PSQL -c "select count(*) from audit_log")" 1
}
tc_pg02_37() {  # AC5 — định nghĩa hai trigger (SRS 5.2)
  local u t
  u=$($PSQL -c "select pg_get_triggerdef(oid) from pg_trigger where tgname='audit_log_no_update'")
  t=$($PSQL -c "select pg_get_triggerdef(oid) from pg_trigger where tgname='audit_log_no_truncate'")
  chk_re "audit_log_no_update BEFORE UPDATE OR DELETE" "$u" 'BEFORE (UPDATE OR DELETE|DELETE OR UPDATE) ON public.audit_log'   # QC v2: pg_get_triggerdef in sự kiện theo thứ tự DELETE OR UPDATE
  chk_re "audit_log_no_update FOR EACH ROW" "$u" 'FOR EACH ROW'
  chk_re "audit_log_no_update gọi audit_log_block_mutation" "$u" 'audit_log_block_mutation\(\)'
  chk_re "audit_log_no_truncate BEFORE TRUNCATE" "$t" 'BEFORE TRUNCATE ON public.audit_log'
  chk_re "audit_log_no_truncate FOR EACH STATEMENT" "$t" 'FOR EACH STATEMENT'
}
tc_pg02_38() { gt ./internal/store 'TestSchema_AuditAppendOnly'; }   # AC5 — test Go của AC

# ================= AC6 — goose hai chiều =================
tc_pg02_39() {  # AC6 — pg_dump trước / sau (down && up) giống hệt
  # pg_dump so chính nó: bảng thử `_test_items` (nếu có) không đổi giữa hai lần dump nên không cần DB sạch.
  local a=$QC_OUT/tc-pg02-39-a.sql b=$QC_OUT/tc-pg02-39-b.sql rcd rcu
  $C exec -T postgres pg_dump -U edupilot -d edupilot -s > "$a" 2>/dev/null
  mig down; rcd=$?
  mig up;   rcu=$?
  $C exec -T postgres pg_dump -U edupilot -d edupilot -s > "$b" 2>/dev/null
  chk "migrate down rc" "$rcd" 0
  chk "migrate up rc" "$rcu" 0
  # QC v2: pg_dump (≥ 17.6) chèn dòng \restrict/\unrestrict kèm token NGẪU NHIÊN mỗi lần dump — không phải lược đồ; bỏ hai dòng này trước khi so
  if diff <(grep -vE '^\\(un)?restrict ' "$a") <(grep -vE '^\\(un)?restrict ' "$b") > "$QC_OUT/tc-pg02-39.diff" 2>&1; then echo "    ok   lược đồ sau down+up giống hệt (diff rỗng)"
  else fail_tc "lược đồ đổi sau down+up — xem $QC_OUT/tc-pg02-39.diff"; head -20 "$QC_OUT/tc-pg02-39.diff" | sed 's/^/      /'; fi
}
tc_pg02_40() {  # AC6 — sau `down`: 5 bảng biến mất, extension vector GIỮ LẠI
  local rcd n v rcu
  mig down; rcd=$?
  n=$($PSQL -c "select count(*) from pg_tables where schemaname='public' and tablename in ('users','audit_log','outbox','jobs','idempotency_keys')")
  v=$($PSQL -c "select extname from pg_extension where extname='vector'")
  mig up; rcu=$?                      # khôi phục NGAY, trước mọi chk
  wait_ready 90
  chk "migrate down rc" "$rcd" 0
  chk "số bảng của 00001 còn lại sau down" "$n" 0
  chk "extension vector còn sau down" "$v" vector
  chk "migrate up rc (khôi phục)" "$rcu" 0
  chk "5 bảng trở lại sau up" "$($PSQL -c "select count(*) from pg_tables where schemaname='public' and tablename in ('users','audit_log','outbox','jobs','idempotency_keys')")" 5
}
tc_pg02_41() {  # AC6 — `migrate up` lần hai là no-op, mã 0
  local n0 n1 rc
  n0=$($PSQL -c "select count(*) from goose_db_version")
  mig up; rc=$?
  n1=$($PSQL -c "select count(*) from goose_db_version")
  chk "migrate up lần hai rc" "$rc" 0
  chk "goose_db_version không thêm dòng" "$n1" "$n0"
  chk "vẫn đúng 5 bảng + goose_db_version" "$($PSQL -c "select count(*) from pg_tables where schemaname='public' and tablename not like '\_test\_%'")" 6
}
tc_pg02_42() {  # AC6 — version_id mới nhất = 1
  chk "goose_db_version.version_id mới nhất" "$($PSQL -c "select version_id from goose_db_version where is_applied order by id desc limit 1")" 1
}
tc_pg02_43() { gt ./db 'TestMigrations_RoundTrip'; }   # AC6 — test Go của AC

# ================= AC7 — compose: migrate chạy một lần trước gateway/worker =================
tc_pg02_44() {  # AC7 — volume trống + `pnpm dev`: migrate exited 0, không thao tác tay
  local t0 t1 rc
  pnpm dev:down >/dev/null 2>&1
  docker volume rm edupilot_postgres_data >/dev/null 2>&1
  t0=$(now_ms); pnpm dev > "$QC_OUT/tc-pg02-44-pnpm-dev.log" 2>&1; rc=$?; t1=$(now_ms)
  echo "    thời gian 'pnpm dev' từ volume trống: $(( (t1-t0)/1000 )) s (chứng cứ, không phải ngưỡng AC)"
  chk "pnpm dev rc" "$rc" 0
  chk "trạng thái service migrate" "$($C ps -a --format '{{.Service}} {{.State}} {{.ExitCode}}' | grep '^migrate ' | head -1)" "migrate exited 0"
  chk "số lần migrate chạy (restart: no)" "$($C ps -aq migrate | grep -c .)" 1
  wait_ready 120
  chk_ge "dòng goose_db_version is_applied" "$($PSQL -c "select count(*) from goose_db_version where is_applied")" 2
  chk "gateway healthy" "$(code $GW/api/v1/readyz)" 200
}
tc_pg02_45() {  # AC7 — thứ tự: gateway/worker khởi động SAU khi migrate xong (chạy ngay sau TC-44)
  local mid fin id st n=0
  mid=$($C ps -aq migrate | head -1)
  [ -n "$mid" ] || { fail_tc "không tìm thấy container migrate"; return 0; }
  fin=$(docker inspect -f '{{.State.FinishedAt}}' "$mid")
  chk_re "migrate có FinishedAt" "$fin" '^[0-9]{4}-'
  for id in $($C ps -q gateway) $($C ps -q worker); do
    st=$(docker inspect -f '{{.State.StartedAt}}' "$id")
    chk "StartedAt($(docker inspect -f '{{.Name}}' "$id")) > migrate.FinishedAt" "$($PSQL -c "select '$st'::timestamptz > '$fin'::timestamptz")" t
    n=$((n+1))
  done
  chk_ge "số container gateway+worker đã kiểm" "$n" 2
  chk_re "nhãn depends_on của gateway" "$(docker inspect -f '{{index .Config.Labels "com.docker.compose.depends_on"}}' "$($C ps -q gateway | head -1)")" 'migrate:service_completed_successfully'
}
tc_pg02_46() {  # AC7 — migrate nối thẳng Postgres, không qua PgBouncer
  local mid e g
  mid=$($C ps -aq migrate | head -1)
  e=$(docker inspect -f '{{range .Config.Env}}{{println .}}{{end}}' "$mid" | grep '^DATABASE_URL=')
  chk_re "migrate DATABASE_URL trỏ thẳng postgres" "$e" '@postgres(:5432)?/'
  chk_nre "migrate không trỏ pgbouncer" "$e" 'pgbouncer'
  chk "migrate không có PGBOUNCER_URL có giá trị" "$(docker inspect -f '{{range .Config.Env}}{{println .}}{{end}}' "$mid" | grep -c '^PGBOUNCER_URL=..*')" 0
  g=$(docker inspect -f '{{range .Config.Env}}{{println .}}{{end}}' "$($C ps -q gateway | head -1)")
  chk "gateway runtime có PGBOUNCER_URL" "$(printf '%s\n' "$g" | grep -c '^PGBOUNCER_URL=..*')" 1
}

# ================= AC8 / AC9 — sqlc =================
tc_pg02_47() {  # AC8 — sqlc diff thoát 0, không in gì
  command -v sqlc >/dev/null 2>&1 || { fail_tc "KHÔNG KIỂM ĐƯỢC: chưa cài sqlc trên máy QC (AC8 cần 'sqlc diff'); không tự cài theo phỏng đoán"; return 0; }
  local o rc; o=$( (cd backend-go && sqlc diff) 2>&1 ); rc=$?
  printf '%s\n' "$o" > "$QC_OUT/tc-pg02-47-sqlc-diff.log"
  chk "sqlc diff rc" "$rc" 0
  chk "sqlc diff không in gì (số dòng)" "$(printf '%s' "$o" | grep -c .)" 0
}
tc_pg02_48() {  # AC8 — enum Postgres thành kiểu Go, không float64
  chk "3 kiểu enum Go trong internal/store/models.go" "$(grep -cE '^type (UserRole|UserStatus|JobStatus) string' backend-go/internal/store/models.go 2>/dev/null)" 3
  chk "không float64 trong mã sản xuất internal/store (spec v1.5, #12: _test.go miễn)" "$(grep -rn 'float64' backend-go/internal/store --include=*.go 2>/dev/null | grep -v '_test.go' | grep -c .)" 0
  chk_ge "timestamptz ánh xạ time.Time" "$(grep -c 'time\.Time' backend-go/internal/store/models.go 2>/dev/null)" 1
}
tc_pg02_49() {  # AC8 — không OFFSET trong queries, mọi file truy vấn có '-- name:'
  chk "OFFSET trong internal/store/queries" "$(grep -rniE '\boffset\b' backend-go/internal/store/queries 2>/dev/null | grep -c .)" 0
  local f n bad=0 tot=0
  for f in backend-go/internal/store/queries/*.sql; do
    [ -e "$f" ] || continue
    tot=$((tot+1)); n=$(grep -c -- '-- name:' "$f")
    [ "$n" -ge 1 ] || { bad=$((bad+1)); echo "    thiếu '-- name:' trong $f"; }
  done
  chk_ge "số file queries/*.sql" "$tot" 1
  chk "file queries thiếu '-- name:'" "$bad" 0
}
tc_pg02_50() {  # AC9 — sửa queries mà không generate → sqlc diff khác 0 và nêu tên file (làm trên BẢN SAO)
  command -v sqlc >/dev/null 2>&1 || { fail_tc "KHÔNG KIỂM ĐƯỢC: chưa cài sqlc trên máy QC (AC9)"; return 0; }
  local bg=$QC_TMP/bg q o rc
  rm -rf "$bg"; cp -R backend-go "$bg" || { fail_tc "không sao chép được backend-go"; return 0; }
  q=$bg/internal/store/queries/users.sql
  [ -f "$q" ] || { fail_tc "KHÔNG KIỂM ĐƯỢC: không có internal/store/queries/users.sql"; return 0; }
  printf '\n-- name: QcTmpDrift :one\nselect 1;\n' >> "$q"
  o=$( (cd "$bg" && sqlc diff) 2>&1 ); rc=$?
  printf '%s\n' "$o" > "$QC_OUT/tc-pg02-50-sqlc-diff.log"
  chk_ne "sqlc diff rc (phải khác 0)" "$rc" 0
  chk_re "diff nêu tên file sinh ra lệch" "$o" '\.go'
  chk "repo thật không bị sửa" "$(git status --porcelain backend-go | grep -c .)" 0
  rm -rf "$bg"
}

# ================= AC10 — quy ước vector =================
tc_pg02_51() { gt ./internal/store 'TestVectorConventions'; }                 # AC10 — lệnh 1 của AC
tc_pg02_52() { gt ./internal/platform/db 'TestVector_ThroughPgBouncer'; }     # AC10 — lệnh 2 (phải chạy RIÊNG: hai -run trong một lệnh thì cờ sau đè cờ trước)
tc_pg02_53() {  # AC10 — migration 00001 có khối CHÚ THÍCH mẫu HNSW/halfvec
  [ -f "$MIG" ] || { fail_tc "KHÔNG KIỂM ĐƯỢC: không có $MIG"; return 0; }
  chk_ge "số dòng chứa halfvec_cosine_ops" "$(grep -c 'halfvec_cosine_ops' "$MIG")" 1
  chk "mọi dòng halfvec_cosine_ops đều là chú thích SQL (bắt đầu bằng --)" \
      "$(grep 'halfvec_cosine_ops' "$MIG" | grep -cv '^[[:space:]]*--')" 0
  chk_ge "tham số m = 16 trong khối chú thích" "$(grep -E '^[[:space:]]*--' "$MIG" | grep -c 'm *= *16')" 1
  chk_ge "tham số ef_construction = 64 trong khối chú thích" "$(grep -E '^[[:space:]]*--' "$MIG" | grep -c 'ef_construction *= *64')" 1
  chk_ge "mẫu truy vấn <=> halfvec(1536) trong khối chú thích" "$(grep -E '^[[:space:]]*--' "$MIG" | grep '<=>' | grep -c 'halfvec(1536)')" 1
}

# ================= AC11 / AC12 — blob =================
tc_pg02_54() { gt ./internal/platform/blob 'TestBlob_RoundTrip|TestBlob_Presign|TestBlob_InvalidKey|TestBlob_Cancel'; }
tc_pg02_55() {  # AC11 (hộp đen bổ sung) — MinIO thật của compose: bucket tồn tại, cổng 9000 dùng được
  local b c; b=$(envv BLOB_BUCKET)
  chk_re "BLOB_BUCKET trong .env.local" "$b" '^[a-z0-9][a-z0-9.-]{2,62}$'
  chk "MinIO /minio/health/live" "$(code http://localhost:9000/minio/health/live)" 200
  c=$(code "http://localhost:9000/$b/")
  chk_ne "GET ẩn danh bucket '$b' không phải 404 NoSuchBucket (bucket đã được tạo)" "$c" 404
  chk_re "GET ẩn danh bucket trả 403/200 (bucket có thật)" "$c" '^(403|200)$'
}
tc_pg02_56() { gt ./internal/platform/blob 'TestPresign_PublicHost|TestPresign_NoNetwork'; }
tc_pg02_57() {  # AC12 (hộp đen bổ sung) — gateway thấy minio:9000, ký URL bằng localhost:9000
  local e; e=$(docker inspect -f '{{range .Config.Env}}{{println .}}{{end}}' "$($C ps -q gateway | head -1)")
  chk "BLOB_ENDPOINT của gateway" "$(printf '%s\n' "$e" | grep '^BLOB_ENDPOINT=' | cut -d= -f2-)" 'minio:9000'
  chk "BLOB_PUBLIC_ENDPOINT của gateway" "$(printf '%s\n' "$e" | grep '^BLOB_PUBLIC_ENDPOINT=' | cut -d= -f2-)" 'localhost:9000'
  chk "host localhost:9000 dùng được từ máy (trình duyệt)" "$(code http://localhost:9000/minio/health/live)" 200
}

# ================= AC13 — outbox ghi cùng transaction =================
tc_pg02_58() { gt ./internal/platform/outbox 'TestOutbox_SameTransaction_Commit|TestOutbox_SameTransaction_Rollback'; }
tc_pg02_59() {  # AC13 (hộp đen) — rollback: 0 dòng cả hai, không tin nào vào Stream
  $PSQL -c "BEGIN; insert into jobs(kind,owner_id) values('qc.rollback','$U1'); insert into outbox(topic,payload) values('qc.rollback.test','{}'::jsonb); ROLLBACK;" >/dev/null 2>&1
  sleep 3   # relay quét mỗi OUTBOX_POLL_INTERVAL = 500 ms
  chk "dòng outbox topic qc.rollback.test" "$($PSQL -c "select count(*) from outbox where topic='qc.rollback.test'")" 0
  chk "dòng jobs kind qc.rollback" "$($PSQL -c "select count(*) from jobs where kind='qc.rollback'")" 0
  $RDS xrevrange outbox.dispatch + - COUNT 50 > "$QC_OUT/tc-pg02-59.stream" 2>/dev/null
  chk "tin qc.rollback.test trong outbox.dispatch" "$(grep -c 'qc.rollback.test' "$QC_OUT/tc-pg02-59.stream")" 0
  chk "tin qc.rollback.test trong dead-letter" "$($RDS xrevrange outbox.dispatch.dead + - COUNT 50 2>/dev/null | grep -c 'qc.rollback.test')" 0
}
tc_pg02_60() {  # AC13 (hộp đen) — commit: jobs + outbox job.enqueue cùng MỘT transaction (so xmin)
  ensure_mode test
  local id c
  id=$(curl -sk --max-time 20 -X POST -H "$H" -H 'Content-Type: application/json' -H "Idempotency-Key: qc-pg02-60-$$" \
        -d '{"steps":1,"kind":"test.progress"}' "$GW/api/v1/_test/jobs" | jq -r '.job_id // .id // empty')
  chk_re "POST /_test/jobs trả job id" "$id" '^[0-9a-f-]{36}$'
  [ -n "$id" ] || return 0
  chk "số dòng jobs" "$($PSQL -c "select count(*) from jobs where id='$id'")" 1
  c=$($PSQL -c "select count(*) from outbox o join jobs j on o.xmin::text=j.xmin::text where j.id='$id' and o.topic='job.enqueue'")
  chk_ge "dòng outbox job.enqueue cùng xmin với dòng jobs (cùng transaction)" "$c" 1
}

# ================= AC14 — outbox chạy thường, hai worker =================
tc_pg02_61() { gt ./internal/platform/outbox 'TestOutbox_TwoWorkers_ExactlyOnce'; }
tc_pg02_62() {  # AC14 (hộp đen, ~1–2 phút) — 100 job, hai worker: mọi dòng dispatched, không trùng, không dead
  ensure_mode test
  local t0 dead0 dead1 i n202 tot undisp maxatt succ done100 pend xdisp w el s0
  $CT up -d --scale worker=2 --wait worker >/dev/null 2>&1
  w=$($C ps -q worker | grep -c .)
  dead0=$(xlen outbox.dispatch.dead)
  t0=$($PSQL -c "select now()")
  : > "$QC_OUT/tc-pg02-62.codes"
  i=1; while [ $i -le 100 ]; do
    { newjob test.progress 1 "62-$$-$i"; echo; } >> "$QC_OUT/tc-pg02-62.codes"
    i=$((i+1)); done
  n202=$(grep -c '^202$' "$QC_OUT/tc-pg02-62.codes")
  s0=$(now_ms); i=0
  while [ $i -lt 60 ]; do
    undisp=$($PSQL -c "select count(*) from outbox where topic='job.enqueue' and created_at > '$t0' and dispatched_at is null")
    [ "$undisp" = 0 ] && break
    sleep 1; i=$((i+1)); done
  el=$(( ($(now_ms)-s0)/1000 ))
  tot=$($PSQL -c "select count(*) from outbox where topic='job.enqueue' and created_at > '$t0'")
  maxatt=$($PSQL -c "select coalesce(max(attempts),0) from outbox where created_at > '$t0'")
  succ=$($PSQL -c "select count(*) from jobs where created_at > '$t0' and kind='test.progress' and status='SUCCEEDED'")
  done100=$($PSQL -c "select count(*) from jobs where created_at > '$t0' and kind='test.progress' and progress=100")
  sleep 10                                   # lặng 10 s: consumer XACK rồi XDEL (US.md 02-AC15 "Làm rõ" #Q-QC-02-6)
  pend=$(xpend outbox.dispatch outbox); dead1=$(xlen outbox.dispatch.dead); xdisp=$(xlen outbox.dispatch)
  $CT up -d --scale worker=1 --wait worker >/dev/null 2>&1   # khôi phục TRƯỚC khi chấm
  wait_ready 90
  chk "số worker lúc chạy" "$w" 2
  chk "job nhận 202" "$n202" 100
  chk "dòng outbox job.enqueue mới" "$tot" 100
  echo "    thời gian tới khi dispatched hết: ${el}s"
  chk "dòng outbox chưa dispatched" "$undisp" 0
  chk_le "thời gian dispatched hết (giây; ngưỡng stack compose thật ≤ 30 s — US.md 02-AC14 \"Làm rõ\" #Q-QC-02-2, 10 s chỉ cho test Go)" "$el" 30
  chk "max(attempts) (0 = handler chạy đúng 1 lần mỗi dòng)" "$maxatt" 0
  chk "job SUCCEEDED" "$succ" 100
  chk "job progress=100" "$done100" 100
  chk "XPENDING outbox.dispatch outbox" "$pend" 0
  chk "XLEN outbox.dispatch sau 10 s lặng (đã XACK + XDEL)" "$xdisp" 0
  chk "XLEN outbox.dispatch.dead không đổi" "$dead1" "$dead0"
}

# ================= AC15 — retry rồi dead-letter =================
tc_pg02_63() { gt ./internal/platform/outbox 'TestOutbox_RetryThenDead|TestOutbox_PanicIsFailure|TestOutbox_UnknownTopicDead'; }
tc_pg02_64() {  # AC15 (hộp đen, ~60–90 s) — topic chưa đăng ký: 4 lần gọi rồi dead-letter (SRS 3.4)
  local id dead0 dead1 pend xdisp
  dead0=$(xlen outbox.dispatch.dead)
  id=$(ac15_dead_row)
  chk_re "tạo được dòng outbox topic lạ" "$id" '^[0-9a-f-]{36}$'
  [ -n "$id" ] || return 0
  chk "attempts (1 lần đầu + 3 lần thử lại)" "$($PSQL -c "select attempts from outbox where id='$id'")" 4
  chk "dead_at có giá trị" "$($PSQL -c "select (dead_at is not null) from outbox where id='$id'")" t
  chk "dispatched_at vẫn NULL" "$($PSQL -c "select (dispatched_at is null) from outbox where id='$id'")" t
  chk "last_error khác rỗng" "$($PSQL -c "select (coalesce(last_error,'') <> '') from outbox where id='$id'")" t
  dead1=$(xlen outbox.dispatch.dead)
  chk "XLEN outbox.dispatch.dead tăng 1" "$dead1" "$((dead0+1))"
  $RDS xrevrange outbox.dispatch.dead + - COUNT 3 > "$QC_OUT/tc-pg02-64.dead" 2>/dev/null
  chk_ge "tin dead-letter chứa id của dòng outbox" "$(grep -c "$id" "$QC_OUT/tc-pg02-64.dead")" 1
  chk_ge "tin dead-letter có trường outbox_id" "$(grep -c 'outbox_id' "$QC_OUT/tc-pg02-64.dead")" 1
  chk_ge "tin dead-letter có trường topic" "$(grep -c 'topic' "$QC_OUT/tc-pg02-64.dead")" 1
  chk_ge "tin dead-letter có trường error" "$(grep -c 'error' "$QC_OUT/tc-pg02-64.dead")" 1
  chk_ge "tin dead-letter nêu topic qc.unknown.topic" "$(grep -c 'qc.unknown.topic' "$QC_OUT/tc-pg02-64.dead")" 1
  sleep 10                                   # lặng 10 s rồi mới đo stream (US.md 02-AC15 "Làm rõ" #Q-QC-02-6)
  pend=$(xpend outbox.dispatch outbox); xdisp=$(xlen outbox.dispatch)
  chk "XPENDING outbox.dispatch outbox (tin gốc đã XACK)" "$pend" 0
  chk "XLEN outbox.dispatch sau 10 s lặng (XACK rồi XDEL)" "$xdisp" 0
  chk_re "worker vẫn sống sau topic lạ" "$($C ps --format '{{.Service}} {{.Health}}' | grep '^worker')" 'healthy'
}
tc_pg02_65() {  # AC15 / SRS 5.3 — last_error ≤ 1000 ký tự và KHÔNG chứa PII của payload
  local id r
  id=$(ac15_dead_row)
  [ -n "$id" ] || { fail_tc "KHÔNG KIỂM ĐƯỢC: không tạo được dòng dead-letter"; return 0; }
  r=$($PSQL -c "select length(coalesce(last_error,''))||'|'||(coalesce(last_error,'') like '%qc-pii-canary%')::text from outbox where id='$id'")
  chk_re "last_error có nội dung" "$r" '^[1-9][0-9]*\|'
  chk_le "độ dài last_error" "$(printf '%s' "$r" | cut -d'|' -f1)" 1000
  chk "last_error chứa PII của payload (qc-pii-canary)" "$(printf '%s' "$r" | cut -d'|' -f2)" false
}

# ================= AC16 — consumer chết giữa chừng =================
tc_pg02_66() { gt ./internal/platform/outbox 'TestOutbox_ConsumerCrash_Reclaimed|TestOutbox_StaleEnqueuedRequeued'; }
tc_pg02_67() {  # AC16 (hộp đen, ~3 phút) — giết worker giữa tải: không tin nào mất
  ensure_mode test
  local t0 dead0 dead1 i n202 tot undisp pend wid el s0
  dead0=$(xlen outbox.dispatch.dead)
  t0=$($PSQL -c "select now()")
  : > "$QC_OUT/tc-pg02-67.codes"
  i=1; while [ $i -le 30 ]; do
    { newjob test.progress 3 "67-$$-$i"; echo; } >> "$QC_OUT/tc-pg02-67.codes"
    i=$((i+1)); done
  n202=$(grep -c '^202$' "$QC_OUT/tc-pg02-67.codes")
  wid=$($C ps -q worker | head -1)
  docker kill "$wid" >/dev/null 2>&1                       # giết KHÔNG kịp XACK
  sleep 2
  $CT up -d --wait worker >/dev/null 2>&1                  # khôi phục NGAY, trước mọi chk
  s0=$(now_ms); i=0
  while [ $i -lt 150 ]; do
    undisp=$($PSQL -c "select count(*) from outbox where topic='job.enqueue' and created_at > '$t0' and dispatched_at is null")
    [ "$undisp" = 0 ] && break
    sleep 1; i=$((i+1)); done
  el=$(( ($(now_ms)-s0)/1000 ))
  tot=$($PSQL -c "select count(*) from outbox where topic='job.enqueue' and created_at > '$t0'")
  pend=$(xpend outbox.dispatch outbox); dead1=$(xlen outbox.dispatch.dead)
  $PSQL -c "select status||' '||count(*) from jobs where created_at > '$t0' group by 1" > "$QC_OUT/tc-pg02-67.jobs" 2>/dev/null
  chk "job nhận 202" "$n202" 30
  chk "dòng outbox job.enqueue mới" "$tot" 30
  echo "    thời gian hồi phục sau khi giết worker: ${el}s (OUTBOX_CLAIM_IDLE mặc định 60 s)"
  chk "dòng outbox chưa dispatched (không mất tin)" "$undisp" 0
  chk_le "thời gian hồi phục (giây)" "$el" 150
  chk "XPENDING outbox.dispatch outbox" "$pend" 0
  chk "XLEN outbox.dispatch.dead không đổi" "$dead1" "$dead0"
  echo "    chứng cứ trạng thái job sau sự cố: $(tr '\n' ' ' < "$QC_OUT/tc-pg02-67.jobs")"
  chk "worker còn đúng 1 bản sau khôi phục" "$($C ps -q worker | grep -c .)" 1
}

# ================= AC17 — không áp dụng: kiểm ràng buộc an toàn thay thế =================
tc_pg02_68() {  # AC17 — mật khẩu chỉ ở dạng bcrypt (US-PG-04 AC8) trên DB sau toàn bộ test
  local n t
  # chuỗi SQL để trong nháy đơn + dollar-quoting của Postgres: regex tới psql đúng là ^\$2[aby]\$ (dấu $ literal)
  n=$($PSQL -c 'select count(*) from users where password_hash is not null and password_hash !~ $re$^\$2[aby]\$$re$')
  t=$($PSQL -c "select count(*) from users")
  echo "    tổng số dòng users hiện có: $t (chứng cứ cho sức nặng của phép đo)"
  chk "dòng users có password_hash không phải bcrypt" "$n" 0
}
tc_pg02_69() {  # AC17 — idempotency_keys khoá theo user_id (US-PG-03 AC12)
  local ok
  ok="insert into idempotency_keys(user_id,endpoint,key,request_hash,status_code,response) values('$U1','POST /api/v1/_test/items','$IDEM','h',201,'{}'::jsonb); insert into idempotency_keys(user_id,endpoint,key,request_hash,status_code,response) values('$U2','POST /api/v1/_test/items','$IDEM','h',201,'{}'::jsonb)"
  local o; o=$(qv "BEGIN; $ok; select count(*) from idempotency_keys where key='$IDEM'; ROLLBACK;")
  chk_nre "hai người dùng khác nhau dùng cùng khoá: hợp lệ" "$o" 'ERROR'
  chk_re "chèn được 2 dòng" "$o" '^2$'
  sql_err "cùng user + cùng endpoint + cùng key" 23505 idempotency_keys_user_endpoint_key_key \
    "insert into idempotency_keys(user_id,endpoint,key,request_hash,status_code,response) values('$U1','POST /api/v1/_test/items','$IDEM','h',201,'{}'::jsonb); insert into idempotency_keys(user_id,endpoint,key,request_hash,status_code,response) values('$U1','POST /api/v1/_test/items','$IDEM','h2',200,'{}'::jsonb)"
}

# ================= AC1/AC4 ở chế độ test — phép đếm loại `\_test\_%` vẫn đúng (US.md "Làm rõ" #Q-QC-02-3) =================
tc_pg02_70() {  # chế độ test (có `_test_items`): 5 bảng nền + goose_db_version, 17 index, 5 cột id uuidv7()
  ensure_mode test
  local ntest
  ntest=$($PSQL -c "select count(*) from pg_tables where schemaname='public' and tablename like '\_test\_%'")
  chk_ge "bảng thử _test_% đang tồn tại (điều kiện để phép đo có nghĩa)" "$ntest" 1
  chk "bảng public sau khi loại _test_%" \
      "$($PSQL -c "select tablename from pg_tables where schemaname='public' and tablename not like '\_test\_%' order by 1" | tr '\n' ' ' | sed 's/ $//')" \
      "audit_log goose_db_version idempotency_keys jobs outbox users"
  chk "số index sau khi loại _test_%" \
      "$($PSQL -c "select count(*) from pg_indexes where schemaname='public' and tablename<>'goose_db_version' and tablename not like '\_test\_%'")" 17
  chk "cột id mặc định uuidv7() sau khi loại _test_%" \
      "$($PSQL -c "select count(*) from information_schema.columns where table_schema='public' and table_name not like '\_test\_%' and column_name='id' and column_default='uuidv7()'")" 5
}

main 02 "$@"
