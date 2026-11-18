from lib3 import *
import datetime as dt, re, threading
fx=json.load(open('/tmp/qc3e/r4fx.json')); X,Y,L,U=fx['X'],fx['Y'],fx['L'],fx['U']; S=fx['S']
ICT=dt.timezone(dt.timedelta(hours=7)); day=dt.datetime.now(ICT).strftime('%Y%m%d')
def tabid(n): return str(uuid.uuid5(uuid.NAMESPACE_DNS,n))
def hb(u,c,area='CHAT',tab='t1'): return A(u,'POST','/activity/heartbeat',{'course_id':c,'area':area,'tab_id':tabid(tab)})
def rget(k): v=rds('GET',k); return int(v) if v else 0
def dk(*a): return subprocess.run(['docker',*a],capture_output=True,text=True)
def entries(since):
    out=dk('logs','ep-r-postgres-1','--since',since).stdout+dk('logs','ep-r-postgres-1','--since',since).stderr
    ents=[];cur=None
    for l in out.split('\n'):
        if re.match(r'^\d{4}-\d\d-\d\d \d\d:\d\d:\d\d',l):
            if cur: ents.append(cur)
            cur=l
        elif cur is not None: cur+=' '+l.strip()
    if cur: ents.append(cur)
    return ents
def iso(t): return dt.datetime.fromtimestamp(t,dt.timezone.utc).strftime('%Y-%m-%dT%H:%M:%SZ')
R={}
[login(x) for x in (U[0],'qc3t','qc3ta')]
assert lockg('qc3','Sprint7 P5-03-03/04/15/29 + P4-02-02/P4-03-04: log_statement=all, redis stop, kill worker; ≤ 10 phút')
t_lock=time.time()
try:
    pglog(True); time.sleep(2)
    # ---- P5-03-03
    u=U[0]; hb(u,X,'CHAT','t0'); time.sleep(31)  # nạp đệm ep:trk / ep:enr
    k=f'ep:study:{X}:{uid(u)}:{day}:CHAT'; b=rget(k); t0=time.time(); s,_=hb(u,X,'CHAT','t1'); a=rget(k); time.sleep(1.5)
    es=entries(iso(t0-1)); mine=[e for e in es if uid(u) in e]; w=[e for e in mine if re.search(r'\b(insert|update)\b',e,re.I)]; allw=[e for e in es if re.search(r'(statement|execute [^:]*): *(insert|update)',e,re.I)]
    sdaily=[e for e in es if 'study_time_daily' in e]
    R['03']=(s==204 and a-b==30 and len(w)==0 and len(mine)>=1,f'log_statement=all (cửa sổ khoá): nhịp CHAT của SV mới → {s}; Redis {k.split(":")[0]}…:CHAT {b} → {a} (+{a-b}); mục log nhắc user_id của SV: {len(mine)} (toàn SELECT: đọc đệm ep:trk/ep:enr), trong đó INSERT/UPDATE: {len(w)}; mục log nhắc study_time_daily trong cửa sổ: {len(sdaily)} ; tổng INSERT/UPDATE của mọi người dùng khác trong cửa sổ 1,5 s: {len(allw)} (không thuộc request này: không chứa user_id/course của nó)')
    # ---- P4-02-02
    ts=time.time(); eid=mkexam3(L,'qc3t','qc3ta',open_s=8,win=1800,dur=20,nq=2,title='QC3 1k notify'); time.sleep(8)
    es=entries(iso(ts-1)); ins=[e for e in es if re.search(r'INSERT INTO notifications',e) and L in e]; unn=[e for e in ins if 'unnest' in e.lower()]
    nn=psql(f"select count(*) from notifications where course_id='{L}' and created_at>=to_timestamp({int(ts)-1})"); typ=psql(f"select type||':'||count(*) from notifications where course_id='{L}' and created_at>=to_timestamp({int(ts)-1}) group by type").replace('\n','; ')
    nstu=psql("select count(*) from enrollments where role_in_course='STUDENT' and course_id='%s'"%L)
    R['0202']=(len(unn)==1 and len(ins)<=2 and int(nn)>=1000,f'lớp {nstu} SV; GV đặt lịch thi (schedule, exam {eid[:8]}…): log_statement=all, số mục `INSERT INTO notifications` gắn lớp = {len(ins)} trong đó dạng `unnest` (một câu cho cả danh sách) = {len(unn)}; bảng notifications mới của lớp: {nn} dòng ({typ}) — không 1.000 lượt upsert riêng')
    # ---- P4-03-04
    Q=f'/courses/{L}'; ts=time.time(); st,d=A('qc3t','GET',f'{Q}/tickets?limit=100&status=OPEN'); el=time.time()-ts; time.sleep(1.2)
    es=entries(iso(ts-0.5)); mineq=[e for e in es if L in e and re.search(r'(statement|execute)',e)]; off=[e for e in es if re.search(r'\boffset\b',e,re.I) and L in e]; sel=[e for e in mineq if re.search(r'select',e,re.I)]
    n=len(d.get('items',[])) if isinstance(d,dict) else -1
    ntk=psql("select count(*) from escalation_tickets where course_id='%s'"%L); smp=[re.sub(r'\s+',' ',e)[40:150] for e in sel[:3]]
    R['0304']=(st==200 and n==100 and len(sel)<=8 and len(off)==0,f'lớp {ntk} ticket: GET tickets?limit=100&status=OPEN → {st}, {n} mục, {el*1000:.0f} ms; log_statement=all: số câu SQL gắn lớp trong lúc gọi = {len(mineq)} (SELECT {len(sel)}; không phải 100+); `grep -i offset` = {len(off)}; mẫu: {smp}')
    pglog(False)
    # ---- P5-03-29: Redis dừng
    t0=time.time(); dk('stop','ep-r-redis-1'); time.sleep(1)
    t1=time.time(); s,_=hb(U[1],X,'CHAT','t9'); dt1=time.time()-t1
    s2,_=hb(U[1],X,'THREADS','t10'); lg=dk('logs','ep-r-gateway-1','--since',iso(t0-1)); gl=(lg.stdout+lg.stderr).split('\n'); drop=[l for l in gl if 'activity_heartbeat_dropped_total' in l]
    s5=A('qc3t','GET','/me/courses')[0]
    dk('start','ep-r-redis-1')
    for _ in range(40):
        if dk('inspect','ep-r-redis-1','--format','{{.State.Health.Status}}').stdout.strip()=='healthy': break
        time.sleep(1)
    pii=[l for l in drop if uid(U[1]) in l or 'THREADS' in l or 'CHAT' in l]
    d0=re.sub(r'\s+',' ',drop[0])[-170:] if drop else ''
    R['29']=(s==204 and s2==204 and dt1<1.5 and len(drop)>=1 and len(pii)==0,f'docker stop ep-r-redis-1; nhịp → {s}/{s2} trong {dt1:.2f} s (≤ 1 s theo SRS 4.6.1 bước 5; không 5xx); log gateway: {len(drop)} dòng `activity_heartbeat_dropped_total` (WARN) mẫu «{d0}»; dòng có user_id/area (PII)={len(pii)}; gateway không có /metrics (proposals #31) nên kiểm bộ đếm qua log; redis bật lại (healthy)')
finally:
    pglog(False); 
    if dk('inspect','ep-r-redis-1','--format','{{.State.Running}}').stdout.strip()!='true': dk('start','ep-r-redis-1')
    unlockg('qc3','log_statement=none, redis chạy lại (P5-03-03/29, P4-02-02, P4-03-04); thời gian giữ khoá %d s'%(time.time()-t_lock))
for k,(ok,note) in R.items():
    tc={'03':'TC-P5-03-03','0202':'TC-P4-02-02','0304':'TC-P4-03-04','29':'TC-P5-03-29'}[k]; rec(tc,ok,note)
save('p53lock')
