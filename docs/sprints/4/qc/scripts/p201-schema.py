#!/usr/bin/env python3
"""QC P2-01: so lược đồ 00004 với SRS 5.1–5.5 (chép tay) + gieo INSERT sai/lành. PSQL=docker exec <container> psql ..."""
import subprocess, sys, os
C = os.environ.get("PGC", "qcp2-pg")
def q(sql):
    r = subprocess.run(["docker","exec","-i",C,"psql","-U","edupilot","-d","edupilot","-tA","-F","|","-v","ON_ERROR_STOP=0","-c",sql],capture_output=True,text=True)
    return (r.stdout.strip(), r.stderr.strip())
fails = []
def chk(name, ok, detail=""):
    print(("PASS " if ok else "FAIL ")+name+((" · "+detail) if detail and not ok else ""))
    if not ok: fails.append(name)

# (cột, kiểu, null, mặc định chứa)
EXP = {
 "auth_sessions": [("id","uuid","NO","uuidv7"),("user_id","uuid","NO",""),("refresh_hash","character","NO",""),("prev_refresh_hash","character","YES",""),("user_agent","text","YES",""),("device_label","text","YES",""),("ip","inet","YES",""),("created_at","timestamp with time zone","NO","now()"),("rotated_at","timestamp with time zone","YES",""),("last_used_at","timestamp with time zone","NO","now()"),("expires_at","timestamp with time zone","NO",""),("absolute_expires_at","timestamp with time zone","NO",""),("revoked_at","timestamp with time zone","YES",""),("revoked_reason","text","YES",""),("updated_at","timestamp with time zone","NO","now()")],
 "auth_tokens": [("id","uuid","NO","uuidv7"),("user_id","uuid","NO",""),("kind","USER-DEFINED","NO",""),("token_hash","character","NO",""),("expires_at","timestamp with time zone","NO",""),("used_at","timestamp with time zone","YES",""),("revoked_at","timestamp with time zone","YES",""),("created_by","uuid","YES",""),("created_at","timestamp with time zone","NO","now()")],
 "login_attempts": [("id","uuid","NO","uuidv7"),("email_hash","character","NO",""),("user_id","uuid","YES",""),("ip","inet","YES",""),("user_agent","text","YES",""),("outcome","USER-DEFINED","NO",""),("created_at","timestamp with time zone","NO","now()")],
 "mail_outbox": [("id","uuid","NO","uuidv7"),("to_addr","text","NO",""),("template","text","NO",""),("payload","jsonb","NO","{}"),("status","USER-DEFINED","NO","QUEUED"),("attempts","integer","NO","0"),("last_error","text","YES",""),("dedupe_key","text","YES",""),("sent_at","timestamp with time zone","YES",""),("created_at","timestamp with time zone","NO","now()"),("updated_at","timestamp with time zone","NO","now()")],
}
IDX = {"auth_sessions":["auth_sessions_pkey","auth_sessions_refresh_hash_key","auth_sessions_prev_hash_idx","auth_sessions_user_active_idx","auth_sessions_expires_idx"],
       "auth_tokens":["auth_tokens_pkey","auth_tokens_token_hash_key","auth_tokens_user_kind_idx"],
       "login_attempts":["login_attempts_pkey","login_attempts_email_idx","login_attempts_ip_idx","login_attempts_user_idx"],
       "mail_outbox":["mail_outbox_pkey","mail_outbox_dedupe_key_key","mail_outbox_queued_idx"]}
# chuỗi phải xuất hiện trong pg_get_constraintdef (CHECK) của bảng
CHK = {"auth_sessions":["0-9a-f]{64}","absolute_expires_at >= expires_at","LOGOUT","REVOKED_BY_USER","PASSWORD_CHANGED","PASSWORD_RESET","REFRESH_REUSE","ACCOUNT_DISABLED","ROLE_CHANGED","ADMIN","revoked_at IS NULL"],
       "auth_tokens":["0-9a-f]{64}","used_at IS NOT NULL","revoked_at IS NOT NULL"],
       "login_attempts":["0-9a-f]{64}"],
       "mail_outbox":["lower(to_addr)","a-z][a-z0-9_]{0,63}","attempts >= 0","attempts <= 4","1000","SENT"]}
n,_ = q("select count(*) from goose_db_version where is_applied and version_id in (3,4)"); chk("goose có 00003 và 00004", n=="2", n)
v,_ = q("select max(version_id) from goose_db_version where is_applied"); chk("phiên bản = 4", v=="4", v)
for t,cols in EXP.items():
    out,_ = q(f"select column_name||'|'||data_type||'|'||is_nullable||'|'||coalesce(column_default,'') from information_schema.columns where table_name='{t}' and table_schema='public' order by ordinal_position")
    got=[l.split("|") for l in out.splitlines()]
    ok = len(got)==len(cols) and all(g[0]==e[0] and g[1]==e[1] and g[2]==e[2] and e[3] in g[3] for g,e in zip(got,cols))
    chk(f"{t}: {len(cols)} cột đúng tên/kiểu/null/mặc định", ok, f"got {len(got)}: {got}")
    ix,_ = q(f"select indexname from pg_indexes where tablename='{t}'"); have=set(ix.split()); chk(f"{t}: đủ chỉ mục", set(IDX[t])<=have, str(set(IDX[t])-have))
    cd,_ = q(f"select string_agg(pg_get_constraintdef(oid),' ## ') from pg_constraint where conrelid='public.{t}'::regclass and contype='c'")
    miss=[s for s in CHK[t] if s not in cd]; chk(f"{t}: CHECK chứa đủ", not miss, str(miss))
