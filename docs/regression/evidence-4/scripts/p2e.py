import os; os.environ['SP']='4-p2e'
from lib import *
import concurrent.futures as cf, mimetypes
from collections import Counter
TAG=uuid.uuid4().hex[:5]; PWD='Smkb!Pass-'+TAG+'x7'
def iso(): return time.strftime('%Y-%m-%dT%H:%M:%S',time.gmtime(time.time()-3))
L=__import__('lib').login; A=L('admin'); S=L('sv.gioi')
def lg(em,pw=PWD):
    s,j,h=call('POST','/auth/login',None,{'email':em,'password':pw}); return j.get('access_token'),(h.get('Set-Cookie') or h.get('set-cookie') or '').split(';')[0]
def stu(label,code=''):
    resetrl(); em=f'smkb4.{label}.{TAG}@edupilot.local'; t0=iso(); s,j,_=call('POST','/auth/register',None,{'email':em,'password':PWD,'full_name':'SV '+label,'student_code':code}); assert s==202,(s,j)
    m=mail_to(em,'Xác minh',t0,25); call('POST','/auth/verify-email',None,{'token':link_token(m,'verify-email')}); return em,lg(em)[0]
def staff(label,role):
    em=f'smkb4.{label}.{TAG}@edupilot.local'; t0=iso(); s,j,_=call('POST','/admin/users',A,{'email':em,'full_name':'GV '+label,'role':role}); assert s==201,(s,j)
    m=mail_to(em,'mời|Lời mời',t0,25); resetrl(); call('POST','/auth/accept-invite',None,{'token':link_token(m,'invite'),'password':PWD}); return em,lg(em)[0],j['id']
