import sys; sys.path.insert(0,'/tmp/qc3oc')
from routes import *
LK='/tmp/qc-global.lock'
os.mkdir(LK); open(LK+'/owner','w').write('qc3 OpenCode Go stub/fallback/breaker '+datetime.datetime.now().isoformat())
say('LOCK qc3 (run3)',datetime.datetime.now().isoformat())
orig={r['task']:[c['model_id'] for c in r['chain']] for r in routes()['items'] if r['task']!='EMBEDDING'}
course=psql("select id from courses where class_code='QCOC1'")
def stublog(since): 
    out=[]
    for l in open('/tmp/qc3oc/stub.log'):
        d=json.loads(l)
        if d['t']>=since: out.append(d)
    return out
def circ(): return {n:p['circuit'] for n,p in provs().items() if n.startswith('QC3-')}
def waitreload(): time.sleep(1.5)
try:
    ok=mid('QC3-stub-ok','m1'); r429=mid('QC3-stub-429','m429'); r500=mid('QC3-stub-500','m500'); bad=mid('QC3-badkey','deepseek-v4.1-flash')
    # TC-06 e2e: nhà khác host → không cờ OpenCode
    put('CHAT',[ok]); waitreload(); t0=time.time(); s,d,dt=call('CHAT',course); L=stublog(t0)
    say('E2E-HOST stub-ok:',s,d.get('provider'),repr(d.get('text')),'| stub nhận',len(L),'request; UA=',L[0]['ua'],'session=',L[0]['session'],'reasoning_effort trong thân=','reasoning_effort' in L[0]['body'])
    say(audit(course,t0))
    # TC-41: AUTH → nhà tiếp
    put('CHAT',[bad,ok]); waitreload(); t0=time.time()
    outs=[call('CHAT',course) for _ in range(6)]
    say('TC41 6 lời gọi [badkey(OpenCode khoá sai), stub-ok]:',[(o[1].get('provider'),o[1].get('degraded'),o[2]) for o in outs])
    say(audit(course,t0)); say('TC41 circuit sau 6 lần:',circ())
    s,d=A('admin','DELETE',f"/admin/llm/providers/{provs()['QC3-badkey']['id']}"); say('TC31 DELETE nhà đang trong tuyến →',s,(d or {}).get('code'))
    # TC-42a: 429
    put('CHAT',[r429,ok]); waitreload(); t0=time.time(); outs=[call('CHAT',course) for _ in range(6)]
    L=[x for x in stublog(t0) if x['mode']=='r429']; say('TC42a 429: 6 lời gọi →',[(o[1].get('provider'),o[2]) for o in outs],'| stub429 nhận',len(L),'request'); say(audit(course,t0)); say('TC42a circuit:',circ())
    # TC-42b: 500 → mạch
    put('CHAT',[r500,ok]); waitreload(); t0=time.time(); hits=[]
    for i in range(8):
        n0=len([x for x in stublog(t0) if x['mode']=='r500']); o=call('CHAT',course); n1=len([x for x in stublog(t0) if x['mode']=='r500']); hits.append((i+1,o[1].get('provider'),n1-n0,circ()['QC3-stub-500']))
    say('TC42b 500 (lần, provider, số request tới stub500, circuit):',hits); say(audit(course,t0))
    say('TC42b chờ 32 s (LLM_BREAKER_OPEN=30 s)…'); time.sleep(32); t1=time.time(); o=call('CHAT',course); n=len([x for x in stublog(t1) if x['mode']=='r500']); say('TC42b sau 32 s: provider',o[1].get('provider'),'stub500 nhận',n,'circuit',circ()['QC3-stub-500'])
    # TC-36: PII qua đường chat thật
    put('CLASSIFY',[ok]); put('CHAT',[ok]); put('UTILITY',[ok]); waitreload()
    nm=psql("select full_name from users where email='sv.gioi@edupilot.local'"); sc=psql(f"select student_code_snapshot from enrollments e join users u on u.id=e.user_id where u.email='sv.gioi@edupilot.local' and e.course_id='{course}'")
    s,sess=A('sv.gioi','POST','/chat/sessions',{'course_id':course,'title':'qcoc pii'}); sid=sess['id']
    t0=time.time(); import urllib.request
    req=urllib.request.Request(B+f'/chat/sessions/{sid}/messages',data=json.dumps({'content':f'Em là {nm}, MSSV {sc}, cho em hỏi hạn nộp bài tập tuần này là khi nào ạ?'}).encode(),method='POST',headers={'content-type':'application/json','authorization':'Bearer '+login('sv.gioi'),'Idempotency-Key':str(uuid.uuid4()),'Accept':'text/event-stream'})
    try: r=urllib.request.urlopen(req,context=ctx,timeout=60); r.read()
    except Exception as e: say('chat err',e)
    time.sleep(3); L=stublog(t0); blob='\n'.join(x['body'] for x in L)
    say('TC36 stub nhận',len(L),'request; chứa tên thật?',nm in blob,'| chứa MSSV thật?',sc in blob,'| có placeholder?',bool(__import__('re').search(r'\[\[|\{\{|<SV|\[SV',blob)))
    say('TC36 trích:',blob[:420].replace('\\n',' '))
    say('TC36 llm_audit pii_masked_count:',psql(f"select task,provider,pii_masked_count,status from llm_audit where course_id='{course}' and created_at>=to_timestamp({t0}) order by created_at"))
finally:
    for task,ids in orig.items():
        if [c['model_id'] for c in route_of(task)['chain']]!=ids:
            s,d,_=put(task,ids); say('RESTORE',task,s)
    cur={r['task']:[c['model_id'] for c in r['chain']] for r in routes()['items'] if r['task']!='EMBEDDING'}
    say('RESTORED đúng?',cur==orig)
    subprocess.run("docker exec ep-qc-redis-1 sh -c \"redis-cli --scan --pattern 'ep:llm:cb:*' | xargs -r redis-cli del\"",shell=True,capture_output=True)
    os.remove(LK+'/owner'); os.rmdir(LK); say('UNLOCK qc3 (run3)',datetime.datetime.now().isoformat())
