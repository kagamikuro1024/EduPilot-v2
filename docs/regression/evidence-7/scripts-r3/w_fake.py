import sys
sys.argv=['x']
from lib3 import *
import fx as FXm   # tái dùng fixture QC3S6 (ctx + ask)
QA=FXm.QA
A_=C7['A']; Q=f'/courses/{A_}'
for u in ('sv04','sv05','teacher','qc3s11','qc3t','qc3s12','qc3s13','admin','qc3s14'): login(u)
R={}
orig=fakeroute('qc3','Sprint7 cửa sổ fake-LLM (P4-03-24, P4-05-42, P4-06-44, P4-07-E1/E2, P4-01-33); ≤ 6 phút')
try:
    fakeset(error_rate=0,latency_min_ms=0,latency_max_ms=30,stream_delay_ms=0)
    # --- 24: STAFF_ANSWER / ESCALATION_FOLLOWUP không vào lịch sử gửi LLM
    r1=FXm.ask('sv04','Quy định nộp bài muộn của lớp là gì?'); sid=r1['sid']
    am=psql(f"select id from chat_messages where session_id='{sid}' and role='ASSISTANT' order by created_at desc limit 1")
    s,d=A('sv04','POST',f'/chat/messages/{am}/escalate'); tid=d['ticket']['id']; QQ=f'/courses/{QA}'
    A('teacher','POST',f'{QQ}/tickets/{tid}/claim'); sa,_=A('teacher','POST',f'{QQ}/tickets/{tid}/answer',{'body':'Chào em, QCSTAFF24 nộp muộn trừ 10 phần trăm mỗi ngày.'})
    sf,_=A('sv04','POST',f'/chat/tickets/{tid}/followup',{'content':'QCFOLLOW24 em chưa hiểu ý thầy','client_msg_id':str(uuid.uuid4())})
    stsess=psql(f"select session_id from escalation_tickets where id='{tid}'")
    r2=FXm.ask('sv04','Quy định về điểm danh của lớp là gì?',sid=stsess)
    fl=FXm.flat(r2['payload']) if r2['payload'] else ''
    R['24']=(s,sa,sf,len(r2['payload']),'QCSTAFF24' in fl,'QCFOLLOW24' in fl,r2['row'])
    # --- 33 auto ticket thật
    r3=FXm.ask('sv05','Giải thích chi tiết định lý Fermat nhỏ trong số học là gì?',pre_ctx=False)
    am3=psql(f"select id from chat_messages where session_id='{r3['sid']}' and role='ASSISTANT' order by created_at desc limit 1")
    ss,dd=A('sv05','GET',f"/chat/sessions/{r3['sid']}/messages"); R['33']=(r3['row'],tk(am3))
    # --- E1: LLM chết, câu khủng hoảng vẫn câu mẫu
    fakeset(error_rate=1.0,error_kind='SERVER'); 
    r4=chat('qc3s13',A_,'Em không muốn sống nữa.') 
    R['E1']=(r4[4], psql(f"select intent||'/'||stream_status||'/'||degraded||'/'||left(content,60) from chat_messages where id='{r4[1]}'"))
    # --- 05-42 / 06-44: embedding chết (fake error_rate=1 đang bật)
    ta,_=tick('qc3s11',A_); A('qc3t','POST',f'{Q}/tickets/{ta}/claim'); A('qc3t','POST',f'{Q}/tickets/{ta}/answer',{'body':'Chào em, nộp bài muộn bị trừ điểm theo quy chế của lớp.'})
    s,pv=A('qc3t','POST',f'{Q}/tickets/{ta}/knowledge-preview',{}); R['pv']=(s,json.dumps(pv,ensure_ascii=False)[:200])
    qn=(pv.get('question') if isinstance(pv,dict) else None) or 'Quy định nộp bài muộn QCKB42 của lớp là gì?'; an=(pv.get('answer') if isinstance(pv,dict) else None) or 'Nộp muộn bị trừ điểm theo quy chế QCKB42.'
    sk,dk=A('qc3t','POST',f'{Q}/tickets/{ta}/knowledge',{'question':qn,'answer':an}); R['kb_save']=(sk,json.dumps(dk,ensure_ascii=False)[:200])
    time.sleep(8)
    R['kb_emb']=psql(f"select count(*)||'/'||count(embedding) from content_chunks c join knowledge_items k on k.chunk_id=c.id where k.source_id='{ta}'")
    sdet,ddet=A('qc3t','GET',f'{Q}/tickets/{ta}'); R['kb_state']=json.dumps(ddet.get('knowledge'),ensure_ascii=False)[:200]
    # 06-44 VERIFY khi embedding chết
    s,th=A('qc3s14','POST',f'{Q}/threads',{'title':'Thảo luận QCV44 AI','body':'Nội dung câu hỏi cho bài AI chờ duyệt QCV44.','tags':[]}); thid=th['thread']['id']
    psql(f"update forum_threads set ai_state='PENDING' where id='{thid}'")
    pid=psql(f"insert into forum_posts(course_id,thread_id,kind,body,verification_state,confidence,ai_body) values('{A_}','{thid}','AI','Câu trả lời AI QCV44','PENDING',0.5,'Câu trả lời AI QCV44') returning id").split('\n')[0]
    psql(f"update forum_threads set ai_state='ANSWERED' where id='{thid}'")
    ai0=int(A('qc3t','GET','/me/today')[1].get('count',0))
    sv,dv=A('qc3t','POST',f'/courses/{A_}/posts/{pid}/verify',{'version':1}); R['verify']=(sv,json.dumps(dv,ensure_ascii=False)[:160])
    R['verify_state']=psql(f"select verification_state from forum_posts where id='{pid}'")
    # phục hồi embedding, thử lại lưu tri thức
    fakeset(error_rate=0)
    sk2,dk2=A('qc3t','POST',f'{Q}/tickets/{ta}/knowledge',{'question':qn,'answer':an}); time.sleep(8)
    R['kb_retry']=(sk2,psql(f"select count(*)||'/'||count(embedding) from content_chunks c join knowledge_items k on k.chunk_id=c.id where k.source_id='{ta}'"))
    R['tid']=ta; R['pid']=pid
finally:
    fakeset(error_rate=0)
    restoreroute('qc3',orig,'cửa sổ fake-LLM sprint 7 xong')
print(json.dumps(R,ensure_ascii=False,indent=0)[:3500]); json.dump(R,open('/tmp/qc3e/w_fake.json','w'),ensure_ascii=False)
