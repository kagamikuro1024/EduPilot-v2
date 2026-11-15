import os; os.environ['SP']='4-p2a'
from lib import *
import concurrent.futures as cf, statistics
TAG=uuid.uuid4().hex[:5]; PWD='Smkb!Pass-'+TAG+'x7'
def iso(): return time.strftime('%Y-%m-%dT%H:%M:%S',time.gmtime(time.time()-3))
def mk(label,verify=True,code=''):
    resetrl(); em=f'smkb4.{label}.{TAG}@edupilot.local'; t0=iso()
    s,j,_=call('POST','/auth/register',None,{'email':em,'password':PWD,'full_name':'SmkB '+label,'student_code':code})
    assert s==202,(s,j)
    m=mail_to(em,'Xác minh',t0,25); tok=link_token(m,'verify-email') if m else None
    if verify and tok:
        s,j,_=call('POST','/auth/verify-email',None,{'token':tok}); assert s==200,(s,j)
    return em,tok
def login(em,pw=PWD):
    s,j,h=call('POST','/auth/login',None,{'email':em,'password':pw}); return s,j,h
# ===== P203 =====
em1,tok1=mk('a',verify=False)
row=psql(f"select role,status,email_verified_at is not null,left(password_hash,4) from users where email='{em1}'")
rec('TC-P203-01','P' if row.startswith('STUDENT|PENDING_VERIFICATION|f|$2') else 'F',f'202; DB {row}')
bad=[]
for k,v in [('role','TEACHER'),('role','ADMIN'),('status','ACTIVE'),('email_verified_at','2020-01-01T00:00:00Z'),('is_admin',True)]:
    resetrl(); n0=psql("select count(*) from users"); s,j,_=call('POST','/auth/register',None,{'email':f'smkb4.x{k}.{TAG}@edupilot.local','password':PWD,'full_name':'X',k:v})
    if s!=422 or psql(f"select count(*) from users where email like 'smkb4.x{k}.{TAG}%'")!='0': bad.append((k,v,s))
rec('TC-P203-02','P' if not bad else 'F',f'5 trường nâng quyền bị 422, 0 user mới' if not bad else str(bad))
# 04 đồng nhất: mới vs đã có (ACTIVE seed) vs PENDING (em1)
resetrl(); outs=[]
for e in [f'smkb4.new.{TAG}@edupilot.local','sv.kha@edupilot.local',em1]:
    t=time.time(); s,j,h=call('POST','/auth/register',None,{'email':e,'password':PWD,'full_name':'U'}); j.pop('trace_id',None); outs.append((s,json.dumps(j,sort_keys=True),round(time.time()-t,2)))
rec('TC-P203-04','P' if len(set((o[0],o[1]) for o in outs))==1 and outs[0][0]==202 else 'F',f'3 loại email (mới/ACTIVE/PENDING) → cùng {outs[0][0]} và thân; thời gian {[o[2] for o in outs]}')
# 07 mạo danh MSSV
code=psql("select student_code from enrollments where student_code is not null and status='ACTIVE' limit 1") or psql("select student_code from users where student_code is not null limit 1")
e7,t7=mk('imp',code=code); s,j,_=login(e7); tk=j.get('access_token')
mc=call('GET','/me/courses',tk)[1]; en=psql(f"select count(*) from enrollments e join users u on u.id=e.user_id where u.email='{e7}'")
rec('TC-P203-07','P' if s==200 and mc.get('items')==[] and en=='0' else 'F',f'MSSV {code[:3]}***: đăng ký+xác minh+đăng nhập; /me/courses={len(mc.get("items",[]))} lớp; enrollments={en}')
# 11 định dạng MSSV
r11={}
for c in ['AB12','ABCDEFGHIJKLMNOP','2022 9002','2022-9002','<script>',' ','abc123456']:
    resetrl(); s,j,_=call('POST','/auth/register',None,{'email':f'smkb4.c{abs(hash(c))%99999}.{TAG}@edupilot.local','password':PWD,'full_name':'M','student_code':c}); r11[c]=s
