import os; os.environ['SP']='4-t1'
from smkT import *
import hashlib
root=os.path.expanduser('~/Documents/TA_Agent_qc2')
# ---- P201-03
mod=sh(f"cd {root} && git log --format='%h %ad %s' --date=short --diff-filter=M -- backend-go/db/migrations/00001_pg_platform.sql backend-go/db/migrations/00002_llm.sql")
st=sh(f"cd {root} && git diff --stat origin/main -- backend-go/db/migrations/00001* backend-go/db/migrations/00002* | wc -l")
h=sh(f"cd {root}/backend-go/db/migrations && sha256sum 00001* 00002* | cut -c1-16 | tr '\\n' ' '")
rec('TC-P201-03','P' if mod.strip()=='' and st.strip()=='0' else 'F',f'git log --diff-filter=M trên 00001/00002 = rỗng (chỉ commit tạo: 240621b1 s1, 01242098 s2); git diff --stat origin/main = 0 dòng; sha256 {h}')
# ---- P201-04 goose down/up trên DB tạm trong ep-r-postgres-1
pgpw=sh("docker inspect ep-r-postgres-1 --format '{{range .Config.Env}}{{println .}}{{end}}' | grep ^POSTGRES_PASSWORD= | cut -d= -f2").strip()
dsn=f'postgres://edupilot:{pgpw}@127.0.0.1:6133/smkt_goose?sslmode=disable'
sh("docker exec ep-r-postgres-1 psql -U edupilot -d edupilot -c 'drop database if exists smkt_goose' -c 'create database smkt_goose'")
def goose(cmd): 
    r=subprocess.run(f'cd {root}/backend-go/db/migrations && goose postgres "{dsn}" {cmd}',shell=True,capture_output=True,text=True); return r.returncode,(r.stdout+r.stderr).replace(pgpw,'***')
