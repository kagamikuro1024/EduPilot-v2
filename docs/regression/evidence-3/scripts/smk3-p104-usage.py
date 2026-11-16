# TC-P104-30/31/32: gieo 20.000 dòng llm_audit (provider='smk3', tháng 2/2026, ngoài cửa sổ dùng của nhóm khác), so API /admin/llm/usage
# với số TỰ TÍNH bằng Python từ dữ liệu thô; đo thời gian phản hồi; dọn sau.
import os,sys,time,statistics,datetime as dt,json,collections
sys.path.insert(0,os.path.dirname(__file__)); os.environ['SP']='x'
from lib import *
A=login('admin'); T=login('teacher')
N=20000
psql("delete from llm_audit where provider='smk3'")
psql(f"""insert into llm_audit(task,lane,provider,model,tokens_in,tokens_out,latency_ms,status,degraded,trace_id,course_id,created_at,cost_est)
select (array['CHAT','CLASSIFY','UTILITY','GRADING','QUESTION_GEN','INSIGHT'])[1+i%6],'INTERACTIVE','smk3','m',100+i%50,50+i%30,10+(i*37)%900,
 (array['ok','ok','ok','error','timeout','rate_limited'])[1+i%6], i%7=0,'smk3-'||i,
 case when i%4=0 then null else (select id from courses order by id limit 1) end,
 timestamptz '2026-02-01 00:00:00+00' + (i*127 % (27*86400))*interval '1 second', ((i%100)*0.0125)::numeric(14,4)
from generate_series(1,{N}) i""")
rows=[l.split('|') for l in psql("select task,tokens_in,tokens_out,latency_ms,status,degraded,cost_est,to_char(created_at at time zone 'Asia/Ho_Chi_Minh','YYYY-MM-DD') from llm_audit where provider='smk3'").split('\n')]
print('rows',len(rows))
def pct(v,p):
    v=sorted(v); k=(len(v)-1)*p; f=int(k); c=min(f+1,len(v)-1); return v[f]+(v[c]-v[f])*(k-f)
def agg(rs):
    lat=[int(r[3]) for r in rs]
    return dict(calls=len(rs),tokens_in=sum(int(r[1]) for r in rs),tokens_out=sum(int(r[2]) for r in rs),cost_est='%.4f'%sum(float(r[6]) for r in rs),
      latency_p50_ms=int(statistics.median(lat)+0.5) if False else int(round(pct(lat,.5))),latency_p95_ms=int(round(pct(lat,.95))),
      errors=sum(1 for r in rs if r[4] in('error','timeout','rate_limited','circuit_open','not_configured')),degraded=sum(1 for r in rs if r[5]=='t'))
bad=[]
for grp,ix in (('task',0),('day',7)):
    t0=time.time(); s,j,_=call('GET','/admin/llm/usage?group=%s&from=2026-02-01&to=2026-03-01'%grp,A); dt_=time.time()-t0
    print(grp,s,len(j['items']),'%.0f ms'%(dt_*1000))
    by=collections.defaultdict(list)
    for r in rows: by[r[ix]].append(r)
    got={x['key']:x for x in j['items']}
    for k,rs in by.items():
        if k not in got: bad.append((grp,k,'thiếu')); continue
        e=agg(rs)
        for f,v in e.items():
            gv=got[k][f]
            if f=='cost_est': ok=abs(float(gv)-float(v))<1e-9 and isinstance(gv,str)
            else: ok=gv==v
            if not ok: bad.append((grp,k,f,gv,v))
    if grp=='task' : T_MS=dt_
    else: D_MS=dt_
print('LỆCH',len(bad),bad[:5])
# 31
r={}
s,j,_=call('GET','/admin/llm/usage',A); fr=dt.datetime.fromisoformat(j['from'].replace('Z','+00:00')); to=dt.datetime.fromisoformat(j['to'].replace('Z','+00:00')); r['default7d']=(s,round((to-fr).total_seconds()/86400,2))
r['93d']=call('GET','/admin/llm/usage?from=2026-01-01&to=2026-04-05',A)[:2]
r['94d']=call('GET','/admin/llm/usage?from=2026-01-01&to=2026-04-05',A)[0]
r['exact93']=call('GET','/admin/llm/usage?from=2026-01-01&to=2026-04-04',A)[0]
r['from>to']=call('GET','/admin/llm/usage?from=2026-03-01&to=2026-02-01',A)[:2]
r['group=lạ']=call('GET','/admin/llm/usage?group=foo',A)[:2]
cid=psql("select id from courses order by id limit 1")
r['teacher+course']=call('GET','/admin/llm/usage?course_id='+cid,T)[:2]
r['teacher total']=call('GET','/admin/llm/usage',T)[0]
r['admin+course']=call('GET','/admin/llm/usage?course_id='+cid,A)[0]
for k,v in r.items(): print(k,str(v)[:160])
# 32 EXPLAIN + thời gian
import subprocess
print(psql("analyze llm_audit"))
ex=psql("explain (format json) select task as key,count(*) from llm_audit where created_at >= '2026-02-10' and created_at < '2026-02-12' group by task")
print('EXPLAIN','Seq Scan' in ex, 'llm_audit_created_idx' in ex, 'Index' in ex or 'Bitmap' in ex)
ts=[]
for _ in range(5):
    t0=time.time(); call('GET','/admin/llm/usage?group=task&from=2026-02-01&to=2026-03-01',A); ts.append((time.time()-t0)*1000)
print('ms task',['%.0f'%x for x in ts])
psql("delete from llm_audit where provider='smk3'")
print('cleanup',psql("select count(*) from llm_audit where provider='smk3'"))
json.dump(dict(bad=bad[:10],r={k:str(v) for k,v in r.items()},ms=ts,T_MS=T_MS,D_MS=D_MS,explain_seq='Seq Scan' in ex),open('/tmp/smk34/p104-usage.json','w'))