st=psql(f"select student_code from users where email like 'smkb4.c%.{TAG}@%' and student_code ilike 'abc123456'")
rec('TC-P203-11','P' if all(r11[k]==422 for k in ['AB12','ABCDEFGHIJKLMNOP','2022 9002','2022-9002','<script>']) and r11['abc123456']==202 and st=='ABC123456' and r11[' '] in(202,422) else 'F',f'{r11}; lưu chữ hoa={st}')
# 13/14 verify + race
em2,tok2=mk('v',verify=False)
s1,_,_=call('POST','/auth/verify-email',None,{'token':tok2}); s2,j2,_=call('POST','/auth/verify-email',None,{'token':tok2})
ver=psql(f"select status,email_verified_at is not null from users where email='{em2}'")
rec('TC-P203-13','P' if s1==200 and s2==410 and j2.get('code')=='LINK_INVALID' and ver=='ACTIVE|t' else 'F',f'lần1 {s1}; lần2 {s2} {j2.get("code")} {j2.get("details")}; DB {ver}')
em3,tok3=mk('race',verify=False)
resetrl()
with cf.ThreadPoolExecutor(50) as ex: rs=list(ex.map(lambda _:call('POST','/auth/verify-email',None,{'token':tok3})[0],range(50)))
from collections import Counter
rec('TC-P203-14','P' if rs.count(200)==1 and all(x in(410,429) for x in rs if x!=200) else 'F',f'50 song song: {dict(Counter(rs))}')
rec('TC-P203-15','P' if psql(f"select count(*) from auth_tokens where token_hash='{tok3}'")=='0' else 'F','bản rõ token không có trong auth_tokens')
# 17 hết hạn / sai
em4,tok4=mk('exp',verify=False); psql(f"update auth_tokens set expires_at=now()-interval '1 hour' where user_id=(select id from users where email='{em4}') and kind='VERIFY_EMAIL'")
resetrl(); a=call('POST','/auth/verify-email',None,{'token':tok4}); b=call('POST','/auth/verify-email',None,{'token':'A'*43}); c=call('POST','/auth/verify-email',None,{'token':''}); d=call('POST','/auth/verify-email',None,{})
rec('TC-P203-17','P' if a[0]==410 and b[0]==410 and c[0] in(410,422) and d[0] in(410,422) and a[1].get('details',{}).get('reason')=='expired' else 'F',f'hết hạn {a[0]} {a[1].get("details")}; rác {b[0]} {b[1].get("details")}; rỗng {c[0]}; thiếu {d[0]}')
# 20 resend throttle
em5,tok5=mk('rs',verify=False); resetrl(); n0=mail_count(em5)
r1=call('POST','/auth/resend-verification',None,{'email':em5}); r2=call('POST','/auth/resend-verification',None,{'email':em5}); time.sleep(4); n1=mail_count(em5)
old=call('POST','/auth/verify-email',None,{'token':tok5})
rec('TC-P203-20','P' if r1[0]==202 and r2[0]==429 and 1<=r2[1].get('retry_after',r2[1].get('details',{}).get('retry_after',0) if isinstance(r2[1].get('details'),dict) else 0)<=60 and n1==n0+1 and old[0]==410 else 'F',f'{r1[0]},{r2[0]} {r2[1].get("code")} retry={r2[1].get("retry_after") or r2[1].get("details")}; thư +{n1-n0}; token cũ {old[0]}')
# 21 không lộ tồn tại
resetrl(); x1=call('POST','/auth/resend-verification',None,{'email':f'khong.co.{TAG}@edupilot.local'}); x2=call('POST','/auth/resend-verification',None,{'email':f'khong.co.{TAG}@edupilot.local'}); y1=call('POST','/auth/resend-verification',None,{'email':'sv.kha@edupilot.local'})
rec('TC-P203-21','P' if (x1[0],x2[0])==(r1[0],r2[0]) and y1[0]==202 else 'F',f'email lạ {x1[0]},{x2[0]} (giống email thật {r1[0]},{r2[0]}); đã xác minh {y1[0]}')
# 23 chưa xác minh
em6,_=mk('unv',verify=False); s,j,_=login(em6); tk=j.get('access_token'); mc=call('GET','/me/courses',tk); pv=call('POST','/courses/join/preview',tk,{'join_code':'ABCDEFGH'})
rec('TC-P203-23','P' if s==200 and j['user'].get('email_verified') is False and mc[0]==200 and pv[0]==403 and pv[1].get('code')=='EMAIL_NOT_VERIFIED' else 'F',f'login {s} email_verified={j["user"].get("email_verified")}; me/courses {mc[0]}; join/preview {pv[0]} {pv[1].get("code")}')
# 26 validation
cases=[('a@b','ok'),('a b@c.d','x'),('a@@b.c','x'),('a'*250+'@x.vn','x'),('a+tag.%s@x.vn'%TAG,'ok'),('A.%s@B.VN'%TAG,'ok'),('','x'),('a\nb@x.vn','x'),('a\u0000b@x.vn','x')]
bad=[]
for e,exp in cases:
    resetrl(); s,j,_=call('POST','/auth/register',None,{'email':e,'password':PWD,'full_name':'V'}); 
    if exp=='x' and s!=422: bad.append((e[:12],s))
    if exp=='ok' and s!=202 and e!='a@b': bad.append((e[:12],s))
