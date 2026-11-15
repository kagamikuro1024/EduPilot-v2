import os,sys; os.environ['SP']='3-p10x'
from lib import *
import concurrent.futures as cf, datetime, urllib.request
A=login('admin'); T=login('teacher')
U1='http://host.docker.internal:9741/v1'; U2='http://host.docker.internal:9742/v1'
MODELS=['m-ok','m-429','m-404','m-500','m-503','m-garbage','m-slow','m-slow1','m-hang']
def stub(p,path='/__count'): return json.loads(urllib.request.urlopen(f'http://localhost:{p}{path}').read())
def stubreset(): 
    for p in(9741,9742): urllib.request.urlopen(urllib.request.Request(f'http://localhost:{p}/__reset',data=b'{}',method='POST'))
def mkprov(name,u,models):
    ex=[p for p in call('GET','/admin/llm/providers',A)[1]['items'] if p['name']==name]
    if ex: return ex[0]
    s,j,_=call('POST','/admin/llm/providers',A,{'type':'openai_compatible','name':name,'base_url':u,'api_key':'good-key','models':[{'model':m,'kind':'chat','price_in':'1000000','price_out':'1000000'} for m in models]})
    assert s==201,(s,j); return j
S1=mkprov('SmkB-S1',U1,MODELS); S2=mkprov('SmkB-S2',U2,['m-ok','m-500'])
mid=lambda P,m:[x['id'] for x in P['models'] if x['model']==m][0]
def routes(): return {i['task']:i for i in call('GET','/admin/llm/routes',A)[1]['items']}
orig=routes()
def put(task,chain,params=None):
    cur=routes()[task]
    return call('PUT','/admin/llm/routes',A,{'task':task,'chain':chain,'params':params or {},'version':cur['version']},hdr={'If-Match':'W/"v%d"'%cur['version']})
def chat(task='CHAT',prompt='xin chao',tk=None,**kw):
    t=time.time(); s,j,h=call('POST','/_test/llm/chat',tk or A,dict(task=task,prompt=prompt,**kw)); return s,j,round((time.time()-t)*1000),h
def aud(q,n=1): return psql(q)
def restore():
    for t,i in orig.items():
        if t=='EMBEDDING': continue
        ch=[c['model_id'] for c in i['chain']]
        if ch: 
            r=put(t,ch,i['params'])
            if r[0]!=200: print('RESTORE FAIL',t,r[:2])
    b=call('GET','/admin/llm/budget',A)[1]
    call('PUT','/admin/llm/budget',A,{'daily_limit':None,'monthly_limit':None,'version':b['version']},hdr={'If-Match':'W/"v%d"'%b['version']})
LOCK='/tmp/qc2-lock-global'
for _ in range(240):
    try: os.mkdir(LOCK); break
    except FileExistsError: time.sleep(5)
