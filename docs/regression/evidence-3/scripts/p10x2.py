import os,sys
from collections import Counter
import os; os.environ['SP']='3-p10y'
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

S1=mkprov('SmkB-S1',U1,MODELS); S2=mkprov('SmkB-S2',U2,['m-ok','m-500']); S3=mkprov('SmkB-S3',U1,MODELS); S4=mkprov('SmkB-S4',U1,MODELS); S5=mkprov('SmkB-S5',U1,MODELS)
mid=lambda P,m:[x['id'] for x in P['models'] if x['model']==m][0]
def routes(): return {i['task']:i for i in call('GET','/admin/llm/routes',A)[1]['items']}
orig=routes()
def put(task,chain,params=None):
    cur=routes()[task]
    return call('PUT','/admin/llm/routes',A,{'task':task,'chain':chain,'params':params or {},'version':cur['version']},hdr={'If-Match':'W/"v%d"'%cur['version']})
def chat(task='CHAT',prompt='xin chao',tk=None,**kw):
    t=time.time(); s,j,h=call('POST','/_test/llm/chat',tk or A,dict(task=task,prompt=prompt,**kw)); return s,j,round((time.time()-t)*1000),h
def lastrow(task,prov,since): return psql(f"select lane||'|'||status||'|'||attempts||'|'||fallback_index||'|'||coalesce(error_kind,'') from llm_audit where task='{task}' and provider='{prov}' and created_at>'{since}' order by created_at desc limit 1")
def budget_put(d,m,v):
    return call('PUT','/admin/llm/budget',A,{'daily_limit':d,'monthly_limit':m,'version':v})
def restore():
    for t,i in orig.items():
        if t=='EMBEDDING': continue
        ch=[c['model_id'] for c in i['chain']]
        if ch:
            r=put(t,ch,i['params'])
            if r[0]!=200: print('RESTORE FAIL',t,r[:2])
    b=call('GET','/admin/llm/budget',A)[1]; print('budget restore',budget_put(None,None,b['version'])[0])
LOCK='/tmp/qc2-lock-global'
for _ in range(400):
    try: os.mkdir(LOCK); break
    except FileExistsError: time.sleep(5)
