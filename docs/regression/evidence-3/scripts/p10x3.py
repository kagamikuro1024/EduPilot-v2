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
    t=time.time(); import lib as _lib; s,j,h=_lib._call('POST','/_test/llm/chat',tk or A,dict(task=task,prompt=prompt,**kw)); return s,j,round((time.time()-t)*1000),h
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
    # --- R3-05 (c561c25b): đối chiếu bằng llm_audit (attempts, queue_wait_ms, latency_ms), không đếm request ở stub
    SS=lambda sql_: psql(sql_)
    put('CHAT',[mid(S1,'m-503')]); put('GRADING',[mid(S1,'m-503')]); time.sleep(0.3); stubreset()
    t_=datetime.datetime.utcnow(); chat('CHAT'); n1=stub(9741)['n']; a1=SS(f"select status||'/'||coalesce(error_kind,'')||'/att='||attempts||'/qwait='||queue_wait_ms||'/lat='||latency_ms from llm_audit where provider='SmkB-S1' and task='CHAT' and created_at>'{t_}' order by created_at")
    stubreset(); t2=datetime.datetime.utcnow(); chat('GRADING'); n2=stub(9741)['n']; time.sleep(1)
    a2=SS(f"select status||'/'||coalesce(error_kind,'')||'/att='||attempts||'/qwait='||queue_wait_ms||'/lat='||latency_ms from llm_audit where provider='SmkB-S1' and task='GRADING' and created_at>'{t2}' order by created_at")
    other=SS(f"select count(*) from llm_audit where task='GRADING' and created_at>'{t2}' and provider<>'SmkB-S1'")
    rec('TC-P102-14','P' if n1==2 and n2==3 and 'att=3' in a2 and other=='0' else 'F',f'stub: INTERACTIVE={n1} (kỳ vọng 2), BATCH={n2} (kỳ vọng 3); llm_audit CHAT: {a1!r}; GRADING: {a2!r}; GRADING của nhà khác trong cửa sổ: {other}')
    for _ in range(6): chat('CHAT')
    prov=[p for p in call('GET','/admin/llm/providers',A)[1]['items'] if p['name']=='SmkB-S1'][0]
    stubreset(); t3=datetime.datetime.utcnow(); s,j,ms,_=chat('CHAT'); time.sleep(0.5)
    a3=SS(f"select status||'/'||coalesce(error_kind,'')||'/att='||attempts||'/lat='||latency_ms from llm_audit where task='CHAT' and created_at>'{t3}' order by created_at")
    rec('TC-P103-19','P' if prov['circuit']=='open' and stub(9741)['n']==0 and ms<500 else 'F',f'circuit={prov["circuit"]}; lời gọi kế tiếp {s} {j.get("code")} {ms} ms, stub {stub(9741)["n"]} request; llm_audit: {a3!r}')
    time.sleep(31); stubreset(); t4=datetime.datetime.utcnow()
    with cf.ThreadPoolExecutor(3) as ex: rs=list(ex.map(lambda _:chat('CHAT'),range(3)))
    time.sleep(1); a4=SS(f"select status||'/att='||attempts||'/lat='||latency_ms from llm_audit where task='CHAT' and created_at>'{t4}' order by created_at")
    rec('TC-P103-20','P' if stub(9741)['n']<=2 else 'F',f'bán mở: 3 lời gọi đồng thời → stub {stub(9741)["n"]} (kỳ vọng thăm dò 1 lần x ≤2 thử); kết quả {[r[0] for r in rs]}; llm_audit: {a4!r}')

finally:
    restore()
    os.rmdir(LOCK)
    open('/tmp/qc2-global-changes.log','a').write(datetime.datetime.now().strftime('%H:%M:%S')+" SmkB: UNLOCK — route CHAT/CLASSIFY/UTILITY/GRADING/QUESTION_GEN/INSIGHT + ngân sách hoàn lại (9router)\n")
    print('window s',time.time()-t_lock)
