# SmkT2: kiểm chứng trực tiếp bằng HTTP trên ep-r cho GATEP2-02, 10 (phần API), 25, 29, 46, 48
import os,sys,json,random; exec(open(os.path.join(os.path.dirname(os.path.abspath(__file__)),'smkt2lib.py')).read())
from smkt2lib import rec
TAG=uuid.uuid4().hex[:5]; random.seed(TAG)
def jn(t,code,pv=False): return call('POST','/courses/join/preview' if pv else '/courses/join',t,{'code':code})
def multipart(data,fname='roster.csv'):
    b='----q2'+uuid.uuid4().hex
    return (f'--{b}\r\nContent-Disposition: form-data; name="file"; filename="{fname}"\r\nContent-Type: application/octet-stream\r\n\r\n').encode()+data+f'\r\n--{b}--\r\n'.encode(),f'multipart/form-data; boundary={b}'
def imp(cid,tok,csv,q=''):
    body,ct=multipart(csv.encode()); return call('POST',f'/courses/{cid}/roster/import{q}',tok,raw=body,hdr={'content-type':ct,'Idempotency-Key':str(uuid.uuid4())},idem=False)
def mssv(): return str(random.randint(700000000,799999999))
pool=student_pool(40); P=iter(pool)
def tk(): u=next(P); return u,L(u)
resetrl(); resetjoin()
# ---------- GATEP2-02: 7 hành vi ----------
res={}
uS,tS=tk()
adm=[call('GET','/admin/users',tS)[0],call('GET','/admin/courses',tS)[0],call('POST','/admin/users',tS,{'email':'x%s@e.local'%TAG,'full_name':'x','role':'TA'})[0],call('PATCH','/admin/users/'+psql("select id from users where email='sv.kha@edupilot.local'"),tS,{'full_name':'x','version':1})[0]]
res['1 SV gọi /admin/*']=adm; ok1=adm==[403]*4
r=call('POST','/auth/register',None,{'email':f'smkt2.g02.{TAG}@edupilot.local','password':'Smkt2!Pass-'+TAG+'x7','full_name':'G02','role':'TEACHER'})
r2=call('POST','/auth/register',None,{'email':f'smkt2.g02b.{TAG}@edupilot.local','password':'Smkt2!Pass-'+TAG+'x7','full_name':'G02b'})
roles=psql(f"select role||':'||status from users where email like 'smkt2.g02%.{TAG}@edupilot.local'")
res['2 /register không tạo TEACHER']=(r[0],r[1].get('code'),r2[0],roles); ok2=r[0] in(400,422) and 'TEACHER' not in roles and r2[0]==202 and set(roles.split())<= {'STUDENT:PENDING_VERIFICATION'}
cid,cc,code=new_course('g'); settings(cid)
u3,t3=tk(); resetjoin()
tT=L('teacher'); s,j,_=call('POST',f'/courses/{cid}/join-code/regenerate',tT); newc=j.get('join_code')
old=jn(t3,code); nw=jn(t3,newc,True)
res['3 tạo lại mã → mã cũ bị từ chối']=(s,old[0],old[1].get('code'),nw[0]); ok3=s==200 and newc!=code and old[0]==404 and old[1].get('code')=='JOIN_CODE_INVALID' and nw[0]==200
resetjoin(); a=jn(t3,newc); b=jn(t3,newc); n=psql(f"select count(*) from enrollments where course_id='{cid}' and user_id=(select id from users where email='{u3}@edupilot.local')")
res['4 join hai lần → một enrollment']=(a[0],b[0],n); ok4=a[0]==200 and b[0]==200 and n=='1'
u5,t5=tk(); resetjoin(); st=[jn(t5,'AAAAAAA')[0] for _ in range(6)]
res['5 lần thử thứ 6 → 429']=st; ok5=st==[404]*5+[429]
# 6: tắt / hết hạn / đầy
cidS,_,codeS=new_course('h'); u6,t6=tk(); resetjoin(); eff={}
settings(cidS,enabled=False); eff['tắt']=jn(t6,codeS)
settings(cidS,enabled=True); resetjoin(); settings(cidS,expires_at=time.strftime('%Y-%m-%dT%H:%M:%SZ',time.gmtime(time.time()+3))); time.sleep(5); eff['hết hạn']=jn(t6,codeS)
settings(cidS,expires_at=None,capacity=1); u7,t7=tk(); resetjoin(); jn(t7,codeS); u8,t8=tk(); resetjoin(); eff['đầy']=jn(t8,codeS)
res['6 tắt/hết hạn/đầy']={k:(v[0],v[1].get('code')) for k,v in eff.items()}; ok6=eff['tắt'][0]==404 and eff['hết hạn'][0]==404 and eff['đầy'][1].get('code')=='COURSE_FULL' and eff['tắt'][1].get('code')==eff['hết hạn'][1].get('code')=='JOIN_CODE_INVALID'
# 7: SV trong roster nhập mã
cid7,_,code7=new_course('r'); settings(cid7); uB,tB=tk(); bcode=psql(f"select student_code from users where email='{uB}@edupilot.local'")
csvt='Email,Họ và tên,MSSV\r\n'+f'{uB}@edupilot.local,B trong roster,{bcode}\r\n'+'\r\n'.join(f'smkt2.{TAG}.{i}@example.test,SV {i},{mssv()}' for i in range(5))+'\r\n'
ri=imp(cid7,tT,csvt); resetjoin(); jj=jn(tB,code7); n7=psql(f"select count(*) from enrollments where course_id='{cid7}' and user_id=(select id from users where email='{uB}@edupilot.local')")
res['7 SV trong roster nhập mã']=(ri[0],jj[0],n7); ok7=ri[0] in(200,201) and jj[0]==200 and n7=='1'
oks=[ok1,ok2,ok3,ok4,ok5,ok6,ok7]
rec('TC-GATEP2-02','P' if all(oks) else 'F','chạy trực tiếp 7 hành vi bằng HTTP trên ep-r (đều khớp: %s): '%oks+'; '.join(f'{k}: {v}' for k,v in res.items())+'; ca nền TC-P202-43, P203-02, P209-18, P209-05, P209-11, P209-22, P210-20 đều P; smkt2-gate-http.py')
# ---------- GATEP2-25: mã cũ bị từ chối bằng câu chung ----------
b=old[1]; bad=jn(t3,'ZZZZZZZ')[1]; dis=eff['tắt'][1]; exp=eff['hết hạn'][1]
same=(b.get('code'),b.get('message'))==(bad.get('code'),bad.get('message'))==(dis.get('code'),dis.get('message'))==(exp.get('code'),exp.get('message'))
leak=any(w in json.dumps(x,ensure_ascii=False).lower() for x in (b,dis,exp) for w in ('đã bị đổi','đã tắt','hết hạn','đã đổi','cũ'))
rec('TC-GATEP2-25','P' if same and not leak else 'F',f'tạo lại mã rồi dùng mã cũ → {old[0]} {b.get("code")} «{b.get("message")}»; mã không tồn tại, mã tắt, mã hết hạn cho đúng cùng mã lỗi và câu ({same}); không có chữ nêu lý do (đã đổi / tắt / hết hạn): {not leak}; ca nền TC-P209-18, P209-07 đều P')
# ---------- GATEP2-29: roster 2 dòng lỗi ----------
cid29,_,_=new_course('m'); rows=[(f'smkt2.m29.{TAG}.{i}@example.test',f'SV {i}',mssv()) for i in range(30)]
rows[5]=(rows[5][0],'Lỗi MSSV','ab'); rows[17]=(rows[17][0],'Lỗi MSSV 2','x1')   # dòng dữ liệu 6 và 18 → dòng tệp 7 và 19 (có tiêu đề)
csv29='Email,Họ và tên,MSSV\r\n'+'\r\n'.join(','.join(r) for r in rows)+'\r\n'
d=imp(cid29,tT,csv29,'?dry_run=true'); er=sorted(x.get('row') for x in d[1].get('errors',[]))
rec('TC-GATEP2-29','P' if er==[7,19] else 'F',f'tệp 30 dòng dữ liệu, 2 dòng sai MSSV (dòng 7 và 19 của tệp có hàng tiêu đề): dry_run → {d[0]}, errors.row = {er}; ca nền TC-P210-06 (UI) P')
# ---------- GATEP2-10 (phần API) ----------
A=L('admin'); key=str(uuid.uuid4()); cc10='SK2G'+TAG.upper()+'A'
b10={'subject_code':'INT1006','class_code':cc10,'name':'G10 '+TAG,'semester':'2026-2027-HK1'}
c1=call('POST','/admin/courses',A,b10,hdr={'Idempotency-Key':key}); c2=call('POST','/admin/courses',A,b10,hdr={'Idempotency-Key':key})
n10=psql(f"select count(*) from courses where class_code='{cc10}'")
cid10=c1[1].get('id'); v=c1[1].get('version',1)
ok_v=call('PUT',f'/admin/courses/{cid10}',A,{'name':'G10 sửa','version':v+7})
um=call('GET','/admin/users?limit=2',A); cur=um[1].get('next_cursor'); um2=call('GET','/admin/users?limit=2&cursor='+str(cur),A) if cur else (0,{},{})
ids=[x['id'] for x in um[1].get('items',[])]+[x['id'] for x in um2[1].get('items',[])]
nt=call('GET','/notifications?limit=1',L('teacher')); ntc=nt[1].get('next_cursor')
rec('TC-GATEP2-10','P' if c1[0]==201 and c2[0] in(200,201) and c1[1].get('id')==c2[1].get('id') and n10=='1' and ok_v[0]==409 and um[0]==200 and len(set(ids))==len(ids)==4 and nt[0]==200 else 'F',f'POST /admin/courses hai lần cùng Idempotency-Key → {c1[0]},{c2[0]} cùng id, đếm lớp {n10}; PUT sai version → {ok_v[0]} {ok_v[1].get("code")}; cursor /admin/users: trang 1 {len(um[1].get("items",[]))} + trang 2 {len(um2[1].get("items",[]))} mục không trùng; /notifications {nt[0]}; (phần Go ở dòng go-g10)')
# ---------- GATEP2-46: IDOR /me/sessions và thông báo ----------
uA,tA=tk(); uB2,tB2=tk()
sB=call('GET','/me/sessions',tB2)[1]; sidB=(sB.get('items') or sB.get('sessions') or [{}])[0].get('id') if isinstance(sB,dict) else None
del_a=call('DELETE',f'/me/sessions/{sidB}',tA); del_adm=call('DELETE',f'/me/sessions/{sidB}',A); del_none=call('DELETE','/me/sessions/'+str(uuid.uuid4()),tA)
still=call('GET','/me/sessions',tB2)[0]
nidB=psql(f"select id from notifications where user_id=(select id from users where email='{uB2}@edupilot.local') limit 1")
if not nidB:
    psql(f"insert into notifications(user_id,type,title) select id,'SMKT2_TEST','Thử IDOR' from users where email='{uB2}@edupilot.local'"); nidB=psql(f"select id from notifications where user_id=(select id from users where email='{uB2}@edupilot.local') limit 1")
