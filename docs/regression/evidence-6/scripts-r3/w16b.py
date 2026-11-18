from w7lib import *
import fx
fx.use('QC3S6C','QKTWQ3C'); Q=fx.QA
def clr(): subprocess.run(['docker','exec','ep-r-redis-1','sh','-c',"redis-cli --scan --pattern 'ep:ans:*' | xargs -r redis-cli del"],capture_output=True)
tid=psql(f"select t.id from forum_threads t join forum_posts p on p.thread_id=t.id and p.kind='AI' where t.course_id='{Q}' limit 1")
# ---- 07-05: ngưỡng (chunk có cos≈0,5 với câu hỏi → confidence trung gian)
import math
q5='Phân tích điểm yếu của giao thức SSL cũ trong bối cảnh hiện đại'
vq=[float(x) for x in fx.vec([q5])[0].strip('[]').split(',')]; vr=[float(x) for x in fx.vec(['nội dung khác hoàn toàn xyz'])[0].strip('[]').split(',')]
mix=[0.5*a+0.866*b for a,b in zip(vq,vr)]; n=math.sqrt(sum(x*x for x in mix)); mix=[x/n for x in mix]
cosq=sum(a*b for a,b in zip(vq,mix))
d5=fx.doc('QC3 chunk cos 0,5'); nx=int(psql(f"select coalesce(max(ord)+1,0) from content_chunks where document_id='{d5}'"))
psql(f"delete from content_chunks where document_id='{d5}'")
psql(f"insert into content_chunks (document_id,course_ids,audience,ord,text,embedding) values ('{d5}',array['{Q}'::uuid],'ALL',0,'Ghi chú liên quan SSL cũ và các điểm yếu','[{','.join(f'{x:.8f}' for x in mix)}]')")
def setthr(v):
    s,d=A('teacher','GET',f'/courses/{Q}'); d=d['settings']; return A('teacher','PUT',f'/courses/{Q}/settings',{'escalation_threshold':v,'version':d['version']}),d
(s1,d1),_=setthr('0.95'); clr(); lo=chat('sv.gioi',q5,course=Q)
low1=psql(f"select low_confidence||'|'||no_context from chat_messages where session_id='{lo['sid']}' and role='ASSISTANT'")
(s2,d2),_=setthr('0.50'); clr(); lo2=chat('sv.kha',q5,course=Q)
low2=psql(f"select low_confidence||'|'||no_context from chat_messages where session_id='{lo2['sid']}' and role='ASSISTANT'")
(s3,d3),_=setthr('0.99'); (s4,d4),_=setthr('0.40')
(s5,_),_=setthr('0.80')
g=subprocess.run(GO+"go test ./internal/agent -run 'TestLowConfidenceUsesCourseThreshold' -v -count=1 2>&1 | grep -E '^(--- |ok|FAIL)'",shell=True,capture_output=True,text=True).stdout
rec('TC-P3-07-05',s1==200 and s2==200 and s3==422 and s4==422 and low1.startswith('true') and low2.startswith('false') and 'FAIL' not in g,f'chunk có cos với câu hỏi ≈{cosq:.2f} (confidence trung gian); ngưỡng 0,95 → tin kế tiếp low_confidence|no_context={low1}; ngưỡng 0,50 → {low2} (cùng câu, hiệu lực ở tin kế tiếp); 0,99 và 0,40 bị từ chối → {s3},{s4}; trả lại 0,80 → {s5}; go test: {g.replace(chr(10)," | ")[:120]}')
# ---- 07-06: SV không thấy con số ở mọi đường đọc
sid=lo['sid']; bad=[]
def keys(o,acc):
    if isinstance(o,dict):
        for k,v in o.items(): acc.add(k); keys(v,acc)
    elif isinstance(o,list):
        for v in o: keys(v,acc)
    return acc
FORB={'confidence','retrieval_score','groundedness'}
srcs={}
srcs['chat messages']=A('sv.gioi','GET',f'/chat/sessions/{sid}/messages')[1]
srcs['threads list']=A('sv.gioi','GET',f'/courses/{Q}/threads')[1]; srcs['thread detail']=A('sv.gioi','GET',f'/courses/{Q}/threads/{tid}')[1]
srcs['thread similar']=A('sv.gioi','GET',f'/courses/{Q}/threads/{tid}/similar')[1]
for k,v in srcs.items():
    if keys(v,set())&FORB: bad.append(k)
sse_txt=open('/tmp/qc3e/sse_all.txt').read(); sseb=[l for l in sse_txt.splitlines() if re.search(r'"(confidence|retrieval_score|groundedness)"',l)]
g=subprocess.run(GO+"go test ./internal/chat ./internal/thread ./internal/contract -run 'TestStudentNeverSeesConfidence' -v -count=1 2>&1 | grep -E '^(--- |ok|FAIL)'",shell=True,capture_output=True,text=True).stdout
rec('TC-P3-07-06',not bad and not sseb and 'FAIL' not in g,f'khoá confidence/retrieval_score/groundedness ở: chat messages, GET threads, GET thread, similar = {bad or "0"}; tất cả khung SSE đã ghi (>{len(sse_txt.splitlines())} dòng) chứa chúng={len(sseb)}; chỉ có low_confidence (bool); go test: {g.replace(chr(10)," | ")[:140]}')
json.dump(R,open('/tmp/qc3e/w16.json','w'),ensure_ascii=False)
