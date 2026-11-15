import os; os.environ['SP']='4-p2c'
from lib import *
import concurrent.futures as cf
from collections import Counter
TAG=uuid.uuid4().hex[:5]; PWD='Smkb!Pass-'+TAG+'x7'
def iso(): return time.strftime('%Y-%m-%dT%H:%M:%S',time.gmtime(time.time()-3))
L=__import__('lib').login; A=L('admin'); T=L('teacher'); TA=L('ta'); S=L('sv.gioi')
def lg(em,pw=PWD):
    s,j,h=call('POST','/auth/login',None,{'email':em,'password':pw}); return s,j,(h.get('Set-Cookie') or h.get('set-cookie') or '').split(';')[0]
# ===== P206 =====
em=f'smkb4.gv1.{TAG}@edupilot.local'; t0=iso()
s,j,_=call('POST','/admin/users',A,{'email':em,'full_name':'GV SmkB Một','role':'TEACHER'})
uid=j.get('id'); shape=set(['id','email','full_name','role','status','version'])<=set(j) and j.get('status')=='INVITED' and not any(k in j for k in ['password','token','password_hash'])
rec('TC-P206-01','P' if s==201 and shape else 'F',f'{s} {sorted(j)} status={j.get("status")}')
bad=[]
for r in ['STUDENT','ADMIN','admin','Teacher','',None,'SUPERUSER',['ADMIN']]:
    n0=psql("select count(*) from users"); s2,j2,_=call('POST','/admin/users',A,{'email':f'smkb4.bad{abs(hash(str(r)))%9999}.{TAG}@x.vn','full_name':'B','role':r})
    if s2!=422: bad.append((r,s2))
