import json,os,ssl,subprocess,urllib.request,uuid,time,re
B='https://localhost:1143/api/v1'
PW='Edupilot#Seed-2026'
ctx=ssl._create_unverified_context()
_tok={}
def raw(method,path,token=None,body=None,idem=None,headers=None,timeout=60,stream=False):
    h={'content-type':'application/json'}
    if token: h['authorization']='Bearer '+token
    if method in('POST',) : h['Idempotency-Key']=idem or str(uuid.uuid4())
    if headers: h.update(headers)
    data=None if body is None else (body if isinstance(body,(bytes,str)) else json.dumps(body)).encode() if not isinstance(body,bytes) else body
    req=urllib.request.Request(B+path if path.startswith('/') else path,data=data,method=method,headers=h)
    try:
        r=urllib.request.urlopen(req,context=ctx,timeout=timeout)
        return r.status,r.read().decode(),dict(r.headers)
    except urllib.error.HTTPError as e:
        return e.code,e.read().decode(),dict(e.headers)
def j(method,path,token=None,body=None,**kw):
    s,t,h=raw(method,path,token,body,**kw)
    try: return s,json.loads(t) if t else None
    except Exception: return s,t
def login(u):
    if u in _tok: return _tok[u]
    try:
        c=json.load(open('/tmp/qc3r7/tokcache.json'))
        if u in c and time.time()-c[u][1]<600: _tok[u]=c[u][0]; return _tok[u]
    except Exception: pass
    email=u if '@' in u else u+'@edupilot.local'
    for i in range(3):
        s,d=j('POST','/auth/login',None,{'email':email,'password':PW})
        if s==200:
            _tok[u]=d['access_token']
            try: c=json.load(open('/tmp/qc3r7/tokcache.json'))
            except Exception: c={}
            c[u]=[_tok[u],time.time()]; json.dump(c,open('/tmp/qc3r7/tokcache.json','w'))
            return _tok[u]
        if s==429: time.sleep(20)
        time.sleep(5)
    raise Exception(f'login {u} {s} {d}')
def A(u,method,path,body=None,**kw): return j(method,path,login(u),body,**kw)
def psql(sql):
    r=subprocess.run(['docker','exec','-i','ep-r-postgres-1','psql','-U','edupilot','-d','edupilot','-At','-F','|','-v','ON_ERROR_STOP=0','-c',sql],capture_output=True,text=True)
    return (r.stdout+r.stderr).strip()
def rds(*a):
    r=subprocess.run(['docker','exec','ep-r-redis-1','redis-cli',*a],capture_output=True,text=True); return r.stdout.strip()
def sse(method,path,token,body=None,idem=None,maxsec=90):
    """đọc SSE -> list (event,data)"""
    h={'content-type':'application/json','authorization':'Bearer '+token,'Accept':'text/event-stream'}
    if method=='POST': h['Idempotency-Key']=idem or str(uuid.uuid4())
    req=urllib.request.Request(B+path,data=None if body is None else json.dumps(body).encode(),method=method,headers=h)
    ev=[];t0=time.time()
    try:
        r=urllib.request.urlopen(req,context=ctx,timeout=maxsec)
    except urllib.error.HTTPError as e: return e.code,[('http',e.read().decode())]
    cur=None;st=r.status
    try:
        for ln in r:
            ln=ln.decode().rstrip('\n')
            if ln.startswith('event:'): cur=ln[6:].strip()
            elif ln.startswith('data:'): ev.append((cur,ln[5:].strip(),round(time.time()-t0,3)))
            if time.time()-t0>maxsec: break
    except Exception: pass
    return st,ev
R={}
def rec(tc,ok,note=''):
    R[tc]=('P' if ok else 'F',note); print(('PASS' if ok else 'FAIL'),tc,note[:140])
def save(name):
    json.dump(R,open(f'/tmp/qc3r7/res-{name}.json','w'),ensure_ascii=False,indent=0)
