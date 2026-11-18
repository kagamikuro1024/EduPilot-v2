import os; os.environ['SP']='4-t8'
from smkT import *
import hashlib
def ck(h): return (h.get('Set-Cookie') or h.get('set-cookie') or '').split(';')[0]
A_=A()
gv1m,GV1,gv1=staff('g1','TEACHER'); gv2m,GV2,gv2=staff('g2','TEACHER'); ta1m,TA1,ta1=staff('t1','TA'); ta2m,TA2,ta2=staff('t2','TA'); ta3m,TA3,ta3=staff('t3','TA')
svm,SV=stu('s1'); svid=uid(svm); svbm,SVB=stu('s2'); svbid=uid(svbm)
# ---- P208-04 teacher_id trong thân
cc=f'ST04{TAG}'.upper()
s,c,_=call('POST','/admin/courses',A_,{'subject_code':'INT1006','class_code':cc,'name':'Lớp SmkT 04','semester':'2026-2027-HK1','teacher_id':gv1})
en=psql(f"select role_in_course||'|'||status||'|'||joined_via from enrollments where course_id='{c.get('id')}' and user_id='{gv1}'")
bad=[]
for nm,tid in [('SV',svid),('TA',ta1),('lạ',str(uuid.uuid4())),('rác','abc')]:
    cn=f'ST04{nm[0]}{TAG}'.upper().replace('Ạ','A').replace('Á','A')
    s2,_,_=call('POST','/admin/courses',A_,{'subject_code':'INT1006','class_code':cn,'name':'Lớp SmkT 04x','semester':'2026-2027-HK1','teacher_id':tid})
    if s2!=422 or psql(f"select count(*) from courses where class_code='{cn}'")!='0': bad.append((nm,s2))
