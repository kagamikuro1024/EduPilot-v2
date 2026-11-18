from w7lib import *
import fx,threading,urllib.request,urllib.error,ssl
fx.use('QC3S6C','QKTWQ3C'); Q=fx.QA
def setfake(**k): return A('admin','POST','/_test/llm/fake',k)
def mid_of(sid): return psql(f"select id from chat_messages where session_id='{sid}' and role='ASSISTANT' order by created_at desc limit 1")
def start_slow(u,text,delay=600,course=Q):
    sid=newsess(u,course); setfake(stream_delay_ms=delay); tk=login(u); res={}
    def run(): res['r']=sse('POST',f'/chat/sessions/{sid}/messages',tk,{'content':text},maxsec=60)
    t=threading.Thread(target=run); t.start(); return sid,t,res
TITLE='Bài ghi chú của Vũ Hoàng Giang'
q05='Quy chế bảo mật mật khẩu nói gì về độ dài tối thiểu của mật khẩu'
ask('sv.gioi',q05,title=TITLE,course=Q)   # tạo ngữ cảnh (đoạn trùng vectơ, tiêu đề chứa tên → tên thật phải hiện sau khôi phục)
# ===== 05 + 07 + 11 (cùng một lượt chậm)
subprocess.run(['docker','exec','ep-r-redis-1','sh','-c',"for p in 'ep:ans:*'; do redis-cli --scan --pattern $p | xargs -r redis-cli del; done"],capture_output=True)
sid,t,res=start_slow('sv.gioi',q05,delay=700)
seen=[];lens=[];snap=None;mid=None;bufdel=False
t0=time.time()
while t.is_alive() and time.time()-t0<40:
    row=psql(f"select role||'/'||stream_status||'/'||(partial_content is null)::text||'/'||coalesce(length(partial_content),0) from chat_messages where session_id='{sid}' order by created_at desc, role desc limit 2")
    if row: seen.append(row.replace('\n',' | '))
    pl=psql(f"select coalesce(length(partial_content),0) from chat_messages where session_id='{sid}' and role='ASSISTANT'")
    if pl: lens.append((round(time.time()-t0,2),int(pl)))
    if not mid: mid=mid_of(sid) or None
    if mid and not bufdel and time.time()-t0>2.0:
        rds('del',f'ep:chat:buf:{mid}:1'); bufdel=True
        tk=login('sv.gioi'); st,ev=sse('GET',f'/chat/messages/{mid}/stream',tk,maxsec=8); snap=(st,ev[0] if ev else None)
    time.sleep(0.4)
t.join(); st,ev=res['r']
fin=psql(f"select stream_status||'|'||(partial_content is null)::text||'|'||(content=(select string_agg(x,'' order by o) from (select (json_array_elements_text('[]'::json)) x, 0 o) z)) from chat_messages where id='{mid}'")
fin=psql(f"select stream_status||'|'||(partial_content is null)::text||'|'||length(content) from chat_messages where id='{mid}'")
text=''.join(json.loads(d)['t'] for e,d,*_ in ev if e=='token')
dbtext=psql(f"select content from chat_messages where id='{mid}'")
first_seen=next((s for s in seen if 'ASSISTANT/STREAMING' in s),None)
rec('TC-P3-05-05',first_seen is not None and 'USER/DONE' in first_seen and fin.startswith('DONE|true|') and dbtext==text,f'trong khi sinh: {seen[1] if len(seen)>1 else seen[:1]} (USER/DONE + ASSISTANT/STREAMING); sau done: {fin} (ASSISTANT/DONE, partial NULL), content==văn bản SSE: {dbtext==text}')
ch=[(a,b) for (a,b) in lens if b>0]; incr=[ch[i] for i in range(len(ch)) if i==0 or ch[i][1]!=ch[i-1][1]]
gaps=[round(incr[i+1][0]-incr[i][0],2) for i in range(len(incr)-1)]
pc=psql(f"select partial_content from chat_messages where id='{mid}'")
rec('TC-P3-05-07',len(incr)>=3 and all(incr[i+1][1]>incr[i][1] for i in range(len(incr)-1)) and 0.5<=sum(gaps)/max(len(gaps),1)<=1.6 and '[[SV_' not in text and 'Vũ Hoàng Giang' in text,f'độ dài partial_content đo mỗi 0,4 s: các lần đổi {incr[:6]}; khoảng cách giữa hai lần đổi (s)={gaps[:6]} (TB {sum(gaps)/max(len(gaps),1):.2f}); văn bản cuối có tên thật="Vũ Hoàng Giang" in text={"Vũ Hoàng Giang" in text}, không có [[SV_')
rec('TC-P3-05-11',snap is not None and snap[1] is not None and snap[1][0]=='snapshot' and '"off":0' in snap[1][1].replace(' ',''),f'DEL ep:chat:buf:{{mid}}:1 giữa lượt rồi GET …/stream không Last-Event-ID → HTTP {snap[0] if snap else None}, khung đầu = {snap[1][:2] if snap and snap[1] else None}')
# ===== 13: rớt kết nối sau token đầu
sid=newsess('sv.gioi',Q); setfake(stream_delay_ms=300)
req=urllib.request.Request(B+f'/chat/sessions/{sid}/messages',data=json.dumps({'content':q05+' (13)'}).encode(),method='POST',headers={'content-type':'application/json','authorization':'Bearer '+login('sv.gioi'),'Idempotency-Key':str(uuid.uuid4()),'Accept':'text/event-stream'})
r=urllib.request.urlopen(req,context=ctx,timeout=30); got=False
for ln in r:
    if ln.decode().startswith('event: token'): got=True; break