def sqlx(sql):
    """chạy trong giao dịch rồi ROLLBACK; trả (SQLSTATE hoặc 'OK', thông điệp, stdout)"""
    r=subprocess.run(['docker','exec','-i','ep-r-postgres-1','psql','-U','edupilot','-d','edupilot','-At','-F','|','-v','VERBOSITY=verbose','-v','ON_ERROR_STOP=1','-c','begin;\n'+sql+'\nrollback;'],capture_output=True,text=True)
    m=re.search(r'ERROR:\s+(\w{5}):',r.stderr)
    return (m.group(1) if m else 'OK'),r.stderr.strip()[:200],r.stdout.strip()
def rec(tc,ok,note='',status=None):
    st=status or ('P' if ok else 'F')
    R[tc]=(st,note); print({'P':'PASS','F':'FAIL','K':'K'}[st],tc,note[:140])
    spr='7'
    f=f'/tmp/qc3r7/master{spr}.json'
    try: m=json.load(open(f))
    except Exception: m={}
    m[tc]=[st,note]; json.dump(m,open(f,'w'),ensure_ascii=False)
def mkusers(prefix,role,n,start=1):
    out=[]
    for i in range(start,start+n):
        em=f'{prefix}{i:02d}@edupilot.local' if n>1 or prefix.endswith('s') else f'{prefix}@edupilot.local'
        psql(f"insert into users(email,password_hash,full_name,role,status,email_verified_at,student_code) select '{em}',password_hash,'QC3 {prefix}{i:02d}','{role}','ACTIVE',now(),{'null' if role!='STUDENT' else repr('3'+str(i).zfill(7))} from users where email='sv08@edupilot.local' on conflict do nothing")
        out.append(em.split('@')[0])
    return out
def mkcourse(code,teacher,tas,jc):
    ex=psql(f"select id from courses where class_code='{code}'")
    if ex: return ex
    tid=psql(f"select id from users where email='{teacher}@edupilot.local'"); taids=[psql(f"select id from users where email='{t}@edupilot.local'") for t in tas]
    s,d=A('admin','POST','/admin/courses',{'subject_code':'QC3','class_code':code,'name':'QC3 '+code,'semester':'2026-2027-HK1','capacity':50,'teacher_id':tid,'ta_ids':taids,'join_code':jc})
    assert s in(200,201),(s,d); return d['id']
def uid(u): return psql(f"select id from users where email='{u}@edupilot.local'")
import threading
C7=json.load(open('/tmp/qc3r7/c7.json')) if os.path.exists('/tmp/qc3r7/c7.json') else {}
def sq(s): return s.replace("'","''")
def mkchat(u,course,q='Câu hỏi thử QC3',a='Trả lời AI thử QC3',intent='COURSE_QA',stream='DONE',lowc=False,conf=0.9,session=None):
    """tạo phiên bằng API, chèn USER + ASSISTANT bằng SQL; trả (sid, uid_msg, am)"""
    if session: sid=session
    else:
        s,d=A(u,'POST','/chat/sessions',{'course_id':course,'title':'qc3-s7'}); assert s==201,(s,d); sid=d['id']
    ui=uid(u)
    um=psql(f"insert into chat_messages(course_id,session_id,user_id,role,content,stream_status,client_msg_id,completed_at) values('{course}','{sid}','{ui}','USER','{sq(q)}','DONE',gen_random_uuid(),now()) returning id").split('\n')[0]
    am=psql(f"insert into chat_messages(course_id,session_id,user_id,role,content,stream_status,reply_to,intent,confidence,low_confidence,completed_at) values('{course}','{sid}','{ui}','ASSISTANT','{sq(a)}','{stream}','{um}','{intent}',{conf},{'true' if lowc else 'false'},{'null' if stream=='STREAMING' else 'now()'}) returning id").split('\n')[0]
    return sid,um,am
