import os; os.environ['SP']='3-gwenv'
from lib import *
import base64
IMG='edupilot-gateway-test'
base=['-e','APP_ENV=test','-e','JWT_SECRET_KEY=qc2-b-jwt-secret-0123456789abcdef0123','-e','DATABASE_URL=postgres://u:p@127.0.0.1:1/x?sslmode=disable','-e','PGBOUNCER_URL=postgres://u:p@127.0.0.1:1/x?sslmode=disable','-e','REDIS_URL=redis://127.0.0.1:1/0','-e','BLOB_ENDPOINT=127.0.0.1:1','-e','BLOB_BUCKET=b','-e','BLOB_ACCESS_KEY=a','-e','BLOB_SECRET_KEY=s','-e','CORS_ORIGINS=http://localhost:3000']
good=base64.b64encode(os.urandom(32)).decode()
def run(env):
    args=['docker','run','--rm']+base
    for k,v in env.items(): args+=['-e',f'{k}={v}']
    args.append(IMG); t=time.time()
    try: r=subprocess.run(args,capture_output=True,text=True,timeout=40); out=r.stdout+r.stderr; rc=r.returncode
    except subprocess.TimeoutExpired as e: out=((e.stdout or b'')+(e.stderr or b'')).decode() if isinstance(e.stdout,(bytes,type(None))) else str(e.stdout); rc='timeout'
    return rc,round(time.time()-t,1),out
# 18 khoá hỏng
res={}
for name,v in [('rỗng',''),('abc','abc'),('16 byte',base64.b64encode(os.urandom(16)).decode())]:
    rc,sec,out=run({'APP_ENCRYPTION_KEY':v}); res[name]=(rc,sec,'APP_ENCRYPTION_KEY' in out,(v in out) if v else False,out.strip().splitlines()[-1][:160] if out.strip() else '')
rec('TC-P101-18','P' if all(r[0] not in(0,'timeout') and r[2] and not r[3] for r in res.values()) else 'F',str({k:(v[0],v[1],'nêu tên biến' if v[2] else 'KHÔNG nêu tên biến','lộ khoá' if v[3] else 'không lộ khoá') for k,v in res.items()})+' | '+res['abc'][4])
# 19
r19={}
for name,v in [('xuống dòng cuối',good+'\n'),('33 byte',base64.b64encode(os.urandom(33)).decode()),('base64url',base64.urlsafe_b64encode(b'\xfb'*32).decode()),('khoảng trắng 2 đầu',' '+good+' ')]:
    rc,sec,out=run({'APP_ENCRYPTION_KEY':v}); r19[name]=(rc,'panic' in out.lower(),'APP_ENCRYPTION_KEY' in out)
rec('TC-P101-19','P' if all(not x[1] for x in r19.values()) and r19['33 byte'][0] not in(0,) and r19['base64url'][0] not in (0,) else 'F',str(r19)+' (rc≠0 & có tên biến = từ chối; rc=timeout/1 sau khi qua bước khoá = chấp nhận)')
# 32 LLM_EMBED_DIMS
rc,sec,out=run({'APP_ENCRYPTION_KEY':good,'LLM_EMBED_DIMS':'768'}); rec('TC-P102-32','P' if rc not in(0,'timeout') and 'LLM_EMBED_DIMS' in out and sec<=10 else 'F',f'LLM_EMBED_DIMS=768 → rc={rc} sau {sec}s; '+(out.strip().splitlines()[-1][:150] if out.strip() else ''))
# 47 biến sai
r47={}
for var,val in [('LLM_BATCH_SHARE','1.5'),('LLM_BATCH_SHARE','0'),('LLM_BATCH_SHARE','-1'),('LLM_MAX_CONCURRENCY','0'),('LLM_MAX_CONCURRENCY','-3'),('LLM_QUEUE_MAX','-1'),('LLM_BREAKER_FAILS','0')]:
    rc,sec,out=run({'APP_ENCRYPTION_KEY':good,var:val}); r47[f'{var}={val}']=(rc,var in out)
rec('TC-P103-47','P' if all(v[0] not in(0,'timeout') and v[1] for v in r47.values()) else 'F',str(r47))