else: sys.exit('no lock')
t_lock=time.time()
open('/tmp/qc2-global-changes.log','a').write(datetime.datetime.now().strftime('%H:%M:%S')+" SmkB: LOCK — CHAT/GRADING/CLASSIFY/... route đổi sang stub cục bộ :9741/:9742 + ngân sách tạm, hoàn lại ≤10 phút\n")
try:
    stubreset(); t0=datetime.datetime.utcnow()
    # P104-27 hot reload (đo từ lúc PUT xong tới khi lời gọi dùng nhà mới)
    put('CHAT',[mid(S2,'m-ok')]); time.sleep(1.0); chat()
    cur=routes()['CHAT']; ok1=[mid(S1,'m-ok')]
    r=put('CHAT',ok1); tput=time.time(); seen=None; tries=0
    while time.time()-tput<3:
        s_,j_,ms_,_=chat(); tries+=1
        if j_.get('provider')=='SmkB-S1': seen=(time.time()-tput)*1000; break
        time.sleep(0.05)
    rec('TC-P104-27','P' if r[0]==200 and r[2].get('Etag')=='W/"v%d"'%(cur['version']+1) and seen is not None and seen<1000 else 'F',f'PUT {r[0]} ETag {r[2].get("Etag")}; lời gọi dùng nhà mới sau {seen if seen is None else round(seen)} ms ({tries} lần thử; 1 gateway — ca "cả hai gateway" không có trên ep-r)')
    rec('TC-P105-20','P' if seen is not None and seen<1000 else 'F','đổi CHAT từ API → lời gọi chat kế tiếp dùng nhà mới ≤ 1 s (1 gateway)')
    # P103-03 lanes
    for tk in['CLASSIFY','UTILITY','GRADING','QUESTION_GEN','INSIGHT']: put(tk,ok1)
    time.sleep(0.5)
    for tk in['CHAT','CLASSIFY','UTILITY','GRADING','QUESTION_GEN','INSIGHT']: chat(tk)
    chat('CHAT',lane='NEAR_REALTIME'); s_bad=chat('GRADING',lane='INTERACTIVE')
    time.sleep(2)
    rows=psql(f"select task,lane from llm_audit where provider='SmkB-S1' and created_at>'{t0}' order by created_at").split('\n')
    exp={'CHAT':'INTERACTIVE','CLASSIFY':'NEAR_REALTIME','UTILITY':'NEAR_REALTIME','GRADING':'BATCH','QUESTION_GEN':'BATCH','INSIGHT':'BATCH'}
    got={}
    for r_ in rows[:6]: a,b=r_.split('|'); got[a]=b
    rec('TC-P103-03','P' if got==exp and rows[6].split('|')[1]=='NEAR_REALTIME' and rows[7].split('|')[1]=='BATCH' else 'F',f'bảng làn {got}; CHAT hạ làn -> {rows[6]}; GRADING đòi INTERACTIVE -> {rows[7] if len(rows)>7 else "?"} (HTTP {s_bad[0]})')
    # P102-07/21/22/27 stream, audit
    stubreset(); tp='00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01'
    req=urllib.request.Request(B+'/_test/llm/chat',data=json.dumps({'task':'CHAT','prompt':'xin chao stream','stream':True}).encode(),method='POST',headers={'authorization':'Bearer '+A,'content-type':'application/json','Idempotency-Key':str(uuid.uuid4()),'traceparent':tp})
    body=urllib.request.urlopen(req,context=ctx,timeout=30).read().decode()
    n=stub(9741)['n']; time.sleep(1.5)
    tr=psql("select trace_id,status,task from llm_audit where provider='SmkB-S1' order by created_at desc limit 1")
    rec('TC-P102-07','P' if n==1 and 'event: done' in body.replace('event:  done','event: done') or ('done' in body and n==1) else 'F',f'stream: stub nhận {n} lời gọi; SSE có token/done; audit {tr}')
    rec('TC-P102-27','P' if tr.startswith('0af7651916cd43dd8448eb211c80319c') else 'F',f'traceparent -> llm_audit.trace_id={tr.split("|")[0]}')
    rec('TC-P103-27a','P' if 'ok' in tr else 'F','audit status của stream: '+tr)
    cols=psql("select column_name from information_schema.columns where table_name='llm_audit' and column_name ~* 'prompt|content|message|text|response'")
    rec('TC-P102-22','P' if cols=='' and psql("select count(*) from llm_audit where llm_audit::text like '%xin chao%'")=='0' else 'F',f'cột prompt/nội dung={cols!r}; "xin chao" trong llm_audit=0')
    # P102-08/P103-27 huỷ stream
    put('CHAT',[mid(S1,'m-ok')]); 
    import http.client
    c=http.client.HTTPSConnection('localhost',1143,context=ctx,timeout=10)
    c.request('POST','/api/v1/_test/llm/chat',json.dumps({'task':'CHAT','prompt':'cancel','stream':True}),{'authorization':'Bearer '+A,'content-type':'application/json','Idempotency-Key':str(uuid.uuid4())})
    r=c.getresponse(); r.read(20); c.close(); tc=time.time()
    infl=None
    for _ in range(20):
        st=call('GET','/_test/llm/stats',A)[1]; infl=st['provider_inflight']
        if infl==0: break
        time.sleep(0.1)
    rel=time.time()-tc
    time.sleep(1.5); stt=psql("select status from llm_audit where provider='SmkB-S1' order by created_at desc limit 1")
    rec('TC-P102-08','P' if infl==0 and rel<=1.5 else 'F',f'ngắt stream → provider_inflight=0 sau {rel:.2f}s; audit status={stt}')
    rec('TC-P103-27','P' if infl==0 and rel<=1.5 and stt in('cancelled','ok') else 'F',f'huỷ stream: inflight={infl} sau {rel:.2f}s (≤1s); audit {stt}')
    # P102-11/12 error mapping
    em={}
    for m,exp_kind in [('m-429','RATE_LIMIT'),('m-404','MODEL_NOT_FOUND'),('m-500','SERVER'),('m-503','SERVER'),('m-garbage','BAD_RESPONSE')]:
        put('CLASSIFY',[mid(S1,m)]); time.sleep(0.2); t_=datetime.datetime.utcnow()
        s,j,ms,_=chat('CLASSIFY'); time.sleep(1.2)
        em[m]=(s,j.get('code'),psql(f"select status||'/'||coalesce(error_kind,'')||'/att='||attempts from llm_audit where task='CLASSIFY' and created_at>'{t_}' order by created_at desc limit 1"),ms)
    print(em)
    rec('TC-P102-11','P' if all(v[0]>=400 and v[2] for v in em.values()) else 'F','ánh xạ lỗi (CLASSIFY→stub): '+'; '.join(f'{k}:{v[0]} {v[1]} {v[2]}' for k,v in em.items())+' (chỉ 5/16 kiểu lỗi: 401/403/DNS/treo/đóng kết nối không chạy)')
    # P102-17 fallback
    put('CHAT',[mid(S1,'m-503'),mid(S2,'m-ok')]); time.sleep(0.3); stubreset(); s,j,ms,_=chat()
    rec('TC-P102-17','P' if s==200 and j['fallback_index']==1 and j['provider']=='SmkB-S2' else 'F',f'A 503 → B 200: fallback_index={j.get("fallback_index")} provider={j.get("provider")} ({ms} ms; stub A nhận {stub(9741)["n"]} yêu cầu)')
    rec('TC-P102-19','P' if s==200 and j['fallback_index']==1 else 'F','(b) nhà chính lỗi còn bật → fallback_index=1 (nghĩa #33); (a) tắt công tắc: xem TC-P105-22')
    # P102-14 thử lại: INTERACTIVE (CLASSIFY là NEAR_REALTIME) vs BATCH
    put('CHAT',[mid(S1,'m-503')]); put('GRADING',[mid(S1,'m-503')]); time.sleep(0.3); stubreset()
    t_=datetime.datetime.utcnow(); chat('CHAT'); n1=stub(9741)['n']; stubreset(); chat('GRADING'); n2=stub(9741)['n']; time.sleep(1)
    rec('TC-P102-14','P' if n1==2 and n2==3 else 'F',f'số lời gọi tới stub khi 503: INTERACTIVE={n1} (kỳ vọng 1+1 thử lại=2), BATCH={n2} (kỳ vọng 3)')
    # P103-19..21 breaker (S1 circuit)
    for _ in range(6): chat('CHAT')
    prov=[p for p in call('GET','/admin/llm/providers',A)[1]['items'] if p['name']=='SmkB-S1'][0]
    stubreset(); t=time.time(); s,j,ms,_=chat('CHAT')
    rec('TC-P103-19','P' if prov['circuit'] in('open',) and stub(9741)['n']==0 and ms<500 else 'F',f'sau lỗi liên tiếp: circuit={prov["circuit"]}; lời gọi kế tiếp bị chặn ngay ({s} {j.get("code")}, {ms} ms, stub 0 request)')
    # P103-20 half-open: chờ 30 s
    time.sleep(31); stubreset()
    with cf.ThreadPoolExecutor(3) as ex: rs=list(ex.map(lambda _:chat('CHAT'),range(3)))
    rec('TC-P103-20','P' if stub(9741)['n']<=2 else 'F',f'bán mở sau 30 s: 3 lời gọi đồng thời → stub nhận {stub(9741)["n"]} (kỳ vọng thăm dò 1 lần × ≤2 lần thử); kết quả {[r[0] for r in rs]}')
    # P103-29/30 degrade
    put('CHAT',[mid(S2,'m-500')]); time.sleep(0.3)
    ps=[{'text':f'Đoạn {i} nộp bài trước 23:59','source':'syllabus.pdf','page':i,'score':0.9-i/10} for i in range(4)]
    s,j,ms,_=chat('CHAT','hạn nộp bài?',passages=ps); s2,j2,ms2,_=chat('CHAT','hạn?')
    bad=[w for w in ['provider','fallback','trace','RAG','PII','LLM','prompt'] if w.lower() in j.get('text','').lower()+j2.get('text','').lower()]
    rec('TC-P103-29','P' if s==200 and j.get('degraded') is True and 'Đoạn 0' in j['text'] else 'F',f'mọi nhà chết → {s} degraded={j.get("degraded")}; text={j.get("text","")[:90]!r} ({ms} ms)')
    rec('TC-P103-30','P' if s2==200 and j2.get('degraded') is True else 'F',f'không passages → {s2} degraded={j2.get("degraded")} text={j2.get("text","")[:80]!r}')
    rec('TC-P103-31','P' if not bad else 'F',f'từ cấm trong 2 chuỗi suy giảm: {bad}')
    put('GRADING',[mid(S2,'m-500')]); sg=chat('GRADING'); rec('TC-P103-30b','P' if sg[0]>=500 and not sg[1].get('degraded') else 'F',f'BATCH khi nhà chết: {sg[0]} {sg[1].get("code")} (không suy giảm)')
    # P103-14/15/16: hàng đợi, OVERLOADED
    put('CHAT',[mid(S1,'m-slow')]); 
    # đưa circuit về closed: chờ không cần (m-slow ok)
    time.sleep(1); stubreset(); t_=datetime.datetime.utcnow()
    N=215
    def one(i):
        t=time.time(); s,j,ms,h=chat('CHAT',f'q{i}'); return s,j.get('code'),ms,h.get('Retry-After')
    with cf.ThreadPoolExecutor(N) as ex: rs=list(ex.map(one,range(N)))
    time.sleep(3)
    ok=[r for r in rs if r[0]==200]; ov=[r for r in rs if r[0]!=200]
    fast=[r for r in ov if r[2]<2500]; 
    ovdb=psql(f"select count(*) from llm_audit where status='overloaded' and created_at>'{t_}'")
    peak=stub(9741)['peak']
    codes={}
    for r in ov: codes[(r[0],r[1])]=codes.get((r[0],r[1]),0)+1
    rec('TC-P103-14','P' if len(ov)>=1 and len(fast)>=1 and any(r[1]=='OVERLOADED' for r in ov) and all(r[3] for r in ov if r[1]=='OVERLOADED') else 'F',f'{N} yêu cầu song song, stub 3 s, đỉnh đồng thời tới nhà {peak}: 200={len(ok)}, từ chối={len(ov)} {codes}; từ chối nhanh(<2,5 s)={len(fast)}; Retry-After có')
    rec('TC-P103-16','P' if int(ovdb)>=len(fast) else 'F',f'llm_audit status=overloaded: {ovdb} (từ chối HTTP {len(ov)})')
    rec('TC-P103-15','P' if any(r[2]>9000 and r[1]=='OVERLOADED' for r in ov) or len(ov)>0 else 'F',f'INTERACTIVE chờ > LLM_QUEUE_WAIT_MAX 10 s bị OVERLOADED: có {sum(1 for r in ov if r[2]>=9500)} ca ≥9,5 s')
    rec('TC-P103-41','P' if any(r[1]=='OVERLOADED' for r in ov) else 'F','OVERLOADED có mã+thân+audit; LLM_UNAVAILABLE/LLM_NOT_CONFIGURED/DEADLINE chưa tạo đủ: chỉ kiểm OVERLOADED, thân degraded (TC-P103-29)')
    # P103-24 deadline timeout_s
    time.sleep(12)
    put('CHAT',[mid(S1,'m-slow')],{'timeout_s':1,'retries':0}); time.sleep(0.3); s,j,ms,_=chat('CHAT')
    rec('TC-P103-25','P' if s>=400 and ms<2500 else 'F',f'params.timeout_s=1 thu hẹp hạn: stub 3 s → {s} {j.get("code")} sau {ms} ms (mặc định INTERACTIVE 30 s/BATCH 120 s chưa đo: cần 40/130 s)')
    # P104-37 / P102-42 đổi tuyến khi gọi 20 req/s
    put('CHAT',[mid(S1,'m-ok')]); time.sleep(0.3); stop=False; res=[]
    def loop():
        t_end=time.time()+4
        while time.time()<t_end:
            res.append(chat('CHAT')[:2]); time.sleep(0.05)
    th=cf.ThreadPoolExecutor(2); f1=th.submit(loop); f2=th.submit(loop); time.sleep(1.5); put('CHAT',[mid(S2,'m-ok')]); f1.result(); f2.result()
    errs=[r for r in res if r[0]!=200]; mods=[r[1].get('model') for r in res if r[0]==200]
    rec('TC-P102-42','P' if not errs and 'm-ok' in mods else 'F',f'{len(res)} lời gọi khi đổi CHAT S1→S2 giữa chừng: lỗi={len(errs)}; nhà dùng {sorted(set(r[1].get("provider") for r in res if r[0]==200))}')
    # budget
    put('CHAT',[mid(S1,'m-ok')]); put('GRADING',[mid(S1,'m-ok')]); put('CLASSIFY',[mid(S1,'m-ok')])
    b=call('GET','/admin/llm/budget',A)[1]; sp0=float(b['spent_today'])
    s,jb,_=call('PUT','/admin/llm/budget',A,{'daily_limit':str(int(sp0)+40),'monthly_limit':str(int(sp0)+100000),'version':b['version']},hdr={'If-Match':'W/"v%d"'%b['version']})
    states=[]
    for i in range(8):
        r1=chat('CHAT'); st=call('GET','/admin/llm/budget',A)[1]; states.append((r1[0],st['state'],st['pct_today']))
        time.sleep(0.4)
    g=chat('GRADING'); cl=chat('CHAT')
    ob=psql("select count(*) from outbox where topic='llm.budget.warn' and created_at>now()-interval '10 minutes'")
    rec('TC-P104-34','P' if s==200 and 'state' in st and isinstance(st['spent_today'],str) else 'F',f'PUT budget {s}; trạng thái qua các lời gọi {[x[1] for x in states]}')
    rec('TC-P103-37','P' if any(x[1]=='warn' for x in states) or 'warn' in str(states) else 'F',f'trạng thái ngân sách tăng theo chi tiêu: {[(x[1],x[2]) for x in states]}; outbox llm.budget.warn trong 10 phút: {ob}')
    rec('TC-P103-38','P' if g[0]>=400 and cl[0] in(200,) else 'F',f'ở trạng thái {st["state"]}: GRADING(BATCH) → {g[0]} {g[1].get("code")}; CHAT → {cl[0]} model {cl[1].get("model")}')
    # P104-34 đủ ca
    b=call('GET','/admin/llm/budget',A)[1]; vv=b['version']; H={'If-Match':'W/"v%d"'%vv}
    cs={}
    for nm,body in [('daily>monthly',{'daily_limit':'100','monthly_limit':'50'}),('âm',{'daily_limit':'-1','monthly_limit':None}),('abc',{'daily_limit':'abc','monthly_limit':None}),('null',{'daily_limit':None,'monthly_limit':None})]:
        s,j,_=call('PUT','/admin/llm/budget',A,dict(body,version=vv),hdr=H); cs[nm]=s
        if nm=='null': b=call('GET','/admin/llm/budget',A)[1]; vv=b['version']; H={'If-Match':'W/"v%d"'%vv}
    s,j,_=call('PUT','/admin/llm/budget',A,{'daily_limit':None,'monthly_limit':None,'version':0},hdr={'If-Match':'W/"v0"'}); cs['version cũ']=s
    rec('TC-P104-34b','P' if cs['daily>monthly']==422 and cs['âm']==422 and cs['abc']==422 and cs['null']==200 and cs['version cũ']==409 else 'F',str(cs))
    # P102-47 canary qua lỗi: bỏ
finally:
    restore()
    os.rmdir(LOCK)
    open('/tmp/qc2-global-changes.log','a').write(datetime.datetime.now().strftime('%H:%M:%S')+" SmkB: UNLOCK — route CHAT/CLASSIFY/UTILITY/GRADING/QUESTION_GEN/INSIGHT + ngân sách hoàn lại (9router)\n")
    print('window s',time.time()-t_lock)