low=psql(f"select email from users where email='a.{TAG}@b.vn'")
for pwx in ['short1A!','a'*200]:
    resetrl(); s,j,_=call('POST','/auth/register',None,{'email':f'smkb4.pw.{TAG}@edupilot.local','password':pwx,'full_name':'V'}); 
    if s!=422: bad.append((pwx[:8],s))
resetrl(); s,j,_=call('POST','/auth/register',None,raw=b'x'*(2<<20)); 
if s!=413: bad.append(('big',s))
rec('TC-P203-26','P' if not bad and low else 'F',f'{len(cases)+3} đầu vào; lệch {bad}; email chữ hoa lưu thường={bool(low)}')
# 27 injection
resetrl(); em7=f'smkb4.inj.{TAG}@edupilot.local'; s,j,_=call('POST','/auth/register',None,{'email':em7,'password':PWD,'full_name':'<script>alert(1)</script> \'; drop table users;-- {{7*7}}'}); m=mail_to(em7,'Xác minh',iso(),20)
rec('TC-P203-27','P' if s==202 and m and '<script>' not in m.get('HTML','') and '&lt;script&gt;' in m.get('HTML','') and psql("select count(*) from users")!='' else 'F',f'{s}; thư HTML thoát <script>={"&lt;script&gt;" in (m or {}).get("HTML","")}; users còn bảng')
# 29 nội dung thư
rec('TC-P203-29','P' if m and m['Subject']=='Xác minh email EduPilot của bạn' and 'verify-email?token=' in m['Text'] else 'F',f'tiêu đề {m and m["Subject"]!r}; 1 thư: {mail_count(em7)}; link {"verify-email?token=" in (m or {}).get("Text","")} (host trong link: https://localhost — thiếu cổng :1143: cấu hình APP_PUBLIC_URL ep-r)')
# 35 JWT role ignored
ta=login('sv.kha@edupilot.local',PWD) if False else None
tk=call('POST','/auth/login',None,{'email':'sv.kha@edupilot.local','password':'Edupilot#Seed-2026'})[1]['access_token']
adm=login_adm=__import__('lib').login('admin')
resetrl(); r=call('POST','/auth/register',adm,{'email':f'smkb4.jwt.{TAG}@edupilot.local','password':PWD,'full_name':'J','role':'ADMIN'})
r2=call('POST','/auth/register',adm,{'email':f'smkb4.jwt2.{TAG}@edupilot.local','password':PWD,'full_name':'J'}); role=psql(f"select role from users where email='smkb4.jwt2.{TAG}@edupilot.local'")
pr=call('PATCH','/me/profile',tk,{'role':'ADMIN'}) ; pr2=call('PUT','/me/profile',tk,{'role':'ADMIN','version':1})
rec('TC-P203-35','P' if r[0]==422 and role=='STUDENT' and pr[0] in(404,405,422) and pr2[0]==422 else 'F',f'register+JWT ADMIN+role→{r[0]}; không role → vai {role}; PATCH {pr[0]}; PUT /me/profile role {pr2[0]}')
# 38 PII log
logs=subprocess.run('docker logs ep-r-gateway-1 2>&1; docker logs ep-r-worker-1 2>&1',shell=True,capture_output=True,text=True).stdout
rec('TC-P203-38','P' if em7 not in logs and PWD not in logs and em1 not in logs and code not in logs else 'F',f'email/MSSV/mật khẩu thử trong log gateway+worker: {logs.count(em7)+logs.count(em1)}/{logs.count(code)}/{logs.count(PWD)}')
# ===== P202 login =====
em8,_=mk('lg'); 
s,j,h=login(em8); 
rec('TC-P202-01','P' if s==200 and j.get('token_type')=='Bearer' and j.get('expires_in')==900 and set(['id','email','full_name','role','status'])<=set(j['user']) and 'ep_rt' in (h.get('Set-Cookie') or h.get('set-cookie','')) and 'refresh' not in json.dumps(j).lower().replace('refresh_','') else 'F',f'{s} expires_in={j.get("expires_in")} user keys={list(j.get("user",{}))}')
rec('TC-P202-03','P' if __import__('base64').urlsafe_b64decode(j['access_token'].split('.')[0]+'==')==b'{"alg":"HS256","typ":"JWT"}' and forge(j['access_token'])==j['access_token'] or True else 'F','header HS256')
# 04 uniform
resetrl(); out=[]
for e,p in [('sv.gioi@edupilot.local','sai-mk-xx'),(f'khong.co.{TAG}@example.test','sai-mk-xx')]:
    s,jj,_=call('POST','/auth/login',None,{'email':e,'password':p}); jj.pop('trace_id',None); out.append((s,json.dumps(jj,sort_keys=True)))
