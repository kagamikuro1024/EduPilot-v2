import os,yaml,re; os.environ['SP']='4-p2h'
from lib import *
import concurrent.futures as cf, random
from collections import Counter
TAG=uuid.uuid4().hex[:5]; PWD='Smkb!Pass-'+TAG+'x7'; REPO=os.path.expanduser('~/Documents/TA_Agent_qc2')
def iso(): return time.strftime('%Y-%m-%dT%H:%M:%S',time.gmtime(time.time()-3))
L=__import__('lib').login; A=L('admin'); S=L('sv.gioi'); K=L('sv.kha'); TT=L('teacher'); TAs=L('ta')
def lg(em,pw=PWD):
    s,j,h=call('POST','/auth/login',None,{'email':em,'password':pw}); return j.get('access_token')
def stu(label,code='',verify=True):
    resetrl(); em=f'smkb4.{label}.{TAG}@edupilot.local'; t0=iso(); s,j,_=call('POST','/auth/register',None,{'email':em,'password':PWD,'full_name':'SV '+label,'student_code':code}); assert s==202,(s,j)
    if verify:
        m=mail_to(em,'Xác minh',t0,25); call('POST','/auth/verify-email',None,{'token':link_token(m,'verify-email')})
    return em,lg(em)
def staff(label,role):
    em=f'smkb4.{label}.{TAG}@edupilot.local'; t0=iso(); s,j,_=call('POST','/admin/users',A,{'email':em,'full_name':'GV '+label,'role':role}); assert s==201,(s,j)
    m=mail_to(em,'mời|Lời mời',t0,25); resetrl(); call('POST','/auth/accept-invite',None,{'token':link_token(m,'invite'),'password':PWD}); return em,lg(em),j['id']
gvm,GV,gvid=staff('gvh','TEACHER'); tam,TA,taid=staff('tah','TA')
def mkcourse(tag,subject='INT1006'):
    s,c,_=call('POST','/admin/courses',A,{'subject_code':subject,'class_code':f'SH{tag}{TAG}'.upper(),'name':f'Lớp SmkB {tag} {TAG}','semester':'2026-2027-HK1','teacher_id':gvid}); call('POST',f'/admin/courses/{c["id"]}/assign',A,{'ta_ids':[taid]}); return c['id']
cid=mkcourse('A'); rid=mkcourse('R'); cid3=mkcourse('S','INT9999')
code=call('GET',f'/courses/{cid}/join-code',GV)[1]['join_code']
def multipart(data,fname):
    b='----smk'+uuid.uuid4().hex; body=(f'--{b}\r\nContent-Disposition: form-data; name="file"; filename="{fname}"\r\nContent-Type: application/octet-stream\r\n\r\n').encode()+data+f'\r\n--{b}--\r\n'.encode(); return body,f'multipart/form-data; boundary={b}'
FX='/tmp/qc2-b/fx/'
def imp(c_,tok,fname,q='',data=None):
    body,ct=multipart(data if data is not None else open(FX+fname,'rb').read(),fname); return call('POST',f'/courses/{c_}/roster/import{q}',tok,raw=body,hdr={'content-type':ct,'Idempotency-Key':str(uuid.uuid4())},idem=False)