rec('TC-P208-04','P' if s==201 and en=='TEACHER|ACTIVE|ADMIN' and not bad and psql(f"select count(*) from courses where class_code='{cc}'")=='1' else 'F',f'teacher_id hợp lệ → 201, enrollment GV {en} trong cùng lệnh; sai vai (SV/TA)/uuid lạ/rác → 422 và 0 lớp được tạo (rollback): lệch {bad or "không"}')
# ---- khóa lớp dùng chung cho các ca sau
cid=mkcourse('A',gv1,ta1); jc_=call('GET',f'/courses/{cid}/join-code',GV1)[1]
# ---- P208-13 GV cũ mất quyền ngay
m1=call('GET',f'/courses/{cid}/members',GV1)[0]
as2=call('POST',f'/admin/courses/{cid}/assign',A_,{'teacher_id':gv2}); m2=call('GET',f'/courses/{cid}/members',GV1)[0]; m3=call('GET',f'/courses/{cid}/members',GV2)[0]
st=psql(f"select user_id=(select id from users where email='{gv1m}')::text||'|'||status||'|'||joined_via from enrollments where course_id='{cid}' and role_in_course='TEACHER' order by status")
acts=psql(f"select count(*) from enrollments where course_id='{cid}' and role_in_course='TEACHER' and status='ACTIVE'")
g2=psql(f"select status||'|'||joined_via from enrollments where course_id='{cid}' and user_id='{gv2}'"); g1s=psql(f"select status from enrollments where course_id='{cid}' and user_id='{gv1}'")
rec('TC-P208-13','P' if m1==200 and as2[0]==200 and m2==403 and m3==200 and g1s=='REMOVED' and g2=='ACTIVE|ADMIN' and acts=='1' else 'F',f'GV1 trước {m1}; gán GV2 {as2[0]}; GV1 (access cũ, không đăng nhập lại) {m2}; GV2 {m3}; DB GV1={g1s}, GV2={g2}, số GV ACTIVE={acts}')
# ---- P208-14 TA thay toàn bộ
call('POST',f'/admin/courses/{cid}/assign',A_,{'ta_ids':[ta1,ta2]}); ta_set=lambda: sorted(psql(f"select u.email from enrollments e join users u on u.id=e.user_id where e.course_id='{cid}' and e.role_in_course='TA' and e.status='ACTIVE'").split('\n'))
b0=ta_set(); r1=call('POST',f'/admin/courses/{cid}/assign',A_,{'ta_ids':[ta3]}); b1=ta_set()
rem=psql(f"select count(*) from enrollments where course_id='{cid}' and role_in_course='TA' and status='REMOVED'")
r2=call('POST',f'/admin/courses/{cid}/assign',A_,{}); b2=ta_set(); r3=call('POST',f'/admin/courses/{cid}/assign',A_,{'ta_ids':[]}); b3=ta_set()
rec('TC-P208-14','P' if b0==sorted([ta1m,ta2m]) and b1==[ta3m] and int(rem)>=2 and r2[0] in(200,422) and (b2==b1 if r2[0]==200 else True) else 'F',f'[TA1,TA2]→{len(b0)} TA; assign {{ta_ids:[TA3]}} {r1[0]} → {len(b1)} TA ({b1==[ta3m]}), TA1/TA2 REMOVED={rem}; assign {{}} → {r2[0]} giữ nguyên={b2==b1}; assign {{ta_ids:[]}} → {r3[0]} còn {len(b3)} TA (hành vi ghi nhận: rỗng = {"gỡ hết" if not b3 else "giữ nguyên"})')
# ---- P208-15 nguyên tử + chỉ người mới có outbox
ob=lambda: psql(f"select count(*) from outbox where topic='course.assigned' and payload->>'course_id'='{cid}'")
o0=ob(); ra=call('POST',f'/admin/courses/{cid}/assign',A_,{'teacher_id':gv2,'ta_ids':[ta3]}); time.sleep(1); o1=ob()   # gán lại đúng người đang có
tch=psql(f"select user_id from enrollments where course_id='{cid}' and role_in_course='TEACHER' and status='ACTIVE'")
rb=call('POST',f'/admin/courses/{cid}/assign',A_,{'teacher_id':gv1,'ta_ids':[ta1,svid]}); o2=ob()   # nửa chừng lỗi: TA sai vai → 422, GV phải còn GV2
tch2=psql(f"select user_id from enrollments where course_id='{cid}' and role_in_course='TEACHER' and status='ACTIVE'")
nrow=psql(f"select count(*) from outbox where topic='course.assigned' and payload->>'course_id'='{cid}' and payload->>'user_id' in ('{gv1}','{ta1}')")
rc=call('POST',f'/admin/courses/{cid}/assign',A_,{'teacher_id':gv1}); time.sleep(1); o3=ob()
roles=psql(f"select payload->>'role'||':'||count(*) from outbox where topic='course.assigned' and payload->>'course_id'='{cid}' group by 1 order by 1").replace('\n',' ')
rec('TC-P208-15','P' if ra[0]==200 and o1==o0 and rb[0]==422 and tch2==tch and o2==o1 and rc[0]==200 and int(o3)==int(o2)+1 else 'F',f'outbox course.assigned của lớp: ban đầu {o0}; gán lại đúng người đang có ({ra[0]}) → {o1} (không thêm); gán GV1 + TA sai vai → {rb[0]}, GV vẫn như cũ={tch2==tch}, outbox {o2} (rollback không để lại gì); gán GV1 hợp lệ → {o3} (+1, chỉ người mới); theo vai: {roles}; phần "ép outbox lỗi" do test Go TestAssignNotifyOnce/TestAssignTeacherNotifies (đạt ở lô go test)')
# ---- P208-17 ≤60s, đúng 1, nội dung
cid2=None
cn=f'ST17{TAG}'.upper(); s,c,_=call('POST','/admin/courses',A_,{'subject_code':'INT1006','class_code':cn,'name':f'Lớp SmkT 17 {TAG}','semester':'2026-2027-HK1'}); cid2=c['id']
t0=time.time(); call('POST',f'/admin/courses/{cid2}/assign',A_,{'teacher_id':gv2}); n=0
while time.time()-t0<70:
    n=int(psql(f"select count(*) from notifications where user_id='{gv2}' and type='COURSE_ASSIGNED' and link like '%{cid2}'") or 0)
    if n: break
    time.sleep(0.5)