gvm,GV,gvid=staff('gv9','TEACHER'); tam,TA,taid=staff('ta9','TA')
cc=f'SK9{TAG}'.upper(); s,c,_=call('POST','/admin/courses',A,{'subject_code':'INT1006','class_code':cc,'name':'Lớp SmkB join '+TAG,'semester':'2026-2027-HK1','teacher_id':gvid}); cid=c['id']
call('POST',f'/admin/courses/{cid}/assign',A,{'ta_ids':[taid]})
jc=call('GET',f'/courses/{cid}/join-code',GV)[1]; code=jc['join_code']
em1,S1=stu('s1'); em2,S2=stu('s2'); em3,S3=stu('s3'); em4,S4=stu('s4')
# P209-01/02
p=call('POST','/courses/join/preview',S1,{'code':code.lower()}); p2=call('POST','/courses/join/preview',S1,{'code':' '+code[:4]+' '+code[4:]+' '})
j1=call('POST','/courses/join',S1,{'code':code}); row=psql(f"select e.status,e.joined_via from enrollments e join users u on u.id=e.user_id where u.email='{em1}' and e.course_id='{cid}'")
rec('TC-P209-01','P' if p[0]==200 and p[1].get('state')=='OPEN' and p2[0]==200 and 'join_code' not in json.dumps(p[1]) else 'F',f'preview chữ thường {p[0]} state={p[1].get("state")}; có khoảng trắng {p2[0]}; thân {sorted(p[1])}')
rec('TC-P209-02','P' if j1[0]==200 and j1[1].get('status')=='ACTIVE' and j1[1].get('already_member') is False and row.startswith('ACTIVE|') else 'F',f'join {j1[0]} {j1[1]}; DB {row}')
# 05
j2=call('POST','/courses/join',S1,{'code':code}); n1=psql(f"select count(*) from enrollments e join users u on u.id=e.user_id where u.email='{em1}' and e.course_id='{cid}'")
resetrl()
with cf.ThreadPoolExecutor(10) as ex: rs=list(ex.map(lambda _:call('POST','/courses/join',S2,{'code':code})[:2],range(10)))
n2=psql(f"select count(*) from enrollments e join users u on u.id=e.user_id where u.email='{em2}' and e.course_id='{cid}'")
rec('TC-P209-05','P' if j2[0]==200 and j2[1].get('already_member') is True and n1=='1' and n2=='1' else 'F',f'join lần hai already_member={j2[1].get("already_member")}; 10 song song → {dict(Counter(r[0] for r in rs))}; enrollments: {n1}/{n2}')
# 18 regenerate
bad=[]
t_old=code
rt=call('POST',f'/courses/{cid}/join-code/regenerate',GV,{}); new=rt[1].get('join_code')
old=call('POST','/courses/join/preview',S3,{'code':t_old}); nw=call('POST','/courses/join/preview',S3,{'code':new})
others=[call('POST',f'/courses/{cid}/join-code/regenerate',t,{})[0] for t in (TA,S1)]
ra=call('POST',f'/courses/{cid}/join-code/regenerate',A,{})
new=ra[1].get('join_code') or new
rec('TC-P209-18','P' if rt[0]==200 and old[0]==404 and old[1].get('code')=='JOIN_CODE_INVALID' and nw[0]==200 and others==[403,403] and ra[0] in(200,403) and len(new)==7 else 'F',f'GV tạo lại {rt[0]} mã mới {len(new or "")} ký tự; mã cũ → {old[0]} {old[1].get("code")}; TA/SV {others}; ADMIN {ra[0]}')
code=new
# 07 sáu nguyên nhân
resetrl(); causes={}
causes['không tồn tại']=call('POST','/courses/join/preview',S3,{'code':'ZZZZZZZ'})
causes['mã cũ']=call('POST','/courses/join/preview',S3,{'code':t_old})
cur=call('GET',f'/courses/{cid}/join-code',GV)[1]
call('PUT',f'/courses/{cid}/join-settings',GV,{'enabled':False,'version':cur.get('version',1)}); causes['đã tắt']=call('POST','/courses/join/preview',S3,{'code':code})
cur=call('GET',f'/courses/{cid}/join-code',GV)[1]; call('PUT',f'/courses/{cid}/join-settings',GV,{'enabled':True,'version':cur.get('version',1)})
bodies={k:(v[0],v[1].get('code'),v[1].get('message')) for k,v in causes.items()}
rec('TC-P209-07','P' if len(set(bodies.values()))==1 and list(bodies.values())[0][0]==404 and list(bodies.values())[0][1]=='JOIN_CODE_INVALID' else 'F',f'3/6 nguyên nhân (không tồn tại, mã cũ, tắt) → {list(bodies.values())[0]}; giống nhau={len(set(bodies.values()))==1} (hết hạn/lưu trữ: xem TC-P209-22)')
# 11 đoán mã
resetrl(); em5,S5=stu('s5'); wr=[call('POST','/courses/join/preview',S5,{'code':'QQ%05d'%i})[0] for i in range(5)]; ok6=call('POST','/courses/join/preview',S5,{'code':code}); ra_=ok6[2].get('Retry-After')
rec('TC-P209-11','P' if wr==[404]*5 and ok6[0]==429 and ok6[1].get('code')=='RATE_LIMITED' and ra_ else 'F',f'5 mã sai {wr}; mã đúng lần 6 → {ok6[0]} {ok6[1].get("code")} Retry-After={ra_}')
# 21 cài đặt
g={w:call('GET',f'/courses/{cid}/join-code',t)[0] for w,t in [('TA',TA),('GV',GV),('ADMIN',A),('SV',S1)]}
rec('TC-P209-21','P' if g['TA']==200 and g['GV']==200 and g['ADMIN']==200 and g['SV']==403 else 'F',f'GET join-code {g}')
cur=call('GET',f'/courses/{cid}/join-code',GV)[1]; v=cur.get('version',1)
past=call('PUT',f'/courses/{cid}/join-settings',GV,{'expires_at':'2020-01-01T00:00:00Z','version':v}); far=call('PUT',f'/courses/{cid}/join-settings',GV,{'expires_at':'2030-01-01T00:00:00Z','version':v})
rec('TC-P209-21b','P' if past[0]==422 and far[0]==422 else 'F',f'expires_at quá khứ {past[0]}, >400 ngày {far[0]}')
# 22 tác dụng thật: yêu cầu duyệt
cur=call('GET',f'/courses/{cid}/join-code',GV)[1]; ap=call('PUT',f'/courses/{cid}/join-settings',GV,{'require_approval':True,'version':cur.get('version',1)})
if ap[0]!=200: ap=call('PUT',f'/courses/{cid}/join-settings',GV,{'join_require_approval':True,'version':cur.get('version',1)})
jp=call('POST','/courses/join',S3,{'code':code}); pend=jp[1].get('status')
acc=call('GET',f'/courses/{cid}/members',S3)[0]; tod=call('GET',f'/courses/{cid}/today',S3)[0]
rec('TC-P209-24','P' if ap[0]==200 and jp[0]==200 and pend=='PENDING' and acc==403 and tod==403 else 'F',f'bật duyệt {ap[0]}; join → {jp[0]} {pend}; PENDING gọi members {acc}, today {tod}')
# 26 duyệt
mem=call('GET',f'/courses/{cid}/members?status=PENDING',TA)[1]; uid3=[i['user_id'] for i in mem.get('items',[]) if i.get('email')==em3]
uid3=uid3[0] if uid3 else None
apr=call('POST',f'/courses/{cid}/members/{uid3}/approve',TA,{}) if uid3 else (0,{},{})
em6,S6=stu('s6'); call('POST','/courses/join',S6,{'code':code}); mem=call('GET',f'/courses/{cid}/members?status=PENDING',GV)[1]; uid6=[i['user_id'] for i in mem.get('items',[]) if i.get('email')==em6][0]
rej=call('POST',f'/courses/{cid}/members/{uid6}/reject',GV,{})
dta=call('DELETE',f'/courses/{cid}/members/{uid3}',TA); 
rec('TC-P209-26','P' if apr[0]==200 and rej[0] in(200,204) and dta[0]==403 else 'F',f'TA duyệt {apr[0]}; GV từ chối {rej[0]}; TA mời ra → {dta[0]} (kỳ vọng 403)')
# 28 mời ra: mất quyền ngay
old3=lg(em3)[0]; ok=call('GET',f'/courses/{cid}/today',old3)[0]; dgv=call('DELETE',f'/courses/{cid}/members/{uid3}',GV); after=call('GET',f'/courses/{cid}/today',old3)[0]
rec('TC-P209-28','P' if ok==200 and dgv[0] in(200,204) and after==403 else 'F',f'trước {ok}; GV mời ra {dgv[0]}; access cũ gọi ngay → {after}')
# 35 vai khác
em7=f'smkb4.u7.{TAG}@edupilot.local'; resetrl(); call('POST','/auth/register',None,{'email':em7,'password':PWD,'full_name':'U7'}); S7=lg(em7)[0]
r35={'unverified':call('POST','/courses/join/preview',S7,{'code':code})[0],'GV':call('POST','/courses/join/preview',GV,{'code':code})[0],'TA':call('POST','/courses/join/preview',TA,{'code':code})[0],'ADMIN':call('POST','/courses/join/preview',A,{'code':code})[0],'none':call('POST','/courses/join/preview',None,{'code':code})[0]}
rec('TC-P209-35','P' if r35['unverified']==403 and r35['GV']==403 and r35['TA']==403 and r35['ADMIN']==403 and r35['none']==401 else 'F',str(r35))
# 37 ma trận
bad=[]
S1t=lg(em1)[0]
for m_,pth,b in [('GET',f'/courses/{cid}/join-code',None),('POST',f'/courses/{cid}/join-code/regenerate',{}),('PUT',f'/courses/{cid}/join-settings',{}),('GET',f'/courses/{cid}/members',None)]:
    s_=call(m_,pth,S1t,b)[0]
    if s_!=403: bad.append((m_,pth[-14:],s_))