def esc(u,am,idem=None,**kw): return A(u,'POST',f'/chat/messages/{am}/escalate',idem=idem,**kw)
def tk(am): return psql(f"select id||'/'||status||'/'||origin from escalation_tickets where message_id='{am}'")
def nt(am): return psql(f"select count(*) from escalation_tickets where message_id='{am}'")
def course(code): return psql(f"select id from courses where class_code='{code}'")
def lockg(who,why):
    import time
    for i in range(80):
        r=subprocess.run(['mkdir','/tmp/qc2-lock-global'],capture_output=True)
        if r.returncode==0:
            open('/tmp/qc2-global-changes.log','a').write(time.strftime('%H:%M:%S')+f' {who}: LOCK — {why}\n'); return True
        time.sleep(15)
    return False
def unlockg(who,why):
    open('/tmp/qc2-global-changes.log','a').write(time.strftime('%H:%M:%S')+f' {who}: UNLOCK — {why}\n'); subprocess.run(['rmdir','/tmp/qc2-lock-global'])
def getroute(task='CHAT'):
    s,d=A('admin','GET','/admin/llm/routes')
    return [r for r in d['items'] if r['task']==task][0]
FAKE_MODEL='01a1df2d-a8ee-70d6-91cd-be284dbb4097'
def setroute(task,chain,ver):
    return A('admin','PUT','/admin/llm/routes',{'task':task,'chain':chain,'version':ver})
def fakeroute(who,why):
    lockg(who,why); r=getroute('CHAT'); orig=[c['model_id'] for c in r['chain']]
    s,d=setroute('CHAT',[FAKE_MODEL],r['version']); assert s==200,(s,d); return orig
def restoreroute(who,orig,why):
    r=getroute('CHAT'); s,d=setroute('CHAT',orig,r['version']); unlockg(who,why+f' — CHAT hoàn lại ({s})'); return s
def fakeset(**kw): return A('admin','POST','/_test/llm/fake',kw)
def chat(u,course,text,maxsec=60,sid=None):
    if not sid:
        s,d=A(u,'POST','/chat/sessions',{'course_id':course,'title':'qc3-s7'}); sid=d['id']
    st,ev=sse('POST',f'/chat/sessions/{sid}/messages',login(u),{'content':text},maxsec=maxsec)
    done=[json.loads(x[1]) for x in ev if x[0]=='done']
    am=psql(f"select id from chat_messages where session_id='{sid}' and role='ASSISTANT' order by created_at desc limit 1")
    return sid,am,st,ev,(done[0] if done else None)
import http.client,socket
class SSEConn:
    """kết nối /events trong luồng riêng; frames = [(t_local, id, event, data)]"""
    def __init__(self,u,last=None,path='/events',retry429=True):
        for _try in range(20):
            if self._open(u,last,path): break
            if self.status!=429 or not retry429: break
            time.sleep(1)
    def _open(self,u,last,path):
        self.u=u; self.frames=[]; self.status=None; self.body=''; self.stop=False
        h={'Authorization':'Bearer '+login(u),'Accept':'text/event-stream'}
        if last is not None: h['Last-Event-ID']=last
        self.conn=http.client.HTTPSConnection('localhost',1143,context=ctx,timeout=130)
        self.conn.request('GET','/api/v1'+path,headers=h); self.r=self.conn.getresponse(); self.status=self.r.status
        if self.status!=200: self.body=self.r.read().decode(); return False
        self.th=threading.Thread(target=self._run,daemon=True); self.th.start(); return True
    def _run(self):
        cur={}
        try:
            while not self.stop:
                ln=self.r.readline()
                if not ln: break
                ln=ln.decode().rstrip('\r\n')
                if ln=='':
                    if cur: self.frames.append((time.time(),cur.get('id'),cur.get('event'),cur.get('data'))); cur={}
                elif ln.startswith(':'): pass
                else:
                    k,_,v=ln.partition(':'); cur[k]=v.strip()
        except Exception: pass
    def close(self):
        self.stop=True
        try: self.conn.close()
        except Exception: pass
    def notif(self): return [f for f in self.frames if f[2]=='notification']
    def wait(self,pred,timeout=10):
        t0=time.time()
        while time.time()-t0<timeout:
            if pred(self.frames): return True
            time.sleep(0.1)
        return False