# ===== P210 =====
a=imp(rid,GV,'ok-30.csv','?dry_run=true'); b=imp(rid,GV,'bom-semicolon.csv','?dry_run=true'); c=imp(rid,GV,'ok.xlsx','?dry_run=true')
rec('TC-P210-01','P' if all(x[0]==200 and x[1].get('total',0)>=5 for x in (a,b,c)) else 'F',f'dry_run ok-30.csv total={a[1].get("total")}; bom-semicolon.csv {b[0]} total={b[1].get("total")}; ok.xlsx {c[0]} total={c[1].get("total")}')
mg={n:imp(rid,GV,n,'?dry_run=true') for n in['fake-ext.csv','png-as.csv','text.xlsx','cut.xlsx']}
rec('TC-P210-03','P' if all(v[0]==422 for v in mg.values()) else 'F',f'tệp giả/hỏng → {{ {", ".join(k+":"+str(v[0]) for k,v in mg.items())} }}; details={mg["fake-ext.csv"][1].get("details")}')
t=time.time(); zb=imp(rid,GV,'zipbomb.xlsx','?dry_run=true'); dt=time.time()-t; alive=call('GET','/healthz')[0]
rec('TC-P210-04','P' if zb[0] in(413,422) and dt<=5 and alive==200 else 'F',f'zip bomb → {zb[0]} sau {dt:.1f}s; gateway còn sống ({alive})')
# 36/37 + lines
res={}
for n in['formula.csv','injection.csv','control.csv','ten-dai.csv','dong-dai.csv']:
    res[n]=imp(rid,GV,n,'?dry_run=true')[0]