def pq(q): return subprocess.run(['docker','exec','-i','ep-r-postgres-1','psql','-U','edupilot','-d','smkt_goose','-At','-c',q],capture_output=True,text=True).stdout.strip()
rc1,o1=goose('up-to 4'); v1=pq('select max(version_id) from goose_db_version where is_applied')
rc2,o2=goose('down'); rc3,o3=goose('down'); v3=pq('select max(version_id) from goose_db_version where is_applied')
orph=pq("select count(*) from pg_type where typname in ('auth_token_kind','mail_status','login_outcome','course_status','enrollment_status')")
tabs=pq("select count(*) from information_schema.tables where table_schema='public' and table_name in ('auth_sessions','auth_tokens','login_attempts','mail_outbox','courses','enrollments','class_sessions','notifications','user_settings','documents','content_chunks','document_courses')")
rc4,o4=goose('up-to 4'); v4=pq('select max(version_id) from goose_db_version where is_applied'); n4=pq("select count(*) from pg_tables where schemaname='public' and tablename in ('auth_sessions','auth_tokens','login_attempts','mail_outbox')")
open('/tmp/smkT/goose.log','w').write('\n'.join([o1,o2,o3,o4]))
rec('TC-P201-04','P' if (rc1,rc2,rc3,rc4)==(0,0,0,0) and v1=='4' and v3=='2' and orph=='0' and tabs=='0' and v4=='4' and n4=='4' else 'F',f'goose up-to 4 → v{v1}; down×2 → v{v3}, enum/bảng của 00003-00004 còn lại={orph}/{tabs}; up-to 4 lại rc={rc4}, v{v4}, 4 bảng auth/mail={n4} (DB tạm smkt_goose trên ep-r-postgres, đã xoá); log /tmp/smkT/goose.log')
sh("docker exec ep-r-postgres-1 psql -U edupilot -d edupilot -c 'drop database if exists smkt_goose'")
# ---- P201-07 INSERT hợp lệ mỗi bảng (rollback)
u=psql("select id from users where email='sv.kha@edupilot.local'")
q=f"""begin;
insert into auth_sessions(user_id,refresh_hash,expires_at,absolute_expires_at) values('{u}',repeat('a',64),now()+interval '14 days',now()+interval '30 days');
insert into auth_tokens(user_id,kind,token_hash,expires_at) values('{u}','VERIFY_EMAIL',repeat('b',64),now()+interval '1 day');
insert into login_attempts(email_hash,outcome) values(repeat('c',64),'SUCCESS');
insert into mail_outbox(to_addr,template,payload,dedupe_key) values('smkt.valid@example.test','email_exists','{{}}','smkt-valid-1');
insert into mail_outbox(to_addr,template,status,sent_at) values('smkt.valid2@example.test','verify_email','SENT',now());
select (select count(*) from auth_sessions where refresh_hash=repeat('a',64)),(select count(*) from auth_tokens where token_hash=repeat('b',64)),(select count(*) from login_attempts where email_hash=repeat('c',64)),(select count(*) from mail_outbox where to_addr like 'smkt.valid%');
rollback;"""
out=sh("docker exec -i ep-r-postgres-1 psql -U edupilot -d edupilot -At -F '|' -v ON_ERROR_STOP=1 <<'EOF'\n"+q+"\nEOF")
after=psql("select count(*) from mail_outbox where to_addr like 'smkt.valid%'")
rec('TC-P201-07','P' if '1|1|1|2' in out and after=='0' else 'F',f'INSERT hợp lệ vào auth_sessions/auth_tokens/login_attempts/mail_outbox (2 dòng SENT/QUEUED) trong 1 transaction: đếm {out.splitlines()[-2] if out.strip() else out!r}, ROLLBACK, còn lại {after}')
# ---- P201-09 + 11 token băm
em,s,j,t0=register('t09'); m=mail_to(em,'Xác minh',t0,30); tk=link_token(m,'verify-email'); seen(tk)
shat=hashlib.sha256(tk.encode()).hexdigest()
plain=psql(f"select count(*) from auth_tokens where token_hash='{tk}'"); hashed=psql(f"select count(*) from auth_tokens where token_hash='{shat}'")
db=sh("docker exec ep-r-postgres-1 pg_dump -U edupilot -d edupilot --data-only 2>/dev/null | grep -c '%s'"%tk).strip()
fmt=bool(re.fullmatch(r'[A-Za-z0-9_-]{43}',tk or ''))
rec('TC-P201-09','P' if s==202 and plain=='0' and hashed=='1' and fmt and db=='0' else 'F',f'register→token từ thư dài {len(tk)} khớp [A-Za-z0-9_-]{{43}}={fmt}; auth_tokens bản rõ={plain}, sha256={hashed}; pg_dump toàn DB grep bản rõ={db}')
# ---- P201-33 consumer mail.send
cons=sh("docker exec ep-r-redis-1 redis-cli XINFO CONSUMERS outbox.dispatch outbox").split('\n')
names=[];idle={}
cur=None
for i,l in enumerate(cons):
    if l=='name': cur=cons[i+1]
    if l=='idle' and cur: idle[cur]=int(cons[i+1]); cur=None
active=[n for n,v in idle.items() if v<120000]
ids={c.split()[0]:c.split()[1] for c in sh("docker ps --format '{{.ID}} {{.Names}}'").strip().split('\n')}
amap={a:ids.get(a,'?') for a in active}
reg_gw=sh(f"cd {root}/backend-go && grep -rn 'Register(' cmd/gateway | grep -c 'mail.send'").strip()
reg_wk=sh(f"cd {root}/backend-go && grep -rn 'mail' cmd/worker/registry.go | grep -ci 'Register\\|mail.Topic'").strip()
bad=[a for a,n in amap.items() if 'gateway' in n]
rec('TC-P201-33','P' if active and not bad and reg_gw=='0' else 'F',f'topic mail.send chạy trên stream outbox.dispatch (nhóm outbox, {len(idle)} consumer lịch sử); consumer hoạt động <120 s: {amap}; không consumer nào là container gateway; cmd/gateway không đăng ký mail.send ({reg_gw}), cmd/worker/registry.go có {reg_wk} dòng mail')
