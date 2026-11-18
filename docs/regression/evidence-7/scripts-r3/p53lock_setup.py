from lib3 import *
U=mkusers('qc3p5s','STUDENT',4,start=82)
fx=json.load(open('/tmp/qc3e/r4fx.json')); X,Y=fx['X'],fx['Y']
for u in U:
    for c in (X,Y): psql(f"insert into enrollments(course_id,user_id,role_in_course,status,joined_via,student_code_snapshot) select '{c}',u.id,'STUDENT','ACTIVE','ROSTER',u.student_code from users u where u.email='{u}@edupilot.local' on conflict do nothing")
psql("update users set tracking_notice_ack_at=now() where email like 'qc3p5s8_@edupilot.local'")
L=mkcourse('QC3L','qc3t',['qc3ta'],'QKM2222')
print(psql(f"insert into enrollments(course_id,user_id,role_in_course,status,joined_via,student_code_snapshot) select '{L}',u.id,'STUDENT','ACTIVE','ROSTER',u.student_code from users u where u.email like 'load%@load.local' and u.role='STUDENT' on conflict do nothing"))
n=psql(f"select count(*) from enrollments where course_id='{L}' and role_in_course='STUDENT'"); print('SV',n)
if psql(f"select count(*) from escalation_tickets where course_id='{L}'")=='0':
    psql(f"""with st as (select user_id, row_number() over (order by user_id) rn from enrollments where course_id='{L}' and role_in_course='STUDENT'),
 ss as (insert into chat_sessions(course_id,user_id,title) select '{L}',user_id,'qc3-1k' from st returning id,user_id),
 um as (insert into chat_messages(course_id,session_id,user_id,role,content,stream_status,client_msg_id,completed_at) select '{L}',id,user_id,'USER','Câu hỏi 1k','DONE',gen_random_uuid(),now() from ss returning id,session_id,user_id),
 am as (insert into chat_messages(course_id,session_id,user_id,role,content,stream_status,reply_to,intent,confidence,low_confidence,completed_at) select '{L}',um.session_id,um.user_id,'ASSISTANT','Trả lời 1k','DONE',um.id,'COURSE_QA',0.4,true,now() from um returning id,session_id,user_id)
 insert into escalation_tickets(course_id,origin,source_type,source_id,message_id,session_id,student_id,waiting_since,created_at) select '{L}','AUTO_LOW_CONFIDENCE','CHAT_MESSAGE',id,id,session_id,user_id,now()-(random()*interval '5 days'),now()-(random()*interval '5 days') from am""")
print('tickets',psql(f"select count(*),count(*) filter (where status='OPEN') from escalation_tickets where course_id='{L}'"))
fx.update({'L':L,'U':U}); json.dump(fx,open('/tmp/qc3e/r4fx.json','w')); print(L)
