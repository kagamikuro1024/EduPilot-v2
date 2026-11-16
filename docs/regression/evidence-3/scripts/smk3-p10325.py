from smk3iso import *
V1='http://host.docker.internal:9751/v1'
T4=imkprov('SmkI-T4',V1,['m-slow40','m-slow130'],skip_verify=True)
iput('CHAT',[mid(T4,'m-slow40')],{'retries':0},gw='gw3'); iput('CLASSIFY',[mid(T4,'m-slow40')],{'retries':0,'timeout_s':10},gw='gw3'); iput('UTILITY',[mid(T4,'m-slow40')],{'retries':0,'timeout_s':100},gw='gw3'); iput('GRADING',[mid(T4,'m-slow130')],{'retries':0},gw='gw3')
out={}
def run(task,key):
    t0=time.time(); c=http.client.HTTPConnection('localhost',18203,timeout=260)
    c.request('POST','/api/v1/_test/llm/chat',json.dumps({'task':task,'prompt':'x'}),{'authorization':'Bearer '+itok('gw3'),'content-type':'application/json','Idempotency-Key':str(uuid.uuid4())})
    r=c.getresponse(); b=r.read().decode(); out[key]=(r.status,round(time.time()-t0,1),json.loads(b).get('code'))
ths=[threading.Thread(target=run,args=a) for a in (('CHAT','INTERACTIVE 40s nhà, mặc định'),('CLASSIFY','NEAR_REALTIME timeout_s=10'),('UTILITY','NEAR_REALTIME timeout_s=100 (không nới)'),('GRADING','BATCH 130s nhà, mặc định'))]
[t.start() for t in ths]; [t.join() for t in ths]
for k,v in out.items(): print(k,v)
print(ipsql("select task,status,error_kind,latency_ms from llm_audit where created_at>now()-interval '4 minutes' and provider='SmkI-T4' order by created_at"))