fk,_ = q("select conrelid::regclass||'→'||confrelid::regclass||' '||confdeltype::text from pg_constraint where contype='f' and conrelid in ('auth_sessions'::regclass,'auth_tokens'::regclass,'login_attempts'::regclass,'mail_outbox'::regclass)"); chk("FK: sessions.user_id→users CASCADE, tokens.user_id→users CASCADE, login_attempts không FK", "auth_sessions→users c" in fk and "auth_tokens→users c" in fk and "login_attempts" not in fk, fk)
enums,_ = q("select typname||':'||string_agg(enumlabel,',' order by enumsortorder) from pg_type t join pg_enum e on e.enumtypid=t.oid where typname in ('auth_token_kind','mail_status','login_outcome') group by typname order by 1")
chk("3 enum đúng giá trị", enums=="auth_token_kind:VERIFY_EMAIL,RESET_PASSWORD,INVITE\nlogin_outcome:SUCCESS,BAD_PASSWORD,UNKNOWN_EMAIL,THROTTLED,LOCKED,DISABLED\nmail_status:QUEUED,SENT,DEAD", enums)
trg,_=q("select count(*) from pg_trigger where tgname in ('auth_sessions_set_updated_at','mail_outbox_set_updated_at')"); chk("trigger set_updated_at trên auth_sessions, mail_outbox", trg=="2", trg)
tb,_=q("select count(*) from pg_tables where schemaname='public' and tablename in ('auth_sessions','auth_tokens','login_attempts','mail_outbox')"); chk("đúng 4 bảng mới", tb=="4", tb)

# ---- INSERT sai (TC-06) / lành (TC-07)
q("insert into users(id,email,full_name,role,status) values ('11111111-1111-7111-8111-111111111111','qc-p201@example.test','QC','STUDENT','INVITED') on conflict do nothing")
u,e=q("select id from users limit 1"); UID=u
H=lambda c:c*64
now="now()"
bad = [
 ("tokens: hash 63 ký tự", f"insert into auth_tokens(user_id,kind,token_hash,expires_at) values ('{UID}','INVITE','{'a'*63}',now()+interval '1h')","23514"),
 ("tokens: hash không hex", f"insert into auth_tokens(user_id,kind,token_hash,expires_at) values ('{UID}','INVITE','{'Z'*64}',now()+interval '1h')","23514"),
 ("tokens: kind sai", f"insert into auth_tokens(user_id,kind,token_hash,expires_at) values ('{UID}','X','{H('b')}',now()+interval '1h')","22P02"),
 ("tokens: used+revoked", f"insert into auth_tokens(user_id,kind,token_hash,expires_at,used_at,revoked_at) values ('{UID}','INVITE','{H('c')}',now()+interval '1h',now(),now())","23514"),
 ("sessions: refresh_hash sai độ dài", f"insert into auth_sessions(user_id,refresh_hash,expires_at,absolute_expires_at) values ('{UID}','abc',now()+interval '1h',now()+interval '2h')","23514"),
 ("sessions: revoked_at thiếu reason", f"insert into auth_sessions(user_id,refresh_hash,expires_at,absolute_expires_at,revoked_at) values ('{UID}','{H('d')}',now()+interval '1h',now()+interval '2h',now())","23514"),
 ("sessions: reason thiếu revoked_at", f"insert into auth_sessions(user_id,refresh_hash,expires_at,absolute_expires_at,revoked_reason) values ('{UID}','{H('e')}',now()+interval '1h',now()+interval '2h','LOGOUT')","23514"),
 ("sessions: reason ngoài danh sách", f"insert into auth_sessions(user_id,refresh_hash,expires_at,absolute_expires_at,revoked_at,revoked_reason) values ('{UID}','{H('f')}',now()+interval '1h',now()+interval '2h',now(),'FOO')","23514"),
 ("sessions: absolute < expires", f"insert into auth_sessions(user_id,refresh_hash,expires_at,absolute_expires_at) values ('{UID}','{H('1')}',now()+interval '2h',now()+interval '1h')","23514"),
 ("sessions: prev_refresh_hash sai", f"insert into auth_sessions(user_id,refresh_hash,prev_refresh_hash,expires_at,absolute_expires_at) values ('{UID}','{H('2')}','xyz',now()+interval '1h',now()+interval '2h')","23514"),
 ("login_attempts: email_hash sai", "insert into login_attempts(email_hash,outcome) values ('abc','SUCCESS')","23514"),
 ("login_attempts: outcome sai", f"insert into login_attempts(email_hash,outcome) values ('{H('3')}','NOPE')","22P02"),
 ("mail: to_addr hoa", "insert into mail_outbox(to_addr,template) values ('A@x.vn','verify_email')","23514"),
 ("mail: to_addr không @", "insert into mail_outbox(to_addr,template) values ('abc','verify_email')","23514"),
 ("mail: template sai", "insert into mail_outbox(to_addr,template) values ('a@x.vn','Verify-Email')","23514"),
 ("mail: status sai", "insert into mail_outbox(to_addr,template,status) values ('a@x.vn','verify_email','X')","22P02"),
 ("mail: attempts 5", "insert into mail_outbox(to_addr,template,attempts) values ('a@x.vn','verify_email',5)","23514"),
 ("mail: SENT thiếu sent_at", "insert into mail_outbox(to_addr,template,status) values ('a@x.vn','verify_email','SENT')","23514"),
 ("mail: payload không phải object", "insert into mail_outbox(to_addr,template,payload) values ('a@x.vn','verify_email','[]')","23514"),
 ("mail: last_error > 1000", "insert into mail_outbox(to_addr,template,last_error) values ('a@x.vn','verify_email',repeat('x',1001))","23514"),
]
for name,sql,code in bad:
    _,err=q(sql); chk("INSERT sai bị chặn: "+name, ("ERROR" in err) and (code in q("select 1")[0] or True), err[:120])
    # kiểm mã SQLSTATE qua VERBOSITY
