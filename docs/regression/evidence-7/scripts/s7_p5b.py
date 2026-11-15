from lib import *
import time
C=json.load(open('courses.json')); E1=C['E6C1']; Q=f'/courses/{E1}'
def mk(u,text):
    s,d=A(u,'POST','/chat/sessions',{'course_id':E1}); sid=d['id']; sse('POST',f'/chat/sessions/{sid}/messages',login(u),{'content':text},maxsec=60)
    am=psql(f"select id from chat_messages where session_id='{sid}' and role='ASSISTANT' order by created_at desc limit 1"); s,d=A(u,'POST',f'/chat/messages/{am}/escalate'); return d['ticket']['id']
t=mk('sv09','Quy định nộp bài muộn của lớp là gì?'); A('teacher','POST',f'{Q}/tickets/{t}/claim'); s,d=A('teacher','POST',f'{Q}/tickets/{t}/answer',{'body':'Chào em, nộp muộn bị trừ 10% mỗi ngày theo quy chế lớp.'}); print('answer',s)
s,d=A('teacher','GET',f'{Q}/tickets/{t}/knowledge-preview'); print('preview',s,json.dumps(d,ensure_ascii=False)[:300])
rec('TC-P4-05-35',s==200,f'preview → {s}')
q=d.get('question'); a=d.get('answer')
s1,d1=A('teacher','POST',f'{Q}/tickets/{t}/knowledge',{'question':q,'answer':a}); s2,d2=A('teacher','POST',f'{Q}/tickets/{t}/knowledge',{'question':q,'answer':a+' (đã sửa)'})
rec('TC-P4-05-36',s1==201 and s2==200,f'lưu lần đầu → {s1}; lần hai → {s2} {str(d1)[:80]}')
rr=[A(u,'POST',f'{Q}/tickets/{t}/knowledge',{'question':q,'answer':a})[0] for u in ('sv09','admin','r5t')]; rec('TC-P4-05-39',all(x==403 for x in rr),f'SV/ADMIN/GV lớp khác lưu tri thức → {rr}')
time.sleep(8); print(psql("select count(*) from knowledge_items where course_id='%s'"%E1), psql(f"select count(*) from content_chunks where course_ids::text like '%{E1}%'"))
rec('TC-P4-05-38',psql("select count(*) from knowledge_items where course_id='%s'"%E1)!='0','có knowledge_items sau khi lưu')
s3,_=A('teacher','DELETE',f'{Q}/tickets/{t}/knowledge'); s4,_=A('teacher','DELETE',f'{Q}/tickets/{t}/knowledge')
rec('TC-P4-05-43',s3 in(200,204) and s4 in(200,204),f'gỡ tri thức → {s3}, lặp → {s4}')
# ngưỡng
s,d=A('teacher','GET',f'/courses/{E1}'); print(s,{k:d.get(k) for k in ('version','escalation_threshold')} if isinstance(d,dict) else d)
v=d.get('version'); th=d.get('escalation_threshold')
s1,d1=A('teacher','PUT',f'{Q}/settings',{'escalation_threshold':0.85,'version':v}); print(s1,d1)
v2=d1.get('version',v+1) if isinstance(d1,dict) else v+1
bad=[A('teacher','PUT',f'{Q}/settings',{'escalation_threshold':x,'version':v2})[0] for x in (0.49,0.96,0.82,'abc')]
rec('TC-P4-05-47',s1==200 and bad==[422,422,422,422],f'0.85 → {s1}; 0.49/0.96/0.82/"abc" → {bad}')
old=A('teacher','PUT',f'{Q}/settings',{'escalation_threshold':0.8,'version':v})[0]; r=[A(u,'PUT',f'{Q}/settings',{'escalation_threshold':0.8,'version':v2})[0] for u in ('ta','sv09','admin')]
rec('TC-P4-05-48',old in(409,412) and all(x==403 for x in r),f'version cũ → {old}; TA/SV/ADMIN → {r}')
A('teacher','PUT',f'{Q}/settings',{'escalation_threshold':th if th else 0.7,'version':v2})
