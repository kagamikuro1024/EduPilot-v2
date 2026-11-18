from lib3 import *
T=mkusers('qc3pc','TEACHER',1)+mkusers('qc3pa','TEACHER',1)+mkusers('qc3pb','TEACHER',1)
TA=mkusers('qc3pf','TA',1)
S=['qc3at%d'%i for i in range(100,130)]
C=mkcourse('QC3U3','qc3pc',['qc3pf'],'QKU3333'); C0=mkcourse('QC3U4','qc3pc',['qc3pf'],'QKU4444'); CE=mkcourse('QC3U5','qc3pc',['qc3pf'],'QKU5555')
CA=mkcourse('QC3U6','qc3pa',['qc3pf'],'QKU6666'); CB=mkcourse('QC3U7','qc3pb',['qc3pf'],'QKU7777')
for u in S:
    psql(f"insert into enrollments(course_id,user_id,role_in_course,status,joined_via,student_code_snapshot) select '{C}',u.id,'STUDENT','ACTIVE','ROSTER',u.student_code from users u where u.email='{u}@edupilot.local' on conflict do nothing")
json.dump({'C':C,'C0':C0,'CE':CE,'CF':CB,'CA':CA,'S':S,'teacher':'qc3pc','ta':'qc3pf','tA':'qc3pa','tB':'qc3pb'},open('/tmp/qc3e/fx_att.json','w'))
print(open('/tmp/qc3e/fx_att.json').read()); print(psql(f"select count(*) from enrollments where course_id='{C}' and status='ACTIVE'"))