r.close(); m13=mid_of(sid)
for i in range(60):
    st13=psql(f"select stream_status||'|'||length(content)||'|'||extract(epoch from completed_at-created_at)::int from chat_messages where id='{m13}'")
    if st13.startswith('DONE') or st13.startswith('FAILED'): break
    time.sleep(1)
rec('TC-P3-05-13',got and st13.startswith('DONE') and int(st13.split('|')[1])>20 and int(st13.split('|')[2])<=120,f'ngắt kết nối ngay sau token đầu: sau {i+1}s hàng tin = {st13} (stream_status|độ dài content|giây hoàn tất) — rớt kết nối không huỷ')
# ===== 14 (API): huỷ giữa lượt
sid,t,res=start_slow('sv.gioi',q05+' (14)',delay=700); time.sleep(3); m=mid_of(sid); c0=time.time(); s,_=A('sv.gioi','POST',f'/chat/messages/{m}/cancel'); t.join(); 
row=psql(f"select stream_status||'|'||(partial_content is not null)::text||'|'||length(coalesce(partial_content,'')) from chat_messages where id='{m}'"); time.sleep(2)
aud=psql("select status from llm_audit where user_id=(select id from users where email='sv.gioi@edupilot.local') and task='CHAT' order by created_at desc limit 1")
rec('TC-P3-05-14-api',s in(200,204) and row.startswith('CANCELLED|true') and aud=='cancelled',f'cancel → {s}; hàng tin={row}; llm_audit.status={aud} (UI "Đã dừng." kiểm ở nhóm UI)')
# ===== 17 / 18
setfake(stream_delay_ms=20,error_rate=1,error_kind='RATE_LIMIT',retry_after_ms=20000)
sid=newsess('sv.gioi',Q); tk=login('sv.gioi'); st,ev=sse('POST',f'/chat/sessions/{sid}/messages',tk,{'content':q05+' (17)'},maxsec=60)
errf=[d for e,d,*_ in ev if e=='error']; m17=mid_of(sid); rw=psql(f"select stream_status||'|'||coalesce(error_code,'') from chat_messages where id='{m17}'")
rec('TC-P3-05-17-api',bool(errf) and '"OVERLOADED"' in errf[0].replace(' ','') and '"retry_after":20' in errf[0].replace(' ','') and rw=='FAILED|OVERLOADED',f'khung error={errf[:1]}; hàng tin={rw} (UI "AI đang bận. Thử lại sau khoảng 20 giây." kiểm ở nhóm UI)')
setfake(error_rate=0,retry_after_ms=0)
ub0=psql(f"select count(*) from chat_messages where session_id='{sid}' and role='USER'"); a0=psql(f"select attempt from chat_messages where id='{m17}'")
s,_=A('sv.gioi','POST',f'/chat/messages/{m17}/retry'); time.sleep(6); a1=psql(f"select attempt||'|'||stream_status from chat_messages where id='{m17}'"); ub1=psql(f"select count(*) from chat_messages where session_id='{sid}' and role='USER'"); asst=psql(f"select count(*) from chat_messages where session_id='{sid}' and role='ASSISTANT'")
g=subprocess.run(GO+"go test ./internal/chat -run 'TestOverloadedEvent|TestRetryReusesAssistantRow' -v -count=1 2>&1 | grep -E '^(--- |ok|FAIL)'",shell=True,capture_output=True,text=True).stdout
rec('TC-P3-05-18',s in(200,202) and a1.split('|')[0]==str(int(a0)+1) and ub0==ub1 and asst=='1' and 'FAIL' not in g,f'retry → {s}; cùng hàng ASSISTANT {m17[:8]} attempt {a0}→{a1}; hàng USER {ub0}→{ub1}; số hàng ASSISTANT={asst}; go test: {g.replace(chr(10)," | ")[:120]}')
# ===== 20/21 suy giảm
q20='Hướng dẫn cấu hình nhật ký kiểm toán trên máy chủ web ra sao'
ask('sv.gioi',q20,course=Q)
setfake(error_rate=1,error_kind='SERVER'); r=chat('sv.gioi',q20,course=Q); setfake(error_rate=0)
dg=[n for n in r['notice'] if n.get('degraded')]
rec('TC-P3-05-20',bool(dg) and 'Trả lời tạm thời, trích nguyên văn từ tài liệu của lớp.' in r['text'] and not re.search(r'(?i)provider|fallback|tạm thời không khả dụng',r['text']),f'mọi provider lỗi: notice={r["notice"]}; văn bản={r["text"][:140]!r}')
setfake(error_rate=1,error_kind='SERVER'); r=chat('sv.gioi','Giá vàng hôm nay bao nhiêu?',course=Q); setfake(error_rate=0)
g=subprocess.run(GO+"go test ./internal/chat -run 'TestAllProvidersDownExtractive|TestAllProvidersDownNoContext|TestDegradedDropsLLMText|TestNoInstructorPromiseTextToStudent' -v -count=1 2>&1 | grep -E '^(--- |ok|FAIL)'",shell=True,capture_output=True,text=True).stdout
rec('TC-P3-05-21','AI đang gián đoạn. Thử lại sau.' in r['text'] and 'giảng viên sẽ xem' not in r['text'] and 'FAIL' not in g,f'câu không ngữ cảnh khi mọi provider lỗi: {r["text"]!r}; không có "giảng viên sẽ xem"; go test: {g.replace(chr(10)," | ")[:150]}')
setfake(stream_delay_ms=20,error_rate=0,retry_after_ms=0)
json.dump(R,open('/tmp/qc3e/w11.json','w'),ensure_ascii=False)