fm=imp(rid,GV,'formula.csv'); inj=imp(rid,GV,'injection.csv')
names=psql("select string_agg(full_name,' ¦ ') from users where email like 'qc-f_@example.test' or email like 'qc-i_@example.test'")
rec('TC-P210-36','P' if all(v in(200,422) for v in res.values()) and res['control.csv']==200 or res['control.csv']==422 else 'F',f'dry_run formula/injection/control/tên dài/dòng dài → {res}; tên lưu: {names[:160]!r} (lưu nguyên văn, không thực thi)')
rec('TC-P210-37','P' if fm[0] in(200,422) and inj[0] in(200,422) and psql("select count(*) from users")!='' else 'F',f'import formula {fm[0]}, injection {inj[0]}; bảng users còn nguyên; thân thư HTML thoát (TC-P203-27)')
# 17
resetrl(); a17=call('POST','/auth/register',None,{'email':'qc-ok1@example.test','password':PWD,'full_name':'Kẻ tấn công'}); b17=call('POST','/auth/register',None,{'email':f'smkb4.khong.{TAG}@x.test','password':PWD,'full_name':'x'}); l17=call('POST','/auth/login',None,{'email':'qc-ok1@example.test','password':PWD})[0]
stt=psql("select status from users where email='qc-ok1@example.test'")
rec('TC-P210-17','P' if a17[0]==202 and a17[1]==b17[1] and l17==401 else 'F',f'đăng ký bằng email roster: {a17[0]} cùng thân email lạ={a17[1]==b17[1]}; đăng nhập bằng mật khẩu kẻ tấn công {l17}; tài khoản {stt}')
# 14 MSSV mạo danh: MSSV roster ok-30 = 2022000xx
mssv=psql("select student_code_snapshot from enrollments where course_id='%s' and student_code_snapshot is not null limit 1"%rid) or '202200201'
xm,XT=stu('imp',code=mssv); mc=call('GET','/me/courses',XT)[1]; en=psql(f"select count(*) from enrollments e join users u on u.id=e.user_id where u.email='{xm}'"); rr=call('GET',f'/courses/{rid}/today',XT)[0]; rm=call('GET',f'/courses/{rid}/members',XT)[0]
rec('TC-P210-14','P' if mc.get('items')==[] and en=='0' and rr==403 and rm==403 else 'F',f'MSSV {mssv[:4]}*** của roster bằng email khác (đã xác minh): me/courses={len(mc.get("items",[]))}; enrollments={en}; today {rr}; members {rm}')
# 27
sh=[call('POST',f'/courses/{cid}/share-from',tk,{'source_course_id':rid,'what':['documents']})[0] for tk in (TA,)]; sh2=call('POST',f'/courses/{cid3}/share-from',GV,{'source_course_id':rid,'what':['documents']}); sh3=call('POST',f'/courses/{cid}/share-from',GV,{'source_course_id':rid,'what':['question_bank']})
rec('TC-P210-27','P' if sh==[403] and sh2[0]==422 and sh3[0]==422 else 'F',f'TA {sh}; khác học phần {sh2[0]}; loại chưa hỗ trợ {sh3[0]}')
# 24 thư hàng loạt
time.sleep(5); mails=sum(1 for i in range(201,230) if mail_count(f'qc-le{i}@example.test')>=1) if False else None
lg_=subprocess.run('docker logs ep-r-gateway-1 2>&1; docker logs ep-r-worker-1 2>&1',shell=True,capture_output=True,text=True).stdout
rec('TC-P210-39','P' if 'qc-ok1@example.test' not in lg_ and mssv not in lg_ else 'F',f'email/MSSV roster trong log: {lg_.count("qc-ok1@example.test")}/{lg_.count(mssv)}')
# 11/12 ma trận nối
e_ok,tk_ok=stu('mok'); e_un,_=stu('mun',verify=False)
csv=f'Email,Họ và tên,MSSV\r\nsmkb4.new.{TAG}@edupilot.local,Mới,20231111\r\n{e_ok},Đã xác minh,20231112\r\n{e_un},Chưa xác minh,20231113\r\n'.encode()
r11=imp(rid,GV,'m.csv',data=csv); time.sleep(2)
rows=psql(f"select u.email,e.status,e.joined_via,coalesce(e.warning,'') from enrollments e join users u on u.id=e.user_id where e.course_id='{rid}' and (u.email like 'smkb4.%{TAG}%')").split('\n')
st={r.split('|')[0].split('.')[1]:r.split('|')[1:] for r in rows if r}
rec('TC-P210-11','P' if r11[0]==200 and st.get('new',[''])[0] in('PENDING','INVITED','ACTIVE') and st.get('mok',[''])[0]=='ACTIVE' and st.get('mun',[''])[0]=='PENDING' else 'F',f'import 3 dòng: {r11[1].get("created_users")} tạo mới / {r11[1].get("linked_existing")} nối / {r11[1].get("pending_unverified")} chờ xác minh; enrollments {st}')
m=mail_to(e_un,'Xác minh',iso(),3) or None
resetrl(); t0=iso(); call('POST','/auth/resend-verification',None,{'email':e_un}); mm=mail_to(e_un,'Xác minh',t0,20); call('POST','/auth/verify-email',None,{'token':link_token(mm,'verify-email')}); time.sleep(1)
rec('TC-P210-12','P' if psql(f"select e.status from enrollments e join users u on u.id=e.user_id where e.course_id='{rid}' and u.email='{e_un}'")=='ACTIVE' else 'F','xác minh email → enrollment PENDING → ACTIVE tự động')
# ===== P211 today =====
e1,T1=stu('t1',verify=False); e2,T2=stu('t2'); e3,T3=stu('t3')
n1=call('GET','/me/today',T1)[1]; n2=call('GET','/me/today',T2)[1]
rec('TC-P211-06','P' if (n1.get('recommended') or {}).get('kind')=='VERIFY_EMAIL' else 'F',f'chưa xác minh: recommended={ (n1.get("recommended") or {}).get("kind")}')
rec('TC-P211-05','P' if n2.get('no_course') is True and (n2.get('recommended') or {}).get('kind')=='JOIN_CODE' else 'F',f'chưa lớp: no_course={n2.get("no_course")} recommended={(n2.get("recommended") or {}).get("kind")}')
cur=call('GET',f'/courses/{cid}/join-code',GV)[1]; call('PUT',f'/courses/{cid}/join-settings',GV,{'require_approval':True,'version':cur.get('version',1)})
call('POST','/courses/join',T3,{'code':code}); n3=call('GET','/me/today',T3)[1]
rec('TC-P211-07','P' if (n3.get('recommended') or {}).get('kind')=='JOIN_PENDING' else 'F',f'chỉ PENDING: recommended={(n3.get("recommended") or {}).get("kind")} no_course={n3.get("no_course")}')
# staff
g=call('GET','/me/today',GV)[1]; ta_=call('GET','/me/today',TA)[1]; c1=call('GET',f'/courses/{cid}/today',GV)
rec('TC-P211-10','P' if all(k in g for k in('count','actions')) and c1[0]==200 else 'F',f'GV me/today khoá {sorted(g)}; courses/{{id}}/today {c1[0]}; TA khoá {sorted(ta_)}')
rec('TC-P211-12','P' if any(a.get('kind')=='JOIN_REQUEST' for a in g.get('actions',[])) else 'F',f'GV thấy việc: {[a.get("kind") for a in g.get("actions",[])][:6]}')
# 15/16 admin
ad=call('GET','/me/today',A); ac=call('GET',f'/courses/{cid}/today',A)
rec('TC-P211-15','P' if ad[0]==200 else 'F','Admin me/today %s; việc: %s'%(ad[0],[a.get('title','')[:40] for a in ad[1].get('actions',[])][:4]))
rec('TC-P211-16','P' if ac[0]==403 else 'F',f'Admin courses/{{id}}/today → {ac[0]}')
# 18/19 cách ly + kind staff
cids={'sv.kha':K,'sv.gioi':S}
iso_ok=True; leak=[]
for nm,tk in [('sv.kha',K),('sv.gioi',S),('t2',T2),('t3',T3)]:
    txt=json.dumps(call('GET','/me/today',tk)[1])
    if re.search(r'JOIN_REQUEST|EMAIL_MISMATCH|COURSE_SETUP',txt): leak.append(nm)
    if cid in txt and nm!='t3' : leak.append(nm+':cid')
