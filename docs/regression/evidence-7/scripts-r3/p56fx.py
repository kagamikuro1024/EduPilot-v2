from lib3 import *
import datetime as dt
U=mkusers('qc3me','STUDENT',4,start=1)   # qc3me01..04
for u in U: assert uid(u),u
M=mkcourse('QC3M5','qc3t',['qc3ta'],'QKM5MMM'); M2=mkcourse('QC3M6','qc3t',['qc3ta'],'QKM6MMM')
g=A('qc3t','GET',f'/courses/{M2}')[1]
if psql(f"select count(*) from enrollments where course_id='{M2}' and role_in_course='STUDENT'")=='0': print(A('qc3t','PUT',f'/courses/{M2}/settings',{'allow_study_time_opt_out':False,'version':g['settings']['version']})[0])
tid=psql("select id from users where email='qc3t@edupilot.local'")
def enr(c,u): psql(f"insert into enrollments(course_id,user_id,role_in_course,status,joined_via,student_code_snapshot) select '{c}',u.id,'STUDENT','ACTIVE','ROSTER',u.student_code from users u where u.email='{u}@edupilot.local' on conflict do nothing")
for u in U: enr(M,u)
for u in U[:1]+U[2:3]: enr(M2,u)
psql(f"update users set tracking_notice_ack_at=now() where email like 'qc3me0%@edupilot.local'")
for u in U: rds('DEL',f'ep:trk:{uid(u)}')
# 9 buổi đã qua ở M (meA: 7 có mặt + 2 vắng; meC: 9 có mặt), lịch cũ
if psql(f"select count(*) from class_sessions where course_id='{M}'")=='0':
    for i in range(1,10):
        sid=psql(f"insert into class_sessions(course_id,session_no,starts_at,ends_at,room,attendance_done_at,attendance_done_by) values('{M}',{i},now()-interval '{(10-i)*7} days',now()-interval '{(10-i)*7} days'+interval '2 hours','P.1',now(),'{tid}') returning id").split('\n')[0]
        for k,u in enumerate(U):
            st='ABSENT' if (u=='qc3me01' and i in (3,6)) else 'PRESENT'
            if u=='qc3me04': continue
            psql(f"insert into attendance_records(course_id,session_id,student_id,status,marked_by) values('{M}','{sid}','{uid(u)}','{st}','{tid}')")
    sid=psql(f"select id from class_sessions where course_id='{M}' and session_no=2")
    for n in range(2): psql(f"insert into participation_events(course_id,student_id,session_id,type,points,client_event_id,created_by) values('{M}','{uid('qc3me01')}','{sid}','SPEAK',0.25,gen_random_uuid(),'{tid}')")
# thời gian học 8 tuần: meA M (tuần này 45 phút), M2; meC tuần thấp
def study(c,u,mins):
    for w,m in enumerate(mins):   # w=0 là tuần này
        d=f"((date_trunc('week',(now() at time zone 'Asia/Ho_Chi_Minh'))::date) - interval '{w*7} days')::date"
        psql(f"insert into study_time_daily(course_id,student_id,date,area,active_seconds) select '{c}','{uid(u)}',{d},'CHAT',{m*60} where {m}>0 on conflict (course_id,student_id,date,area) do update set active_seconds=excluded.active_seconds")
study(M,'qc3me01',[45,120,95,150,80,110,130,70]); study(M2,'qc3me01',[30,60,90,60,45,75,100,20]); study(M,'qc3me03',[50,45,30,120,130,110,90,100])
json.dump({'M':M,'M2':M2,'U':U},open('/tmp/qc3e/fx_me.json','w'))
print(M,M2,psql(f"select count(*) from attendance_records where course_id='{M}'"),psql(f"select count(*) from study_time_daily where course_id in ('{M}','{M2}')"))
