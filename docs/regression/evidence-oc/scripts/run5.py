import sys; sys.path.insert(0,'/tmp/qc3oc')
from routes import *
LK='/tmp/qc-global.lock'
os.mkdir(LK); open(LK+'/owner','w').write('qc3 P-11-108 fallback rules '+datetime.datetime.now().isoformat())
say('LOCK qc3 (run5)',datetime.datetime.now().isoformat())
orig={r['task']:[c['model_id'] for c in r['chain']] for r in routes()['items'] if r['task']!='EMBEDDING'}
course=psql("select id from courses where class_code='QCOC1'")
def stubhits(mode,since): return len([1 for l in open('/tmp/qc3oc/stub.log') if json.loads(l)['mode']==mode and json.loads(l)['t']>=since])
def circ(n): return provs()[n]['circuit']
try:
    ok=mid('QC3-stub-ok','m1'); r404=mid('QC3-stub-404','m404'); r403=mid('QC3-stub-403','m403'); bm=mid('QC3-badmodel','deepseek-v9-khong-co')
    # 400 model lạ: không chuyển, không tính mạch
    put('CHAT',[bm,ok]); time.sleep(1.5); t0=time.time(); outs=[call('CHAT',course) for _ in range(6)]
    say('P108-400 [OpenCode model lạ, stub-ok] 6 lời gọi:',[(o[1].get('provider') or '-',o[1].get('degraded'),o[2]) for o in outs],'| stub-ok nhận',stubhits('ok',t0),'request | circuit badmodel =',circ('QC3-badmodel'))
    say(audit(course,t0))
    # 404: chuyển nhà tiếp, không tính mạch
    put('CHAT',[r404,ok]); time.sleep(1.5); t0=time.time(); outs=[call('CHAT',course) for _ in range(6)]
    say('P108-404 [stub-404, stub-ok] 6 lời gọi:',[(o[1].get('provider'),o[2]) for o in outs],'| stub404 nhận',stubhits('r404',t0),'| circuit stub404 =',circ('QC3-stub-404')); say(audit(course,t0))
    # 403: chuyển nhà tiếp, tính mạch
    put('CHAT',[r403,ok]); time.sleep(1.5); t0=time.time(); outs=[call('CHAT',course) for _ in range(7)]
    say('P108-403 [stub-403, stub-ok] 7 lời gọi:',[(o[1].get('provider'),o[2]) for o in outs],'| stub403 nhận',stubhits('r403',t0),'| circuit stub403 =',circ('QC3-stub-403')); say(audit(course,t0))
finally:
    for task,ids in orig.items():
        if [c['model_id'] for c in route_of(task)['chain']]!=ids:
            s,d,_=put(task,ids); say('RESTORE',task,s)
    cur={r['task']:[c['model_id'] for c in r['chain']] for r in routes()['items'] if r['task']!='EMBEDDING'}
    say('RESTORED đúng?',cur==orig)
    subprocess.run("docker exec ep-qc-redis-1 sh -c \"redis-cli --scan --pattern 'ep:llm:cb:*' | xargs -r redis-cli del\"",shell=True,capture_output=True)
    os.remove(LK+'/owner'); os.rmdir(LK); say('UNLOCK qc3 (run5)',datetime.datetime.now().isoformat())