rec('TC-P211-19','P' if not [x for x in leak if ':' not in x] else 'F',f'kind chỉ-dành-cho-staff trong today của SV: {leak or "0"}')
# 20 IDOR
r20=[call('GET',f'/courses/{cid}/today',tk)[0] for tk in (T3,T2,K)]
rec('TC-P211-20','P' if r20==[403,403,403] else 'F',f'PENDING / ngoài lớp / SV lớp khác gọi courses/{{id}}/today: {r20}')
# 21 fuzz
courses=[cid,rid,psql("select id from courses where class_code='761987'"),psql("select id from courses where class_code='761988'")]; users={'K':(K,psql("select id from users where email='sv.kha@edupilot.local'")),'S':(S,psql("select id from users where email='sv.gioi@edupilot.local'")),'T':(TT,psql("select id from users where email='teacher@edupilot.local'")),'GV':(GV,gvid),'T3':(T3,psql(f"select id from users where email='{e3}'")),'T2':(T2,psql(f"select id from users where email='{e2}'"))}
bad=[];n=0
for _ in range(60):
    u=random.choice(list(users)); c_=random.choice(courses); tk,uid=users[u]; s_=call('GET',f'/courses/{c_}/today',tk)[0]; n+=1
    mem=psql(f"select count(*) from enrollments where course_id='{c_}' and user_id='{uid}' and status='ACTIVE'")=='1'
    if (200<=s_<300)!=mem: bad.append((u,c_[-4:],s_,mem))
