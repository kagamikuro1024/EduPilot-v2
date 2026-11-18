import os; os.environ['SP']='4-t9'
from smkT import *
import itertools
A_=A()
gm,GV,gv=staff('m1','TEACHER'); cid=mkcourse('M',gv)
modes={'Member':('GET',f'/courses/{cid}/today',None),'Staff':('POST',f'/courses/{cid}/sessions/generate',{}),'Teacher':('GET',f'/courses/{cid}/share-sources',None),
 'StaffOrAdmin':('GET',f'/courses/{cid}/members',None),'Manage':('GET',f'/courses/{cid}/assistant-candidates',None),'MemberOrAdmin':('GET',f'/courses/{cid}',None)}
def allowed(mode,jwt,enr):
    st,role=enr
    member=st=='ACTIVE'; staff_=st=='ACTIVE' and role in('TEACHER','TA'); teacher=st=='ACTIVE' and role=='TEACHER'; adm=jwt=='ADMIN'
    return {'Member':member,'Staff':staff_,'Teacher':teacher,'StaffOrAdmin':staff_ or adm,'Manage':teacher or adm,'MemberOrAdmin':member or adm}[mode]
states=[('ACTIVE','STUDENT'),('ACTIVE','TA'),('ACTIVE','TEACHER'),('PENDING','STUDENT'),('REMOVED','STUDENT'),(None,None)]
cases=0; bad=[]; users={}
for jwt in ['STUDENT','TA','TEACHER','ADMIN']:
    for st,role in states:
        lab=f'{jwt[:2]}{(st or "NO")[:2]}{(role or "")[:2]}'.lower()
        em,uid_=mkuser(lab,role=jwt)
        if st:
            if st=='ACTIVE' and role=='TEACHER': psql(f"update enrollments set status='REMOVED',removed_at=now() where course_id='{cid}' and role_in_course='TEACHER' and status='ACTIVE'")
            psql(f"insert into enrollments(course_id,user_id,role_in_course,status,joined_via,student_code_snapshot{',removed_at' if st=='REMOVED' else ''}) values('{cid}','{uid_}','{role}','{st}','ADMIN',null{',now()' if st=='REMOVED' else ''}) on conflict do nothing")
        users[(jwt,st,role)]=(em,tokof(em))
        for mode,(m,p,b) in modes.items():
            s,j,_=call(m,p,users[(jwt,st,role)][1],b); cases+=1
            exp=allowed(mode,jwt,(st,role)); got_allow=s!=403
            rs=(j.get('details') or {}).get('reason') if s==403 else None
            if got_allow!=exp or (not exp and rs!='course' and not (rs=='role')): bad.append((jwt,st,role,mode,s,rs))
# vai trong lớp lấy từ enrollments: đổi vai DB → kết quả đổi (cùng token)
em,tk=users[('STUDENT','ACTIVE','STUDENT')]; u_=uid(em)
a0=call('GET',f'/courses/{cid}/members',tk)[0]; psql(f"update enrollments set role_in_course='TA' where course_id='{cid}' and user_id='{u_}'"); a1=call('GET',f'/courses/{cid}/members',tk)[0]
psql(f"update enrollments set role_in_course='STUDENT' where course_id='{cid}' and user_id='{u_}'"); a2=call('GET',f'/courses/{cid}/members',tk)[0]
rec('TC-P207-09','P' if cases>=48 and not bad and a0==403 and a1==200 and a2==403 else 'F',f'{cases} ca (4 vai JWT × 6 tình trạng [ACTIVE STUDENT/TA/TEACHER, PENDING, REMOVED, không ghi danh] × 6 chế độ: Member=GET today, Staff=POST sessions/generate, Teacher=GET share-sources, StaffOrAdmin=GET members, Manage=GET assistant-candidates, MemberOrAdmin=GET /courses/{{id}}): lệch ma trận SRS 4.1 = {bad[:3] or "không"}; mọi từ chối là 403 details.reason=course|role; đổi vai DB STUDENT→TA→STUDENT với cùng JWT: members {a0}→{a1}→{a2} (vai lấy từ enrollments, không từ JWT)')
# ---- P207-11
svm,SV=users[('STUDENT','ACTIVE','STUDENT')]
x1=call('GET','/courses/abc/today',SV)[0]; x2=call('GET','/courses/00000000-0000-0000-0000-000000000000/today',SV)[0]; x3=call('GET','/courses/00000000-0000-0000-0000-000000000000/members',users[('TEACHER','ACTIVE','TEACHER')][1])[0]
tk_=users[('STUDENT','ACTIVE','STUDENT')][1]; call('GET',f'/courses/{cid}/today',tk_); u2=uid(users[('STUDENT','ACTIVE','STUDENT')][0])
psql(f"update enrollments set status='REMOVED',removed_at=now() where course_id='{cid}' and user_id='{u2}'"); y1=call('GET',f'/courses/{cid}/today',tk_)[0]
psql(f"update enrollments set status='ACTIVE',removed_at=null where course_id='{cid}' and user_id='{u2}'"); y2=call('GET',f'/courses/{cid}/today',tk_)[0]
rec('TC-P207-12','P' if all(call('GET',f'/courses/{cid}/share-sources',t)[0]==401 for t in [forge(GV,alg='none'),forge(GV,sec='sai-secret-sai-secret-sai-secret-1234'),forge(GV,role='TEACHER')[:-4]+'AAAA','x.y.z']) else 'F','JWT alg=none / ký secret sai / chữ ký hỏng / rác vào route Teacher (GET share-sources) → 401 cả 4')
open('/tmp/smkT/p207-11.json','w').write(json.dumps([x1,x2,x3,y1,y2]))
print('p207-11 parts',x1,x2,x3,y1,y2)
