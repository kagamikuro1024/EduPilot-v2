import json,ssl,urllib.request,uuid,subprocess,collections
B='https://localhost:943/api/v1'; C='01a20df5-5f4d-759d-9083-23a0e184130f'  # QCOC1
ctx=ssl._create_unverified_context()
def call(m,p,t,body=None):
    h={'authorization':'Bearer '+t,'content-type':'application/json','idempotency-key':str(uuid.uuid4())}
    r=urllib.request.Request(B+p,method=m,headers=h,data=(json.dumps(body or {}).encode() if m in('POST','PUT','PATCH') else None))
    try:
        with urllib.request.urlopen(r,context=ctx,timeout=30) as x: return x.status,x.read()[:200]
    except urllib.error.HTTPError as e: return e.code,e.read()[:200]
def tok(u):
    r=urllib.request.Request(B+'/auth/login',method='POST',headers={'content-type':'application/json'},data=json.dumps({'email':u+'@edupilot.local','password':'Edupilot#Seed-2026'}).encode())
    return json.load(urllib.request.urlopen(r,context=ctx))['access_token']
def psql(q): return subprocess.run(['docker','exec','ep-qc-postgres-1','psql','-U','edupilot','-d','edupilot','-At','-c',q],capture_output=True,text=True).stdout.strip()
routes=[l.rstrip('\n').split('\t') for l in open('/tmp/qc3lt/exam-routes.tsv')]
res={}
for u in ('sv.nguyco','sv04','qct2','admin'):
    inc=psql(f"select count(*) from enrollments e join users x on x.id=e.user_id where x.email='{u}@edupilot.local' and e.course_id='{C}'")
    t=tok(u); cnt=collections.Counter(); bad=[]
    for no,m,p,mode in routes:
        if no=='43': continue
        path='/courses/'+C+p
        for k in ('qid','eid','aid','sid','pid','tid','cid','iid','rid'): path=path.replace('{'+k+'}',str(uuid.uuid4()))
        s,b=call(m,path,t); cnt[s]+=1
        if s<300: bad.append((no,m,p,s))
    res[u]={'in_class':inc,'status_hundreds':dict(cnt),'2xx':bad}
    print(u,'enrolled_in_QCOC1=',inc,dict(cnt),'2xx:',bad)
json.dump(res,open('/tmp/qc3lt/outsider.json','w'),ensure_ascii=False)