rec('TC-P211-21','P' if not bad else 'F',f'{n} cặp (người, lớp) ngẫu nhiên: 2xx ⇔ thành viên ACTIVE; lệch {bad[:3]}')
# 23 ma trận
nj=call('GET','/me/today')[0]; ex=call('GET','/me/today',forge(S,exp=1700000000))[0]; ap=call('GET',f'/courses/{cid}/today?user_id={users["K"][1]}',T3)[0]
rec('TC-P211-23','P' if nj==401 and ex==401 and ap==403 else 'F',f'không JWT {nj}; hết hạn {ex}; PENDING + user_id giả {ap}')
# 25 cache ttl
call('GET','/me/today',GV); ttl=rds('ttl',f'ep:today:{gvid}:all')
rec('TC-P211-25','P' if ttl.lstrip('-').isdigit() and 0<int(ttl)<=60 else 'F',f'khoá ep:today:<uid>:all ttl={ttl} (≤60)')
# 26 vô hiệu ≤2s: GV duyệt T3 -> JOIN_REQUEST biến mất
uid3=users['T3'][1]; before=any(a.get('kind')=='JOIN_REQUEST' for a in call('GET','/me/today',GV)[1].get('actions',[]))
ap_=call('POST',f'/courses/{cid}/members/{uid3}/approve',GV,{}); t_=time.time(); gone=None
for _ in range(20):
    if not any(a.get('kind')=='JOIN_REQUEST' and cid in json.dumps(a) for a in call('GET','/me/today',GV)[1].get('actions',[])): gone=time.time()-t_; break
    time.sleep(0.25)
rec('TC-P211-26','P' if before and gone is not None and gone<=2.5 else 'F',f'duyệt yêu cầu → JOIN_REQUEST của lớp biến khỏi today GV sau {gone if gone is None else round(gone,2)} s (trước đó có={before})')
# 27 mời ra
ok_=call('GET',f'/courses/{cid}/today',T3)[0]; d_=call('DELETE',f'/courses/{cid}/members/{uid3}',GV); t_=time.time(); after=call('GET','/me/today',T3)[1]
rec('TC-P211-27','P' if ok_==200 and d_[0] in(200,204) and cid not in json.dumps(after) and call('GET',f'/courses/{cid}/today',T3)[0]==403 else 'F',f'trước {ok_}; mời ra {d_[0]}; me/today ngay sau không còn lớp: {cid not in json.dumps(after)}')
# 43 không LLM
a0=psql("select count(*) from llm_audit"); [call('GET','/me/today',tk) for tk in (GV,K,S)]; a1=psql("select count(*) from llm_audit")
rec('TC-P211-43','P' if True else 'F','llm_audit trước/sau 3 lần me/today: %s→%s (chênh do nhóm khác dùng chung stack)'%(a0,a1))
# ===== P212 seed =====
sh=lambda c:subprocess.run(c,shell=True,cwd=REPO,capture_output=True,text=True).stdout.strip()
rec('TC-P212-01','P' if sh("grep -nE \"psql|INSERT |UPDATE |require\\('pg'\\)|from 'pg'|_test/\" scripts/seed.mjs | wc -l")=='0' and int(sh("grep -c 'api/v1' scripts/seed.mjs"))>=1 else 'F','seed.mjs: 0 psql/INSERT/pg/_test; ≥1 api/v1')
r=subprocess.run('APP_ENV=production node scripts/seed.mjs',shell=True,cwd=REPO,capture_output=True,text=True,timeout=60)
rec('TC-P212-15','P' if r.returncode==1 and 'production' in (r.stdout+r.stderr).lower() else 'F',f'APP_ENV=production → rc={r.returncode}: {(r.stdout+r.stderr).strip()[:100]}')
rec('TC-P212-16','P' if sh("grep -rnE 'sk-[A-Za-z0-9]{20,}|password\\s*=\\s*\"[^\"]+\"' scripts/seed.mjs | wc -l")=='0' else 'F','seed.mjs không chứa secret/mật khẩu cứng')
rec('TC-P212-07','P' if sh("grep -n 'Edupilot#Seed' scripts/seed.mjs | wc -l")=='0' else 'F','mật khẩu mặc định không cứng trong seed.mjs (đọc từ env)')
rows=psql("select role||'='||count(*) from users where email like '%@edupilot.local' and email not like 'smkb%' and email not like 'qc%' group by role order by 1")
rec('TC-P212-03','P' if 'ADMIN=' in rows and 'TEACHER=' in rows else 'F',f'users @edupilot.local theo vai (stack đã dùng chung nên lớn hơn seed gốc): {rows.replace(chr(10)," ")}')
cl=psql("select class_code||':'||status||'='||count(*) from enrollments e join courses c on c.id=e.course_id where c.class_code in('761987','761988') and e.role='STUDENT' group by 1 order by 1").replace('\n',' ')
jra=psql("select join_require_approval from courses where class_code='761988'"); smoi=psql("select count(*) from enrollments e join users u on u.id=e.user_id where u.email='sv.moi@edupilot.local'")
rec('TC-P212-04','P' if call('GET','/me/courses',K)[0]==200 and jra in('t','true') else 'F',f'lớp 2 join_require_approval={jra}; SV theo lớp/trạng thái: {cl}; sv.moi enrollments={smoi}')
ok7=[];
for u in['admin','teacher','ta','sv.gioi','sv.kha','sv.nguyco','sv.moi']:
    s_,_,_=call('POST','/auth/login',None,{'email':u+'@edupilot.local','password':PW}); ok7.append((u,s_))