s2,_,_=call('POST','/admin/users',A,{'email':f'smkb4.badx.{TAG}@x.vn','full_name':'B','role':'TEACHER','status':'ACTIVE','password':'x'})
if s2!=422: bad.append(('extra',s2))
rec('TC-P206-02','P' if not bad and psql(f"select count(*) from users where email like 'smkb4.bad%.{TAG}@%'")=='0' else 'F',f'9 vai/trường lạ → 422, 0 user mới' if not bad else str(bad))
d1=call('POST','/admin/users',A,{'email':em,'full_name':'X','role':'TEACHER'}); d2=call('POST','/admin/users',A,{'email':em.upper(),'full_name':'X','role':'TEACHER'}); k=str(uuid.uuid4()); body={'email':f'smkb4.gv2.{TAG}@edupilot.local','full_name':'GV Hai','role':'TA'}
e1=call('POST','/admin/users',A,body,hdr={'Idempotency-Key':k}); e2=call('POST','/admin/users',A,body,hdr={'Idempotency-Key':k}); e3=call('POST','/admin/users',A,body,idem=False)
rec('TC-P206-03','P' if d1[0]==409 and d2[0]==409 and d1[1].get('details',{}).get('field')=='email' and e1[0]==201 and e2[2].get('Idempotent-Replayed')=='true' and e3[0]==422 and psql(f"select count(*) from users where email='{body['email']}'")=='1' else 'F',f'email trùng {d1[0]}/{d2[0]}(hoa-thường); idem replay={e2[2].get("Idempotent-Replayed")}; thiếu key {e3[0]}')
m=mail_to(em,'mời|Lời mời',t0,25); tok=link_token(m,'invite')
rec('TC-P206-04','P' if m and tok else 'F',f'thư mời tới GV: {m and m["Subject"]!r}')
prv=call('POST','/auth/tokens/preview',None,{'kind':'INVITE','token':tok})
ac=call('POST','/auth/accept-invite',None,{'token':tok,'password':PWD}); ac2=call('POST','/auth/accept-invite',None,{'token':tok,'password':PWD})
st=psql(f"select role,status,email_verified_at is not null from users where email='{em}'")
rec('TC-P206-05','P' if ac[0]==200 and 'access_token' in ac[1] and st=='TEACHER|ACTIVE|t' else 'F',f'accept-invite {ac[0]}; DB {st}; preview {prv[0]}')
rec('TC-P206-06','P' if ac2[0]==410 and ac2[1].get('details',{}).get('reason')=='used' else 'F',f'dùng lại token mời → {ac2[0]} {ac2[1].get("details")}')
em3=f'smkb4.gv3.{TAG}@edupilot.local'; call('POST','/admin/users',A,{'email':em3,'full_name':'GV Ba','role':'TEACHER'}); m3=mail_to(em3,'mời|Lời mời',t0,25); tk3=link_token(m3,'invite')
resetrl()
with cf.ThreadPoolExecutor(30) as ex: rs=list(ex.map(lambda _:call('POST','/auth/accept-invite',None,{'token':tk3,'password':PWD})[0],range(30)))
sess=psql(f"select count(*) from auth_sessions where user_id=(select id from users where email='{em3}')")
rec('TC-P206-07','P' if rs.count(200)==1 and all(x in(410,429) for x in rs if x!=200) and sess=='1' else 'F',f'30 song song {dict(Counter(rs))}; phiên={sess}')
rec('TC-P206-09','P' if psql(f"select count(*) from auth_tokens where token_hash='{tok}'")=='0' else 'F','token mời không ở DB (chỉ băm)')
# 13 resend
em4=f'smkb4.gv4.{TAG}@edupilot.local'; r0=call('POST','/admin/users',A,{'email':em4,'full_name':'GV Bốn','role':'TEACHER'}); uid4=r0[1]['id']; resetrl(); n0=mail_count(em4)
a=call('POST',f'/admin/users/{uid4}/resend-invite',A,{}); b=call('POST',f'/admin/users/{uid4}/resend-invite',A,{}); act=call('POST',f'/admin/users/{uid}/resend-invite',A,{}); tchr=call('POST',f'/admin/users/{uid4}/resend-invite',T,{})
rec('TC-P206-13','P' if a[0]==200 and b[0]==429 and act[0] in(409,422) and tchr[0]==403 else 'F',f'resend {a[0]},{b[0]}; người ACTIVE {act[0]}; TEACHER {tchr[0]}')
# 15/16/17 khoá
sg=lg(em)[1]['access_token']; cur=call('GET','/admin/users?q='+em,A)[1]['items'][0]; ver=cur['version']
dis=call('PATCH',f'/admin/users/{uid}',A,{'status':'DISABLED','version':ver}); t1=time.time(); time.sleep(1.1); old=call('GET','/me/courses',sg)
rv=psql(f"select string_agg(distinct revoked_reason,',') from auth_sessions where user_id='{uid}' and revoked_at is not null")
rec('TC-P206-15','P' if dis[0]==200 and old[0]==401 and 'ACCOUNT_DISABLED' in rv else 'F',f'PATCH DISABLED {dis[0]}; access cũ {old[0]} {old[1].get("code")}; revoked_reason={rv}')
l1=lg(em); l2=lg(em,'Sai-mk-123x')
rec('TC-P206-16','P' if l1[0]==403 and l1[1].get('code')=='ACCOUNT_DISABLED' and l2[0]==401 else 'F',f'mật khẩu đúng → {l1[0]} {l1[1].get("code")}; sai → {l2[0]}')
cur=call('GET','/admin/users?q='+em,A)[1]['items'][0]; en=call('PATCH',f'/admin/users/{uid}',A,{'status':'ACTIVE','version':cur['version']}); bv=call('PATCH',f'/admin/users/{uid}',A,{'status':'DISABLED','version':1})
rec('TC-P206-17','P' if en[0]==200 and lg(em)[0]==200 and bv[0]==409 else 'F',f'mở khoá {en[0]}, đăng nhập lại {lg(em)[0]}; sai version {bv[0]}')
# 19 vai
cur=call('GET','/admin/users?q='+em,A)[1]['items'][0]; sgn=lg(em)[1]['access_token']
rt=call('PATCH',f'/admin/users/{uid}',A,{'role':'TA','version':cur['version']}); cur=call('GET','/admin/users?q='+em,A)[1]['items'][0]
ra=call('PATCH',f'/admin/users/{uid}',A,{'role':'ADMIN','version':cur['version']}); rs_=call('PATCH',f'/admin/users/{uid}',A,{'role':'STUDENT','version':cur['version']})
admin_id=psql("select id from users where email='admin@edupilot.local'"); ac_=call('GET','/admin/users?q=admin@edupilot',A)[1]['items'][0]
sf=call('PATCH',f'/admin/users/{admin_id}',A,{'status':'DISABLED','version':ac_['version']}); sr=call('PATCH',f'/admin/users/{admin_id}',A,{'role':'TEACHER','version':ac_['version']})
rec('TC-P206-19','P' if rt[0]==200 and ra[0]==422 and rs_[0]==422 and sf[0] in(403,409,422) and sr[0] in(403,409,422) and psql("select status from users where email='admin@edupilot.local'")=='ACTIVE' else 'F',f'TEACHER→TA {rt[0]}; →ADMIN {ra[0]}; →STUDENT {rs_[0]}; tự khoá {sf[0]}; tự đổi vai {sr[0]}; admin còn ACTIVE')
# 21 ma trận
ops=[('GET','/admin/users',None),('POST','/admin/users',{}),('PATCH',f'/admin/users/{uid}',{}),('POST',f'/admin/users/{uid4}/resend-invite',{})]
bad=[]
for m_,p,b in ops:
    for who,tk in [('TEACHER',T),('TA',TA),('STUDENT',S),('none',None)]:
        s_,j_,_=call(m_,p,tk,b)
        if (who=='none' and s_!=401) or (who!='none' and (s_!=403 or j_.get('details',{}).get('reason')!='role')): bad.append((who,m_,s_))
