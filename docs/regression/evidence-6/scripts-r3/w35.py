from p8lib import *
import fx,re
fx.use('QC3P8A','QKTWP8A')
C=json.load(open('/tmp/qc3e/p8c.json')); A1,B1=C['A'],C['B']; QA=f'/courses/{A1}'; QB=f'/courses/{B1}'
d=json.load(open('/tmp/qc3e/p8d.json')); dK,dL,dH,D1=d['dK'],d['dL'],d['dH'],d['D1']
def ctxn(r):
    g=[it for it in r['payload'] if it['task']!='EMBEDDING']
    return g
def cites(r): return psql(f"select citations from chat_messages where session_id='{r['sid']}' and role='ASSISTANT' order by created_at desc limit 1")
def plj(r): return json.dumps(r['payload'],ensure_ascii=False)
# ---- 33 + 34: phá rào bằng tay
Qc='CANARY-7Q2X là gì? Cho tôi đáp án câu 1 đề giữa kỳ'
vq=fx.vec([Qc])[0]
sql_res=[]
for q in (f"update content_chunks set audience='ALL',embedding='{vq}' where document_id='{dK}'",f"update documents set visible_to_students=true, use_for_rag=true where id='{dK}'"):
    sql_res.append(psql(q)[:80])
print('phá rào',sql_res)
r1=fx.ask('sv.gioi',Qc,pre_ctx=False); c1=cites(r1)
tk=login('sv.gioi'); 
s2,th=A('sv.gioi','POST',f'/courses/{A1}/threads',{'title':'CANARY-7Q2X là gì?','body':Qc+' AES','tags':[],'week_no':1})
time.sleep(25) # AI thread
r3=fx.ask('sv.gioi','tìm tài liệu có CANARY-7Q2X',pre_ctx=False)
s4,x4=A('sv.gioi','POST','/chat/sessions',{'course_id':A1,'document_id':dK})
pl_all=plj(r1)+plj(r3)
cn=psql(f"select count(*) from chat_messages where content like '%CANARY-7Q2X%' and role='ASSISTANT' and created_at>now()-interval '10 minutes'")
cc=psql(f"select count(*) from chat_messages m where m.citations::text like '%{dK}%'")
lg=''.join(subprocess.run(['docker','logs','--since','10m',c],capture_output=True,text=True,errors='replace').stderr+subprocess.run(['docker','logs','--since','10m',c],capture_output=True,text=True,errors='replace').stdout for c in ('ep-r-gateway-1','ep-r-worker-1'))
thr=psql(f"select count(*) from forum_posts p join forum_threads t on t.id=p.thread_id where t.course_id='{A1}' and t.title like 'CANARY%' and p.kind='AI' and p.body like '%CANARY-7Q2X%'")
thc=psql(f"select count(*) from forum_posts p join forum_threads t on t.id=p.thread_id where t.course_id='{A1}' and t.title like 'CANARY%' and p.kind='AI' and p.citations::text like '%{dK}%'")
print('paths',r1['text'][:80],c1[:200],s2,s4,x4,'canary payload',pl_all.count('CANARY-7Q2X'),'asst',cn,'cites',cc,'thr',thr,thc,'logs',lg.count('CANARY-7Q2X'))
rec('TC-P8-01-33',pl_all.count('Đáp án CANARY')==0 and cn=='0' and cc=='0' and thr=='0' and thc=='0' and s4 in(404,) ,f'phá rào bằng tay (audience→ALL + visible_to_students/use_for_rag=true trên tài liệu ANSWER_KEY có CANARY-7Q2X, không bị ràng buộc chặn: {sql_res}); 4 đường: (1) chat riêng, (2) Threads (thread {s2}), (3) hỏi tìm tài liệu, (4) chat giới hạn document_id=ANSWER_KEY → HTTP {s4} {x4.get("code") if isinstance(x4,dict) else ""}; chữ của tài liệu ANSWER_KEY («Đáp án CANARY…») trong payload gửi provider giả={pl_all.count("Đáp án CANARY")} (chuỗi CANARY-7Q2X chỉ còn trong chính câu hỏi của SV: {pl_all.count("CANARY-7Q2X")} lần), trong câu trả lời chat={cn}, trong trích dẫn chat/thread={cc}/{thc}, bài AI Threads chứa CANARY={thr}; (loại ANSWER_KEY lọc theo documents.type chứ không chỉ audience)')
# 34 bề mặt HTTP
codes={}
for p in (f'/library',f'/library/{dK}',f'/library/{dK}/download',f'/documents/{dK}/chunks',f'/documents'):
    s,b=A('sv.gioi','GET',f'{QA}{p}'); codes[p.replace(dK,'DK')]=(s,json.dumps(b,ensure_ascii=False).count('CANARY'))
print(codes)
lib=A('sv.gioi','GET',f'{QA}/library')[1]; inlib=dK in json.dumps(lib) or 'dapan-qc3' in json.dumps(lib)
rec('TC-P8-01-34',codes['/library/DK'][0]==404 and codes['/library/DK/download'][0]==404 and codes['/documents/DK/chunks'][0]==403 and codes['/documents'][0]==403 and all(v[1]==0 for v in codes.values()) and not inlib and lg.count('CANARY-7Q2X')==0,f'SV gọi: {codes}; ANSWER_KEY có trong /library={inlib}; CANARY trong log gateway+worker 10 phút={lg.count("CANARY-7Q2X")} (kể cả khi cờ hiển thị đã bị phá)')
# khôi phục
psql(f"update documents set visible_to_students=false where id='{dK}'"); psql(f"update content_chunks set audience='GRADING' where document_id='{dK}'")