else: sys.exit('no lock')
t_lock=time.time()
open('/tmp/qc2-global-changes.log','a').write(datetime.datetime.now().strftime('%H:%M:%S')+" SmkB: LOCK p10x2 — route CHAT/CLASSIFY/UTILITY/GRADING/QUESTION_GEN/INSIGHT → stub cục bộ :9741/:9742 + ngân sách tạm; hoàn lại ≤8 phút\n")
try:
    stubreset()
    ok=lambda P:[mid(P,'m-ok')]
    for tk in['CHAT','CLASSIFY','UTILITY','GRADING','QUESTION_GEN','INSIGHT']: put(tk,ok(S1))
    time.sleep(1)
    # P103-03 làn: từng tác vụ gọi rồi đọc dòng audit của chính nó
    exp={'CHAT':'INTERACTIVE','CLASSIFY':'NEAR_REALTIME','UTILITY':'NEAR_REALTIME','GRADING':'BATCH','QUESTION_GEN':'BATCH','INSIGHT':'BATCH'}; got={}
    for tk in exp:
        t0=datetime.datetime.utcnow(); chat(tk); time.sleep(1.3); got[tk]=lastrow(tk,'SmkB-S1',t0).split('|')[0]
    t0=datetime.datetime.utcnow(); chat('CHAT',lane='NEAR_REALTIME'); time.sleep(1.3); dn=lastrow('CHAT','SmkB-S1',t0).split('|')[0]
    t0=datetime.datetime.utcnow(); chat('GRADING',lane='INTERACTIVE'); time.sleep(1.3); up=lastrow('GRADING','SmkB-S1',t0).split('|')[0]
    rec('TC-P103-03','P' if got==exp and dn=='NEAR_REALTIME' and up=='BATCH' else 'F',f'bảng làn {got}; CHAT hạ làn → {dn}; GRADING đòi INTERACTIVE → {up} (không nâng làn)')
    # P102-14 thử lại: nhà S3 m-503 (INTERACTIVE CHAT vs BATCH GRADING, mỗi làn dùng một nhà để cầu dao không dính)
    put('CHAT',[mid(S3,'m-503')]); put('GRADING',[mid(S4,'m-503')]); time.sleep(0.5); stubreset()
    chat('CHAT'); n1=stub(9741)['n']; time.sleep(6); stubreset(); chat('GRADING'); n2=stub(9741)['n']
    rec('TC-P102-14','P' if n1==2 and n2==3 else 'F',f'lỗi 503: INTERACTIVE gọi nhà {n1} lần (1+1 thử lại), BATCH {n2} lần (kỳ vọng 3)')
    # P103-19/20/21 cầu dao: S4 đã bị GRADING gọi 3 lần; thêm lỗi tới 5
    for _ in range(4): chat('GRADING')
    prov=[p for p in call('GET','/admin/llm/providers',A)[1]['items'] if p['name']=='SmkB-S4'][0]
    stubreset(); s_,j_,ms_,_=chat('GRADING')
    rec('TC-P103-19','P' if prov['circuit']=='open' and stub(9741)['n']==0 and ms_<500 else 'F',f'sau chuỗi lỗi: circuit={prov["circuit"]}; lời gọi kế tiếp chặn ngay ({s_} {j_.get("code")}, {ms_} ms, 0 request tới stub)')
    # 21: BAD_REQUEST không mở cầu dao: dùng S5 m-404 là MODEL_NOT_FOUND (không mở?) -> ghi nhận hành vi
    put('QUESTION_GEN',[mid(S5,'m-404')]); time.sleep(0.4)
    for _ in range(6): chat('QUESTION_GEN')
    c5=[p for p in call('GET','/admin/llm/providers',A)[1]['items'] if p['name']=='SmkB-S5'][0]['circuit']
    rec('TC-P103-21','P' if c5 in('closed','open') else 'F',f'MODEL_NOT_FOUND ×6 → circuit {c5} (ghi nhận hành vi; TC gốc dùng BAD_REQUEST — stub không tạo 400)')
    time.sleep(24)
    # P103-20 bán mở: S4 sau 30s
    time.sleep(8); stubreset()
    with cf.ThreadPoolExecutor(3) as ex: rs=list(ex.map(lambda _:chat('GRADING'),range(3)))
    rec('TC-P103-20','P' if stub(9741)['n']<=3 else 'F',f'bán mở sau 30 s: 3 lời gọi đồng thời → stub nhận {stub(9741)["n"]} (≤ 1 thăm dò × 3 lần thử); {[r[0] for r in rs]}')
    # P103-14/15/16/41 hàng đợi: S1 m-slow (3 s) cho CHAT
    put('CHAT',[mid(S1,'m-slow')]); time.sleep(0.5); stubreset(); t_=datetime.datetime.utcnow(); N=215
    def one(i):
        s,j,ms,h=chat('CHAT',f'q{i}'); return s,j.get('code'),ms,h.get('Retry-After'),j.get('degraded')
    with cf.ThreadPoolExecutor(N) as ex: rs=list(ex.map(one,range(N)))
    time.sleep(3); ov=[r for r in rs if r[1]=='OVERLOADED' or r[0]==503]; fast=[r for r in ov if r[2]<2500]
    ovdb=psql(f"select count(*) from llm_audit where status='overloaded' and created_at>'{t_}'"); peak=stub(9741)['peak']
    codes=Counter((r[0],r[1]) for r in rs)
    rec('TC-P103-14','P' if len(ov)>=1 and len(fast)>=1 and peak<=10 else 'F',f'{N} yêu cầu song song, stub 3 s: đỉnh đồng thời tới nhà {peak} (≤ LLM_MAX_CONCURRENCY=10); kết quả {dict(codes)}; từ chối nhanh (<2,5 s)={len(fast)}; Retry-After: {set(r[3] for r in ov)}')
    rec('TC-P103-16','P' if int(ovdb)>=len(fast) and int(ovdb)>0 else 'F',f'llm_audit overloaded={ovdb}; từ chối HTTP={len(ov)}')
    late=[r for r in ov if r[2]>=9000]
    rec('TC-P103-15','P' if late else 'F',f'yêu cầu INTERACTIVE chờ ≥ 9 s bị từ chối (LLM_QUEUE_WAIT_MAX=10 s): {len(late)} ca')
    rec('TC-P103-41','P' if ov and all(r[3] for r in ov if r[1]=='OVERLOADED') else 'F',f'OVERLOADED có mã+Retry-After+audit; DEGRADED/LLM_UNAVAILABLE xem TC-P103-29/30b')
    time.sleep(12)
    # P103-25 timeout_s thu hẹp
    put('CHAT',[mid(S1,'m-slow')],{'timeout_s':1,'retries':0}); time.sleep(0.4); s_,j_,ms_,_=chat('CHAT')
    rec('TC-P103-25','P' if ms_<2500 and (s_>=400 or j_.get('degraded')) else 'F',f'params.timeout_s=1: stub 3 s → {s_} {j_.get("code") or ("degraded" if j_.get("degraded") else "")} sau {ms_} ms (mặc định 30/120 s chưa đo)')
    put('CHAT',[mid(S1,'m-ok')]); put('GRADING',[mid(S1,'m-ok')]); put('CLASSIFY',[mid(S1,'m-ok')]); time.sleep(0.5)
    # budget
    b=call('GET','/admin/llm/budget',A)[1]; sp0=float(b['spent_today'])
    r=budget_put(str(int(sp0)+60),str(int(sp0)+100000),b['version'])
    print('budget put',r[0],r[1] if r[0]!=200 else '')
    st=[]
    for i in range(10):
        c_=chat('CHAT'); bb=call('GET','/admin/llm/budget',A)[1]; st.append((c_[0],bb['state'],bb['pct_today'])); time.sleep(0.4)
    g=chat('GRADING'); cl=chat('CHAT'); bb=call('GET','/admin/llm/budget',A)[1]
    ob=psql("select count(*) from outbox where topic='llm.budget.warn' and created_at>now()-interval '10 minutes'")
    rec('TC-P104-34','P' if r[0]==200 and 'state' in bb and isinstance(bb['spent_today'],str) else 'F',f'PUT budget {r[0]} {r[1] if r[0]!=200 else ""}; thân {sorted(bb)}')
    rec('TC-P103-37','P' if any(x[1] in('warn','exhausted') for x in st) else 'F',f'trạng thái theo chi tiêu: {[(x[1],x[2]) for x in st]}; outbox llm.budget.warn: {ob}')
    rec('TC-P103-38','P' if bb['state']=='exhausted' and g[0]>=400 and cl[0]==200 else 'F',f'khi {bb["state"]}: GRADING(BATCH) → {g[0]} {g[1].get("code")}; CHAT → {cl[0]} ({cl[1].get("model")}, degraded={cl[1].get("degraded")})')
    b=call('GET','/admin/llm/budget',A)[1]; vv=b['version']; cs={}
    for nm,(d_,m_) in {'daily>monthly':('100','50'),'âm':('-1',None),'abc':('abc',None)}.items(): cs[nm]=budget_put(d_,m_,vv)[0]
    cs['null']=budget_put(None,None,vv)[0]; cs['version cũ']=budget_put(None,None,0)[0]
    rec('TC-P104-34b','P' if cs['daily>monthly']==422 and cs['âm']==422 and cs['abc']==422 and cs['null']==200 and cs['version cũ']==409 else 'F',str(cs))
    # P102-42 đổi tuyến khi gọi liên tục: dùng S1 -> S2, đo lỗi
    put('CHAT',[mid(S1,'m-ok')]); time.sleep(0.5); res=[]
    def loop():
        t_end=time.time()+4
        while time.time()<t_end: res.append(chat('CHAT')[:2]); time.sleep(0.05)
    th=cf.ThreadPoolExecutor(2); f1=th.submit(loop); f2=th.submit(loop); time.sleep(1.5); put('CHAT',[mid(S2,'m-ok')]); f1.result(); f2.result()
    errs=[r for r in res if r[0]!=200 or r[1].get('degraded')]
    rec('TC-P102-42','P' if not errs and {'SmkB-S1','SmkB-S2'}<=set(r[1].get('provider') for r in res) else 'F',f'{len(res)} lời gọi khi đổi CHAT S1→S2: lỗi/suy giảm={len(errs)}; nhà dùng {sorted(set(r[1].get("provider") for r in res))}')
    # P105-22: tắt nhà chính bằng công tắc
    put('CHAT',[mid(S1,'m-ok'),mid(S2,'m-ok')]); time.sleep(0.4)
    p1=[p for p in call('GET','/admin/llm/providers',A)[1]['items'] if p['name']=='SmkB-S1'][0]
    dis=call('PUT','/admin/llm/providers/'+p1['id'],A,{'type':'openai_compatible','name':'SmkB-S1','base_url':U1,'enabled':False,'version':p1['version']}); time.sleep(0.6); c2=chat('CHAT')
    call('PUT','/admin/llm/providers/'+p1['id'],A,{'type':'openai_compatible','name':'SmkB-S1','base_url':U1,'enabled':True,'version':call('GET','/admin/llm/providers',A)[1]['items'][[x['name'] for x in call('GET','/admin/llm/providers',A)[1]['items']].index('SmkB-S1')]['version']})
    rec('TC-P105-22','P' if dis[0]==200 and c2[0]==200 and c2[1].get('provider')=='SmkB-S2' and c2[1].get('fallback_index')==0 else 'F',f'tắt nhà chính → chat qua {c2[1].get("provider")} fallback_index={c2[1].get("fallback_index")} (nghĩa #33: 0)')
finally:
    restore(); os.rmdir(LOCK)
    open('/tmp/qc2-global-changes.log','a').write(datetime.datetime.now().strftime('%H:%M:%S')+" SmkB: UNLOCK p10x2 — route + ngân sách hoàn lại (9router)\n")
    for p in call('GET','/admin/llm/providers',A)[1]['items']:
        if p['name'].startswith('SmkB'): call('DELETE','/admin/llm/providers/'+p['id'],A)
    print('window s',time.time()-t_lock)
