from lib3 import *
for em,role,nm in [('qc3pg','TEACHER','QC3 qc3pg'),('qc3ph','TA','QC3 qc3ph')]:
    psql(f"insert into users(email,password_hash,full_name,role,status,email_verified_at) select '{em}@edupilot.local',password_hash,'{nm}','{role}','ACTIVE',now() from users where email='sv08@edupilot.local' on conflict do nothing")
S=[]
for i in range(1,4):
    em=f'qc3p5s{i:02d}'; S.append(em)
    psql(f"insert into users(email,password_hash,full_name,role,status,email_verified_at,student_code) select '{em}@edupilot.local',password_hash,'QC3 {em}','STUDENT','ACTIVE',now(),'39{i:06d}' from users where email='sv08@edupilot.local' on conflict do nothing")
    psql(f"update users set tracking_notice_ack_at=now() where email='{em}@edupilot.local'")
C=mkcourse('QC3P5','qc3pg',['qc3ph'],'QP5AAAB'); Q=f'/courses/{C}'
for u in S:
    psql(f"insert into enrollments(course_id,user_id,role_in_course,status,joined_via,student_code_snapshot) select '{C}',u.id,'STUDENT','ACTIVE','ROSTER',u.student_code from users u where u.email='{u}@edupilot.local' on conflict do nothing")
for n,d in ((1,'3 days'),(2,'2 days'),(3,'1 day')):
    psql(f"insert into class_sessions(course_id,session_no,starts_at,ends_at) values('{C}',{n},now()-interval '{d}',now()-interval '{d}'+interval '2 hours') on conflict do nothing")
import json
for n in (1,2,3):
    sid=psql(f"select id from class_sessions where course_id='{C}' and session_no={n}")
    G=f'{Q}/sessions/{sid}/attendance'
    g=A('qc3pg','GET',G)
    ids=[r['id'] for r in g[1]['students']]
    st=['PRESENT','ABSENT','PRESENT'] if n!=2 else ['LATE','PRESENT','ABSENT']
    s,d=A('qc3pg','PUT',G,{'records':[{'student_id':i,'status':st[k%3]} for k,i in enumerate(ids)],'complete':True}); print(n,s)
json.dump({'C':C,'T':'qc3pg','TA':'qc3ph','S':S,'SV':uid(S[0])},open('/tmp/qc3e/fx_p5.json','w'))
print(open('/tmp/qc3e/fx_p5.json').read())
