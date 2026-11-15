# TC-P104-31 (khoảng/nhóm/phân quyền) + TC-P104-32 (20.000 dòng: thời gian phản hồi) + TC-P104-44 (số truy vấn GET providers) — gateway thử :9761
import sys,os,json,subprocess,time,datetime
sys.path.insert(0,os.path.dirname(__file__)); os.makedirs('/tmp/qc2-b',exist_ok=True)
from lib import call,PW
G='http://localhost:9761/api/v1'
def sq(q,db='qc2i_r3go'):
    r=subprocess.run(['docker','exec','-i','ep-r-postgres-1','psql','-U','edupilot','-d',db,'-At','-F','|','-v','ON_ERROR_STOP=1'],input=q,capture_output=True,text=True); return (r.stdout+r.stderr).strip()
def tok(u):
    s,j,_=call('POST','/auth/login',body={'email':u+'@edupilot.local','password':PW},base=G); assert s==200,(s,j); return j['access_token']
A=tok('admin'); T=tok('teacher')
d=datetime.date.today
def U(q,t=A): return call('GET','/admin/llm/usage'+q,t,base=G)
res={}
s,j,_=U(''); fr=datetime.datetime.fromisoformat(j['from'].replace('Z','+00:00')); to=datetime.datetime.fromisoformat(j['to'].replace('Z','+00:00'))
res['mặc định']=(s,(to-fr).days,j['group']); 
ok31=[s==200 and 6<=(to-fr).days<=8 and j['group']=='task']
c=[]
for name,q,t,want in [('92 ngày',f'?from={d()-datetime.timedelta(days=92)}&to={d()}',A,200),('93 ngày',f'?from={d()-datetime.timedelta(days=93)}&to={d()}',A,422),
  ('from>to',f'?from={d()}&to={d()-datetime.timedelta(days=3)}',A,422),('group lạ','?group=hour',A,422),
  ('TEACHER course_id','?course_id=00000000-0000-7000-8000-000000000001',T,403),('TEACHER tổng hệ thống','',T,200),('group=day mặc định','?group=day',A,200)]:
    s,j,_=U(q,t); c.append((name,s,want)); ok31.append(s==want)
print('P104-31',res,c); print('P104-31','P' if all(ok31) else 'F')
# 32: 20.000 dòng
sq("""truncate llm_audit; insert into llm_audit(task,lane,provider,model,tokens_in,tokens_out,latency_ms,status,degraded,trace_id,course_id,created_at,cost_est)
select (array['CHAT','CLASSIFY','UTILITY','GRADING','QUESTION_GEN','INSIGHT'])[1+i%6],'INTERACTIVE','p','m',100+i%50,50+i%30,10+i%900,(array['ok','ok','error'])[1+i%3],i%7=0,'t'||i,
 case when i%4=0 then null else ('00000000-0000-7000-8000-00000000000'||(i%3))::uuid end, now()-((i%90)||' days')::interval-((i%1440)||' minutes')::interval,(i%17)*0.0123 from generate_series(1,20000) i; analyze llm_audit""")
print('rows',sq('select count(*) from llm_audit'))
ms=[];out={}
for q in ['','?group=day','?from=%s&to=%s'%(d()-datetime.timedelta(days=90),d()+datetime.timedelta(days=1)),'?group=day&from=%s&to=%s'%(d()-datetime.timedelta(days=90),d()+datetime.timedelta(days=1)),'?course_id=00000000-0000-7000-8000-000000000001']:
    t=[]
    for _ in range(10):
        t0=time.time(); s,j,_=U(q); t.append((time.time()-t0)*1000); assert s==200,(s,j)
    t.sort(); out[q or 'mặc định']=(round(t[4]),round(t[-1])); ms.append(t[-1])
print('P104-32 p50/max ms',out); print('P104-32','P' if max(ms)<=300 else 'F')
# 44: số truy vấn GET providers với 20 nhà × 10 mô hình
sq("delete from llm_providers where name like 'qc2i-n1-%'")
sq("""insert into llm_providers(type,name) select 'fake','qc2i-n1-'||i from generate_series(1,20) i;
insert into llm_models(provider_id,model,kind,dims) select p.id,'m'||j,'chat',null from llm_providers p, generate_series(1,10) j where p.name like 'qc2i-n1-%'""")
sq("alter database qc2i_r3go set log_statement='all'"); sq("select pg_terminate_backend(pid) from pg_stat_activity where datname='qc2i_r3go' and pid<>pg_backend_pid()")
time.sleep(3); call('GET','/healthz',base=G)
A=tok('admin'); time.sleep(2)
t0=datetime.datetime.now(datetime.timezone.utc)
s,j,_=call('GET','/admin/llm/providers',A,base=G); n=len(j.get('items',[])); nm=sum(len(i['models']) for i in j['items']); t1=datetime.datetime.now(datetime.timezone.utc); time.sleep(1)
log=subprocess.run(['docker','logs','--since',(t0-datetime.timedelta(seconds=1)).strftime('%Y-%m-%dT%H:%M:%SZ'),'ep-r-postgres-1'],capture_output=True,text=True); L=log.stdout+log.stderr
import re
ents=re.split(r'\n(?=\d{4}-\d\d-\d\d \d\d:\d\d:\d\d\.\d+ UTC)',L)
q=[]
for e in ents:
    m=re.match(r'(\S+ \S+) UTC \[\d+\] LOG:  (?:execute [^:]*|statement): (.*)',e,re.S)
    if not m: continue
    ts=datetime.datetime.strptime(m.group(1)[:26],'%Y-%m-%d %H:%M:%S.%f').replace(tzinfo=datetime.timezone.utc)
    if t0<=ts<=t1+datetime.timedelta(milliseconds=50): q.append(m.group(2).strip().replace('\n',' ')[:110])
print('providers',n,'models',nm,'| truy vấn trong yêu cầu GET providers:',len(q)); [print('  ',x) for x in q]
print('P104-44','P' if s==200 and n>=20 and len(q)<=3 else 'F')
sq("alter database qc2i_r3go reset log_statement"); sq("delete from llm_providers where name like 'qc2i-n1-%'")
