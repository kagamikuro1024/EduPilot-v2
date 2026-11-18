from lib3 import *
A_=C7['A']; Q=f'/courses/{A_}'; T='qc3t'; TA='qc3ta'
def ans(u,tid,body,key=True): return A(u,'POST',f'{Q}/tickets/{tid}/answer',{'body':body}) if key else raw2('POST',f'{Q}/tickets/{tid}/answer',login(u),{'body':body})
def audit_n(): return int(psql("select count(*) from audit_log where entity='escalation_ticket'"))
# 13
t,am=tick('qc3s21'); s,d=A(T,'POST',f'{Q}/tickets/{t}/claim')
row=psql(f"select status||'/'||(claimed_by='{uid(T)}')||'/'||(claimed_at is not null) from escalation_tickets where id='{t}'")
rec('TC-P4-03-13',s==200 and row=='CLAIMED/true/true',f'claim → {s}; DB {row}')
# 25: T6 CLAIMED bởi TA, teacher answer → 409
t6,_=tick('qc3s22'); A(TA,'POST',f'{Q}/tickets/{t6}/claim'); s,d=ans(T,t6,'Chào em, thầy trả lời.')
rec('TC-P4-03-25',s==409 and d.get('code')=='TICKET_ALREADY_CLAIMED' and 'claimed_by' in json.dumps(d),f'teacher answer ticket TA đã nhận → {s} {d.get("code")} details={json.dumps(d.get("details"),ensure_ascii=False)[:120]}')
# 12 audit
t12,_=tick('qc3s23'); t12b,_=tick('qc3s24'); a0=audit_n()
for _ in range(3): A(T,'GET',f'{Q}/tickets/{t12}')
a1=audit_n(); A(T,'POST',f'{Q}/tickets/{t12}/claim'); a2=audit_n(); ans(T,t12,'Chào em, CANARY-AUD-12 không được vào audit.'); a3=audit_n(); A(T,'POST',f'{Q}/tickets/{t12b}/dismiss',{'reason':'OUT_OF_SCOPE'}); a4=audit_n()
can=psql(f"select count(*) from audit_log where entity='escalation_ticket' and (coalesce(before::text,'')||coalesce(after::text,'')) like '%CANARY-AUD-12%'")
has=psql(f"select count(*) from audit_log where entity='escalation_ticket' and entity_id in ('{t12}','{t12b}') and (coalesce(before::text,'')||coalesce(after::text,'')) like '%{t12[:8]}%'")
rec('TC-P4-03-12',a1==a0 and a2==a1+1 and a3==a2+1 and a4==a3+1 and can=='0',f'audit trước {a0}; 3 lần mở chi tiết → {a1}; claim → {a2}; answer → {a3}; dismiss → {a4}; canary trong audit={can}')
# 21 thiếu Idempotency-Key
t21,_=tick('qc3s25'); A(T,'POST',f'{Q}/tickets/{t21}/claim'); n0=psql(f"select count(*) from chat_messages where role='ASSISTANT' and intent='STAFF_ANSWER' and course_id='{A_}'")
s,d=ans(T,t21,'Chào em, trả lời không có key.',key=False); n1=psql(f"select count(*) from chat_messages where role='ASSISTANT' and intent='STAFF_ANSWER' and course_id='{A_}'")
rec('TC-P4-03-21',s in(400,422) and n0==n1,f'answer không Idempotency-Key → {s} {(d or {}).get("code") if isinstance(d,dict) else d}; số tin STAFF_ANSWER {n0}→{n1}')
# 23 phiên xoá mềm
t23,am=tick('qc3s26'); sid0=psql(f"select session_id from escalation_tickets where id='{t23}'"); A('qc3s26','DELETE',f'/chat/sessions/{sid0}')
A(T,'POST',f'{Q}/tickets/{t23}/claim'); s,d=ans(T,t23,'Chào em, thầy trả lời sau khi em xoá phiên.')
sess=psql(f"select session_id from escalation_tickets where id='{t23}'")
msg=psql(f"select s.title||'/'||(s.deleted_at is null)||'/'||s.id from chat_messages m join chat_sessions s on s.id=m.session_id where m.course_id='{A_}' and m.user_id='{uid('qc3s26')}' and m.intent='STAFF_ANSWER' order by m.created_at desc limit 1")
sg,dg=A(T,'GET',f'{Q}/tickets/{t23}')
f1,_=A('qc3s26','POST',f'/chat/tickets/{t23}/followup',{'content':'Em chưa hiểu bước 2'}); 
rec('TC-P4-03-23',s==200 and sess==sid0 and 'Tin nhắn từ giảng viên' in msg and sg==200,f'answer → {s}; ticket.session_id đổi={sess!=sid0}; tin STAFF_ANSWER nằm ở phiên «{msg}»; GV đọc ngữ cảnh → {sg}; Hỏi tiếp → {f1}')
# 27
t27,_=tick('qc3s27'); A(T,'POST',f'{Q}/tickets/{t27}/claim'); ans(T,t27,'Trả lời lần một.'); st0=psql(f"select answered_at::text from escalation_tickets where id='{t27}'"); n_not0=nrows('qc3s27',"and type='TICKET_ANSWERED'"); time.sleep(1.2)
s2,d2=ans(T,t27,'Bổ sung lần hai.'); s3,d3=ans(TA,t27,'TA trả lời chen.'); time.sleep(3)
row=psql(f"select status||'/'||(answered_at::text<>'{st0}')::text from escalation_tickets where id='{t27}'"); nmsg=psql(f"select count(*) from chat_messages where intent='STAFF_ANSWER' and user_id='{uid('qc3s27')}'"); n_not1=nrows('qc3s27',"and type='TICKET_ANSWERED'")
rec('TC-P4-03-27',s2==200 and s3==409 and row=='ANSWERED/true' and nmsg=='2' and int(n_not1)==int(n_not0)+1,f'teacher thêm → {s2}; TA → {s3}; ticket {row}; tin STAFF_ANSWER={nmsg}; TICKET_ANSWERED {n_not0}→{n_not1}')
# 29 PII
t29,_=tick('qc3s28'); A(T,'POST',f'{Q}/tickets/{t29}/claim')
m2=psql(f"select student_code from users where email='qc3s05@edupilot.local'")
r=[ans(T,t29,b) for b in (f'Mã số sinh viên {m2} cũng hỏi giống em.','Em hỏi thử bạn qc3s05@edupilot.local nhé.','Số CCCD 079203004512 của em được ghi nhận.')]
rec('TC-P4-03-29',all(x[0]==422 and x[1].get('code')=='PII_IN_ANSWER' for x in r),f'MSSV/email/CCCD → {[(x[0],x[1].get("code")) for x in r]}')
# 41 feedback STAFF_ANSWER
sm=psql(f"select id from chat_messages where intent='STAFF_ANSWER' and user_id='{uid('qc3s27')}' order by created_at limit 1"); ssid=psql(f"select session_id from chat_messages where id='{sm}'")
fb=A('qc3s27','PUT',f'/chat/messages/{sm}/feedback',{'value':'HELPFUL'}); rt=A('qc3s27','POST',f'/chat/messages/{sm}/retry'); fv=psql(f"select coalesce(feedback::text,'-') from chat_messages where id='{sm}'")
rec('TC-P4-03-41',fb[0]==422 and fb[1].get('code')=='MESSAGE_NOT_RATEABLE' and fv=='-' and rt[0] in(409,422,404),f'feedback → {fb[0]} {fb[1].get("code")}; retry → {rt[0]} {(rt[1] or {}).get("code") if isinstance(rt[1],dict) else rt[1]}; feedback DB={fv}')
# E1: claim ticket ANSWERED/CLOSED; answer ticket CLOSED
tA,_=tick('qc3s29'); A(T,'POST',f'{Q}/tickets/{tA}/claim'); ans(T,tA,'Đã trả lời.'); c1=A(T,'POST',f'{Q}/tickets/{tA}/claim')
tC,_=tick('qc3s30'); A(T,'POST',f'{Q}/tickets/{tC}/claim'); A(T,'POST',f'{Q}/tickets/{tC}/dismiss',{'reason':'OUT_OF_SCOPE'}); c2=A(T,'POST',f'{Q}/tickets/{tC}/claim'); a3=ans(T,tC,'Trả lời vào ticket đã đóng.')
rec('TC-P4-03-E1',c1[0]==409 and c1[1].get('code')=='TICKET_STATE_INVALID' and c2[0]==409 and a3[0]==409,f'claim ANSWERED → {c1[0]} {c1[1].get("code")}; claim CLOSED → {c2[0]} {c2[1].get("code")}; answer CLOSED → {a3[0]} {a3[1].get("code")}')
# E2: gửi lại cùng key không tin đôi
tE,_=tick('qc3s20'); A(T,'POST',f'{Q}/tickets/{tE}/claim'); k=str(uuid.uuid4())
r1=A(T,'POST',f'{Q}/tickets/{tE}/answer',{'body':'Chào em, đây là lời đáp E2.'},idem=k); r2=A(T,'POST',f'{Q}/tickets/{tE}/answer',{'body':'Chào em, đây là lời đáp E2.'},idem=k)
nn=psql(f"select count(*) from chat_messages where intent='STAFF_ANSWER' and user_id='{uid('qc3s20')}' and content like '%lời đáp E2%'")
rec('TC-P4-03-E2',r1[0]==200 and r2[0]==200 and nn=='1',f'gửi hai lần cùng key → {r1[0]}/{r2[0]}; số tin STAFF_ANSWER={nn}')
# P1: lớp khác
B_=C7['B']; tb,_=tick('qc3s01',B_)
r=[A(T,'GET',f'/courses/{B_}/tickets/{tb}')[0]]
Cc=C7['A']; tA2,_=tick('qc3s21')
x1=A('r5t','GET',f'{Q}/tickets/{tA2}')[0]; x2=A('r5ta','POST',f'{Q}/tickets/{tA2}/answer',{'body':'Chào em, TA lớp khác.'})[0]
sv=[A('qc3s02',m,p)[0] for m,p in (('GET',f'{Q}/tickets'),('GET',f'{Q}/tickets/{tA2}'),('POST',f'{Q}/tickets/{tA2}/claim'),('POST',f'{Q}/tickets/{tA2}/answer'),('POST',f'{Q}/tickets/{tA2}/dismiss'))]
rec('TC-P4-03-P1',x1 in(403,404) and x2 in(403,404) and all(v==403 for v in sv),f'GV lớp khác mở ticket → {x1}; TA lớp khác answer → {x2}; SV gọi route staff → {sv} (spec: 404 cho cross-course; ghi mã thực)')
save('p403')
