import sys,json,time,subprocess,os,datetime
sys.path.insert(0,'/tmp/qc3oc')
from oc import *
EV='/Users/kuro/Documents/TA_Agent_qc3/docs/regression/evidence-oc'
LOG=open(f'{EV}/routes-run.log','a')
def say(*a):
    l=' '.join(str(x) for x in a); print(l); LOG.write(l+'\n'); LOG.flush()
def started(c): return subprocess.run(['docker','inspect',c,'--format','{{.State.StartedAt}}'],capture_output=True,text=True).stdout.strip()
def routes(): return A('admin','GET','/admin/llm/routes')[1]
def provs(): return {p['name']:p for p in A('admin','GET','/admin/llm/providers')[1]['items']}
def mid(pname,model):
    for m in provs()[pname]['models']:
        if m['model']==model: return m['id']
def route_of(task): return next(r for r in routes()['items'] if r['task']==task)
def put(task,model_ids,version=None):
    r=route_of(task); v=r['version'] if version is None else version
    t0=time.time(); s,d=A('admin','PUT','/admin/llm/routes',{'task':task,'chain':model_ids,'version':v}); return s,d,t0
def call(task,course,prompt='Trả lời đúng một từ: chào'):
    t0=time.time(); s,d=A('admin','POST','/_test/llm/chat',{'task':task,'prompt':prompt,'course_id':course,'session_id':str(uuid.uuid4())},timeout=120); return s,d,round(time.time()-t0,1)
def audit(course,since):
    time.sleep(2.5)
    return psql(f"select task,provider,model,status,coalesce(error_kind,''),fallback_index,latency_ms from llm_audit where course_id='{course}' and created_at>=to_timestamp({since}) order by created_at")