rec('TC-P206-21','P' if not bad else 'F',f'4 thao tác × 4 danh tính: chỉ ADMIN {bad}')
# 22 vai JWT thắng DB
suv=L('sv.kha'); psql("update users set role='ADMIN' where email='sv.kha@edupilot.local'"); r22=call('GET','/admin/users',suv); psql("update users set role='STUDENT' where email='sv.kha@edupilot.local'")
rec('TC-P206-22','P' if r22[0]==403 else 'F',f'JWT STUDENT + DB role=ADMIN → /admin/users {r22[0]} (vai trong JWT thắng); đã trả vai về STUDENT')
# 23/24
fk=forge(A,sec='x'*40); n=forge(A,alg='none'); b=[call('GET','/admin/users',t)[0] for t in(fk,n)]
c24=[call('PATCH',f'/admin/users/{admin_id}',T,{'status':'DISABLED','version':1})[0],call('GET','/admin/users?q=nguyen',S)[0],call('GET','/admin/users',S,hdr={'X-Role':'ADMIN','X-User-Id':admin_id})[0]]
rec('TC-P206-23','P' if b==[401,401] else 'F',f'token giả {b}'); rec('TC-P206-24','P' if c24==[403,403,403] else 'F',f'IDOR {c24}')
# 26/27 list
pg=call('GET','/admin/users?limit=30',A)[1]; items=list(pg['items']); cur_=pg.get('next_cursor'); n_=1
while cur_ and n_<40: pg=call('GET','/admin/users?limit=30&cursor='+urllib.parse.quote(cur_),A)[1]; items+=pg['items']; cur_=pg.get('next_cursor'); n_+=1
l101=call('GET','/admin/users?limit=101',A); fr=call('GET','/admin/users?role=TEACHER&q=gv%20smkb',A)[1]; fk_=call('GET','/admin/users?q=Nguyen',A)[1]
rec('TC-P206-26','P' if len(items)==len(set(i['id'] for i in items)) and l101[0]==422 and all(i['role']=='TEACHER' for i in fr['items']) else 'F',f'đi hết {len(items)} người ({n_} trang, không trùng); limit=101 → {l101[0]}; lọc role+q {len(fr["items"])}')
keys=set(items[0]); rec('TC-P206-27','P' if keys=={'id','email','full_name','role','status','last_login_at','version'} and not any(x in json.dumps(items).lower() for x in ['student_code','password','ics_token','failed']) else 'F',f'khoá {sorted(keys)}')
# 31 chỉ CLI tạo ADMIN
resetrl(); r31=call('POST','/auth/register',None,{'email':f'smkb4.adm.{TAG}@x.vn','password':PWD,'full_name':'x','role':'ADMIN'}); r31b=call('POST','/admin/users',A,{'email':f'smkb4.adm2.{TAG}@x.vn','full_name':'x','role':'ADMIN'})
rec('TC-P206-31','P' if r31[0]==422 and r31b[0]==422 else 'F',f'register role ADMIN {r31[0]}; admin/users role ADMIN {r31b[0]}')
lg_=subprocess.run('docker logs ep-r-gateway-1 2>&1; docker logs ep-r-worker-1 2>&1',shell=True,capture_output=True,text=True).stdout
rec('TC-P206-42','P' if em not in lg_ and tok not in lg_ and PWD not in lg_ else 'F',f'email GV/token/mật khẩu trong log gateway+worker: {lg_.count(em)}/{lg_.count(tok)}/{lg_.count(PWD)}')
# ===== P208 =====
cc=f'SK{TAG}'.upper()
k=str(uuid.uuid4()); cb={'subject_code':'INT1006','class_code':cc,'name':'Lớp SmkB '+TAG,'semester':'2026-2027-HK1'}
c1=call('POST','/admin/courses',A,cb,hdr={'Idempotency-Key':k}); c2=call('POST','/admin/courses',A,cb,hdr={'Idempotency-Key':k})
cid=c1[1].get('id'); n=psql(f"select count(*) from courses where class_code='{cc}'")
rec('TC-P208-01','P' if c1[0]==201 and c2[2].get('Idempotent-Replayed')=='true' and n=='1' and c1[1].get('status')=='ACTIVE' else 'F',f'{c1[0]}; replay={c2[2].get("Idempotent-Replayed")}; count={n}; join_code={c1[1].get("join_code")} ')
r2=[call('POST','/admin/courses',A,dict(cb,class_code=cc.lower()))[0],call('POST','/admin/courses',A,dict(cb,class_code='ab'))[0],call('POST','/admin/courses',A,dict(cb,class_code='a b'))[0],call('POST','/admin/courses',A,dict(cb,class_code='X'*21))[0],call('POST','/admin/courses',A,dict(cb,class_code='<x>'))[0],call('POST','/admin/courses',A,dict(cb,class_code='SKB1'+TAG.upper(),semester='2026'))[0],call('POST','/admin/courses',A,dict(cb,class_code='SKB2'+TAG.upper(),name=''))[0]]
rec('TC-P208-02','P' if r2[0]==409 and all(x==422 for x in r2[1:]) else 'F',f'class_code trùng(hoa-thường) {r2[0]}; còn lại {r2[1:]}')
cc2=f'SJ{TAG}'.upper(); bodyr=[dict(cb,class_code=cc2)]*10
with cf.ThreadPoolExecutor(10) as ex: rr=list(ex.map(lambda b:call('POST','/admin/courses',A,b)[0],bodyr))
rec('TC-P208-03','P' if rr.count(201)==1 and rr.count(409)==9 and psql(f"select count(*) from courses where class_code='{cc2}'")=='1' else 'F',f'10 song song cùng class_code: {dict(Counter(rr))}')
# assign
tid=psql("select id from users where email='teacher@edupilot.local'"); gid=uid; tr_=psql(f"select id from users where email='{em}'")
as1=call('POST',f'/admin/courses/{cid}/assign',A,{'teacher_id':tid}); 
bad=[]
for nm,body in [('SV',{'teacher_id':psql("select id from users where email='sv.gioi@edupilot.local'")}),('TA',{'teacher_id':psql("select id from users where email='ta@edupilot.local'")}),('ADMIN',{'teacher_id':admin_id}),('rác',{'teacher_id':'abc'}),('lạ',{'teacher_id':str(uuid.uuid4())}),('ta_ids chứa TEACHER',{'ta_ids':[tid]})]:
    s_,_,_=call('POST',f'/admin/courses/{cid}/assign',A,body)
    if s_!=422: bad.append((nm,s_))
