# TC-P206-30: gateway admin create trên DB trống riêng qc2i_empty (không đụng ep-r / qc2i_auth)
import os,subprocess,re,json; exec(open(os.path.join(os.path.dirname(os.path.abspath(__file__)),'smkt2lib.py')).read())
from smkt2lib import rec
E='qc2i_empty'
def pg(q): return subprocess.run(['docker','exec','-i','ep-r-postgres-1','psql','-U','edupilot','-d',E,'-At','-F','|','-c',q],capture_output=True,text=True).stdout.strip()
def run(args,env):
    e=' '.join(f'-e {k}={v}' for k,v in env.items())
    r=subprocess.run(f'docker run --rm --network ep-r_default -e DATABASE_URL="postgres://edupilot:edupilot-dev@postgres:5432/{E}?sslmode=disable" -e PGBOUNCER_URL="postgres://edupilot:edupilot-dev@postgres:5432/{E}?sslmode=disable" -e APP_ENV=test {e} edupilot-gateway:latest admin {args}',shell=True,capture_output=True,text=True)
    return r.returncode,(r.stdout+r.stderr).strip()[:300]
subprocess.run('HEAVY= bash '+os.path.dirname(os.path.abspath(__file__))+'/qc2i-r4apia-stack.sh up0',shell=True,capture_output=True)
n0=pg("select count(*) from users"); PW='Smkt2!Quan-Tri-Pw-'+uuid.uuid4().hex[:6]+'x'
r1=run("create --email admin@edupilot.local --name 'Quản trị'",{'ADMIN_PASSWORD':f"'{PW}'"})
row=pg("select role||'|'||status||'|'||(email_verified_at is not null)::text from users where email='admin@edupilot.local'"); h1=pg("select md5(password_hash) from users where email='admin@edupilot.local'")
aud=pg("select actor_id is null, action from audit_log where action='admin_bootstrap'")
r2=run("create --email admin@edupilot.local --name 'Quản trị'",{'ADMIN_PASSWORD':f"'{PW}x'"}); h2=pg("select md5(password_hash) from users where email='admin@edupilot.local'"); cnt=pg("select count(*) from users where role='ADMIN'")
r3=run("create --email weak@edupilot.local --name 'Yếu'",{'ADMIN_PASSWORD':'123'}); weak=pg("select count(*) from users where email='weak@edupilot.local'")
r4=run("create --email argv@edupilot.local --name 'Tham số' --password Smkt2-Argv-Pw-1x",{'ADMIN_PASSWORD':"''"}); argv=pg("select count(*) from users where email='argv@edupilot.local'")
# không có route HTTP tạo ADMIN (ep-r)
tS=L('sv.gioi'); A=L('admin')
h1_=call('POST','/auth/register',None,{'email':f'smkt2.adm.{uuid.uuid4().hex[:5]}@edupilot.local','password':'Smkt2!Pass-x7y','full_name':'x','role':'ADMIN'})
h2_=call('POST','/admin/users',A,{'email':f'smkt2.adm2.{uuid.uuid4().hex[:5]}@edupilot.local','full_name':'x','role':'ADMIN'})
h3_=call('POST','/admin/users',tS,{'email':'x@e.local','full_name':'x','role':'ADMIN'})
ok=r1[0]==0 and row=='ADMIN|ACTIVE|true' and aud=='t|admin_bootstrap' and r2[0]==0 and 'tồn tại' in r2[1] and h1==h2 and cnt=='1' and r3[0]==1 and weak=='0' and r4[0]!=0 and argv=='0' and h1_[0] in(400,422) and h2_[0] in(400,422) and h3_[0]==403
rec('TC-P206-30','P' if ok else 'F',f'DB trống qc2i_empty (migrate up0, users={n0}): admin create với ADMIN_PASSWORD hợp lệ → rc={r1[0]}, bản ghi {row}, audit_log (actor_id NULL, admin_bootstrap) {aud}; lần hai → rc={r2[0]} «{r2[1][:80]}», md5(password_hash) không đổi ({h1==h2}), count(ADMIN)={cnt}; ADMIN_PASSWORD=123 → rc={r3[0]} «{r3[1][:80]}», chưa tạo ({weak}); --password trên dòng lệnh → rc={r4[0]} «{r4[1][:90]}», chưa tạo ({argv}) (mật khẩu chỉ đọc từ biến môi trường, cli không có cờ --password nên không thể lộ ở ps); không route HTTP tạo ADMIN: /auth/register role=ADMIN → {h1_[0]}, POST /admin/users role=ADMIN → {h2_[0]} (Admin), {h3_[0]} (SV); smkt2-p206-30.py')