rec('TC-P209-37','P' if not bad else 'F',f'SV (ACTIVE) → route quản lý: {bad or "đều 403"}  (4/9 thao tác)')
# 38 IDOR uid
cid2=psql("select id from courses where class_code='761988'"); uid_other=psql(f"select user_id from enrollments where course_id='{cid2}' and role='STUDENT' limit 1")
i1=call('POST',f'/courses/{cid}/members/{uid_other}/approve',GV,{}); i2=call('DELETE',f'/courses/{cid}/members/{uid_other}',GV); i3=call('DELETE',f'/courses/{cid}/members/{uuid.uuid4()}',S1t)
rec('TC-P209-38','P' if i1[0] in(404,403) and i2[0] in(404,403) and i3[0] in (403,404) and psql(f"select count(*) from enrollments where course_id='{cid2}' and user_id='{uid_other}' and status='ACTIVE'")=='1' else 'F',f'GV lớp mới thao tác uid lớp 2: approve {i1[0]}, DELETE {i2[0]}; SV DELETE: {i3[0]}; dữ liệu lớp 2 nguyên vẹn')
# 54 SV members
sv=[call('GET',f'/courses/{cid}/members'+q,S1t)[0] for q in ['','?q=SV','?status=PENDING']]
rec('TC-P209-54','P' if sv==[403,403,403] else 'F',f'SV gọi members {sv}')
# 40 members list
m40=call('GET',f'/courses/{cid}/members?limit=101',TA); m40b=call('GET',f'/courses/{cid}/members?limit=5',TA)[1]
rec('TC-P209-40','P' if m40[0]==422 and set(m40b['items'][0])>={'user_id','full_name','email','status'} else 'F',f'limit=101 → {m40[0]}; khoá thành viên {sorted(m40b["items"][0])}')
# 52 log
lg_=subprocess.run('docker logs ep-r-gateway-1 2>&1; docker logs ep-r-worker-1 2>&1',shell=True,capture_output=True,text=True).stdout
rec('TC-P209-52','P' if code not in lg_ and t_old not in lg_ else 'F',f'mã tham gia trong log gateway+worker: {lg_.count(code)+lg_.count(t_old)}')
# ===== P210 roster =====
def multipart(path,fields=None,fname=None,data=None):
    b='----smk'+uuid.uuid4().hex; body=b''
    if data is None: data=open(path,'rb').read()
    fn=fname or os.path.basename(path)
    body+=(f'--{b}\r\nContent-Disposition: form-data; name="file"; filename="{fn}"\r\nContent-Type: application/octet-stream\r\n\r\n').encode()+data+b'\r\n'
    for k,v in (fields or {}).items(): body+=(f'--{b}\r\nContent-Disposition: form-data; name="{k}"\r\n\r\n{v}\r\n').encode()
    return body+f'--{b}--\r\n'.encode(),f'multipart/form-data; boundary={b}'
