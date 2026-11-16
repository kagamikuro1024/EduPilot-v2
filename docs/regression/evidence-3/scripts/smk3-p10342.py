from smk3iso import *
V1='http://host.docker.internal:9751/v1'; V2='http://host.docker.internal:9752/v1'
SM=['m-ok','m-400','m-401','m-big','m-canary','m-ra2','m-ra30','m-slow10','m-503','m-500','m-midkill','m-prefail','m-slow']
T1=imkprov('SmkI-T1',V1,SM); T2=imkprov('SmkI-T2',V2,SM)
print([m['model'] for m in T1['models']])
def sreset():
    for p in (9751,9752): urllib.request.urlopen(urllib.request.Request(f'http://localhost:{p}/__reset',data=b'{}',method='POST'),timeout=10)
def stream(tid,gw='gw1'):
    c=http.client.HTTPConnection('localhost',18201,timeout=30)
    c.request('POST','/api/v1/_test/llm/chat',json.dumps({'task':'CHAT','prompt':'s','stream':True}),{'authorization':'Bearer '+itok(),'content-type':'application/json','Idempotency-Key':str(uuid.uuid4()),**tp(tid)})
    r=c.getresponse(); b=r.read().decode(); c.close(); return r.status,b
r=iput('CHAT',[mid(T1,'m-midkill'),mid(T2,'m-ok')],{'retries':0}); print('PUT',r[0],json.dumps(r[1])[:300]); sreset()
st,b=stream('e1'+'0'*30); print(st); print(b[-700:]); print(stub(9751)['log'])
time.sleep(2); print(ipsql("select status,error_kind,attempts,provider from llm_audit where trace_id like 'e1%' order by created_at desc limit 1"))
