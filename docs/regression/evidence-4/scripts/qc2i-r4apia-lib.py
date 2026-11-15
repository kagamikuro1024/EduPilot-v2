# QC2-I r4apia: thư viện chung (B mặc định = gw3 giới hạn nới; G1/G2 giới hạn mặc định) (sao từ evidence-3/scripts/lib.py, chỉnh cho stack auth riêng qc2i-*). Dùng: exec(open(<thư mục>/qc2i-r4apia-lib.py).read())
import json,ssl,urllib.request,urllib.parse,uuid,time,subprocess,os,re,hmac,hashlib,base64,concurrent.futures as cf
from collections import Counter
G1=os.environ.get('QC2I_G1','http://127.0.0.1:18221/api/v1'); G2=os.environ.get('QC2I_G2','http://127.0.0.1:18222/api/v1')
MP=os.environ.get('QC2I_MP','http://127.0.0.1:18223/api/v1')
PGC=os.environ.get('QC2I_PGC','ep-r-postgres-1'); DB=os.environ.get('QC2I_DB','qc2i_auth'); RDC=os.environ.get('QC2I_RDC','qc2i-redis')
G3=os.environ.get('QC2I_G3','http://127.0.0.1:18224/api/v1'); B=G3; ADMIN_EMAIL='qc2ia-admin@edupilot.local'; ADMIN_PW='Qc2ia!Admin-Pw-2026x'
OUT='/tmp/qc2-i/res/r4apia.tsv'; EV=os.path.expanduser('~/Documents/TA_Agent_qc2/docs/regression/evidence-4')
ctx=ssl.create_default_context(); ctx.check_hostname=False; ctx.verify_mode=ssl.CERT_NONE
def _call(m,p,tok=None,body=None,hdr=None,raw=None,idem=True,base=None):
    h={'content-type':'application/json'}
    if tok: h['authorization']='Bearer '+tok
    if m=='POST' and idem and not (hdr and 'Idempotency-Key' in hdr): h['Idempotency-Key']=str(uuid.uuid4())
    if hdr: h.update(hdr)
    d=raw if raw is not None else (json.dumps(body).encode() if body is not None else None)
    r=urllib.request.Request((base or B)+p,data=d,method=m,headers=h)
    try: x=urllib.request.urlopen(r,context=ctx,timeout=90); s=x.status; t=x.read(); hd=dict(x.headers)
    except urllib.error.HTTPError as e: s=e.code; t=e.read(); hd=dict(e.headers)
    except Exception as e: return 0,{'err':str(e)},{}
    try: j=json.loads(t)
    except Exception: j={'_raw':t.decode('utf8','replace')[:500]}
    return s,j,hd
def call(m,p,tok=None,body=None,hdr=None,raw=None,idem=True,base=None,retry=True,_try=0):
    r=_call(m,p,tok,body,hdr,raw,idem,base)
    if retry and r[0] in(0,502,503,504) and _try<8: time.sleep(8); return call(m,p,tok,body,hdr,raw,idem,base,retry,_try+1)
    return r
def ck(h): return (h.get('Set-Cookie') or h.get('set-cookie') or '').split(';')[0]
def ckval(c): return c.split('=',1)[1]
def sh(cmd,timeout=120): return subprocess.run(cmd,shell=True,capture_output=True,text=True,timeout=timeout).stdout
def psql(q): return subprocess.run(['docker','exec','-i',PGC,'psql','-U','edupilot','-d',DB,'-At','-F','|','-c',q],capture_output=True,text=True).stdout.strip()
def rds(*a): return subprocess.run(['docker','exec',RDC,'redis-cli',*a],capture_output=True,text=True).stdout.strip()
def resetrl():
    subprocess.run(['docker','exec',RDC,'sh','-c',"for p in 'ep:rl:*' 'ep:auth:ipfail:*' 'ep:auth:blockip:*'; do redis-cli --scan --pattern \"$p\" | xargs -r redis-cli del >/dev/null; done"],capture_output=True)