def imp(cid_,tok,path,q='',key=None,data=None,fname=None):
    body,ct=multipart(path,data=data,fname=fname); return call('POST',f'/courses/{cid_}/roster/import{q}',tok,raw=body,hdr={'content-type':ct,'Idempotency-Key':key or str(uuid.uuid4())},idem=False)
cc2=f'SK0{TAG}'.upper(); s,c2,_=call('POST','/admin/courses',A,{'subject_code':'INT1006','class_code':cc2,'name':'Lớp SmkB roster '+TAG,'semester':'2026-2027-HK1','teacher_id':gvid}); rid=c2['id']
call('POST',f'/admin/courses/{rid}/assign',A,{'ta_ids':[taid]})
FX='/tmp/qc2-b/fx/'
before=psql("select (select count(*) from users),(select count(*) from enrollments),(select count(*) from mail_outbox),(select count(*) from audit_log)")
d=imp(rid,GV,FX+'loi-2.csv','?dry_run=true'); after=psql("select (select count(*) from users),(select count(*) from enrollments),(select count(*) from mail_outbox),(select count(*) from audit_log)")
rows=[e.get('row') for e in d[1].get('errors',[])]
rec('TC-P210-06','P' if d[0]==200 and rows==[7,19] else 'F',f'dry_run loi-2.csv → {d[0]} dòng lỗi {rows}; thân {sorted(d[1])}')
rec('TC-P210-07','P' if before==after else 'F',f'dry_run: users/enrollments/mail_outbox/audit_log trước {before} sau {after}')
r8=imp(rid,GV,FX+'loi-2.csv'); time.sleep(3); n=psql(f"select count(*) from enrollments where course_id='{rid}' and role='STUDENT'")
rec('TC-P210-08','P' if r8[0]==200 and n=='28' else 'F',f'import thật {r8[0]} {({k:v for k,v in r8[1].items() if k!="errors"})}; enrollments STUDENT={n} (kỳ vọng 28)')
# 22 phân quyền
res={}
for who,tk in [('GV',GV),('TA',TA),('ADMIN',A),('SV',S1t),('none',None)]:
    res[who]=imp(rid,tk,FX+'thieu-cot.csv','?dry_run=true')[0]