rd=call('POST',f'/notifications/{nidB}/read',tA); rdn=call('POST',f'/notifications/{nidB}/read',A)
unread=psql(f"select read_at is null from notifications where id='{nidB}'")
rec('TC-GATEP2-46','P' if del_a[0]==404 and del_adm[0]==404 and del_none[0]==404 and del_a[1].get('code')==del_none[1].get('code') and still==200 and rd[0]==404 and rdn[0]==404 and unread=='t' else 'F',f'SV A xoá phiên của B → {del_a[0]} {del_a[1].get("code")}; Admin xoá qua /me/sessions → {del_adm[0]}; id không tồn tại → {del_none[0]} {del_none[1].get("code")} (đồng nhất); phiên B vẫn dùng được ({still}); SV A / Admin đánh dấu đọc thông báo của B → {rd[0]}/{rdn[0]}, thông báo B vẫn chưa đọc ({unread}); ca nền TC-P204-23, P204-24, P208-23 đều P')
# ---------- GATEP2-48: token giả / leo thang ----------
tokT=L('teacher'); before=psql("select md5(string_agg(id::text||role||status,',' order by id)) from users")
fk={'alg=none':forge(tokT,alg='none',role='ADMIN'),'sửa role=ADMIN, giữ chữ ký sai':forge(tokT,sec='sai-secret-'+TAG,role='ADMIN'),'HS256 secret sai':forge(tS,sec='x'*32,role='ADMIN')}
st48={k:call('GET','/admin/users',v)[0] for k,v in fk.items()}
hdr=call('GET','/admin/users',tS,hdr={'X-Role':'ADMIN','X-User-Id':psql("select id from users where email='admin@edupilot.local'")})[0]
body=call('POST','/admin/users',tS,{'email':f'smkt2.rol.{TAG}@edupilot.local','full_name':'x','role':'ADMIN'})[0]
asg=call('POST','/admin/courses/'+cid+'/assign',tT,{'teacher_id':psql("select id from users where email='ta@edupilot.local'")})[0]
after=psql("select md5(string_agg(id::text||role||status,',' order by id)) from users")
rec('TC-GATEP2-48','P' if all(v==401 for v in st48.values()) and hdr==403 and body==403 and asg==403 and before==after else 'F',f'/admin/users với alg=none / sửa role / secret sai → {st48} (kỳ vọng 401); SV gửi X-Role/X-User-Id giả → {hdr}; SV POST /admin/users role=ADMIN → {body}; GV gọi assign → {asg}; md5 bảng users trước/sau khớp: {before==after}; ca nền TC-P202-29, P203-02, P206-02, P206-22, P206-23, P208-27 đều P')