rec('TC-P208-12','P' if as1[0]==200 and not bad else 'F',f'assign hợp lệ {as1[0]}; sai vai/rác → 422 trừ {bad}')
# 13 GV cũ mất quyền ngay: teacher (lớp mới) -> gán GV mới tr_
mem1=call('GET',f'/courses/{cid}/members',T)[0]
as2=call('POST',f'/admin/courses/{cid}/assign',A,{'teacher_id':tr_}); mem2=call('GET',f'/courses/{cid}/members',T)[0]
gvt=lg(em)[1]['access_token']; mem3=call('GET',f'/courses/{cid}/members',gvt)[0]
rec('TC-P208-13','P' if mem1==200 and as2[0]==200 and mem2==403 and mem3==200 else 'F',f'GV1 trước {mem1}; gán GV2 {as2[0]}; GV1 (access cũ) sau {mem2}; GV2 {mem3}')
# 17 thông báo
t_=time.time(); n_=0
for _ in range(30):
    n_=psql(f"select count(*) from notifications where user_id='{tr_}' and type='COURSE_ASSIGNED'")
    if n_ and int(n_)>=1: break
    time.sleep(1)
ns=call('GET','/notifications',gvt)[1]; ti=[i.get('title') for i in ns.get('items',[])]
rec('TC-P208-17','P' if int(n_)==1 and any('phân công' in (x or '') for x in ti) else 'F',f'thông báo COURSE_ASSIGNED sau {time.time()-t_:.1f}s: count={n_}; tiêu đề {ti[:1]}')
# 23 IDOR thông báo
nid=ns['items'][0]['id'] if ns.get('items') else None
sv=call('POST',f'/notifications/{nid}/read',S,{}); sv2=call('POST',f'/notifications/{uuid.uuid4()}/read',S,{}); 
rec('TC-P208-23','P' if nid and sv[0]==404 and sv2[0]==404 and sv[1].get('code')==sv2[1].get('code') else 'F',f'SV đọc thông báo của GV2 → {sv[0]}; id lạ → {sv2[0]}')
# 26 ma trận admin courses
bad=[]
for m_,p,b in [('GET','/admin/courses',None),('POST','/admin/courses',{}),('PUT',f'/admin/courses/{cid}',{}),('POST',f'/admin/courses/{cid}/assign',{}),('POST',f'/admin/courses/{cid}/archive',{})]:
    for who,tk in [('TEACHER',T),('TA',TA),('STUDENT',S),('none',None)]:
        s_,j_,_=call(m_,p,tk,b)
        if (who=='none' and s_!=401) or (who!='none' and s_!=403): bad.append((who,m_,p[-8:],s_))
