from w7lib import *
import fx
fx.use('QC3S6C','QKTWQ3C'); Q=fx.QA
def clr(): subprocess.run(['docker','exec','ep-r-redis-1','sh','-c',"redis-cli --scan --pattern 'ep:ans:*' | xargs -r redis-cli del"],capture_output=True)
# ---- 07-02: confidence trong [0,1], 3 chữ số, ResponseMetadata
tid=psql(f"select t.id from forum_threads t join forum_posts p on p.thread_id=t.id and p.kind='AI' where t.course_id='{Q}' limit 1")
s,d=A('teacher','GET',f'/courses/{Q}/threads/{tid}'); ai=[p for p in d['posts'] if p['kind']=='AI']
conf=ai[0].get('confidence') if ai else None
cs=psql(f"select p.confidence, (1-(t.embedding <=> c.embedding)) cos from forum_posts p join forum_threads t on t.id=p.thread_id cross join lateral (select embedding from content_chunks where course_ids @> array[t.course_id] and embedding is not null order by embedding <=> t.embedding limit 1) c where p.thread_id='{tid}' and p.kind='AI'")
cf,cos=[float(x) for x in cs.split('|')]
retr=min(max((cos-0.25)/(0.65-0.25),0),1)
rec('TC-P3-07-02',conf is not None and 0<=float(conf)<=1 and round(float(conf),3)==float(conf) and 0.6*retr-1e-3<=float(conf)<=0.6*retr+0.4+1e-3,f'GV GET thread: confidence={conf} (nhiều nhất 3 chữ số, ∈[0,1]); DB confidence (trước khi ca 09 ghi 0,72 lên một thread khác)={cf}; cos top1 thật={cos:.3f} → retr=clamp((cos−0,25)/0,40)={retr:.3f} ⇒ confidence phải ∈ [0,6·retr, 0,6·retr+0,4]=[{0.6*retr:.3f},{0.6*retr+0.4:.3f}]; go test TestConfidence* (công thức, đủ trường ResponseMetadata) ở dòng ghi chú kế')
g=subprocess.run(GO+"go test ./internal/agent -run 'Confidence' -v -count=1 2>&1 | grep -E '^(--- |ok|FAIL)'",shell=True,capture_output=True,text=True).stdout
print(g[:400])
# ---- 07-03: không thêm lời gọi LLM
clr(); st0=A('admin','GET','/_test/llm/stats')[1]['fake_calls']; e0=sum(st0.values())
r=ask('sv.gioi','Hãy giải thích cơ chế điều khiển truy cập dựa trên vai trò RBAC',course=Q); st1=A('admin','GET','/_test/llm/stats')[1]['fake_calls']; e1=sum(st1.values())
emb=len([i for i in r['payload'] if i['task']=='EMBEDDING'])
rec('TC-P3-07-03',e1-e0==1 and emb<=1,f'một câu COURSE_QA: fake_calls {e0}→{e1} (+{e1-e0} lời gọi sinh chữ) + {emb} lần nhúng; không có lời gọi thêm để chấm độ tin cậy (D47)')
# ---- 07-05: ngưỡng
def setthr(v): 
    s,d=A('teacher','GET',f'/courses/{Q}/settings'); return A('teacher','PUT',f'/courses/{Q}/settings',{'escalation_threshold':v,'version':d['version']}),d
(s1,d1),_=setthr('0.95'); lo=ask('sv.gioi','Phân tích điểm yếu của giao thức SSL cũ',course=Q)
low1=psql(f"select low_confidence from chat_messages where session_id='{lo['sid']}' and role='ASSISTANT'")
(s2,d2),_=setthr('0.50'); clr(); lo2=ask('sv.kha','Phân tích điểm yếu của giao thức SSL cũ',course=Q)
low2=psql(f"select low_confidence from chat_messages where session_id='{lo2['sid']}' and role='ASSISTANT'")
(s3,d3),_=setthr('0.99'); (s4,d4),_=setthr('0.40')
(s5,_),_=setthr('0.80')
g=subprocess.run(GO+"go test ./internal/agent -run 'TestLowConfidenceUsesCourseThreshold' -v -count=1 2>&1 | grep -E '^(--- |ok|FAIL)'",shell=True,capture_output=True,text=True).stdout
rec('TC-P3-07-05',s1==200 and s2==200 and s3==422 and s4==422 and low1=='t' and low2=='f' and 'FAIL' not in g,f'ngưỡng 0,95 → tin kế tiếp low_confidence={low1}; ngưỡng 0,50 → low_confidence={low2} (cùng câu, có hiệu lực ở tin kế tiếp); 0,99 và 0,40 bị từ chối → {s3},{s4}; trả lại 0,80 → {s5}; go test: {g.replace(chr(10)," | ")[:120]}')
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
sse_txt=open('/tmp/qc3e/sse_all.txt').read(); sseb=[l for l in sse_txt.splitlines() if re.search(r'confidence|retrieval_score|groundedness',l)]
g=subprocess.run(GO+"go test ./internal/chat ./internal/thread ./internal/contract -run 'TestStudentNeverSeesConfidence' -v -count=1 2>&1 | grep -E '^(--- |ok|FAIL)'",shell=True,capture_output=True,text=True).stdout
rec('TC-P3-07-06',not bad and not sseb and 'FAIL' not in g,f'khoá confidence/retrieval_score/groundedness ở: chat messages, GET threads, GET thread, similar = {bad or "0"}; tất cả khung SSE đã ghi (>{len(sse_txt.splitlines())} dòng) chứa chúng={len(sseb)}; chỉ có low_confidence (bool); go test: {g.replace(chr(10)," | ")[:140]}')
json.dump(R,open('/tmp/qc3e/w16.json','w'),ensure_ascii=False)
