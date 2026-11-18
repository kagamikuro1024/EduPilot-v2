import json,ssl,urllib.request,uuid,time,subprocess,os
B='https://localhost:1143/api/v1'; PW='Edupilot#Seed-2026'
ctx=ssl.create_default_context(); ctx.check_hostname=False; ctx.verify_mode=ssl.CERT_NONE
_tok={}; _tt={}
def call(m,p,tok=None,body=None,hdr=None,raw=None,idem=True,base=None,_try=0):
    r_=_call(m,p,tok,body,hdr,raw,idem,base)
    if r_[0] in(0,502,503) and _try<5 and not (p.startswith('/readyz')):
        time.sleep(8); return call(m,p,tok,body,hdr,raw,idem,base,_try+1)
    return r_
def _call(m,p,tok=None,body=None,hdr=None,raw=None,idem=True,base=None):
    h={'content-type':'application/json'}
    if tok: h['authorization']='Bearer '+tok
    if m in('POST',) and idem and not (hdr and 'Idempotency-Key' in hdr): h['Idempotency-Key']=str(uuid.uuid4())
    if hdr: h.update(hdr)
    d=raw if raw is not None else (json.dumps(body).encode() if body is not None else None)
    r=urllib.request.Request((base or B)+p,data=d,method=m,headers=h)
    try:
        x=urllib.request.urlopen(r,context=ctx,timeout=60); s=x.status; t=x.read(); hd=dict(x.headers)
    except urllib.error.HTTPError as e: s=e.code; t=e.read(); hd=dict(e.headers)
    except Exception as e: return 0,{'err':str(e)},{}
    try: j=json.loads(t)
    except Exception: j={'_raw':t.decode('utf8','replace')[:500]}
    return s,j,hd
def login(u,pw=PW):
    k=u
    if k in _tok and time.time()-_tt[k]<600: return _tok[k]
    s,j,_=call('POST','/auth/login',body={'email':u if '@' in u else u+'@edupilot.local','password':pw})
    if s!=200: raise Exception(f'login {u} {s} {j}')
    _tok[k]=j['access_token']; _tt[k]=time.time(); return _tok[k]
def psql(q):
    return subprocess.run(['docker','exec','-i','ep-r-postgres-1','psql','-U','edupilot','-d','edupilot','-At','-F','|','-c',q],capture_output=True,text=True).stdout.strip()
def rds(*a):
    return subprocess.run(['docker','exec','ep-r-redis-1','redis-cli',*a],capture_output=True,text=True).stdout.strip()
R={}
def rec(tc,res,note=''):
    R[tc]=(res,note); 
    json.dump(R,open('/tmp/qc2-b/results-'+os.environ.get('SP','x')+'.json','w'),ensure_ascii=False)
    print(tc,res,note[:140])

import hmac,hashlib,base64
def _b(x): return base64.urlsafe_b64encode(x).rstrip(b'=').decode()
def secret():
    out=subprocess.run(['docker','inspect','ep-r-gateway-1','--format','{{range .Config.Env}}{{println .}}{{end}}'],capture_output=True,text=True).stdout
    return [l.split('=',1)[1] for l in out.splitlines() if l.startswith('JWT_SECRET_KEY=')][0]
def forge(claims_from,sec=None,alg='HS256',**ch):
    p=json.loads(base64.urlsafe_b64decode(claims_from.split('.')[1]+'=='))
    p.update(ch)
    for k,v in list(p.items()):
        if v is None: del p[k]
    h=_b(json.dumps({'alg':alg,'typ':'JWT'}).encode()); b=_b(json.dumps(p).encode())
    if alg=='none': return h+'.'+b+'.'
    return h+'.'+b+'.'+_b(hmac.new((sec or secret()).encode(),(h+'.'+b).encode(),hashlib.sha256).digest())
import re as _re
MP='http://localhost:8725/api/v1'
def mp(path):
    try: return json.loads(urllib.request.urlopen(MP+path,timeout=20).read())
    except Exception as e: return {'err':str(e)}
