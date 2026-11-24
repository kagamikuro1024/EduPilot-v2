import sys,json,hashlib,time,copy,urllib.request,ssl
sys.path.insert(0,'/tmp/qc3oc')
from oc import *
FX="/Users/kuro/Documents/TA_Agent_qc2/docs/specs/FEAT-assignments/fixtures"
base=json.load(open(f"{FX}/create-bt03.json")); print({k:(str(v)[:60]) for k,v in base.items()})
c=ensure_course(); Q=f"/courses/{c}"
b=copy.deepcopy(base); b.update({"submission_code":"OC01","title":"QCOC bài thử OpenCode Go","due_at":time.strftime("%Y-%m-%dT%H:%M:%SZ",time.gmtime(time.time()+3*86400))})
s,a=A('teacher','POST',f"{Q}/assignments",b); print('create',s,str(a)[:200])
