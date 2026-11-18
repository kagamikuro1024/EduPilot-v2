# SmkT2: gateway riêng smkt2-gw trên DB trống qc2i_empty (không đụng ep-r, không đụng qc2i_auth): TC-P207-11 (DB chết → 503) và TC-P208-39 (API trên DB trống)
import os,subprocess,json,uuid,time,re; exec(open(os.path.join(os.path.dirname(os.path.abspath(__file__)),'qc2i-r4apia-lib.py')).read())
OUTT2=os.path.expanduser('~/Documents/TA_Agent_qc2/docs/regression/evidence-4/results-smk34/results-smk4t2.tsv')
def rec2(tc,k,n): open(OUTT2,'a').write(f'{tc}\t{k}\t{" ".join(n.split())}\n'); print(k,tc,n[:200],flush=True)
E='qc2i_empty'; PORT=18230; GW=f'http://127.0.0.1:{PORT}/api/v1'
def pg(q,db=E): return subprocess.run(['docker','exec','-i',PGC,'psql','-U','edupilot','-d',db,'-At','-F','|','-c',q],capture_output=True,text=True).stdout.strip()
def sh2(c): return subprocess.run(c,shell=True,capture_output=True,text=True).stdout.strip()
url=f'postgres://edupilot:edupilot-dev@postgres:5432/{E}?sslmode=disable'
AP='Smkt2!Quan-Tri-Pw-'+uuid.uuid4().hex[:6]+'x'
try:
    sh2(f'HEAVY= bash {os.path.dirname(os.path.abspath(__file__))}/qc2i-r4apia-stack.sh up0')
    assert pg('select count(*) from users')=='0'
    sh2(f"docker run --rm --network ep-r_default --env-file /tmp/qc2i-r4apia.env -e DATABASE_URL='{url}' -e PGBOUNCER_URL='{url}' -e ADMIN_PASSWORD='{AP}' edupilot-gateway:latest admin create --email admin@edupilot.local --name 'Quản trị'")
    sh2(f"docker rm -f smkt2-gw >/dev/null 2>&1; docker run -d --name smkt2-gw --network ep-r_default --env-file /tmp/qc2i-r4apia.env -e DATABASE_URL='{url}' -e PGBOUNCER_URL='{url}' -e REDIS_URL=redis://qc2i-redis:6379/5 -e AUTH_LOGIN_IP_PER_MIN=100000 -e AUTH_LOGIN_IP_FAIL_PER_15M=1000000 -v $HOME/Documents/TA_Agent_qc2/seed/demo:/seed/demo:ro -p 127.0.0.1:{PORT}:8080 edupilot-gateway-test")
    sh2(f"docker rm -f smkt2-wk >/dev/null 2>&1; docker run -d --name smkt2-wk --network ep-r_default --env-file /tmp/qc2i-r4apia.env -e DATABASE_URL='{url}' -e PGBOUNCER_URL='{url}' -e REDIS_URL=redis://qc2i-redis:6379/5 -e INSTANCE_ID=smkt2-wk -v $HOME/Documents/TA_Agent_qc2/seed/demo:/seed/demo:ro edupilot-worker-test")
    for i in range(40):
        if sh2(f'curl -s -o /dev/null -w "%{{http_code}}" {GW}/readyz')=='200': break
        time.sleep(2)
    B=GW
    def L(em,pw): s,j,_=call('POST','/auth/login',None,{'email':em,'password':pw},base=GW); assert s==200,(s,j); return j['access_token']
    A=L('admin@edupilot.local',AP)
    # nạp người dùng DB trống (GV, SV) bằng SQL từ hàng Admin (cùng mật khẩu)
    for em,role,name in [('gv.iso@edupilot.local','TEACHER','GV Cô Lập'),('sv.iso@edupilot.local','STUDENT','SV Cô Lập'),('sv2.iso@edupilot.local','STUDENT','SV Hai')]:
        sc='null' if role=='TEACHER' else "'7"+str(abs(hash(em))%100000000).zfill(8)+"'"
        pg(f"insert into users(email,password_hash,full_name,role,status,email_verified_at,student_code) select '{em}',password_hash,'{name}','{role}','ACTIVE',now(),{sc} from users where email='admin@edupilot.local'")
    # ---- P208-39: Admin mở lớp + gán GV trên DB trống → GV thấy chuông + mã ≤ 60 s; mã khớp courses.join_code
    t0=time.time(); s,j,_=call('POST','/admin/courses',A,{'subject_code':'INT1006','class_code':'ISO101','name':'Lớp cô lập','semester':'2026-2027-HK1'},base=GW); cid=j.get('id'); assert s==201,(s,j)
    gvid=pg("select id from users where email='gv.iso@edupilot.local'")
    s2,j2,_=call('POST',f'/admin/courses/{cid}/assign',A,{'teacher_id':gvid},base=GW); assert s2 in(200,201),(s2,j2)
    GV=L('gv.iso@edupilot.local',AP); got=None
    while time.time()-t0<60:
        n=call('GET','/notifications?limit=30',GV,base=GW)[1]; its=n.get('items',[])
        got=next((x for x in its if x.get('type')=='COURSE_ASSIGNED' or 'phân công' in (x.get('title') or '')),None)
        if got: break
        time.sleep(1)
    dt=time.time()-t0; jc=pg(f"select join_code from courses where id='{cid}'",E); jcs=call('GET',f'/courses/{cid}/join-code',GV,base=GW)
    mine=call('GET','/me/courses',GV,base=GW)[1]; nm=len(mine.get('items',[]) if isinstance(mine,dict) else mine)
    ok39=bool(got) and dt<=60 and (jc in (got.get('body') or '') or jc in json.dumps(got,ensure_ascii=False)) and jcs[0]==200 and jcs[1].get('join_code')==jc and nm==1
    rec2('TC-P208-39','P' if ok39 else 'F',f'DB trống riêng qc2i_empty (gateway smkt2-gw + worker smkt2-wk, chỉ Admin bootstrap): Admin mở lớp (201) + gán GV (200/201) → GV thấy thông báo «{(got or {}).get("title")}» sau {dt:.1f} s (≤ 60 s), nội dung có mã tham gia {jc}={jc in json.dumps(got,ensure_ascii=False)}; GET /courses/{{id}}/join-code của GV trả {jcs[1].get("join_code")} = courses.join_code ({jcs[1].get("join_code")==jc}); /me/courses của GV {nm} lớp. Phần giao diện (chuông, «Quản lý lớp này») xem TC-GATEP2-24; smkt2-iso.py')
    # ---- P207-11
    sv=pg("select id from users where email='sv.iso@edupilot.local'"); SV=L('sv.iso@edupilot.local',AP)
    sA=pg(f"insert into enrollments(course_id,user_id,role_in_course,status,joined_via) values ('{cid}','{sv}','STUDENT','ACTIVE','CODE') returning id")
    a1=call('GET','/courses/abc/today',SV,base=GW); a2=call('GET','/courses/00000000-0000-0000-0000-000000000000/today',SV,base=GW)
    ok_pre=call('GET',f'/courses/{cid}/today',SV,base=GW)[0]
    # không cache: gỡ thành viên → request kế tiếp 403 ngay
    pg(f"update enrollments set status='REMOVED',removed_at=now() where course_id='{cid}' and user_id='{sv}'"); rm=call('GET',f'/courses/{cid}/today',SV,base=GW)
    pg(f"update enrollments set status='ACTIVE',removed_at=null where course_id='{cid}' and user_id='{sv}'"); back=call('GET',f'/courses/{cid}/today',SV,base=GW)[0]
    # DB chết 3 s khi gọi
    pg("select pg_terminate_backend(pid) from pg_stat_activity where datname='%s' and pid<>pg_backend_pid()"%E,'postgres') if False else None
    try:
        subprocess.run(['docker','exec','-i',PGC,'psql','-U','edupilot','-d','postgres','-c',f"alter database {E} allow_connections false"],capture_output=True)
        subprocess.run(['docker','exec','-i',PGC,'psql','-U','edupilot','-d','postgres','-c',f"select pg_terminate_backend(pid) from pg_stat_activity where datname='{E}'"],capture_output=True)
        time.sleep(1); dead=[call('GET',f'/courses/{cid}/today',SV,base=GW,retry=False) for _ in range(3)]
        rz=call('GET','/readyz',None,base=GW,retry=False)
    finally:
        subprocess.run(['docker','exec','-i',PGC,'psql','-U','edupilot','-d','postgres','-c',f"alter database {E} allow_connections true"],capture_output=True)
    time.sleep(3); after=[call('GET',f'/courses/{cid}/today',SV,base=GW,retry=False)[0] for _ in range(3)]
    ok11=a1[0]==404 and a2[0] in(403,404) and ok_pre==200 and rm[0]==403 and back==200 and all(d[0]==503 for d in dead) and 200 in after
    rec2('TC-P207-11','P' if ok11 else 'F',f'gateway riêng smkt2-gw: /courses/abc/today → {a1[0]}; uuid không có → {a2[0]} {a2[1].get("code")}; SV đang ACTIVE → {ok_pre}; gỡ thành viên (REMOVED) → request kế tiếp {rm[0]} ngay (không cache); phục hồi → {back}; Postgres chết (alter database {E} allow_connections false + ngắt phiên, ~4 s, đã bật lại): 3 lần gọi → {[(d[0],d[1].get("code")) for d in dead]} (không 200), /readyz {rz[0]}; sau khi DB về → {after}; smkt2-iso.py')
finally:
    subprocess.run(['docker','exec','-i',PGC,'psql','-U','edupilot','-d','postgres','-c',f"alter database {E} allow_connections true"],capture_output=True)
    sh2('docker rm -f smkt2-gw smkt2-wk >/dev/null 2>&1'); sh2(f"docker exec {PGC} psql -U edupilot -d postgres -c 'drop database if exists {E} with (force)'")
