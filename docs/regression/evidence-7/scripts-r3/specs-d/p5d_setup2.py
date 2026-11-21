from lib3 import *
import json
for i in (4,5):
    em=f'qc3p5s{i:02d}'
    psql(f"insert into users(email,password_hash,full_name,role,status,email_verified_at,student_code,tracking_notice_ack_at) select '{em}@edupilot.local',password_hash,'QC3 {em}','STUDENT','ACTIVE',now(),'39{i:06d}',now() from users where email='sv08@edupilot.local' on conflict do nothing")
B=mkcourse('QC3P5B','qc3pg',['qc3ph'],'QP5BAAB'); C=mkcourse('QC3P5C','qc3pg',['qc3ph'],'QP5CAAB')
for c,u in ((B,'qc3p5s04'),(C,'qc3p5s05')):
    psql(f"insert into enrollments(course_id,user_id,role_in_course,status,joined_via,student_code_snapshot) select '{c}',u.id,'STUDENT','ACTIVE','ROSTER',u.student_code from users u where u.email='{u}@edupilot.local' on conflict do nothing")
for n in (1,2):
    psql(f"insert into class_sessions(course_id,session_no,starts_at,ends_at) values('{C}',{n},now()-interval '{n} days',now()-interval '{n} days'+interval '2 hours') on conflict do nothing")
fx=json.load(open('/tmp/qc3e/fx_p5.json')); fx.update({'B':B,'Cc':C}); json.dump(fx,open('/tmp/qc3e/fx_p5.json','w')); print(fx)