def nrows(u,where=''): return psql(f"select count(*) from notifications where user_id='{uid(u)}' {where}")
def raw2(method,path,token,body=None,idem=None):
    h={'content-type':'application/json'}
    if token: h['authorization']='Bearer '+token
    if idem: h['Idempotency-Key']=idem
    req=urllib.request.Request(B+path,data=None if body is None else json.dumps(body).encode(),method=method,headers=h)
    try:
        r=urllib.request.urlopen(req,context=ctx,timeout=60); t=r.read().decode(); return r.status,(json.loads(t) if t else None)
    except urllib.error.HTTPError as e:
        t=e.read().decode()
        try: return e.code,json.loads(t)
        except Exception: return e.code,t
def pglog(on):
    psql(f"alter system set log_statement='{'all' if on else 'none'}'"); psql("select pg_reload_conf()")
def pgfind(since,pat):
    out=subprocess.run(f"docker logs ep-r-postgres-1 --since {since} 2>&1",shell=True,capture_output=True,text=True).stdout
    return [l for l in out.split('\n') if re.search(pat,l)]
def clearopen(u,course):
    Qc=f'/courses/{course}'
    for r in psql(f"select id from escalation_tickets where course_id='{course}' and status in ('OPEN','CLAIMED')").split('\n'):
        if r:
            A(u,'POST',f'{Qc}/tickets/{r}/claim'); A(u,'POST',f'{Qc}/tickets/{r}/dismiss',{'reason':'OUT_OF_SCOPE'})
def tick(u,course=None,**kw):
    course=course or C7['A']; _,_,am=mkchat(u,course,**kw); s,d=esc(u,am); assert s==201,(s,d); return d['ticket']['id'],am
def runjob(name,now=None):
    cmd=['docker','exec','-e','APP_ENV=dev','ep-r-gateway-1','/gateway','admin','run-job',name]+(['--now',now] if now else [])
    r=subprocess.run(cmd,capture_output=True,text=True); return (r.stdout+r.stderr).strip()[-300:]
def seedage(course_code,tid,hours): 
    r=subprocess.run(['docker','exec','-e','APP_ENV=dev','ep-r-gateway-1','/gateway','admin','seed-age','--course',course_code,'--tickets','--ticket',tid,'--hours',str(hours)],capture_output=True,text=True); return (r.stdout+r.stderr).strip()[-200:]
def tstate(tid): return psql(f"select status||'/'||coalesce(closed_reason,'-')||'/stage='||sla_stage||'/ws='||to_char(waiting_since at time zone 'Asia/Ho_Chi_Minh','MM-DD HH24:MI') from escalation_tickets where id='{tid}'")
def mp(path):
    return json.loads(urllib.request.urlopen('http://localhost:8725'+path,timeout=20).read())