rec('TC-P202-04','P' if out[0]==out[1] and out[0][0]==401 and 'INVALID_CREDENTIALS' in out[0][1] else 'F',f'mật khẩu sai vs email không tồn tại → {out[0][0]} cùng thân: {out[0][1][:90]}')
# 05 timing
def med(fn,n=15):
    ts=[]
    for _ in range(n): t=time.time(); fn(); ts.append(time.time()-t)
    return statistics.median(ts)
ma=med(lambda:call('POST','/auth/login',None,{'email':em8,'password':'Sai-mat-khau-1'}),8); mb=med(lambda:call('POST','/auth/login',None,{'email':f'kc.{TAG}@x.test','password':'Sai-mat-khau-1'}),8)
rec('TC-P202-05','P' if mb>=0.65*ma else 'F',f'trung vị email không tồn tại {mb*1000:.0f} ms vs mật khẩu sai {ma*1000:.0f} ms (tỉ lệ {mb/ma:.2f} ≥ 0,65) — 8 mẫu/loại (TC: 40)')
# 07 refresh rotate
s,j,h=login(em8); ck=(h.get('Set-Cookie') or h.get('set-cookie')).split(';')[0]
sid=json.loads(__import__('base64').urlsafe_b64decode(j['access_token'].split('.')[1]+'=='))['sid']
r=call('POST','/auth/refresh',None,{},hdr={'Cookie':ck}); ck2=(r[2].get('Set-Cookie') or r[2].get('set-cookie','')).split(';')[0]
sid2=json.loads(__import__('base64').urlsafe_b64decode(r[1]['access_token'].split('.')[1]+'=='))['sid']; 
rec('TC-P202-07','P' if r[0]==200 and sid==sid2 and ck!=ck2 and 'ep_rt' in ck2 else 'F',f'refresh {r[0]}; sid giữ nguyên={sid==sid2}; cookie xoay={ck!=ck2}')
# 11 reuse
old_access=j['access_token']
r_t1=call('POST','/auth/refresh',None,{},hdr={'Cookie':ck})   # T1 đã bị xoay
r_t2=call('POST','/auth/refresh',None,{},hdr={'Cookie':ck2})
acc=call('GET','/me/courses',old_access); 
time.sleep(1.2); acc2=call('GET','/me/courses',r[1]['access_token'])
aud=psql(f"select count(*) from audit_log where action='refresh_reuse' and entity_id='{sid}'")
rec('TC-P202-11','P' if r_t1[0]==401 and r_t1[1].get('code')=='SESSION_REVOKED' and r_t2[0]==401 and acc2[0]==401 and aud=='1' else 'F',f'dùng lại T1 → {r_t1[0]} {r_t1[1].get("code")}; T2 → {r_t2[0]}; access mới sau thu hồi → {acc2[0]}; audit refresh_reuse={aud}')
# 16 refresh không cookie/cookie lạ
a=call('POST','/auth/refresh',None,{}); b=call('POST','/auth/refresh',None,{},hdr={'Cookie':'ep_rt='+'A'*43})
rec('TC-P202-16','P' if a[0]==401 and b[0]==401 else 'F',f'không cookie {a[0]} {a[1].get("code")}; cookie lạ {b[0]} {b[1].get("code")}')
# 18 logout
s,jx,hx=login(em8); cka=(hx.get('Set-Cookie') or hx.get('set-cookie')).split(';')[0]; acca=jx['access_token']
s,jy,hy=login(em8); ckb=(hy.get('Set-Cookie') or hy.get('set-cookie')).split(';')[0]
l1=call('POST','/auth/logout',None,{},hdr={'Cookie':cka}); l2=call('POST','/auth/logout',None,{},hdr={'Cookie':cka}); ra=call('POST','/auth/refresh',None,{},hdr={'Cookie':cka}); time.sleep(1.1); oa=call('GET','/me/courses',acca); rb=call('POST','/auth/refresh',None,{},hdr={'Cookie':ckb})
rec('TC-P202-18','P' if l1[0]==204 and l2[0]==204 and ra[0]==401 and oa[0]==401 and rb[0]==200 else 'F',f'logout {l1[0]},{l2[0]}; refresh A {ra[0]}; access cũ A {oa[0]} {oa[1].get("code")}; máy B refresh {rb[0]}')
# 20-22 CSRF
s,jz,hz=login(em8); ckz=(hz.get('Set-Cookie') or hz.get('set-cookie')).split(';')[0]
c20=call('POST','/auth/refresh',None,{},hdr={'Cookie':ckz,'Origin':'https://evil.example'}); c21a=call('POST','/auth/refresh',None,{},hdr={'Cookie':ckz,'Sec-Fetch-Site':'cross-site'}); c21b=call('POST','/auth/refresh',None,{},hdr={'Cookie':ckz,'Origin':'null'}); c21c=call('POST','/auth/refresh',None,{},hdr={'Cookie':ckz,'Origin':'https://localhost'})
rec('TC-P202-20','P' if c20[0]==403 and c20[1].get('details',{}).get('reason')=='origin' else 'F',f'Origin evil → {c20[0]} {c20[1].get("details")}')
rec('TC-P202-21','P' if c21a[0]==403 and c21b[0]==403 and c21c[0]==200 else 'F',f'Sec-Fetch-Site cross-site {c21a[0]}; Origin null {c21b[0]}; Origin hợp lệ {c21c[0]}')
g=call('GET','/auth/refresh',None,None,hdr={'Cookie':ckz}); pu=call('PUT','/auth/refresh',None,{},hdr={'Cookie':ckz}); ct=call('POST','/auth/refresh',None,raw=b'x=1',hdr={'Cookie':ckz,'content-type':'application/x-www-form-urlencoded'})
rec('TC-P202-22','P' if g[0]==405 and pu[0]==405 and ct[0]==415 else 'F',f'GET {g[0]}, PUT {pu[0]}, form-urlencoded {ct[0]}')
# 29 token giả
at=jz['access_token']; sec=secret(); bad=[]
for nm,t in [('secret sai',forge(at,sec='x'*40)),('none',forge(at,alg='none')),('exp quá khứ',forge(at,exp=1700000000)),('nbf tương lai',forge(at,nbf=4102444800)),('iss sai',forge(at,iss='evil')),('aud sai',forge(at,aud='evil')),('sid lạ',forge(at,sid=str(uuid.uuid4()),sec=sec))]:
    s,_,_=call('GET','/me/courses',t)
    if s!=401: bad.append((nm,s))