own=call('POST',f'/admin/courses/{cid}/assign',gvt,{'teacher_id':tr_})[0]
rec('TC-P208-26','P' if not bad and own==403 else 'F',f'5 thao tác × 4 danh tính {bad}; GV của lớp tự assign {own}')
# 9/10 archive
cur=call('GET',f'/admin/courses?q={cc}',A)[1]['items'][0]
pu=call('PUT',f'/admin/courses/{cid}',A,dict(cb,version=cur.get('version',1),name='Lớp SmkB sửa')) ; pbad=call('PUT',f'/admin/courses/{cid}',A,dict(cb,version=999))
ar=call('POST',f'/admin/courses/{cid}/archive',A,{}); ar2=call('POST',f'/admin/courses/{cid}/archive',A,{})
wr=[call('PUT',f'/admin/courses/{cid}',A,dict(cb,version=cur.get('version',1)+5))[0], call('POST',f'/admin/courses/{cid}/assign',A,{'teacher_id':tid})[0], call('POST',f'/courses/{cid}/join-code/regenerate',gvt,{})[0]]
rec('TC-P208-09','P' if pu[0]==200 and pbad[0]==409 and ar[0] in(200,204) else 'F',f'sửa {pu[0]}; sai version {pbad[0]} {pbad[1].get("code")}; archive {ar[0]}; lần hai {ar2[0]}')
rec('TC-P208-10','P' if all(x==409 for x in wr) else 'F',f'ghi vào lớp đã lưu trữ: {wr} (kỳ vọng 409 COURSE_ARCHIVED)')
# 40 admin không thêm SV
ad=[call('POST',f'/admin/courses/{cid}/members',A,{'user_id':psql("select id from users where email='sv.gioi@edupilot.local'")})[0],call('POST',f'/admin/courses/{cid}/students',A,{})[0],call('POST',f'/admin/courses/{cid}/enroll',A,{})[0]]
rec('TC-P208-40','P' if all(x in(404,405,409,422) for x in ad) and psql(f"select count(*) from enrollments where course_id='{cid}' and role='STUDENT'")=='0' else 'F',f'đường Admin thêm SV: {ad}')
print('ok',TAG,cc)