def clearfail(): # xoá bộ đếm sai theo email (chỉ redis riêng)
    subprocess.run(['docker','exec',RDC,'sh','-c',"redis-cli --scan --pattern 'ep:auth:fail:*' | xargs -r redis-cli del >/dev/null"],capture_output=True)
def mp(path):
    try: return json.loads(urllib.request.urlopen(MP+path,timeout=30).read())
    except Exception as e: return {'err':str(e)}
def mails(addr): return [m for m in mp('/messages?limit=500').get('messages',[]) if any(a['Address']==addr for a in m['To'])]
def mail_count(addr): return len(mails(addr))
def mail_to(addr,subj=None,since='',wait=40,after=None):
    end=time.time()+wait
    while True:
        ms=[m for m in mails(addr) if (not subj or re.search(subj,m['Subject'])) and m['Created']>=since]
        if ms:
            m=sorted(ms,key=lambda x:x['Created'])[-1]; full=mp('/message/'+m['ID']); full['Subject']=m['Subject']; return full
        if time.time()>end: return None
        time.sleep(1)
def iso(): return time.strftime('%Y-%m-%dT%H:%M:%S',time.gmtime(time.time()-3))+'Z'
def link_token(m,needle):
    for u in re.findall(r'https?://[^\s<>")]+',m.get('Text','')):
        if re.search(needle,u): return urllib.parse.parse_qs(urllib.parse.urlparse(u).query).get('token',[''])[0]
TAG=uuid.uuid4().hex[:5]
PWD='Qc2ia!Pass-'+TAG+'x7'
def claims(t): return json.loads(base64.urlsafe_b64decode(t.split('.')[1]+'=='))
def lg(em,pw=PWD,base=None):
    s,j,h=call('POST','/auth/login',None,{'email':em,'password':pw},base=base); return s,j,ck(h)
def sha(x): return hashlib.sha256(x.encode()).hexdigest()
_adm={}
def admin_tok():
    if 'k' not in _adm or time.time()-_adm['t']>600:
        s,j,_=call('POST','/auth/login',None,{'email':ADMIN_EMAIL,'password':ADMIN_PW}); assert s==200,(s,j); _adm.update(k=j['access_token'],t=time.time())
    return _adm['k']
def reg(label,verify=True,code='',pw=None):
    """đăng ký → thư → (xác minh). Trả email."""
    resetrl(); em=f'qc2ia-{label}.{TAG}@edupilot.local'; t0=iso()
    s,j,_=call('POST','/auth/register',None,{'email':em,'password':pw or PWD,'full_name':'QC2IA '+label,'student_code':code}); assert s==202,(s,j)
    if verify:
        m=mail_to(em,'Xác minh',t0,60); assert m,'không có thư xác minh'
        s,j,_=call('POST','/auth/verify-email',None,{'token':link_token(m,'verify-email')}); assert s==200,(s,j)
    return em
def invite(label,role='TEACHER',accept=True):
    em=f'qc2ia-{label}.{TAG}@edupilot.local'; t0=iso()
    s,j,_=call('POST','/admin/users',admin_tok(),{'email':em,'full_name':'QC2IA '+label,'role':role}); assert s==201,(s,j)
    if accept:
        m=mail_to(em,'mời|Lời mời',t0,60); s2,j2,_=call('POST','/auth/accept-invite',None,{'token':link_token(m,'invite'),'password':PWD}); assert s2==200,(s2,j2)
    return em,j['id']
R={}
def rec(tc,kq,note=''):
    """ghi dòng kết quả (bản cuối thắng khi chạy lại)."""
    note=note.replace('\t',' ').replace('\n',' ')[:600]; R[tc]=(kq,note)
    os.makedirs(os.path.dirname(OUT),exist_ok=True)
    open(OUT,'a').write(f'{tc}\t{kq}\t{note}\n'); print(tc,kq,note[:200],flush=True)
