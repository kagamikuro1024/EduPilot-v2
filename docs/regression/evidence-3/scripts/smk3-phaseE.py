# P103-14/15/17/41: hàng đợi, OVERLOADED (gw3: LLM_MAX_CONCURRENCY=1, LLM_QUEUE_MAX=200, LLM_QUEUE_WAIT_MAX=300s) + hạn chờ 10 s (gw1 mặc định)
from smk3iso import *
V1='http://host.docker.internal:9751/v1'
SM=['m-ok','m-slow10','m-hang','m-slow']
T3=imkprov('SmkI-T3',V1,SM)
def flood(task,n,gw,tmo,lane=None):
    out=[]; lk=threading.Lock()
    tok=itok(gw); port=IB[gw].split(':')[2].split('/')[0]
    def one(i):
        t0=time.time()
        try:
            c=http.client.HTTPConnection('localhost',int(port),timeout=tmo)
            body={'task':task,'prompt':f'q{i}'}
            if lane: body['lane']=lane
            c.request('POST','/api/v1/_test/llm/chat',json.dumps(body),{'authorization':'Bearer '+tok,'content-type':'application/json','Idempotency-Key':str(uuid.uuid4())})
            r=c.getresponse(); b=r.read().decode(); res=(i,r.status,time.time()-t0,r.getheader('Retry-After'),b)
        except Exception as e: res=(i,'timeout',time.time()-t0,None,str(e)[:60])
        with lk: out.append(res)
    ths=[threading.Thread(target=one,args=(i,)) for i in range(n)]
    for t in ths: t.start(); time.sleep(0.004)   # tuần tự hoá nhẹ thứ tự đến
    for t in ths: t.join()
    return sorted(out)
def drain(gw):
    for _ in range(60):
        st=ic('GET','/_test/llm/stats',gw=gw)[1]
        if sum(st['queue_depth'].values())==0 and st['provider_inflight']==0: return
        time.sleep(1)
which=sys.argv[1:] or ['14','17','15']
if '14' in which or '17' in which:
    for task,lane in (('CHAT','INTERACTIVE'),('CLASSIFY','NEAR_REALTIME'),('GRADING','BATCH')):
        if '17' not in which and task!='CHAT': continue
        print('==',task,lane,iput(task,[mid(T3,'m-slow10')],{'retries':0,'timeout_s':300},gw='gw3')[0]); time.sleep(1.5)
        rs=flood(task,215,'gw3',22)
        rej=[r for r in rs if r[1]==503]; ok=[r for r in rs if r[1]==200]; to=[r for r in rs if r[1]=='timeout']
        print(' tổng',len(rs),'200:',len(ok),'503:',len(rej),'chờ-quá-22s(không phản hồi):',len(to))
        if rej:
            bodies=[json.loads(r[4]) for r in rej]; ra=[r[3] for r in rej]
            print(' 503 mã:',{b.get('code') for b in bodies},'| Retry-After:',sorted(set(ra)),'| retry_after trong thân:',sorted({b.get('retry_after',(b.get('details') or {}).get('retry_after')) for b in bodies}),'| message:',bodies[0].get('message'))
            print(' độ trễ từ chối (ms): min %.0f median %.0f max %.0f'%(min(r[2] for r in rej)*1000,sorted(r[2] for r in rej)[len(rej)//2]*1000,max(r[2] for r in rej)*1000))
            print(' thứ tự đến của các yêu cầu bị từ chối (chỉ số):',[r[0] for r in rej][:20])
        drain('gw3'); time.sleep(1)
        print(' sau thả: ',ipsql("select status,count(*) from llm_audit where task='%s' and created_at>now()-interval '90 seconds' group by 1"%task))
if '15' in which:
    for t in ('CHAT',): iput(t,[mid(T3,'m-hang')],{'retries':0,'timeout_s':60})
    print('== P103-15: 10 lời gọi chiếm hết 10 chỗ (m-hang 25 s) + 1 yêu cầu chờ trên gw1 (LLM_QUEUE_WAIT_MAX=10s)')
    res=[]
    def occ(i): res.append(ichat('CHAT',f'occ{i}')[:3])
    ths=[threading.Thread(target=occ,args=(i,)) for i in range(10)]; [t.start() for t in ths]; time.sleep(3)
    t0=time.time(); r=ichat('CHAT','waiter'); el=time.time()-t0
    print(' người chờ:',r[0],json.dumps(r[1],ensure_ascii=False)[:230],'sau %.1f s'%el,'Retry-After',r[3].get('Retry-After'))
    [t.join() for t in ths]; print(' 10 lời gọi chiếm chỗ:',[x[0] for x in res])
