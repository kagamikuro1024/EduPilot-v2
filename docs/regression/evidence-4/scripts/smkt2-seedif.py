# SmkT2: TC-P212-14/GATEP2-09 phần chạy được: scripts/seed.mjs --if-empty trên DB trống riêng (smkt2-gw) và trên DB đã có lớp 761987/761988
import os,subprocess,json,uuid,time; exec(open(os.path.join(os.path.dirname(os.path.abspath(__file__)),'qc2i-r4apia-lib.py')).read())
E='qc2i_empty'; PORT=18230; GW=f'http://127.0.0.1:{PORT}/api/v1'; REPO=os.path.expanduser('~/Documents/TA_Agent_qc2')
def pg(q): return subprocess.run(['docker','exec','-i',PGC,'psql','-U','edupilot','-d',E,'-At','-F','|','-c',q],capture_output=True,text=True).stdout.strip()
def sh2(c,t=300): return subprocess.run(c,shell=True,capture_output=True,text=True,timeout=t)
url=f'postgres://edupilot:edupilot-dev@postgres:5432/{E}?sslmode=disable'; SEEDPW='Smkt2!Seed-Pw-'+uuid.uuid4().hex[:6]+'x'
out={}
try:
    sh2(f'HEAVY= bash {os.path.dirname(os.path.abspath(__file__))}/qc2i-r4apia-stack.sh up0')
    sh2(f"docker rm -f smkt2-gw smkt2-wk >/dev/null 2>&1; docker run -d --name smkt2-gw --network ep-r_default --env-file /tmp/qc2i-r4apia.env -e DATABASE_URL='{url}' -e PGBOUNCER_URL='{url}' -e REDIS_URL=redis://qc2i-redis:6379/6 -e AUTH_LOGIN_IP_PER_MIN=100000 -e AUTH_LOGIN_IP_FAIL_PER_15M=1000000 -e AUTH_REGISTER_IP_PER_HOUR=100000 -p 127.0.0.1:{PORT}:8080 edupilot-gateway-test")
    sh2(f"docker run -d --name smkt2-wk --network ep-r_default --env-file /tmp/qc2i-r4apia.env -e DATABASE_URL='{url}' -e PGBOUNCER_URL='{url}' -e REDIS_URL=redis://qc2i-redis:6379/6 -e INSTANCE_ID=smkt2-wk edupilot-worker-test")
    for i in range(40):
        if sh2(f'curl -s -o /dev/null -w "%{{http_code}}" {GW}/readyz').stdout=='200': break
        time.sleep(2)
    env=f"API_URL={GW} MAILPIT_URL=http://127.0.0.1:18223 SEED_DEFAULT_PASSWORD='{SEEDPW}' NODE_TLS_REJECT_UNAUTHORIZED=0"
    admin_cmd=f"docker run --rm -i --network ep-r_default --env-file /tmp/qc2i-r4apia.env -e DATABASE_URL='{url}' -e PGBOUNCER_URL='{url}' -e ADMIN_PASSWORD='{SEEDPW}' edupilot-gateway:latest admin create"
    # (a) DB trống: seed tự chạy, in từng bước (cắt sau 25 s: seed đầy đủ cần docling/LLM giả, ≈ 10 phút theo README 5.2)
    t0=time.time(); r=subprocess.run(f"cd {REPO} && {env} SEED_ADMIN_CMD=\"{admin_cmd}\" timeout 25 node scripts/seed.mjs --if-empty --verbose 2>&1 | grep -E '^\\[[0-9]+/[0-9]+\\]|Đã có dữ liệu|Seed' | head -6",shell=True,capture_output=True,text=True); out['empty']=r.stdout.strip().replace('\n',' / ')
    # (b) DB đã có hai lớp 761987/761988: --if-empty không seed
    A=call('POST','/auth/login',None,{'email':'admin@edupilot.local','password':SEEDPW},base=GW)[1].get('access_token')
    tid=None
    for cc in ('761987','761988'):
        s,j,_=call('POST','/admin/courses',A,{'subject_code':'INT1006','class_code':cc,'name':'Lớp '+cc,'semester':'2026-2027-HK1'},base=GW); out['c'+cc]=s
    n1=pg("select (select count(*) from users)||'/'||(select count(*) from courses)||'/'||(select count(*) from enrollments)")
    r2=subprocess.run(f"cd {REPO} && {env} timeout 60 node scripts/seed.mjs --if-empty 2>&1 | grep -v -i warning | tail -2",shell=True,capture_output=True,text=True); out['again']=r2.stdout.strip().replace('\n',' / ')
    n2=pg("select (select count(*) from users)||'/'||(select count(*) from courses)||'/'||(select count(*) from enrollments)")
    out['same']=n1==n2
finally:
    sh2('docker rm -f smkt2-gw smkt2-wk >/dev/null 2>&1'); sh2(f"docker exec {PGC} psql -U edupilot -d postgres -c 'drop database if exists {E} with (force)'")
print(json.dumps(out,ensure_ascii=False))