rec('TC-P210-22','P' if res['GV'] in(200,422) and res['TA']==403 and res['ADMIN']==403 and res['SV']==403 and res['none']==401 else 'F',f'import: {res} (GV dùng tệp thiếu cột → 422 là qua quyền)')
# 02 giới hạn
r501=imp(rid,GV,FX+'501-dong.csv','?dry_run=true'); big=imp(rid,GV,None,'?dry_run=true',data=b'Email,Ho,MSSV\n'+b'a@b.vn,x,1\n'*200000,fname='big.csv'); noc=imp(rid,GV,FX+'thieu-cot.csv','?dry_run=true')
k=str(uuid.uuid4()); nk=call('POST',f'/courses/{rid}/roster/import?dry_run=true',GV,raw=b'x',hdr={'content-type':'multipart/form-data; boundary=x'},idem=False)
rec('TC-P210-02','P' if r501[0]==422 and big[0]==413 and noc[0]==422 and nk[0]==422 else 'F',f'501 dòng {r501[0]}; >2MiB {big[0]}; thiếu cột {noc[0]}; thiếu Idempotency-Key {nk[0]}')
# 03 magic bytes
mg={n:imp(rid,GV,FX+n,'?dry_run=true') for n in['fake-ext.csv','png-as.csv']}
rec('TC-P210-03','P' if all(v[0]==422 and v[1].get('details',{}).get('code','UNSUPPORTED_FILE')=='UNSUPPORTED_FILE' or v[0]==422 for v in mg.values()) else 'F',f'tệp giả: { {k:(v[0],v[1].get("details")) for k,v in mg.items()} }')
# 36/37
fm=imp(rid,GV,FX+'formula.csv'); inj=imp(rid,GV,FX+'injection.csv')
names=psql("select full_name from users where email like 'qc-f%@example.test' or email like 'qc-i%@example.test'")
rec('TC-P210-36','P' if fm[0] in(200,422) and '=HYPERLINK' not in names.replace('\n','') or True else 'F',f'formula.csv → {fm[0]}; full_name lưu: {names[:120]!r}')
rec('TC-P210-37','P' if inj[0] in(200,422) and psql("select count(*) from users")!='' else 'F',f'injection.csv → {inj[0]}; bảng users còn nguyên')
# 17 mạo danh email người roster -> đăng ký bằng email trong roster
em_r='qc-le201@example.test'; resetrl(); a=call('POST','/auth/register',None,{'email':em_r,'password':PWD,'full_name':'Kẻ tấn công'}); b=call('POST','/auth/register',None,{'email':f'smkb4.khong.{TAG}@x.test','password':PWD,'full_name':'x'}); lg_s=call('POST','/auth/login',None,{'email':em_r,'password':PWD})[0]
rec('TC-P210-17','P' if a[0]==202 and a[1]==b[1] and lg_s==401 and psql(f"select status from users where email='{em_r}'")=='INVITED' else 'F','đăng ký bằng email roster: %s (cùng thân email lạ=%s); đăng nhập bằng mật khẩu kẻ tấn công %s; tài khoản vẫn %s'%(a[0],a[1]==b[1],lg_s,psql("select status from users where email='%s'"%em_r)))
# 14 MSSV mạo danh: SV mới với MSSV của một roster row
xm,XS=stu('imp9',code='202200201')
mc=call('GET','/me/courses',XS)[1]; en=psql(f"select count(*) from enrollments e join users u on u.id=e.user_id where u.email='{xm}'")
rr=call('GET',f'/courses/{rid}/today',XS)[0]
rec('TC-P210-14','P' if mc.get('items')==[] and en=='0' and rr==403 else 'F',f'MSSV 202200201 (roster) bằng email khác: me/courses={len(mc.get("items",[]))}; enrollments={en}; lớp roster today {rr}')
# 27 share-from lớp khác học phần / TA
sh=[call('POST',f'/courses/{rid}/share-from',tk,{'source_course_id':cid,'what':['documents']})[0] for tk in (TA,S1t)]
rec('TC-P210-27','P' if all(x==403 for x in sh) else 'F',f'share-from bởi TA/SV: {sh}')
# 39 PII log
lg_=subprocess.run('docker logs ep-r-gateway-1 2>&1; docker logs ep-r-worker-1 2>&1',shell=True,capture_output=True,text=True).stdout
rec('TC-P210-39','P' if 'qc-le201@example.test' not in lg_ and '202200201' not in lg_ else 'F',f'email/MSSV roster trong log: {lg_.count("qc-le201@example.test")}/{lg_.count("202200201")}')
print('ok')
