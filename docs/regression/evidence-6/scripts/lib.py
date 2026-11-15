import json,os,ssl,subprocess,urllib.request,uuid,time,re
B='https://localhost:1143/api/v1'
PW='Edupilot#Seed-2026'
ctx=ssl._create_unverified_context()
_tok={}
def raw(method,path,token=None,body=None,idem=None,headers=None,timeout=60,stream=False):
    h={'content-type':'application/json'}
    if token: h['authorization']='Bearer '+token
    if method in('POST',) : h['Idempotency-Key']=idem or str(uuid.uuid4())
    if headers: h.update(headers)
    data=None if body is None else (body if isinstance(body,(bytes,str)) else json.dumps(body)).encode() if not isinstance(body,bytes) else body
    req=urllib.request.Request(B+path if path.startswith('/') else path,data=data,method=method,headers=h)
    try:
        r=urllib.request.urlopen(req,context=ctx,timeout=timeout)
        return r.status,r.read().decode(),dict(r.headers)
    except urllib.error.HTTPError as e:
        return e.code,e.read().decode(),dict(e.headers)
def j(method,path,token=None,body=None,**kw):
    s,t,h=raw(method,path,token,body,**kw)
    try: return s,json.loads(t) if t else None
    except Exception: return s,t
def login(u):
    if u in _tok: return _tok[u]
    try:
        c=json.load(open('/tmp/qc2e/tokcache.json'))
        if u in c and time.time()-c[u][1]<600: _tok[u]=c[u][0]; return _tok[u]
    except Exception: pass
    email=u if '@' in u else u+'@edupilot.local'
    for i in range(3):
        s,d=j('POST','/auth/login',None,{'email':email,'password':PW})
        if s==200:
            _tok[u]=d['access_token']
            try: c=json.load(open('/tmp/qc2e/tokcache.json'))
            except Exception: c={}
            c[u]=[_tok[u],time.time()]; json.dump(c,open('/tmp/qc2e/tokcache.json','w'))
            return _tok[u]
        if s==429: time.sleep(20)
        time.sleep(5)
    raise Exception(f'login {u} {s} {d}')
def A(u,method,path,body=None,**kw): return j(method,path,login(u),body,**kw)
def psql(sql):
    r=subprocess.run(['docker','exec','-i','ep-r-postgres-1','psql','-U','edupilot','-d','edupilot','-At','-F','|','-v','ON_ERROR_STOP=0','-c',sql],capture_output=True,text=True)
    return (r.stdout+r.stderr).strip()
def rds(*a):
    r=subprocess.run(['docker','exec','ep-r-redis-1','redis-cli',*a],capture_output=True,text=True); return r.stdout.strip()
def sse(method,path,token,body=None,idem=None,maxsec=90):
    """đọc SSE -> list (event,data)"""
    h={'content-type':'application/json','authorization':'Bearer '+token,'Accept':'text/event-stream'}
    if method=='POST': h['Idempotency-Key']=idem or str(uuid.uuid4())
    req=urllib.request.Request(B+path,data=None if body is None else json.dumps(body).encode(),method=method,headers=h)
    ev=[];t0=time.time()
    try:
        r=urllib.request.urlopen(req,context=ctx,timeout=maxsec)
    except urllib.error.HTTPError as e: return e.code,[('http',e.read().decode())]
    cur=None;st=r.status
    try:
        for ln in r:
            ln=ln.decode().rstrip('\n')
            if ln.startswith('event:'): cur=ln[6:].strip()
            elif ln.startswith('data:'): ev.append((cur,ln[5:].strip(),round(time.time()-t0,3)))
            if time.time()-t0>maxsec: break
    except Exception: pass
    return st,ev
R={}
def rec(tc,ok,note=''):
    R[tc]=('P' if ok else 'F',note); print(('PASS' if ok else 'FAIL'),tc,note[:140])
def save(name):
    json.dump(R,open(f'/tmp/qc2e/res-{name}.json','w'),ensure_ascii=False,indent=0)
def sqlx(sql):
    """chạy trong giao dịch rồi ROLLBACK; trả (SQLSTATE hoặc 'OK', thông điệp, stdout)"""
    r=subprocess.run(['docker','exec','-i','ep-r-postgres-1','psql','-U','edupilot','-d','edupilot','-At','-F','|','-v','VERBOSITY=verbose','-v','ON_ERROR_STOP=1','-c','begin;\n'+sql+'\nrollback;'],capture_output=True,text=True)
    m=re.search(r'ERROR:\s+(\w{5}):',r.stderr)
    return (m.group(1) if m else 'OK'),r.stderr.strip()[:200],r.stdout.strip()
def rec(tc,ok,note='',status=None):
    st=status or ('P' if ok else 'F')
    R[tc]=(st,note); print({'P':'PASS','F':'FAIL','K':'K'}[st],tc,note[:140])
    spr=os.environ.get('SPR') or ('7' if re.match(r'TC-(P4|P5|CI|S7)',tc) else '6')
    f=f'/tmp/qc2e/master{spr}.json'
    try: m=json.load(open(f))
    except Exception: m={}
    m[tc]=[st,note]; json.dump(m,open(f,'w'),ensure_ascii=False)
