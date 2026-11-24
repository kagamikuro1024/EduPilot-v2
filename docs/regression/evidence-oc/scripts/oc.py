import json,ssl,subprocess,urllib.request,uuid,time,re,os,sys
B='https://localhost:943/api/v1'; PW='Edupilot#Seed-2026'; ctx=ssl._create_unverified_context(); _tok={}
def raw(method,path,token=None,body=None,idem=None,timeout=90):
    h={'content-type':'application/json'}
    if token: h['authorization']='Bearer '+token
    if method=='POST': h['Idempotency-Key']=idem or str(uuid.uuid4())
    data=None if body is None else json.dumps(body).encode()
    req=urllib.request.Request(B+path,data=data,method=method,headers=h)
    try:
        r=urllib.request.urlopen(req,context=ctx,timeout=timeout); return r.status,r.read().decode()
    except urllib.error.HTTPError as e: return e.code,e.read().decode()
def j(method,path,token=None,body=None,**kw):
    s,t=raw(method,path,token,body,**kw)
    try: return s,json.loads(t) if t else None
    except Exception: return s,t
def login(u):
    if u in _tok: return _tok[u]
    s,d=j('POST','/auth/login',None,{'email':u+'@edupilot.local','password':PW}); _tok[u]=d['access_token']; return _tok[u]
def A(u,m,p,b=None,**kw): return j(m,p,login(u),b,**kw)
def psql(sql):
    r=subprocess.run(['docker','exec','-i','ep-qc-postgres-1','psql','-U','edupilot','-d','edupilot','-At','-F','|','-c',sql],capture_output=True,text=True); return (r.stdout+r.stderr).strip()
def ensure_course(code='QCOC1',jc='QKTWC37'):
    c=psql(f"select id from courses where class_code='{code}'")
    if c: return c
    tid=psql("select id from users where email='teacher@edupilot.local'"); taid=psql("select id from users where email='ta@edupilot.local'")
    s,d=A('admin','POST','/admin/courses',{'subject_code':'QCOC','class_code':code,'name':'QC3 OpenCode '+code,'semester':'2026-2027-HK1','capacity':50,'teacher_id':tid,'ta_ids':[taid],'join_code':jc}); assert s in(200,201),(s,d)
    c=d['id']
    for u in ['sv.gioi','sv.kha']:
        psql(f"insert into enrollments (course_id,user_id,role_in_course,status,joined_via,student_code_snapshot) select '{c}',u.id,'STUDENT','ACTIVE','ADMIN',(select student_code_snapshot from enrollments where user_id=u.id and student_code_snapshot is not null limit 1) from users u where u.email='{u}@edupilot.local' on conflict do nothing")
    return c
