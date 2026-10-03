import subprocess,threading,time,json,urllib.request,sys
tok=open('/tmp/qcp1/tok-ADMIN').read().strip()
def call(port):
    t=time.time()
    r=urllib.request.Request(f'http://localhost:{port}/api/v1/_test/llm/chat',data=b'{"task":"CHAT","prompt":"x"}',headers={'Authorization':'Bearer '+tok,'Content-Type':'application/json'})
    try:
        b=json.load(urllib.request.urlopen(r,timeout=10)); return t,b['model'],200
    except urllib.error.HTTPError as e: return t,None,e.code
    except Exception as e: return t,None,str(e)
res=[]; stop=False
def loop(port):
    while not stop:
        res.append((port,)+call(port)); time.sleep(0.05)
ths=[threading.Thread(target=loop,args=(p,)) for p in (8080,8081)]
[t.start() for t in ths]; time.sleep(1)
sql="begin; update llm_task_routes set model_id=(select id from llm_models where model='q2-chat') where task='CHAT' and fallback_order=0; update llm_task_routes set model_id=(select id from llm_models where model='q1-chat') where task='CHAT' and fallback_order=1; commit;"
subprocess.run(['docker','exec','-i','qcp1-pg','psql','-U','edupilot','-d','edupilot','-qc',sql])
t0=time.time()
subprocess.run(['docker','exec','qcp1-redis','redis-cli','publish','ep:llm:reload','x'],capture_output=True)
time.sleep(3); stop=True; [t.join() for t in ths]
for port in (8080,8081):
    first=[r for r in res if r[0]==port and r[1]>=t0 and r[2]=='q2-chat']
    errs=[r for r in res if r[0]==port and r[3]!=200]
    print(port,'first new model after publish: %.0f ms'%((first[0][1]-t0)*1000) if first else 'NEVER', 'errors',len(errs),'total',len([r for r in res if r[0]==port]))
