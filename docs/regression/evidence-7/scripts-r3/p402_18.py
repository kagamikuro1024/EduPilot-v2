from lib3 import *
fm=json.load(open('/tmp/qc3e/fx_me.json')); M=fm['M']; Q=f'/courses/{M}'; T='qc3t'
ST=['qc3me01','qc3me02','qc3me03']; REP='qc3me04'
def th(u,title,body):
    s,d=A(u,'POST',f'{Q}/threads',{'title':title,'body':body,'tags':[]}); assert s==201,(s,d); return d['thread']['id']
tag=str(int(time.time()))[-5:]
rep=[th(u,f'Thảo luận báo cáo {tag}-{i}','Nội dung thảo luận bình thường của lớp về bài giảng tuần này, đủ dài để đăng.') for i,u in enumerate(ST)]
for t in rep: s,d=A(REP,'POST',f'{Q}/threads/{t}/report',{'reason':'SPAM'}); assert s in(200,201),(s,d)
aiT=[th(u,f'Câu hỏi AI {tag}-{i}','Em hỏi về nội dung bài giảng tuần này, mong thầy cô và AI giải đáp giúp em.') for i,u in enumerate(ST)]
tu=uid(T); sid=[uid(u) for u in ST]
for t in aiT:
    pid=psql(f"insert into forum_posts(course_id,thread_id,author_id,kind,body,verification_state,confidence,ai_body) values('{M}','{t}',null,'AI','Trả lời AI thử nghiệm','PENDING',0.7,'Trả lời AI thử nghiệm') on conflict do nothing returning id").split('\n')[0]
    psql(f"insert into outbox(topic,payload) values('thread.ai_answered',jsonb_build_object('course_id','{M}','thread_id','{t}','target_id','{pid or t}'))")
# 3 trả lời cho 3 SV
tids=[]
for u in ST:
    _,_,am=mkchat(u,M,q='Câu hỏi nhờ GV tag '+tag); s,d=esc(u,am); assert s==201,(s,d); t=d['ticket']['id']; A(T,'POST',f'{Q}/tickets/{t}/claim'); s,_=A(T,'POST',f'{Q}/tickets/{t}/answer',{'body':'Chào em, thầy trả lời câu hỏi của em đây.'}); assert s==200; tids.append(t)
for _ in range(40):
    ai=psql(f"select title||'|'||coalesce(read_at::text,'-') from notifications where user_id='{tu}' and course_id='{M}' and type='AI_PENDING'"); rp=psql(f"select title||'|'||coalesce(read_at::text,'-') from notifications where user_id='{tu}' and course_id='{M}' and type='POST_REPORTED'")
    if ai and rp: break
    time.sleep(2)
pers=[psql(f"select count(*) from notifications where user_id='{x}' and type='TICKET_ANSWERED' and created_at>now()-interval '3 minutes'") for x in sid]
cnt=psql(f"select count(*) from notifications where user_id='{tu}' and course_id='{M}' and type in('AI_PENDING','POST_REPORTED')")
rec('TC-P4-02-18',ai.startswith('3 câu trả lời AI chờ duyệt') and rp.startswith('3 bài bị báo cáo chờ xem') and cnt=='2' and pers==['1','1','1'],f'lớp QC3M5: 3 thread có bài AI PENDING (SQL + outbox thread.ai_answered), 3 thread bị báo cáo (API), 3 câu trả lời của GV cho 3 SV: GV có đúng {cnt} dòng gộp: AI_PENDING «{ai}», POST_REPORTED «{rp}» (mỗi loại MỘT dòng); thông báo cá nhân TICKET_ANSWERED của 3 SV = {pers} (3 dòng riêng, không gộp)')