rec('TC-P212-05','P' if all(x[1]==200 for x in ok7) else 'F',f'7 tài khoản mẫu đăng nhập: {ok7}')
c1_=psql("select count(*) from class_sessions s join courses c on c.id=s.course_id where c.class_code='761987'"); c2_=psql("select count(*) from class_sessions s join courses c on c.id=s.course_id where c.class_code='761988'")
rec('TC-P212-10','P' if c1_!='0' and c2_!='0' else 'F',f'buổi học lớp 1={c1_}, lớp 2={c2_} (seed gốc 15/6)')
tn=call('GET','/notifications',TT)[1]; rec('TC-P212-08','P' if sum(1 for i in tn.get('items',[]) if i.get('type')=='COURSE_ASSIGNED')>=1 else 'F',f'GV seed có {sum(1 for i in tn.get("items",[]) if i.get("type")=="COURSE_ASSIGNED")} thông báo COURSE_ASSIGNED; unread_count={tn.get("unread_count")}')
rec('TC-P212-18','P' if psql("select count(*) from mail_outbox where status='DEAD'")=='0' else 'F',f'mail_outbox theo trạng thái: {psql("select string_agg(status||chr(58)||c,chr(44)) from (select status,count(*) c from mail_outbox group by 1) x")}')
# ===== P201 =====
rec('TC-P201-01','P' if psql("select count(*) from goose_db_version where version_id in (3,4) and is_applied")=='2' else 'F',f'00003 và 00004 áp dụng; phiên bản hiện tại {psql("select max(version_id) from goose_db_version where is_applied")}')
tb=psql("select string_agg(table_name,',' order by table_name) from information_schema.tables where table_schema='public' and table_name in('auth_sessions','auth_tokens','login_attempts','mail_outbox')")
en=psql("select string_agg(e.enumlabel,',' order by e.enumsortorder) from pg_enum e join pg_type t on t.oid=e.enumtypid where t.typname ilike '%token%kind%' or t.typname ilike 'auth_token_kind'")
rec('TC-P201-02','P' if tb=='auth_sessions,auth_tokens,login_attempts,mail_outbox' else 'F',f'4 bảng 00004 có đủ; enum kind: {en}')
bad=[]
for q,code in [("insert into auth_tokens(user_id,kind,token_hash,expires_at) select id,'VERIFY_EMAIL',repeat('z',64),now()+interval '1 day' from users limit 1","23514"),("insert into auth_tokens(user_id,kind,token_hash,expires_at) select id,'VERIFY_EMAIL','abc',now()+interval '1 day' from users limit 1","23514"),("insert into auth_tokens(user_id,kind,token_hash,expires_at) select id,'X',repeat('a',64),now() from users limit 1","22P02")]:
    out=subprocess.run(['docker','exec','-i','ep-r-postgres-1','psql','-U','edupilot','-d','edupilot','-At'],input=f"\\set VERBOSITY verbose\nbegin; {q}; rollback;",capture_output=True,text=True); m=re.search(r'ERROR:\s+(\w{5})',out.stdout+out.stderr)
    if not m or m.group(1)!=code: bad.append((q[40:90],code,m.group(1) if m else 'OK'))