r=subprocess.run(["docker","exec","-i",C,"psql","-U","edupilot","-d","edupilot","-tA","-v","VERBOSITY=verbose","-c","select 1"],capture_output=True,text=True)
def code_of(sql):
    r=subprocess.run(["docker","exec","-i",C,"psql","-U","edupilot","-d","edupilot","-tA","-v","VERBOSITY=verbose","-c",sql],capture_output=True,text=True); 
    import re; m=re.search(r"ERROR:\s+([0-9A-Z]{5}):",r.stderr); return m.group(1) if m else ""
for name,sql,code in bad:
    got=code_of(sql); chk(f"SQLSTATE {code}: {name}", got==code, got)
# trùng
q(f"insert into auth_tokens(user_id,kind,token_hash,expires_at) values ('{UID}','INVITE','{H('9')}',now()+interval '1h')")
chk("tokens: trùng token_hash → 23505", code_of(f"insert into auth_tokens(user_id,kind,token_hash,expires_at) values ('{UID}','VERIFY_EMAIL','{H('9')}',now()+interval '1h')")=="23505")
q(f"insert into auth_sessions(user_id,refresh_hash,expires_at,absolute_expires_at) values ('{UID}','{H('8')}',now()+interval '1h',now()+interval '2h')")
chk("sessions: trùng refresh_hash → 23505", code_of(f"insert into auth_sessions(user_id,refresh_hash,expires_at,absolute_expires_at) values ('{UID}','{H('8')}',now()+interval '1h',now()+interval '2h')")=="23505")
q("insert into mail_outbox(to_addr,template,dedupe_key) values ('a@x.vn','verify_email','dk1')")
chk("mail: trùng dedupe_key → 23505", code_of("insert into mail_outbox(to_addr,template,dedupe_key) values ('b@x.vn','verify_email','dk1')")=="23505")
chk("tokens: user_id không tồn tại → 23503", code_of(f"insert into auth_tokens(user_id,kind,token_hash,expires_at) values ('22222222-2222-7222-8222-222222222222','INVITE','{H('7')}',now()+interval '1h')")=="23503")
# lành
ok=[ f"insert into auth_tokens(user_id,kind,token_hash,expires_at,used_at) values ('{UID}','RESET_PASSWORD','{H('6')}',now()+interval '1h',now())",
     f"insert into auth_sessions(user_id,refresh_hash,prev_refresh_hash,expires_at,absolute_expires_at,revoked_at,revoked_reason) values ('{UID}','{H('5')}','{H('4')}',now()+interval '1h',now()+interval '1h',now(),'LOGOUT')",
     f"insert into login_attempts(email_hash,outcome,ip) values ('{H('a')}','THROTTLED','10.0.0.1')",
     "insert into mail_outbox(to_addr,template,status,sent_at,attempts) values ('c@x.vn','invite','SENT',now(),4)"]
for i,s in enumerate(ok): _,err=q(s); chk(f"INSERT lành #{i+1}", err=="", err[:100])
q("delete from auth_tokens; delete from auth_sessions; delete from login_attempts; delete from mail_outbox; delete from users where email='qc-p201@example.test'")
print("\nTỔNG FAIL:",len(fails)); sys.exit(1 if fails else 0)
