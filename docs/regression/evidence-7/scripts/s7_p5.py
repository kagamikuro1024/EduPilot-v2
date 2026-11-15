from lib import *
import time,threading
C=json.load(open('courses.json')); E1=C['E6C1']; E2=C['E6C2']; Q=f'/courses/{E1}'
tix=json.load(open('s7_tix.json')); T1=tix['tid1']; TP=tix['tp']
cid=str(uuid.uuid4())
# P4-05 followup (T1 ANSWERED, SV sv08)
s,d=A('sv08','POST',f'/chat/tickets/{T1}/followup',{'content':'Em chưa hiểu bước 2','client_msg_id':cid}); 
rec('TC-P4-05-01',s in(200,201),f'followup → {s} {str(d)[:100]}')
s2,d2=A('sv08','POST',f'/chat/tickets/{T1}/followup',{'content':'Em chưa hiểu bước 2','client_msg_id':cid})
n=psql(f"select count(*) from chat_messages where client_msg_id='{cid}'")
rec('TC-P4-05-03',s2 in(200,201) and n=='1',f'lặp cùng client_msg_id → {s2}; số tin={n}')
r=[A('sv08','POST',f'/chat/tickets/{TP}/followup',{'content':c,'client_msg_id':str(uuid.uuid4())})[0] for c in ('','a'*2001)]
rec('TC-P4-05-04',all(x==422 for x in r),f'content rỗng/2001 → {r}')
r=[A(u,'POST',f'/chat/tickets/{T1}/followup',{'content':'x','client_msg_id':str(uuid.uuid4())})[0] for u in ('sv09','teacher','admin')]
rec('TC-P4-05-08',r[0]==404 and r[1]==403 and r[2]==403,f'followup bởi SV khác/GV/ADMIN → {r}')
st=psql(f"select status from escalation_tickets where id='{T1}'"); print('T1 status',st)
# confirm / undo trên TP
s,d=A('sv08','POST',f'/chat/tickets/{TP}/confirm'); stp=psql(f"select status||'/'||coalesce(closed_reason::text,'') from escalation_tickets where id='{TP}'")
rec('TC-P4-05-09',s in(200,204) and stp.startswith('CLOSED'),f'confirm → {s}; trạng thái {stp}')
s,d=A('sv08','POST',f'/chat/tickets/{TP}/confirm/undo'); stp=psql(f"select status from escalation_tickets where id='{TP}'")
rec('TC-P4-05-10',s in(200,204) and stp=='ANSWERED',f'undo <5s → {s}; trạng thái {stp}')
A('sv08','POST',f'/chat/tickets/{TP}/confirm'); time.sleep(6); s,d=A('sv08','POST',f'/chat/tickets/{TP}/confirm/undo')
rec('TC-P4-05-10b',s in(409,410,422),f'undo sau 6 s → {s}')
r=[A(u,'POST',f'/chat/tickets/{T1}/confirm')[0] for u in ('sv09','teacher')]; rec('TC-P4-05-12',r[0]==404 and r[1]==403,f'confirm bởi SV khác/GV → {r}')
# dismiss 3 lý do
td=[]
def mk(u,text):
    s,d=A(u,'POST','/chat/sessions',{'course_id':E1}); sid=d['id']; sse('POST',f'/chat/sessions/{sid}/messages',login(u),{'content':text},maxsec=60)
    am=psql(f"select id from chat_messages where session_id='{sid}' and role='ASSISTANT' order by created_at desc limit 1"); s,d=A(u,'POST',f'/chat/messages/{am}/escalate'); return d['ticket']['id']
res=[]
for rs,txt in (('OUT_OF_SCOPE','Khi nào em thi giữa kỳ?'),('DUPLICATE','Lịch học tuần sau thế nào?'),('NEED_MORE_INFO','Tuần này có gì?')):
    t=mk('sv09',txt); s,d=A('teacher','POST',f'/courses/{E1}/tickets/{t}/dismiss',{'reason':rs}); res.append((rs,s,psql(f"select status||'/'||coalesce(closed_reason::text,'') from escalation_tickets where id='{t}'")))
rec('TC-P4-05-32',all(x[1] in(200,204) for x in res),str(res))
# knowledge
s,d=A('teacher','GET',f'{Q}/tickets/{T1}/knowledge-preview'); rec('TC-P4-05-35',s==200,f'preview → {s} {str(d)[:100]}')
if s==200:
    q=d.get('question') or d.get('q'); a=d.get('answer') or d.get('a')
    s1,d1=A('teacher','POST',f'{Q}/tickets/{T1}/knowledge',{'question':q,'answer':a}); s2,d2=A('teacher','POST',f'{Q}/tickets/{T1}/knowledge',{'question':q,'answer':a+' (sửa)'})
    rec('TC-P4-05-36',s1==201 and s2==200,f'lưu lần đầu → {s1}; lần hai → {s2}')
    rr=[A(u,'POST',f'{Q}/tickets/{T1}/knowledge',{'question':q,'answer':a})[0] for u in ('sv08','admin','r5t')]; rec('TC-P4-05-39',all(x==403 for x in rr),f'SV/ADMIN/TA lớp khác lưu tri thức → {rr}')
    time.sleep(6); kc=psql(f"select count(*) from knowledge_items where ticket_id='{T1}'") if False else psql("select count(*) from knowledge_items where course_id='%s'"%E1); print('knowledge items',kc)
    s3,_=A('teacher','DELETE',f'{Q}/tickets/{T1}/knowledge'); s4,_=A('teacher','DELETE',f'{Q}/tickets/{T1}/knowledge')
    rec('TC-P4-05-43',s3 in(200,204) and s4 in(200,204),f'gỡ tri thức → {s3}, lặp (idempotent) → {s4}')
# threshold
s,d=A('teacher','GET',f'{Q}/settings'); print('settings',s,str(d)[:200])
if s==200:
    v=d['version']; th=d.get('escalation_threshold')
    s1,d1=A('teacher','PUT',f'{Q}/settings',{'escalation_threshold':0.85,'version':v}); 
    bad=[A('teacher','PUT',f'{Q}/settings',{'escalation_threshold':x,'version':d1.get('version',v+1) if isinstance(d1,dict) else v+1})[0] for x in (0.49,0.96,0.82,'abc')]
    rec('TC-P4-05-47',s1==200 and bad==[422,422,422,422],f'0.85 → {s1}; 0.49/0.96/0.82/"abc" → {bad}')
    sv=A('teacher','PUT',f'{Q}/settings',{'escalation_threshold':0.8,'version':v})[0]; r=[A(u,'PUT',f'{Q}/settings',{'escalation_threshold':0.8,'version':1})[0] for u in ('ta','sv08','admin')]
    rec('TC-P4-05-48',sv in(409,412) and all(x==403 for x in r),f'version cũ → {sv}; TA/SV/ADMIN → {r}')
    s,d=A('teacher','GET',f'{Q}/settings'); A('teacher','PUT',f'{Q}/settings',{'escalation_threshold':th,'version':d['version']})
    print(psql("select action from audit_log where action like '%threshold%' or action like '%settings%' order by created_at desc limit 2"))
# me/today SV
s,d=A('sv08','GET','/me/today'); rec('TC-P4-05-29',s==200,f'me/today SV → {s} keys {list(d)[:5]}')
r=A('sv08','POST','/notifications/read',{'link':'/chat?session=x'})[0]; rec('TC-P4-05-31',r in(200,204),f'notifications/read {{link}} bởi SV → {r}')