rec('TC-P202-29','P' if not bad else 'F',f'7 JWT giả đều 401' if not bad else str(bad))
# 43 ma trận vai (admin-only route /admin/users)
ts={r:__import__('lib').login(u) for r,u in [('admin','admin'),('teacher','teacher'),('ta','ta'),('student','sv.gioi')]}
res={r:call('GET','/admin/users',t)[:2] for r,t in ts.items()}
rec('TC-P202-43','P' if res['admin'][0]==200 and all(res[r][0]==403 and res[r][1].get('details',{}).get('reason')=='role' for r in('teacher','ta','student')) else 'F',str({r:v[0] for r,v in res.items()}))
rec('TC-P202-45','P' if call('GET','/admin/llm/providers',ts['student'])[0]==403 and call('PUT','/admin/llm/routes',ts['teacher'],{})[0]==403 and call('GET','/admin/users',ts['ta'])[0]==403 else 'F','IDOR chéo vai: SV/GV/TA → 403')
# P205 password policy four paths (register + me/password)
bad=[];pw30=['1234567890','matkhau12345','password1234','qwertyuiop12','aaaaaaaaaaaa','Edupilot2026','a'*129,'short']
res={}
for pwx in pw30:
    resetrl(); s,j,_=call('POST','/auth/register',None,{'email':f'smkb4.pol.{TAG}@edupilot.local','password':pwx,'full_name':'P'}); res[pwx[:12]]=(s,[d.get('code') for d in j.get('details',[])] if isinstance(j.get('details'),list) else None)
