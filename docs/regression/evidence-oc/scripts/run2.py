import sys; sys.path.insert(0,'/tmp/qc3oc')
from routes import *
LK='/tmp/qc-global.lock'
os.mkdir(LK); open(LK+'/owner','w').write('qc3 OpenCode Go reload timing '+datetime.datetime.now().isoformat())
say('LOCK qc3 (run2)',datetime.datetime.now().isoformat())
orig={r['task']:[c['model_id'] for c in r['chain']] for r in routes()['items'] if r['task']!='EMBEDDING'}
course=psql("select id from courses where class_code='QCOC1'")
try:
    oc=mid('OpenCode Go','deepseek-v4.1-flash'); fake=mid('QC fake','fake-chat'); bad=mid('QC lỗi','bad-model')
    def until(pred,limit=6.0):
        t0=time.time(); n=0
        while True:
            s,d,dt=call('CHAT',course); n+=1
            if pred(d): return round(time.time()-t0,2),n,d
            if time.time()-t0>limit: return None,n,d
    res=[]
    for i in range(4):
        # → fake
        s,d,tp=put('CHAT',[fake]); assert s==200
        t0=time.time(); lat=None
        while time.time()-t0<8:
            s2,d2,dt=call('CHAT',course)
            if d2.get('provider')=='QC fake': lat=round(time.time()-tp,2); break
        res.append(('→QC fake',lat))
        # → OpenCode Go (poll: fake answers fast until reload)
        s,d,tp=put('CHAT',[oc]); assert s==200
        t0=time.time(); lat=None; first=None
        while time.time()-t0<40:
            ts=time.time(); s2,d2,dt=call('CHAT',course)
            if first is None: first=d2.get('provider')
            if d2.get('provider')=='OpenCode Go': lat=round(ts-tp,2); break
        res.append(('→OpenCode Go (thời điểm gửi lời gọi đúng tuyến mới, kể từ PUT)',lat,'lời gọi đầu sau PUT đi tới:',first))
        # → bad (fast fail)
        s,d,tp=put('CHAT',[bad]); assert s==200
        t0=time.time(); lat=None
        while time.time()-t0<8:
            s2,d2,dt=call('CHAT',course)
            if d2.get('degraded'): lat=round(time.time()-tp,2); break
        res.append(('→QC lỗi (nhà hỏng, lỗi nhanh)',lat))
    for r in res: say('RELOAD',*r)
finally:
    for task,ids in orig.items():
        if [c['model_id'] for c in route_of(task)['chain']]!=ids:
            s,d,_=put(task,ids); say('RESTORE',task,s)
    cur={r['task']:[c['model_id'] for c in r['chain']] for r in routes()['items'] if r['task']!='EMBEDDING'}
    say('RESTORED đúng?',cur==orig)
    subprocess.run("docker exec ep-qc-redis-1 sh -c \"redis-cli --scan --pattern 'ep:llm:cb:*' | xargs -r redis-cli del\"",shell=True,capture_output=True)
    os.remove(LK+'/owner'); os.rmdir(LK); say('UNLOCK qc3 (run2)',datetime.datetime.now().isoformat())
