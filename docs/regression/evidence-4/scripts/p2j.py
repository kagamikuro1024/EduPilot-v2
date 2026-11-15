import os,yaml,re; os.environ['SP']='4-p2j'
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

# ===== P207 =====
em1,T1=stu('p7a'); em2,T2=stu('p7b'); call('POST','/courses/join',T1,{'code':code})
mc={u:[(i['course']['class_code'],i.get('enrollment_status')) for i in call('GET','/me/courses',tk)[1].get('items',[])] for u,tk in [('sv.gioi',S),('sv.kha',K),('teacher',TT),('t2(chưa lớp)',T2),('t1(mới vào)',T1)]}
rec('TC-P207-19','P' if mc['t2(chưa lớp)']==[] and len(mc['t1(mới vào)'])==1 and len(mc['teacher'])>=2 and len(mc['sv.kha'])>=1 else 'F',f'me/courses: {mc}')
l101=call('GET','/me/courses?limit=101',S); r1=call('GET','/me/courses',S); et=r1[2].get('Etag'); r2=call('GET','/me/courses',S,hdr={'If-None-Match':et}) if et else (0,)
rec('TC-P207-21','P' if l101[0]==422 and et and r2[0]==304 else 'F',f'limit=101 → {l101[0]}; ETag {et}; If-None-Match → {r2[0]}')
d_sv=call('GET',f'/courses/{cid}',T1)[1]; d_gv=call('GET',f'/courses/{cid}',GV)[1]; d_ad=call('GET',f'/courses/{cid}',A)
rec('TC-P207-23','P' if 'join_code' not in d_sv and d_sv.get('my_role')=='STUDENT' and all(set(t)<={'full_name'} for t in d_sv.get('teachers',[{}])) and 'join_code' in d_gv and d_ad[0] in(200,403) else 'F',f'SV: join_code={"join_code" in d_sv}, my_role={d_sv.get("my_role")}, teachers keys={[sorted(t) for t in d_sv.get("teachers",[])][:1]}; GV có join_code={"join_code" in d_gv}; Admin {d_ad[0]}')
p=call('GET','/me/profile',T1)[1]; ver=p.get('version',1)
bad=[(k,call('PUT','/me/profile',T1,{k:v,'version':ver})[0]) for k,v in [('role','ADMIN'),('email','x@y.vn'),('id',str(uuid.uuid4())),('status','ACTIVE'),('full_name',''),('student_code','x')]]
bv=call('PUT','/me/profile',T1,{'full_name':'SV p7a mới','version':999})
sc=call('PUT','/me/profile',T1,{'full_name':'SV p7a mới','student_code':'ABC123456','version':ver})
rec('TC-P207-26','P' if all(x[1]==422 for x in bad) and bv[0]==409 and sc[0]==200 else 'F',f'trường chỉ đọc/sai định dạng → {bad}; sai version {bv[0]}; sửa hợp lệ {sc[0]}')
snap=psql(f"select count(*) from enrollments e join users u on u.id=e.user_id where u.email='{em1}' and e.student_code_snapshot='ABC123456'")
rec('TC-P207-25','P' if snap=='0' else 'F',f'đổi MSSV ở hồ sơ → student_code_snapshot của enrollments không đổi (khớp mới: {snap})')
st0=call('GET','/me/settings',T1); put1=call('PUT','/me/settings',T1,{'khoa_la':True}); other=call('GET',f'/users/{gvid}/settings',T1)[0]
rec('TC-P207-28','P' if st0[0]==200 and put1[0]==422 and other in(404,403,405) else 'F',f'GET {st0[0]} {sorted(st0[1])}; PUT khoá lạ {put1[0]}; GET settings người khác {other}')
no=[call('GET',p_)[0] for p_ in ('/me/courses','/me/profile','/me/settings',f'/courses/{cid}')]
rec('TC-P207-36','P' if no==[401]*4 else 'F',f'không JWT: {no}')
uidT1=psql(f"select id from users where email='{em1}'")
i1=call('GET',f'/courses/{cid}/members/{uidT1}',T1); i2=call('GET',f'/courses/{cid}/members/{gvid}',T1)
rec('TC-P207-38','P' if i1[0] in(403,404) and i2[0] in(403,404) else 'F',f'SV gọi members/{{uid}} của mình {i1[0]} / của GV {i2[0]} (route quản lý: 403)')
# 20 PENDING/REMOVED
cur=call('GET',f'/courses/{cid}/join-code',GV)[1]; call('PUT',f'/courses/{cid}/join-settings',GV,{'require_approval':True,'version':cur.get('version',1)}); call('POST','/courses/join',T2,{'code':code}); m2=[(i['course']['class_code'],i.get('enrollment_status')) for i in call('GET','/me/courses',T2)[1].get('items',[])]
rec('TC-P207-20','P' if any(x[1]=='PENDING' for x in m2) else 'F',f'SV vừa xin vào lớp cần duyệt: me/courses {m2}')
# P207-09/10 admin bị từ chối mặc định
adm={p_:call('GET',p_,A)[0] for p_ in (f'/courses/{cid}/members',f'/courses/{cid}/today',f'/courses/{cid}/sessions')}
rec('TC-P207-10','P' if all(v==403 for v in adm.values()) else 'F',f'Admin gọi route lớp Member/Staff: {adm} (kỳ vọng 403 ngoài route Manage)')
# ===== P208 =====
al=call('GET','/admin/courses?limit=5',A)[1]; ak=set(al['items'][0]) if al.get('items') else set()
rec('TC-P208-29','P' if {'id','class_code','subject_code','name','semester','status'}<=ak and 'join_code' not in ak else 'F',f'khoá mục lớp: {sorted(ak)}')
nl=call('GET','/notifications?limit=101',GV); n1=call('GET','/notifications?limit=1',GV)[1]; un=call('GET','/notifications?unread_only=true',GV)[1]
rec('TC-P208-24','P' if nl[0]==422 and len(n1.get('items',[]))<=1 and all(not i.get('read_at') for i in un.get('items',[])) else 'F',f'limit=101 {nl[0]}; limit=1 → {len(n1.get("items",[]))} mục (next_cursor={bool(n1.get("next_cursor"))}); unread_only: {len(un.get("items",[]))} mục chưa đọc')
c1=call('GET','/courses/%s/assistant-candidates?q=tah'%cid,GV); a1=call('PUT',f'/courses/{cid}/assistants',GV,{'ta_ids':[taid]}); a2=call('PUT',f'/courses/{cid}/assistants',TT,{'ta_ids':[taid]}); a3=call('PUT',f'/courses/{cid}/assistants',TA,{'ta_ids':[taid]}); a4=call('PUT',f'/courses/{cid}/assistants',S,{'ta_ids':[taid]})
rec('TC-P208-21','P' if a1[0]==200 and a2[0]==403 and a3[0]==403 and a4[0]==403 else 'F',f'GV lớp thay TA {a1[0]}; GV lớp khác {a2[0]}; TA {a3[0]}; SV {a4[0]}; candidates {c1[0]}')
n=psql(f"select count(*) from notifications where user_id='{taid}' and type='COURSE_ASSIGNED'"); tt=call('GET','/notifications',TA)[1]; ti=[i.get('title','') for i in tt.get('items',[]) if i.get('type')=='COURSE_ASSIGNED']
rec('TC-P208-18','P' if int(n)>=1 and any('trợ giảng' in x for x in ti) else 'F',f'TA nhận thông báo COURSE_ASSIGNED={n}: {ti[:1]}')
n=psql(f"select count(*) from notifications where user_id='{gvid}' and type='COURSE_ASSIGNED'"); tg=[i.get('title','') for i in call('GET','/notifications',GV)[1].get('items',[]) if i.get('type')=='COURSE_ASSIGNED']
rec('TC-P208-17','P' if int(n)>=3 and any('phân công' in x for x in tg) else 'F',f'GV nhận COURSE_ASSIGNED={n} cho 3 lớp được gán ngay khi mở lớp; tiêu đề {tg[:1]} (≤ 60 s: có ngay)')
other=call('GET','/notifications',S)[1]; nid=tg and [i['id'] for i in call('GET','/notifications',GV)[1]['items'] if i.get('type')=='COURSE_ASSIGNED'][0]
r1=call('POST',f'/notifications/{nid}/read',S,{}); r2=call('POST',f'/notifications/{uuid.uuid4()}/read',S,{})
rec('TC-P208-23','P' if r1[0]==404 and r2[0]==404 and r1[1].get('code')==r2[1].get('code') else 'F',f'SV đọc thông báo của GV → {r1[0]} = id lạ {r2[0]} (cùng mã)')
# 13 GV cũ mất quyền ngay
gv2m,GV2,gv2id=staff('gvh2','TEACHER'); okb=call('GET',f'/courses/{cid3}/members',GV)[0]; call('POST',f'/admin/courses/{cid3}/assign',A,{'teacher_id':gv2id}); a_=call('GET',f'/courses/{cid3}/members',GV)[0]; b_=call('GET',f'/courses/{cid3}/members',GV2)[0]
rec('TC-P208-13','P' if okb==200 and a_==403 and b_==200 else 'F',f'GV1 trước {okb}; gán GV2; GV1 (access cũ) sau {a_}; GV2 {b_}')
call('POST',f'/admin/courses/{cid3}/archive',A,{}); ar=[call('PUT',f'/admin/courses/{cid3}',A,{'subject_code':'INT9999','class_code':'XX'+TAG,'name':'x','semester':'2026-2027-HK1','version':99})[0],call('POST',f'/admin/courses/{cid3}/assign',A,{'teacher_id':gvid})[0],call('POST',f'/courses/{cid3}/join-code/regenerate',GV2,{})[0],call('PUT',f'/courses/{cid3}/join-settings',GV2,{'enabled':False,'version':1})[0]]
rec('TC-P208-10','P' if all(x==409 for x in ar[1:]) else 'F',f'ghi vào lớp đã lưu trữ: assign/regenerate/join-settings → {ar[1:]} (sửa với version sai: {ar[0]})')
print('ok')
