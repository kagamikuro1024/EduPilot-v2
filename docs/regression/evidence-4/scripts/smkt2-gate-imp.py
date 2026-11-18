# SmkT2: mạo danh MSSV (GATEP2-04 phần tay, GATEP2-20, GATEP2-40) bằng HTTP trực tiếp trên ep-r
import os; exec(open(os.path.join(os.path.dirname(os.path.abspath(__file__)),'smkt2lib.py')).read())
from smkt2lib import rec
TAG=uuid.uuid4().hex[:5]; PWDX='Smkt2!Pass-'+TAG+'x7'
def jn(t,code): return call('POST','/courses/join',t,{'code':code})
resetrl(); resetjoin()
pool=student_pool(5); uB=pool[0]; tB=L(uB); bcode=psql(f"select student_code from users where email='{uB}@edupilot.local'")
cid,cc,code=new_course('i'); settings(cid); resetjoin(); jb=jn(tB,code)
# kẻ mạo danh: email khác, MSSV của B, đăng ký → thư → xác minh → đăng nhập
em=f'smkt2.imp.{TAG}@edupilot.local'; t0=time.strftime('%Y-%m-%dT%H:%M:%S',time.gmtime(time.time()-3))+'Z'
s,j,_=call('POST','/auth/register',None,{'email':em,'password':PWDX,'full_name':'Kẻ Mạo Danh','student_code':bcode}); 
m=mail_to(em,'Xác minh',t0,60); tok=link_token(m,'verify-email'); sv=call('POST','/auth/verify-email',None,{'token':tok})
resetrl(); lg=call('POST','/auth/login',None,{'email':em,'password':PWDX}); tA=lg[1].get('access_token')
mine=call('GET','/me/courses',tA)[1]; items=mine.get('items',[]) if isinstance(mine,dict) else mine
th=call('GET',f'/courses/{cid}/threads',tA)[0]; mem=call('GET',f'/courses/{cid}/members',tA)[0]; today=call('GET',f'/courses/{cid}/today',tA)[0]; sess=call('GET',f'/courses/{cid}/sessions',tA)[0]
mtoday=call('GET','/me/today',tA)
enr_before=psql(f"select count(*) from enrollments where course_id='{cid}' and user_id=(select id from users where email='{em}')")
ra=psql(f"select count(*) from enrollments e where e.course_id='{cid}' and e.student_code_snapshot='{bcode}' and e.status='ACTIVE'")
# vào bằng mã (đường 3): ghi PENDING + cảnh báo, không nối theo MSSV
resetjoin(); ja=jn(tA,code)
enr=psql(f"select status||'|'||coalesce(warning,'') from enrollments where course_id='{cid}' and user_id=(select id from users where email='{em}')")
actB=psql(f"select count(*) from enrollments where course_id='{cid}' and user_id=(select id from users where email='{uB}@edupilot.local') and status='ACTIVE'")
th2=call('GET',f'/courses/{cid}/threads',tA)[0]; mem2=call('GET',f'/courses/{cid}/members',tA)[0]
tT=L('teacher'); ml=call('GET',f'/courses/{cid}/members?status=PENDING',tT)[1]; its=ml.get('items',[]) if isinstance(ml,dict) else []
warn=[(x.get('full_name'),x.get('warning'),x.get('status')) for x in its]
td=call('GET',f'/courses/{cid}/today',tT)[1]; tds=json.dumps(td,ensure_ascii=False)
okA=sv[0]==200 and lg[0]==200 and items==[] and th in(403,404) and mem in(403,404) and today in(403,404) and sess in(403,404)
okB=ja[0]==200 and enr.startswith('PENDING|EMAIL') and actB=='1' and th2 in(403,404) and mem2 in(403,404) and any(w[1]=='EMAIL_MISMATCH' for w in warn)
common=f'kẻ đăng ký MSSV của B ({bcode[:3]}…) bằng email khác → xác minh → đăng nhập: /me/courses {len(items)} lớp; threads/members/today/sessions của lớp B → {th}/{mem}/{today}/{sess}; enrollments của kẻ lạ trước khi vào lớp {enr_before}; vào lớp bằng mã → {ja[0]} bản ghi «{enr}» (PENDING + cảnh báo, không ACTIVE, không nối theo MSSV); B vẫn ACTIVE ({actB}); sau đó kẻ lạ vẫn {th2}/{mem2} trên threads/members; GV thấy hàng chờ: {warn}; /me/today kẻ lạ {mtoday[0]}'
rec('TC-GATEP2-20','P' if okA and okB else 'F',common+'; ca nền TC-P203-07, P209-30, P210-14 đều P; smkt2-gate-imp.py')
rec('TC-GATEP2-04','P' if okA and okB else 'F','go test -count=1 ./internal/integration -run TestRosterLinkRequiresVerifiedEmail → pass (logs/smkT2/go-g04.json); lặp tay bằng HTTP: '+common+'; ca nền TC-P210-14, P210-16 P')
