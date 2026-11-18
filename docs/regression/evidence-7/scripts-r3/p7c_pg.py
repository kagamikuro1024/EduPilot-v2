from lib3 import *
import datetime as dt, re
fx=json.load(open('/tmp/qc3e/r4fx.json')); L=fx['L']
def iso(t): return dt.datetime.fromtimestamp(t,dt.timezone.utc).strftime('%Y-%m-%dT%H:%M:%SZ')
def entries(since):
    r=subprocess.run(f"docker logs ep-r-postgres-1 --since {since} 2>&1",shell=True,capture_output=True,text=True).stdout
    ents=[];cur=None
    for l in r.split('\n'):
        if re.match(r'^\d{4}-\d\d-\d\d \d\d:\d\d:\d\d',l):
            if cur: ents.append(cur)
            cur=l
        elif cur is not None: cur+=' '+l.strip()
    if cur: ents.append(cur)
    return ents
[login(x) for x in ('qc3t','qc3ta')]
nstu=psql(f"select count(*) from enrollments where role_in_course='STUDENT' and status='ACTIVE' and course_id='{L}'"); ntk=psql(f"select count(*) from escalation_tickets where course_id='{L}'")
print('students',nstu,'tickets',ntk)
assert lockg('qc3S7c','P4-02-02 + P4-03-04: log_statement=all <=3 phút')
t0=time.time()
try:
    psql("alter system set log_statement='all'"); psql("select pg_reload_conf()"); time.sleep(2)
    ts=time.time(); eid=mkexam3(L,'qc3t','qc3ta',open_s=25,win=1800,dur=20,nq=2,title='QC3 1k notify'); time.sleep(2)
    es=entries(iso(ts-1))
    ins=[e for e in es if re.search(r'insert into notifications',e,re.I) and eid in e]
    allins=[e for e in es if re.search(r'insert into notifications',e,re.I)]
    unn=[e for e in ins if 'unnest' in e.lower()]
    nn=psql(f"select count(*) from notifications where dedupe_key like 'exam.scheduled:{eid}:%'")
    import itertools
    arr=[len(re.findall(r'[0-9a-f]{8}-[0-9a-f]{4}-',e.split('parameters:')[-1])) for e in unn]
    print('P4-02-02: ins(exam)=',len(ins),'unnest=',len(unn),'all notif inserts in window=',len(allins),'rows=',nn,'uuid-ish tokens in params=',arr)
    print('sample:',re.sub(r'\s+',' ',unn[0])[:420] if unn else None)
    open('/tmp/qc3e/p4-02-02.log','w').write('\n\n'.join(re.sub(r'\s+',' ',e)[:1200] for e in ins)+'\n')
    # 03-04
    ts2=time.time(); st,d=A('qc3t','GET',f'/courses/{L}/tickets?limit=100&status=OPEN'); el=time.time()-ts2; time.sleep(1.5)
    es2=entries(iso(ts2-0.5))
    q=[e for e in es2 if re.search(r'escalation_tickets',e)]
    off=[e for e in es2 if re.search(r'\boffset\b',e,re.I) and 'escalation_tickets' in e]
    n=len(d.get('items',[])) if isinstance(d,dict) else -1
    print('P4-03-04:',st,n,'ms',round(el*1000),'stmts touching escalation_tickets=',len(q),'offset=',len(off),'tickets',ntk)
    open('/tmp/qc3e/p4-03-04.log','w').write('\n\n'.join(re.sub(r'\s+',' ',e)[:900] for e in es2 if re.search(r'(statement|execute)',e))[:20000]+'\n')
    print('all statements in window:',len([e for e in es2 if re.search(r'(statement|execute)',e)]))
finally:
    psql("alter system set log_statement='none'"); psql("select pg_reload_conf()")
    unlockg('qc3S7c','log_statement=none (P4-02-02/P4-03-04); %d s'%(time.time()-t0))
