# TC-P202-46: biến cấu hình sai → gateway thoát rc=1 nêu tên biến (docker run --rm, không đụng ep-r). Chạy: python3 qc2i-r4apia-env46.py
import os
exec(open(os.path.join(os.path.dirname(os.path.abspath(__file__)),'qc2i-r4apia-lib.py')).read()) if 'rec' not in globals() else None
IMG=os.environ.get('QC2I_GWIMG','edupilot-gateway-test')
base=['-e','APP_ENV=test','-e','JWT_SECRET_KEY=qc2ia-jwt-secret-0123456789abcdef0123456789','-e','DATABASE_URL=postgres://u:p@127.0.0.1:1/x?sslmode=disable','-e','PGBOUNCER_URL=postgres://u:p@127.0.0.1:1/x?sslmode=disable','-e','REDIS_URL=redis://127.0.0.1:1/0','-e','BLOB_ENDPOINT=127.0.0.1:1','-e','BLOB_BUCKET=b','-e','BLOB_ACCESS_KEY=a','-e','BLOB_SECRET_KEY=s','-e','CORS_ORIGINS=http://localhost:3000','-e','APP_ENCRYPTION_KEY='+base64.b64encode(os.urandom(32)).decode(),'-e','ATTENDANCE_TOKEN_KEY='+base64.b64encode(os.urandom(32)).decode()]
def run(env,secs=40):
    a=['docker','run','--rm']+base  # chạy tức thì (thoát vì cấu hình sai trước khi mở kết nối) → không qua heavy.sh
    for k,v in env.items(): a+=['-e',f'{k}={v}']
    t=time.time()
    try: r=subprocess.run(a+[IMG],capture_output=True,text=True,timeout=secs+200); out=r.stdout+r.stderr; rc=r.returncode
    except subprocess.TimeoutExpired as e: out='timeout'; rc='timeout'
    return rc,round(time.time()-t,1),out
cases=[('ACCESS_TOKEN_TTL','2h',{'ACCESS_TOKEN_TTL':'2h'}),('ACCESS_TOKEN_TTL','30s',{'ACCESS_TOKEN_TTL':'30s'}),('REFRESH_TOKEN_TTL','30m',{'REFRESH_TOKEN_TTL':'30m'}),
 ('SESSION_ABSOLUTE_TTL','1h<refresh',{'SESSION_ABSOLUTE_TTL':'1h'}),('COOKIE_DOMAIN','"exa mple.com"',{'COOKIE_DOMAIN':'exa mple.com'}),('COOKIE_DOMAIN','" example.com "',{'COOKIE_DOMAIN':' example.com '})]
res={}
for var,lab,env in cases:
    for _ in range(3):
        rc,sec,out=run(env)
        if rc!='timeout' and 'Cannot connect to the Docker' not in out: break
        time.sleep(20)
    res[f'{var}={lab}']=(rc,var in out,out.strip().splitlines()[-1][:200] if out.strip() else '')
    print(var,lab,res[f'{var}={lab}'],flush=True)
# mặc định: log "config loaded" của gateway qc2i-gw1 (không đặt biến TTL) nêu giá trị mặc định
lg_=sh('docker logs qc2i-gw1 2>&1 | grep "config loaded" | head -1')
dflt=re.findall(r'"(?:access_token_ttl|refresh_token_ttl|session_absolute_ttl)":"[^"]*"',lg_)
print('default',dflt,lg_[:400])
json.dump({'res':{k:list(v) for k,v in res.items()},'default':dflt,'log':lg_[:1500]},open(EV+'/qc2i-r4apia-env46.json','w'),ensure_ascii=False,indent=1)
bad=[k for k,v in res.items() if not (v[0]==1 and v[1])]
rec('TC-P202-46','P' if not bad else 'F',f'rc/nêu tên biến: '+'; '.join(f'{k}→rc={v[0]},tên={v[1]}' for k,v in res.items())+f'; mặc định (log config loaded của gateway không đặt biến): {dflt}')
