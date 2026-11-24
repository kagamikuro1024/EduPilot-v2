import sys; sys.path.insert(0,'/tmp/qc3oc')
from routes import *
LK='/tmp/qc-global.lock'
os.mkdir(LK); open(LK+'/owner','w').write('qc3 P-11-109 BAD_RESPONSE + audit degraded '+datetime.datetime.now().isoformat())
say('LOCK qc3 (run6)',datetime.datetime.now().isoformat())
orig={r['task']:[c['model_id'] for c in r['chain']] for r in routes()['items'] if r['task']!='EMBEDDING'}
course=psql("select id from courses where class_code='QCOC1'")
def hits(mode,since): return len([1 for l in open('/tmp/qc3oc/stub.log') if json.loads(l)['mode']==mode and json.loads(l)['t']>=since])
try:
    ok=mid('QC3-stub-ok','m1'); bad=mid('QC3-stub-bad','mb')
    put('CHAT',[bad,ok]); time.sleep(1.5); t0=time.time(); rows=[]
    for i in range(7):
        n0=hits('bad',t0); o=call('CHAT',course); rows.append((i+1,o[1].get('provider'),hits('bad',t0)-n0,provs()['QC3-stub-bad']['circuit']))
    say('TC61 [stub-bad (200 không JSON), stub-ok] (lần, provider trả lời, request tới stub-bad, circuit):',rows); say(audit(course,t0))
    # audit degraded: chuỗi chỉ có stub-bad
    put('CHAT',[bad]); time.sleep(1.5); time.sleep(31)  # chờ mạch stub-bad bán mở/đóng? ghi nhận trạng thái
    t1=time.time(); o=call('CHAT',course); say('AUDIT-DEGRADED CHAT=[stub-bad]: HTTP',o[0],'degraded=',o[1].get('degraded'),'provider(API)=',repr(o[1].get('provider')))
    say(psql(f"select task,coalesce(provider,'∅'),coalesce(model,'∅'),status,coalesce(error_kind,''),fallback_index,degraded from llm_audit where course_id='{course}' and created_at>=to_timestamp({t1}) order by created_at") if time.sleep(2.5) is None else '')
    # chuỗi hai nhà cùng lỗi: A=stub-bad, B=stub-bad2? dùng [bad, badmodel-less] -> chỉ một nhà; thêm 2 nhà lỗi bằng chuỗi [bad, bad]
    put('CHAT',[bad,bad]); time.sleep(1.5); t2=time.time(); o=call('CHAT',course); time.sleep(2.5)
    say('AUDIT-DEGRADED CHAT=[stub-bad, stub-bad] (mạch có thể đang mở):',o[1].get('degraded'),'|',psql(f"select coalesce(provider,'∅'),coalesce(model,'∅'),status,coalesce(error_kind,''),fallback_index from llm_audit where course_id='{course}' and created_at>=to_timestamp({t2}) order by created_at"))
finally:
    for task,ids in orig.items():
        if [c['model_id'] for c in route_of(task)['chain']]!=ids:
            s,d,_=put(task,ids); say('RESTORE',task,s)
    cur={r['task']:[c['model_id'] for c in r['chain']] for r in routes()['items'] if r['task']!='EMBEDDING'}
    say('RESTORED đúng?',cur==orig)
    subprocess.run("docker exec ep-qc-redis-1 sh -c \"redis-cli --scan --pattern 'ep:llm:cb:*' | xargs -r redis-cli del\"",shell=True,capture_output=True)
    os.remove(LK+'/owner'); os.rmdir(LK); say('UNLOCK qc3 (run6)',datetime.datetime.now().isoformat())