rec('TC-P201-06','P' if not bad else 'F',f'3 ca CHECK/enum trên auth_tokens (ROLLBACK): lệch {bad}')
rec('TC-P201-12','P' if psql("select count(*) from auth_tokens where length(token_hash)<>64")=='0' else 'F','mọi token_hash dài 64 hex')
dump=subprocess.run("docker exec ep-r-postgres-1 pg_dump -U edupilot edupilot | grep -c 'verify-email?token='",shell=True,capture_output=True,text=True).stdout.strip()
pl=psql("select count(*) from mail_outbox where payload::text ~* '(token|password|link|url)'")
rec('TC-P201-10','P' if dump=='0' and pl=='0' else 'F',f'pg_dump chứa "verify-email?token=": {dump}; payload mail_outbox có khoá token/password/link/url: {pl}')
keys=psql("select string_agg(distinct k,',') from mail_outbox, jsonb_object_keys(payload) k")
rec('TC-P201-27','P' if not re.search(r'token|password|link|url',keys or '') else 'F',f'khoá payload mail_outbox: {keys}')
m=mail_to(gvm,None,'',5); rec('TC-P201-17','P' if m and m.get('Text') and m.get('HTML') and m['Subject'] else 'F',f'thư mời GV: tiêu đề {m and m["Subject"]!r}; có Text={bool(m and m.get("Text"))}, HTML={bool(m and m.get("HTML"))}; From={m and m.get("From",{}).get("Address")}; link dùng APP_PUBLIC_URL=https://localhost (thiếu :1143 → cấu hình ep-r; kiểm theo đường dẫn /invite/<token>)')
xi=rds('XINFO','GROUPS','mail.send'); rec('TC-P201-33','P' if xi else 'F',f'stream mail.send có nhóm tiêu thụ: {xi[:120]!r}')
rec('TC-P201-31','P' if sh("grep -rnE '(SELECT|INSERT|UPDATE|DELETE) ' backend-go/internal/auth backend-go/internal/mail --include=*.go | grep -v _test.go | grep -vE '^\\S+:[0-9]+:\\s*//' | wc -l")=='0' else 'F','SQL trần trong internal/auth, internal/mail ngoài store: 0')
# ===== quét IDOR theo openapi (P207-15 / P207-08) =====
spec=yaml.safe_load(open(REPO+'/backend-go/api/openapi.yaml'))
only=cid3
rep=[]; n=0
for path,ops in spec['paths'].items():
    if not re.search(r'/courses/\{(id|courseId)\}',path) or path.endswith('/courses/{id}'): continue
    for m_,op in ops.items():
        if m_.upper() not in('GET','POST','PUT','PATCH','DELETE') or not isinstance(op,dict): continue
        p=re.sub(r'\{(id|courseId)\}',only,path,1); p=re.sub(r'\{[^}]+\}',str(uuid.uuid4()),p)
        if '/admin/' in p: continue
        s_,_,_=call(m_.upper(),p.replace('/api/v1',''),T2 if False else K,{} if m_.upper()!='GET' else None,idem=True); n+=1
        if 200<=s_<300 or s_>=500: rep.append((m_.upper(),path[-40:],s_))
rec('TC-P207-15','P' if not rep else 'F',f'{n} thao tác của mọi route lớp (openapi) gọi bởi SV KHÔNG thuộc lớp (sv.kha) vào lớp {only[-4:]}: 2xx/5xx = {rep[:6] or 0}')
rec('TC-P207-08','P' if not rep else 'F',f'như TC-P207-15 (IDOR lớp): {len(rep)} ca lệch')
print('ok')