def mpq(q,limit=50): return mp('/api/v1/search?query='+urllib.parse.quote(q)+f'&limit={limit}')['messages']
import urllib.parse
import datetime as dt
def iso(**k): return (dt.datetime.now(dt.timezone.utc)+dt.timedelta(**k)).strftime('%Y-%m-%dT%H:%M:%SZ')
def mkexam3(course,teacher='qc3t',ta='qc3ta',open_s=8,win=1800,dur=20,nq=2,title='QC3 bài'):
    Qb=f'/courses/{course}/questions'; Eb=f'/courses/{course}/exams'; ids=[]
    for i in range(nq):
        s,q=A(teacher,'POST',Qb,{'type':'MCQ_SINGLE','title':f'qc3-{i}','topic':'QC3','stem':f'Câu {i}?','options':[{'body':f'Đáp án {k}'} for k in range(4)],'correct':[1],'explanation':'gt'}); assert s in(200,201),(s,q); qid=q['id']
        g=A(teacher,'GET',f'{Qb}/{qid}')[1]; A(teacher,'PUT',f'{Qb}/{qid}/review',{'decision':'REQUEST','version':g['version']}); g=A(ta,'GET',f'{Qb}/{qid}')[1]; s,_=A(ta,'PUT',f'{Qb}/{qid}/review',{'decision':'APPROVE','version':g['version']}); ids.append(qid)
    s,j=A(teacher,'POST',Eb,{'title':title,'opens_at':iso(seconds=open_s),'closes_at':iso(seconds=open_s+win),'duration_minutes':dur}); assert s==201,(s,j); eid=j['id']
    g=A(teacher,'GET',f'{Eb}/{eid}')[1]; s,_=A(teacher,'PUT',f'{Eb}/{eid}/items',{'items':[{'question_id':q,'points':'1'} for q in ids],'version':g['version']}); assert s==200,(s,_)
    g=A(teacher,'GET',f'{Eb}/{eid}')[1]; s,_=A(teacher,'PUT',f'{Eb}/{eid}',{'title':title,'opens_at':iso(seconds=open_s),'closes_at':iso(seconds=open_s+win),'duration_minutes':dur,'version':g['version']}); assert s==200,(s,_)
    s,_=A(teacher,'POST',f'{Eb}/{eid}/schedule',{}); assert s==200,(s,_)
    time.sleep(open_s+2); return eid
def startexam(course,eid,u):
    Eb=f'/courses/{course}/exams'; s,j=A(u,'POST',f'{Eb}/{eid}/attempts',{}); 
    if s not in(200,201): return None,s,j
    aid=j['attempt']['id']; H={'X-Exam-Tab':str(uuid.uuid4())}; A(u,'POST',f'{Eb}/{eid}/attempts/{aid}/takeover',{},headers=H); return aid,s,j
def submitexam(course,eid,aid,u):
    Eb=f'/courses/{course}/exams'; H={'X-Exam-Tab':str(uuid.uuid4())}; A(u,'POST',f'{Eb}/{eid}/attempts/{aid}/takeover',{},headers=H); return A(u,'POST',f'{Eb}/{eid}/attempts/{aid}/submit',{},headers=H)
class SSEConn2(SSEConn):
    def _open(self,u,last,path):
        self.u=u; self.frames=[]; self.status=None; self.body=''; self.stop=False
        h={'Authorization':'Bearer '+login(u),'Accept':'text/event-stream'}
        if last is not None: h['Last-Event-ID']=last
        self.conn=http.client.HTTPConnection('localhost',self.port,timeout=130)
        self.conn.request('GET','/api/v1'+path,headers=h); self.r=self.conn.getresponse(); self.status=self.r.status
        if self.status!=200: self.body=self.r.read().decode(); return False
        self.th=threading.Thread(target=self._run,daemon=True); self.th.start(); return True
    def __init__(self,u,port=18090,last=None,path='/events'):
        self.port=port; self.u=u
        self._open(u,last,path)
def rawB(method,path,token,body=None,port=18090,idem=None):
    h={'content-type':'application/json','authorization':'Bearer '+token}
    if idem: h['Idempotency-Key']=idem
    c=http.client.HTTPConnection('localhost',port,timeout=60)
    try:
        c.request(method,'/api/v1'+path,body=None if body is None else json.dumps(body),headers=h); r=c.getresponse(); t=r.read().decode()
        try: return r.status,json.loads(t) if t else None
        except Exception: return r.status,t
    except Exception as e: return 0,str(e)
    finally: c.close()
