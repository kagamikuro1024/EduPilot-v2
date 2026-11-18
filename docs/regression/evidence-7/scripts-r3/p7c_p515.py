import subprocess,time,json,datetime as dt,random
def sh(*a,inp=None): return subprocess.run(a,capture_output=True,text=True,input=inp).stdout.strip()
def pg(sql): return sh('docker','exec','ep-r-postgres-1','psql','-U','edupilot','-d','qc3seed','-At','-c',sql)
def rd(*a,inp=None): return sh('docker','exec','-i','ep-r-redis-1','redis-cli','-n','13',*a,inp=inp)
C=pg("select id from courses where class_code='761987'"); U=pg(f"select user_id from enrollments where course_id='{C}' and role_in_course='STUDENT' and status='ACTIVE'").split('\n')
today=dt.datetime.now(dt.timezone(dt.timedelta(hours=7))).date()
AREAS=['CHAT','THREADS','PRACTICE','LIBRARY','CALENDAR']
keys=[]
for u in U:
    for d in range(0,140):
        for a in AREAS[:3]: keys.append(f"ep:study:{C}:{u}:{(today-dt.timedelta(days=d)).strftime('%Y%m%d')}:{a}")
N=len(keys); print('students',len(U),'keys',N)
def load(rnd):
    cmds=[]
    for i,k in enumerate(keys):
        v=30*(1+((i*7)%40))+30*rnd; cmds.append(f"SET {k} {v}"); cmds.append(f"SADD ep:study:dirty {k}")
    rd(inp='\n'.join(cmds)+'\n')
def sumdb(): return int(pg(f"select coalesce(sum(active_seconds),0) from study_time_daily where course_id='{C}'") or 0)
def expect_sum(rnd): return sum(30*(1+((i*7)%40))+30*rnd for i in range(N))
def killed(prev,new):
    t0=time.time()
    while time.time()-t0<45:
        cur=sumdb()
        if cur!=prev and cur!=new:
            sh('docker','kill','-s','KILL','ep-r-worker-s7seed'); return cur
        time.sleep(0.03)
    return None
res=[]
for rnd in range(3):
    pre=sumdb(); load(rnd); n=killed(pre,expect_sum(rnd)); st=sh('docker','inspect','ep-r-worker-s7seed','--format','{{.State.Status}}/{{.State.ExitCode}}')
    print(f'rnd{rnd}: tổng giây DB trước={pre}, kỳ vọng sau={expect_sum(rnd)}, lúc kill={n} (giữa chừng={n is not None}), container {st}')
    sh('docker','start','ep-r-worker-s7seed'); time.sleep(1)
    res.append((rnd,pre,n,st))
    if rnd<2: time.sleep(30)
print('đợi 2 phút sau lượt cuối…'); time.sleep(120)
bad=neg=0; tot=0; dbn=int(pg(f"select count(*) from study_time_daily where course_id='{C}'"))
q=pg(f"select student_id::text,to_char(date,'YYYYMMDD'),area,active_seconds from study_time_daily where course_id='{C}'")
db={}
for r in q.split('\n'):
    s,d,a,v=r.split('|'); db[f"ep:study:{C}:{s}:{d}:{a}"]=int(v)
for k in keys:
    v=int(rd('GET',k) or 0); x=db.get(k)
    if x is None or x!=v: bad+=1
    if x is not None and x<0: neg+=1
print(f'KẾT QUẢ: khoá Redis {N}, hàng DB {dbn}, lệch giá trị {bad}, âm {neg}, DB>Redis (đôi) {sum(1 for k in keys if db.get(k,0)>int(rd("GET",k) or 0))}')
json.dump({'N':N,'db':dbn,'bad':bad,'neg':neg,'rounds':res},open('/tmp/qc3e/p515.json','w'))