dt=time.time()-t0; time.sleep(3)
row=psql(f"select title||'¦'||body||'¦'||link from notifications where user_id='{gv2}' and type='COURSE_ASSIGNED' and link like '%{cid2}'").split('¦'); n2=psql(f"select count(*) from notifications where user_id='{gv2}' and type='COURSE_ASSIGNED' and link like '%{cid2}'")
code=psql(f"select join_code from courses where id='{cid2}'"); ncls=psql(f"select name from courses where id='{cid2}'")
ok17=n2=='1' and row[0]==f'Bạn được phân công lớp {ncls} – {cn}' and row[1]==f'Mã tham gia: {code}. Chia sẻ mã hoặc đường dẫn https://localhost/join/{code} cho sinh viên.' and row[2]==f'/class/settings?course={cid2}'
api=call('GET','/notifications',GV2)[1]; inapi=any(cid2 in (i.get('link') or '') for i in api.get('items',[]))
rec('TC-P208-17','P' if ok17 and dt<=60 and inapi else 'F',f'gán GV2 → thông báo COURSE_ASSIGNED sau {dt:.1f}s (≤60 s); đúng {n2} dòng; tiêu đề/thân/link khớp SRS={ok17} ({row[0][:50]}…); GET /notifications của GV2 thấy={inapi}')
# ---- P208-19 giao lại cùng tin
oid=psql(f"select id from outbox where topic='course.assigned' and payload->>'course_id'='{cid2}' and payload->>'user_id'='{gv2}'")
x=sh(f"docker exec ep-r-redis-1 redis-cli XADD outbox.dispatch '*' outbox_id {oid} topic course.assigned").strip(); time.sleep(6)
n3=psql(f"select count(*) from notifications where user_id='{gv2}' and type='COURSE_ASSIGNED' and link like '%{cid2}'")
rec('TC-P208-19','P' if n3=='1' and x else 'F',f'XADD lại tin course.assigned (outbox_id={oid[:8]}…, entry {x}) vào stream outbox.dispatch của ep-r; sau 6 s notifications vẫn {n3} dòng (dedupe_key course.assigned:<outbox_id>); (kill -9 worker + XAUTOCLAIM không thực hiện vì là thay đổi toàn cục — dùng XADD lại như TC cho phép)')
# ---- P208-23 IDOR thông báo
nid=psql(f"select id from notifications where user_id='{gv2}' and type='COURSE_ASSIGNED' and link like '%{cid2}'")
a=call('POST',f'/notifications/{nid}/read',SVB,{}); b=call('POST',f'/notifications/{uuid.uuid4()}/read',SVB,{}); c_=call('POST','/notifications/rac/read',SVB,{}); lst=call('GET','/notifications',SVB)[1]; seen_other=any(i.get('id')==nid for i in lst.get('items',[]))
ra_=psql(f"select read_at is null from notifications where id='{nid}'")
rec('TC-P208-23','P' if a[0]==404 and b[0]==404 and a[1].get('code')==b[1].get('code') and not seen_other and ra_=='t' else 'F',f'SV-B đọc thông báo của GV2 → {a[0]} {a[1].get("code")}; id lạ → {b[0]} {b[1].get("code")} (cùng mã); id rác → {c_[0]}; GET /notifications của SV-B không có ID_GV={not seen_other}; read_at của ID_GV vẫn NULL={ra_=="t"}')
# ---- P208-27 tấn công
bad=[]
tok=GV1; sec=secret()
cls={'alg=none':forge(tok,alg='none'),'sửa role (không ký lại)':tok.split('.')[0]+'.'+__import__('base64').urlsafe_b64encode(json.dumps({**json.loads(__import__('base64').urlsafe_b64decode(tok.split('.')[1]+'==')),'role':'ADMIN'}).encode()).decode().rstrip('=')+'.'+tok.split('.')[2],'secret sai':forge(tok,sec='sai-secret-sai-secret-sai-secret-1234'),'rác':'abc.def.ghi'}
r27={}
for nm,t in cls.items(): r27[nm]=call('POST','/admin/courses',t,{'subject_code':'INT1006','class_code':'ZZ9'+TAG[:3],'name':'x','semester':'2026-2027-HK1'})[0]
r27['X-Role: ADMIN']=call('POST','/admin/courses',GV1,{'subject_code':'INT1006','class_code':'ZZ8'+TAG[:3],'name':'x','semester':'2026-2027-HK1'},hdr={'X-Role':'ADMIN'})[0]
r27['GV của lớp tự assign']=call('POST',f'/admin/courses/{cid}/assign',GV1,{'teacher_id':gv1})[0]; r27['GV2 (GV lớp) assign']=call('POST',f'/admin/courses/{cid}/assign',GV2,{'teacher_id':gv2})[0]
# vai DB ≠ JWT: đổi vai GV thành TA bằng API admin → token cũ mất hiệu lực
ver=psql(f"select version from users where id='{gv2}'"); pr=call('PATCH',f'/admin/users/{gv2}',A_,{'role':'TA','version':int(ver)}); r27['token cũ sau đổi vai GV→TA']=call('GET',f'/courses/{cid2}/members',GV2)[0]
n_new=psql(f"select count(*) from courses where class_code like 'ZZ_{TAG[:3]}'")
okv=all(v in(401,403) for v in r27.values()) and n_new=='0'
rec('TC-P208-27','P' if okv else 'F',f'POST /admin/courses & assign: {r27}; đổi vai GV2→TA qua API {pr[0]}; lớp ZZ* tạo ra={n_new}')
# ---- P208-40 Admin không thêm SV
oa=sh("cd ~/Documents/TA_Agent_qc2/backend-go && grep -nE 'POST.*(admin/courses/.*(students|members|enroll))' api/openapi.yaml | wc -l").strip()
ad=[call('POST',f'/admin/courses/{cid}/members',A_,{'user_id':svid})[0],call('POST',f'/admin/courses/{cid}/enroll',A_,{'user_id':svid})[0],call('POST',f'/admin/courses/{cid}/students',A_,{'user_id':svid})[0],call('POST',f'/admin/courses/{cid}/assign',A_,{'student_ids':[svid]})[0]]
ns=psql(f"select count(*) from enrollments where course_id='{cid}' and role_in_course='STUDENT'")
rec('TC-P208-40','P' if oa=='0' and all(x in(404,405,422) for x in ad) and ns=='0' else 'F',f'grep openapi POST …/admin/courses/…/(students|members|enroll) = {oa}; thử /members,/enroll,/students,assign{{student_ids}} → {ad}; enrollments STUDENT mới của lớp = {ns}; nút "Thêm sinh viên" kiểm ở TC-P208-31 (r4ui-admin)')
# ---- P208-10 lớp đã lưu trữ
cidL=mkcourse('L9',gv1,ta1); jcL=call('GET',f'/courses/{cidL}/join-code',GV1)[1]
call('PUT',f'/courses/{cidL}/join-settings',GV1,{'require_approval':True,'version':jcL.get('version',1)}); 
call('POST','/courses/join',SV,{'code':jcL['join_code']})   # SV vào PENDING
call('POST','/courses/join',SVB,{'code':jcL['join_code']})
snap=lambda: psql(f"select md5(string_agg(x,',' order by x)) from (select c::text x from courses c where id='{cidL}' union all select e::text from enrollments e where course_id='{cidL}' union all select d::text from class_sessions d where course_id='{cidL}') t")
ar=call('POST',f'/admin/courses/{cidL}/archive',A_,{}); s0=snap(); cur=call('GET',f'/admin/courses?q=ST{"L9"}{TAG}'.upper() if False else f'/admin/courses?q=STL9{TAG}',A_)[1]['items'][0]
ops={'sửa':('PUT',f'/admin/courses/{cidL}',A_,{'name':'x','subject_code':'INT1006','class_code':f'STL9{TAG}','semester':'2026-2027-HK1','version':cur['version']}),
'assign':('POST',f'/admin/courses/{cidL}/assign',A_,{'teacher_id':gv2}),'regenerate':('POST',f'/courses/{cidL}/join-code/regenerate',GV1,{}),
'join-settings':('PUT',f'/courses/{cidL}/join-settings',GV1,{'enabled':True,'version':jcL.get('version',1)+1}),
'approve':('POST',f'/courses/{cidL}/members/{svid}/approve',GV1,{}),'DELETE member':('DELETE',f'/courses/{cidL}/members/{svbid}',GV1,None),
'import roster':None,'share':('POST',f'/courses/{cidL}/share-from',GV1,{'source_course_id':cid,'what':['documents']}),'assistants':('PUT',f'/courses/{cidL}/assistants',GV1,{'ta_ids':[ta2]})}
r10={}
for k,v in ops.items():
    if v is None:
        body,ct=multipart(b'Email,Ho va ten,MSSV\r\nsmkt.arch@example.test,A,20259999\r\n','r.csv'); s_,j_,_=call('POST',f'/courses/{cidL}/roster/import',GV1,raw=body,hdr={'content-type':ct,'Idempotency-Key':str(uuid.uuid4())},idem=False)
    else: s_,j_,_=call(v[0],v[1],v[2],v[3])
    r10[k]=(s_,j_.get('code'))
s1=snap()
rec('TC-P208-10','P' if ar[0] in(200,204) and all(v==(409,'COURSE_ARCHIVED') for v in r10.values()) and s0==s1 else 'F',f'lưu trữ lớp {ar[0]}; thao tác ghi → {r10}; DB (courses+enrollments+sessions md5) trước/sau {s0==s1}')
print('done',TAG)
