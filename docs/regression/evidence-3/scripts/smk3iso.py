# Thư viện cho stack LLM cô lập (/tmp/smk34/mini.sh: smk3-pg + smk3-redis + gateway smk3-gw1:18201 / smk3-gw2:18202). Không chạm ep-r.
import sys,os,time,json,uuid,subprocess,datetime,urllib.request,threading,http.client,concurrent.futures as cf
sys.path.insert(0,os.path.dirname(os.path.abspath(__file__))); os.environ.setdefault('SP','x')
import lib
ROOT=os.path.expanduser('~/Documents/TA_Agent_qc2')
IB={'gw1':'http://localhost:18201/api/v1','gw2':'http://localhost:18202/api/v1','gw3':'http://localhost:18203/api/v1'}
APW='Zx9#Quartz-Lantern-2026'
_itok={}
def itok(gw='gw1'):
    if gw not in _itok or time.time()-_itok[gw][1]>600:
        s,j,_=lib._call('POST','/auth/login',body={'email':'admin@smk3.test','password':APW},base=IB[gw]); assert s==200,(s,j)
        _itok[gw]=(j['access_token'],time.time())
    return _itok[gw][0]
def ic(m,p,body=None,gw='gw1',tok=True,hdr=None,raw=None):
    t=itok(gw) if tok is True else (tok or None)
    return lib._call(m,p,t,body,hdr,raw,True,IB[gw])
def ipsql(q):
    r=subprocess.run(['docker','exec','-i','smk3-pg','psql','-U','edupilot','-d','edupilot','-At','-F','|'],input=q,capture_output=True,text=True,timeout=60)
    return (r.stdout+r.stderr).strip()
def ird(*a): return subprocess.run(['docker','exec','smk3-redis','redis-cli',*a],capture_output=True,text=True,timeout=30).stdout.strip()
U1='http://host.docker.internal:9741/v1'; U2='http://host.docker.internal:9742/v1'
MODELS=['m-ok','m-429','m-404','m-500','m-503','m-garbage','m-slow','m-slow1','m-hang']
def stub(p,path='/__count'): return json.loads(urllib.request.urlopen(f'http://localhost:{p}{path}',timeout=10).read())
def stubreset():
    for p in(9741,9742): urllib.request.urlopen(urllib.request.Request(f'http://localhost:{p}/__reset',data=b'{}',method='POST'),timeout=10)
def imkprov(name,u,models,gw='gw1',**kw):
    ex=[p for p in ic('GET','/admin/llm/providers',gw=gw)[1]['items'] if p['name']==name]
    if ex: return ex[0]
    body={'type':'openai_compatible','name':name,'base_url':u,'api_key':'good-key','models':[{'model':m,'kind':'chat','price_in':'1000000','price_out':'1000000'} for m in models]}; body.update(kw)
    s,j,_=ic('POST','/admin/llm/providers',body,gw=gw); assert s==201,(s,j); return j
def mid(P,m): return [x['id'] for x in P['models'] if x['model']==m][0]
def iroutes(gw='gw1'): return {i['task']:i for i in ic('GET','/admin/llm/routes',gw=gw)[1]['items']}
def iput(task,chain,params=None,gw='gw1'):
    cur=iroutes(gw)[task]
    h={'If-Match':'W/"v%d"'%cur['version']} if cur['version']>0 else None
    return ic('PUT','/admin/llm/routes',{'task':task,'chain':chain,'params':params or {},'version':cur['version']},gw=gw,hdr=h)
def ichat(task='CHAT',prompt='xin chao',gw='gw1',hdr=None,**kw):
    t=time.time(); s,j,h=ic('POST','/_test/llm/chat',dict(task=task,prompt=prompt,**kw),gw=gw,hdr=hdr); return s,j,round((time.time()-t)*1000),h
def tp(t): return {'traceparent':f'00-{t}-b7ad6b7169203331-01'}
def rec(n,code,kq,note):
    note=note.replace('|','/').replace('\t',' ').replace('\n',' ')
    d=f'{ROOT}/docs/regression/evidence-{n}/results-smk34'; os.makedirs(d,exist_ok=True)
    open(f'{d}/smk.tsv','a',encoding='utf-8').write(f'{code}\t{kq}\t{note}\n'); print('REC',code,kq,note[:120])
def gwlogs(since='60s',gw='smk3-gw1'):
    r=subprocess.run(['docker','logs','--since',since,gw],capture_output=True,text=True); return (r.stdout+r.stderr).splitlines()
def iprov_set(name,gw='gw1',**ch):
    """PUT đầy đủ (giữ khoá): đổi vài trường của nhà cung cấp theo tên"""
    p=[x for x in ic('GET','/admin/llm/providers',gw=gw)[1]['items'] if x['name']==name][0]
    body={'type':p['type'],'name':p['name'],'base_url':p['base_url'],'enabled':p['enabled'],'rpm_limit':p['rpm_limit'],'tpm_limit':p['tpm_limit'],
          'models':[{'id':m['id'],'model':m['model'],'kind':m['kind'],'dims':m['dims'],'price_in':m['price_in'],'price_out':m['price_out'],'enabled':m['enabled']} for m in p['models']],'version':p['version']}
    body.update(ch)
    return ic('PUT','/admin/llm/providers/'+p['id'],body,gw=gw,hdr={'If-Match':'W/"v%d"'%p['version']})
import socket
class RC:
    """RESP client tối thiểu tới Redis cô lập (cổng 6145) — lấy mẫu ZCARD mỗi vài ms"""
    def __init__(s,port=6145): s.s=socket.create_connection(('localhost',port)); s.f=s.s.makefile('rb')
    def cmd(s,*a):
        b=b'*%d\r\n'%len(a)+b''.join(b'$%d\r\n%s\r\n'%(len(x.encode()),x.encode()) for x in a); s.s.sendall(b)
        l=s.f.readline().rstrip(b'\r\n'); t=l[:1]
        if t==b':': return int(l[1:])
        if t==b'+': return l[1:].decode()
        if t==b'$':
            n=int(l[1:]); 
            if n<0: return None
            d=s.f.read(n+2)[:-2]; return d.decode()
        if t==b'*':
            return [s._item() for _ in range(int(l[1:]))]
        return l.decode()
    def _item(s):
        l=s.f.readline().rstrip(b'\r\n'); t=l[:1]
        if t==b'$': n=int(l[1:]); return s.f.read(n+2)[:-2].decode()
        if t==b':': return int(l[1:])
        return l.decode()
def mkfake(name='SmkI-F',gw='gw1'):
    ex=[p for p in ic('GET','/admin/llm/providers',gw=gw)[1]['items'] if p['name']==name]
    if ex: return ex[0]
    s,j,_=ic('POST','/admin/llm/providers',{'type':'fake','name':name,'api_key':'k','skip_verify':True,'models':[{'model':'fake-chat','kind':'chat','price_in':'0','price_out':'0'}]},gw=gw); assert s==201,(s,j); return j