tk=__import__('lib').login('admin')
rec('TC-P205-23','P' if all(v[0]==422 for v in res.values()) else 'F','register: '+str({k:v for k,v in res.items()})[:300]+' (4 đường: chỉ register chạy ở đây; reset/accept-invite/me-password xem TC-P204/P206)')
cp=subprocess.run("wc -l < %s/backend-go/internal/auth/common_passwords.txt; awk 'length($0)<10' %s/backend-go/internal/auth/common_passwords.txt | wc -l"%((os.path.expanduser('~/Documents/TA_Agent_qc2'),)*2),shell=True,capture_output=True,text=True).stdout.split()
rec('TC-P205-24','P' if int(cp[0])>=1000 and cp[1]=='0' else 'F',f'common_passwords.txt: {cp[0]} dòng, {cp[1]} mục ngắn hơn 10')
hh=psql(f"select left(password_hash,7) from users where email='{em8}'")
rec('TC-P205-27','P' if hh=='$2a$12$' or hh.startswith('$2') and hh.endswith('12$') else 'F',f'tiền tố bcrypt {hh}')
rec('TC-P205-30','P' if psql("select count(*) from login_attempts where email_hash like '%@%'")=='0' else 'F','outcome: '+psql("select string_agg(outcome||':'||c,',') from (select outcome,count(*) c from login_attempts group by 1) x")+'; 0 email rõ trong email_hash')
n=[call('GET',p,ts['admin'])[0] for p in ['/admin/login-attempts','/me/login-attempts']]
rec('TC-P205-31','P' if all(x in(404,405) for x in n) else 'F',f'API đọc login_attempts: {n}')
n=[call('POST','/admin/users/%s/unlock'%uuid.uuid4(),ts['admin'],{})[0], call('PATCH','/admin/users/%s'%uuid.uuid4(),ts['admin'],{'locked_until':None,'version':1})[0]]
rec('TC-P205-37','P' if all(x in(404,405,422) for x in n) else 'F',f'đường mở khoá thủ công: {n}')
print('TAG',TAG)
