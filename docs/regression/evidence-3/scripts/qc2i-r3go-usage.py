# TC-P104-30/31/32 trên gateway thử qc2i-r3go-gw (:9761, DB tạm qc2i_r3go, redis riêng) — số tay tính từ chính các dòng llm_audit đã gieo
import sys,os,json,subprocess,time,statistics,collections
sys.path.insert(0,os.path.dirname(__file__))
os.environ['SP']='qc2i-r3go-usage'
import lib
from lib import call,psql
G='http://localhost:9761/api/v1'
def sq(q):
    r=subprocess.run(['docker','exec','-i','ep-r-postgres-1','psql','-U','edupilot','-d','qc2i_r3go','-At','-F','|','-v','ON_ERROR_STOP=1'],input=q,capture_output=True,text=True); return (r.stdout+r.stderr).strip()
def tok(u):
    s,j,_=call('POST','/auth/login',body={'email':u+'@edupilot.local','password':lib.PW},base=G); assert s==200,(s,j); return j['access_token']
A=tok('admin'); T=tok('teacher'); S=tok('sv11')
sq("truncate llm_audit")
sq("""insert into llm_audit(task,lane,provider,model,tokens_in,tokens_out,latency_ms,status,degraded,trace_id,course_id,created_at,cost_est)
select (array['CHAT','CLASSIFY','UTILITY','GRADING','QUESTION_GEN','INSIGHT'])[1+i%6],'INTERACTIVE','p','m',100+(i*7)%53,50+(i*3)%31,10+(i*37)%900,
 (array['ok','ok','ok','error','timeout','rate_limited','cancelled'])[1+i%7], i%7=0,'t'||i,
 case when i%4=0 then null else '00000000-0000-7000-8000-00000000000'||(i%3) end::uuid,
 date_trunc('day',now())-((i%5)||' days')::interval+((i%1400)||' minutes')::interval*0.5, ((i*13)%977)*0.0123+0.0001*(i%9)
from generate_series(1,1000) i""")
rows=[l.split('|') for l in sq("select task,tokens_in,tokens_out,latency_ms,status,degraded,cost_est,to_char(created_at at time zone 'Asia/Ho_Chi_Minh','YYYY-MM-DD') from llm_audit").split('\n')]
from decimal import Decimal
def pc(v,p):
    v=sorted(v); k=(len(v)-1)*p; f=int(k); c=min(f+1,len(v)-1); return v[f]+(v[c]-v[f])*(k-f)
ERR={'error','timeout','rate_limited','circuit_open','not_configured'}
def expect(keyf):
    g=collections.defaultdict(list)
    for r in rows: g[keyf(r)].append(r)
    out={}
    for k,rs in g.items():
        lat=[int(r[3]) for r in rs]
        out[k]=dict(calls=len(rs),tokens_in=sum(int(r[1]) for r in rs),tokens_out=sum(int(r[2]) for r in rs),cost_est=str(sum(Decimal(r[6]).quantize(Decimal('0.0001')) for r in rs).quantize(Decimal('0.0001'))),
          latency_p50_ms=round(pc(lat,.5)),latency_p95_ms=round(pc(lat,.95)),errors=sum(r[4] in ERR for r in rs),degraded=sum(r[5]=='t' for r in rs))
    return out
import datetime
FR=(datetime.date.today()-datetime.timedelta(days=10)).isoformat(); TO=(datetime.date.today()+datetime.timedelta(days=2)).isoformat()
bad=[]
for grp,kf in (('task',lambda r:r[0]),('day',lambda r:r[7])):
    t0=time.time(); s,j,_=call('GET',f'/admin/llm/usage?group={grp}&from={FR}&to={TO}',A,base=G); 
    print(grp,s,round((time.time()-t0)*1000),'ms',json.dumps(j)[:200])
    if s!=200: bad.append((grp,s,j)); continue
    e=expect(kf); got={i['key']:i for i in j['items']}
    for k,v in e.items():
        g=got.get(k)
        if not g: bad.append((grp,k,'thiếu')); continue
        for f,x in v.items():
            gv=g[f]
            if f=='cost_est': ok=Decimal(gv)==Decimal(x)
            else: ok=gv==x
            if not ok: bad.append((grp,k,f,'API',gv,'tay',x))
    if set(got)-set(e): bad.append((grp,'thừa',set(got)-set(e)))
print('P104-30 lệch:',bad,'| số key task',len(expect(lambda r:r[0])),'day',len(expect(lambda r:r[7])))
print('P104-30','P' if not bad else 'F')
