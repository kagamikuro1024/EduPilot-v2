# P102-12/15/18/47, P103-24/42 trên stack cô lập + stub mở rộng smk3stub (:9751/:9752)
from smk3iso import *
V1='http://host.docker.internal:9751/v1'; V2='http://host.docker.internal:9752/v1'
SM=['m-ok','m-400','m-401','m-big','m-canary','m-ra2','m-ra30','m-slow10','m-503','m-500','m-midkill','m-prefail','m-slow']
T1=imkprov('SmkI-T1',V1,SM); T2=imkprov('SmkI-T2',V2,SM)
def sreset():
    for p in (9751,9752): urllib.request.urlopen(urllib.request.Request(f'http://localhost:{p}/__reset',data=b'{}',method='POST'),timeout=10)
def sn(p): return stub(p)
def logs(p): return stub(p)['log']
R={}
# ---- P102-12: thân lỗi dài
iput('GRADING',[mid(T1,'m-big')],{'retries':0}); sreset()
s,j,ms,_=ichat('GRADING',hdr=tp('a1'+'0'*30)); time.sleep(1.5)
body=json.dumps(j,ensure_ascii=False)
lg='\n'.join(gwlogs('30s'))
print('P102-12',s,len(body),body[:200]); print(' leak html/trace?', '<html' in body, 'Traceback' in body, 'XXXX' in body, '| in logs:','XXXXX' in lg,'Traceback' in lg)
print(' audit',ipsql("select status,error_kind from llm_audit where trace_id like 'a1%'"))
# ---- P102-15: Retry-After
def timed(model,params,task='GRADING',tid='b1'):
    iput(task,[mid(T1,model)],params); sreset(); t0=time.time(); r=ichat(task,hdr=tp(tid+'0'*30)); el=time.time()-t0; lg=sn(9751)['log']; return r,el,lg
for model,params,tid in [('m-ra2',{'retries':2},'b1'),('m-ra30',{'retries':2},'b2')]:
    r,el,lg=timed(model,params,tid=tid); ts=[x[1] for x in lg]; print('P102-15',model,r[0],r[1].get('code'),'%.1fs'%el,'số lần gọi',len(lg),'khoảng cách',[round(b-a,1) for a,b in zip(ts,ts[1:])])
r,el,lg=timed('m-ra2',{'retries':2,'timeout_s':1},tid='b3'); print('P102-15 ctx<1s',r[0],r[1].get('code'),'%.1fs'%el,'số lần gọi',len(lg))
# ---- P102-18: 400 không chuyển tiếp; cả ba 503
iput('GRADING',[mid(T1,'m-400'),mid(T2,'m-ok')],{'retries':2}); sreset(); ipsql("delete from llm_audit where trace_id like 'c1%'")
r=ichat('GRADING',hdr=tp('c1'+'0'*30)); time.sleep(1.5)
print('P102-18 A=400',r[0],r[1].get('code'),'| stub A',sn(9751)['n'],'| stub B',sn(9752)['n'],'| audit',ipsql("select count(*),string_agg(status||'/'||coalesce(error_kind,''),',') from llm_audit where trace_id like 'c1%'"))
iput('GRADING',[mid(T1,'m-503'),mid(T2,'m-503'),mid(T2,'m-500')],{'retries':0}); sreset()
r=ichat('GRADING',hdr=tp('c2'+'0'*30)); time.sleep(1.5)
print('P102-18 A,B,C=5xx',r[0],r[1].get('code'),'| stub',sn(9751)['n'],sn(9752)['n'],'| audit',ipsql("select count(*),string_agg(status||'/'||coalesce(error_kind,'')||'/'||attempts,',') from llm_audit where trace_id like 'c2%'"))
# ---- P102-47: canary
iput('GRADING',[mid(T1,'m-canary')],{'retries':0}); sreset()
r=ichat('GRADING',hdr=tp('d1'+'0'*30)); time.sleep(2)
body=json.dumps(r[1],ensure_ascii=False); lg='\n'.join(gwlogs('60s'))
pat=['CANARY-SMK3','sk-live','Bearer abc','Bearer good-key','good-key']
print('P102-47',r[0],body[:200]); print(' trong phản hồi',[p for p in pat if p in body],'| log',[p for p in pat if p in lg],'| audit_log',ipsql("select count(*) from audit_log where details::text ~ 'CANARY-SMK3|sk-live|Bearer'"),'| llm_audit.error_kind',ipsql("select error_kind from llm_audit where trace_id like 'd1%'"),'| outbox',ipsql("select count(*) from outbox where payload::text ~ 'CANARY-SMK3'"))
s,j,_=ic('POST','/admin/llm/providers/test',{'type':'openai_compatible','base_url':V1,'api_key':'good-key','model':'m-canary'}); print(' Test kết nối m-canary',s,json.dumps(j,ensure_ascii=False)[:250], 'rò:',[p for p in pat if p in json.dumps(j)])
# ---- P103-42: stream giữa chừng
def stream(tid):
    c=http.client.HTTPConnection('localhost',18201,timeout=30)
    c.request('POST','/api/v1/_test/llm/chat',json.dumps({'task':'CHAT','prompt':'s','stream':True}),{'authorization':'Bearer '+itok(),'content-type':'application/json','Idempotency-Key':str(uuid.uuid4()),**tp(tid)})
    r=c.getresponse(); b=r.read().decode(); c.close(); return r.status,b
iput('CHAT',[mid(T1,'m-midkill'),mid(T2,'m-ok')],{'retries':0}); sreset()
st,b=stream('e1'+'0'*30); print('P103-42 giữa chừng',st,[l for l in b.split('\n') if l.startswith('event')][:8],'| stub',sn(9751)['n'],sn(9752)['n'])
iput('CHAT',[mid(T1,'m-prefail'),mid(T2,'m-ok')],{'retries':0}); sreset()
st,b=stream('e2'+'0'*30); print('P103-42 chưa byte',st,[l for l in b.split('\n') if l.startswith('event')][:6],'fallback tokens:', 'Q' in b,'| stub',sn(9751)['n'],sn(9752)['n'])
# ---- P103-24: hạn 3 s
iput('GRADING',[mid(T1,'m-slow10')],{'retries':0,'timeout_s':3}); sreset()
t0=time.time(); r=ichat('GRADING',hdr=tp('f1'+'0'*30)); el=time.time()-t0
print('P103-24 timeout_s=3',r[0],json.dumps(r[1],ensure_ascii=False)[:200],'%.1fs'%el,'| audit',ipsql("select status,error_kind from llm_audit where trace_id like 'f1%'"))
