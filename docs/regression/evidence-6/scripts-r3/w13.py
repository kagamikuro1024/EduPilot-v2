from w7lib import *
import fx,threading
fx.use('QC3S6C','QKTWQ3C'); Q=fx.QA
q05='Quy chế bảo mật mật khẩu nói gì về độ dài tối thiểu của mật khẩu'
def setfake(**k): return A('admin','POST','/_test/llm/fake',k)
def clr(): subprocess.run(['docker','exec','ep-r-redis-1','sh','-c',"redis-cli --scan --pattern 'ep:ans:*' | xargs -r redis-cli del; redis-cli --scan --pattern 'ep:emb:*' | xargs -r redis-cli del"],capture_output=True)
def mid_of(sid): return psql(f"select id from chat_messages where session_id='{sid}' and role='ASSISTANT' order by created_at desc limit 1")
ask('sv.gioi',q05,course=Q,title='Bài ghi chú của Vũ Hoàng Giang')
# ---- 14: huỷ
clr(); setfake(stream_delay_ms=700); sid=newsess('sv.gioi',Q); tk=login('sv.gioi'); res={}
t=threading.Thread(target=lambda: res.setdefault('r',sse('POST',f'/chat/sessions/{sid}/messages',tk,{'content':q05},maxsec=60))); t.start()
time.sleep(3.5); m=mid_of(sid); t0=time.time(); s,_=A('sv.gioi','POST',f'/chat/messages/{m}/cancel'); t.join(); dt=time.time()-t0
setfake(stream_delay_ms=20); time.sleep(2)
row=psql(f"select stream_status||'|'||(partial_content is not null)::text||'|'||length(coalesce(partial_content,'')) from chat_messages where id='{m}'")
aud=psql("select status from llm_audit where user_id=(select id from users where email='sv.gioi@edupilot.local') and task='CHAT' order by created_at desc limit 1")
lock=rds('exists','ep:chat:active:'+psql("select id from users where email='sv.gioi@edupilot.local'"))
rec('TC-P3-05-14-api',s in(200,204) and row.startswith('CANCELLED|true') and aud=='cancelled' and dt<1.5 and lock=='0',f'huỷ sau 3,5 s: cancel→{s}, luồng SSE kết thúc sau {dt:.2f}s; hàng tin={row}; llm_audit.status={aud}; khoá ep:chat:active nhả (exists={lock}); UI chữ "Đã dừng." kiểm ở nhóm UI')
# ---- 17: quá tải thật bằng rpm_limit=1 trên provider fake (hàng chờ > LLM_QUEUE_WAIT_MAX → ErrOverloaded)
FP='01a1df2d-a8ed-7389-b1d8-d6805e510bed'
prov=[p for p in A('admin','GET','/admin/llm/providers')[1]['items'] if p['id']==FP][0]; ver=prov['version']
def put(rpm): 
    body={'type':'fake','name':prov['name'],'enabled':True,'version':ver if rpm else None}
    return A('admin','PUT','/admin/llm/providers/'+FP,{'type':'fake','name':prov['name'],'enabled':True,'rpm_limit':rpm,'version':A('admin','GET','/admin/llm/providers')[1]['items'][[p['id'] for p in A('admin','GET','/admin/llm/providers')[1]['items']].index(FP)]['version']})
try:
    print('rpm',put(1)[0]); clr()
    chat('sv.gioi',q05,course=Q)  # dùng hết 1 rpm
    clr(); r=chat('sv.gioi',q05,course=Q,maxsec=60)
    errf=[json.loads(d) for e,d,*_ in r['events'] if e=='error']; m17=mid_of(r['sid']); rw=psql(f"select stream_status||'|'||coalesce(error_code,'') from chat_messages where id='{m17}'")
finally:
    print('rpm restore',put(None)[0])
rec('TC-P3-05-17-api',bool(errf) and errf[0].get('code')=='OVERLOADED' and errf[0].get('retry_after',0)>0 and rw=='FAILED|OVERLOADED',f'quá tải thật (provider rpm_limit=1, câu thứ hai chờ quá LLM_QUEUE_WAIT_MAX=10s): khung error={errf[:1]}; hàng tin={rw}; (retry_after theo công thức SRS clamp 1–30 s; giá trị 20 của TC là fixture riêng của sprint 6)')
a0=psql(f"select attempt from chat_messages where id='{m17}'"); s,_=A('sv.gioi','POST',f'/chat/messages/{m17}/retry'); time.sleep(4)
rec('TC-P3-05-17-retry',s in(200,202),f'retry sau quá tải → {s} (ca 18 đã P)')
# ---- 20/21 suy giảm (xoá cache trả lời trước)
q20='Hướng dẫn cấu hình nhật ký kiểm toán trên máy chủ web ra sao'
ask('sv.gioi',q20,course=Q); clr()
setfake(error_rate=1,error_kind='SERVER'); r=chat('sv.gioi',q20,course=Q,maxsec=60); setfake(error_rate=0)
dg=[n for n in r['notice'] if n.get('degraded')]
rec('TC-P3-05-20',bool(dg) and 'Trả lời tạm thời, trích nguyên văn từ tài liệu của lớp.' in r['text'] and not re.search(r'(?i)provider|fallback|không khả dụng',r['text']),f'mọi provider lỗi + có ngữ cảnh: notice={r["notice"]}; văn bản={r["text"][:150]!r}')
clr(); setfake(error_rate=1,error_kind='SERVER'); r=chat('sv.gioi','Giá vàng hôm nay bao nhiêu?',course=Q,maxsec=60); setfake(error_rate=0)
alltext=r['text']+' '+' '.join(json.dumps(x,ensure_ascii=False) for x in r['err'])
rec('TC-P3-05-21','AI đang gián đoạn. Thử lại sau.' in alltext and 'giảng viên sẽ xem' not in alltext,f'mọi provider lỗi + câu không ngữ cảnh: text={r["text"]!r}, khung lỗi={r["err"]}, row={r["row"]}; không có cụm "giảng viên sẽ xem"')
json.dump(R,open('/tmp/qc3e/w13.json','w'),ensure_ascii=False)
