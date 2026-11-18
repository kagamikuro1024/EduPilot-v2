import subprocess,datetime as dt,json
def sh(*a,inp=None): return subprocess.run(a,capture_output=True,text=True,input=inp).stdout
def pg(sql): return sh('docker','exec','ep-r-postgres-1','psql','-U','edupilot','-d','qc3seed','-At','-c',sql).strip()
C=pg("select id from courses where class_code='761987'")
db={}
for r in pg(f"select 'ep:study:{C}:'||student_id||':'||to_char(date,'YYYYMMDD')||':'||area,active_seconds from study_time_daily where course_id='{C}'").split('\n'):
    k,v=r.split('|'); db[k]=int(v)
keys=sh('docker','exec','ep-r-redis-1','redis-cli','-n','13','--scan','--pattern',f'ep:study:{C}:*').split()
rv={}
for i in range(0,len(keys),1000):
    ch=keys[i:i+1000]; out=sh('docker','exec','-i','ep-r-redis-1','redis-cli','-n','13',inp='MGET '+' '.join(ch)+'\n').split('\n')
    for k,v in zip(ch,out): rv[k]=int(v)
bad=[k for k in rv if db.get(k)!=rv[k]]; neg=[k for k in db if db[k]<0]; dbl=[k for k in db if db[k]>rv.get(k,0)]
print(f'khoá Redis {len(rv)}, hàng DB {len(db)}, lệch {len(bad)}, âm {len(neg)}, DB>Redis {len(dbl)}, tổng DB {sum(db.values())}, tổng Redis {sum(rv.values())}')