def mail_to(addr,subj=None,since='',wait=20):
    end=time.time()+wait
    while True:
        j=mp('/messages?limit=300')
        ms=[m for m in j.get('messages',[]) if any(a['Address']==addr for a in m['To']) and (not subj or _re.search(subj,m['Subject'])) and m['Created']>=since]
        if ms: 
            m=sorted(ms,key=lambda x:x['Created'])[-1]; full=mp('/message/'+m['ID']); full['Subject']=m['Subject']; return full
        if time.time()>end: return None
        time.sleep(1)
def mail_count(addr): return len([m for m in mp('/messages?limit=300').get('messages',[]) if any(a['Address']==addr for a in m['To'])])
def link_token(m,needle):
    for u in _re.findall(r'https?://[^\s<>")]+',m.get('Text','')):
        if _re.search(needle,u): return urllib.parse.parse_qs(urllib.parse.urlparse(u).query).get('token',[''])[0] or u.rsplit('/',1)[-1]
def resetrl():
    subprocess.run(['docker','exec','ep-r-redis-1','sh','-c',"redis-cli --scan --pattern 'ep:rl:auth*' | xargs -r redis-cli del; redis-cli --scan --pattern 'ep:auth:forgot:*' | xargs -r redis-cli del; redis-cli --scan --pattern 'ep:auth:ipfail:*' | xargs -r redis-cli del; redis-cli --scan --pattern 'ep:auth:fail:*' | xargs -r redis-cli del"],capture_output=True)
import urllib.parse

# ---- qc2 (sprint 4, P209–P212): ghi kết quả vào evidence-4/results-smk34/results-qc2.tsv (bản ghi sau thắng) ----
OUT4 = os.path.expanduser('~/Documents/TA_Agent_qc2/docs/regression/evidence-4/results-smk34/results-qc2.tsv')
def T(tc, st, note=''):
    note = ' '.join(str(note).split())[:900]
    with open(OUT4, 'a') as f: f.write(f'{tc}\t{st}\t{note}\n')
    print(st, tc, note[:170], flush=True)
def C(tc, cond, ok, bad=None):
    T(tc, 'P' if cond else 'F', ok if cond else (bad or ok)); return cond
def resetrl():
    subprocess.run("docker exec ep-r-redis-1 sh -c \"(redis-cli --scan --pattern 'ep:rl:auth*'; redis-cli --scan --pattern 'ep:rl:ip:*') | xargs -r redis-cli del\"", shell=True, capture_output=True)
def resetjoin(*parts):
    subprocess.run("docker exec ep-r-redis-1 sh -c \"redis-cli --scan --pattern 'ep:join:fail:*' | xargs -r redis-cli del\"", shell=True, capture_output=True)
def L(u):
    try: return login(u)
    except Exception:
        resetrl(); return login(u)
def new_course(tag, **kw):
    A = L('admin'); cc = ('QS' + tag + uuid.uuid4().hex[:3]).upper()
    for _try in range(8):
        s, j, _ = call('POST', '/admin/courses', A, {'subject_code': kw.get('subject', 'INT1006'), 'class_code': cc, 'name': 'Lớp QC2S4 ' + cc, 'semester': '2026-2027-HK1'})
        if s != 429: break
        time.sleep(int(j.get('retry_after', 10)) + 1)
    assert s == 201, (s, j)
    cid = j['id']; tid = psql("select id from users where email='%s@edupilot.local'" % kw.get('teacher', 'teacher'))
    s2, j2, _ = call('POST', f'/admin/courses/{cid}/assign', A, {'teacher_id': tid}); assert s2 in (200, 201), (s2, j2)
    return cid, cc, j.get('join_code') or psql(f"select join_code from courses where id='{cid}'")
def settings(cid, who='teacher', **kw):
    t = L(who); cur = call('GET', f'/courses/{cid}/join-code', t)[1]
    s, j, _ = call('PUT', f'/courses/{cid}/join-settings', t, dict(kw, version=cur['version'])); return s, j
def student_pool(n, exclude=()):
    rows = psql("select email from users where role='STUDENT' and status='ACTIVE' and email ~ '^sv[0-9]+@edupilot' order by email").splitlines()
    return [r.split('@')[0] for r in rows if r.split('@')[0] not in exclude][:n]
