import sys; sys.path.insert(0,'/tmp/qc3oc')
from routes import *
LK='/tmp/qc-global.lock'
os.mkdir(LK); open(LK+'/owner','w').write('qc3 OpenCode Go route cases '+datetime.datetime.now().isoformat())
say('LOCK qc3',datetime.datetime.now().isoformat())
orig={r['task']:[c['model_id'] for c in r['chain']] for r in routes()['items'] if r['task']!='EMBEDDING'}
json.dump(routes(),open(f'{EV}/ep-qc-routes-snapshot-run1.json','w'),ensure_ascii=False)
course=psql("select id from courses where class_code='QCOC1'")
try:
    gw0,wk0=started('ep-qc-gateway-1'),started('ep-qc-worker-1'); say('StartedAt gateway',gw0,'worker',wk0)
    oc=mid('OpenCode Go','deepseek-v4.1-flash'); fake=mid('QC fake','fake-chat'); r9=mid('QC 9router','ag/gemini-3.8-flash')
    # TC-OC-29: phân quyền PUT route
    v=route_of('CHAT')['version']; before=json.dumps(route_of('CHAT')['chain'])
    for u in ('teacher','ta','sv.gioi'):
        s,d=A(u,'PUT','/admin/llm/routes',{'task':'CHAT','chain':[fake],'version':v}); say('TC29 PUT route',u,s,(d or {}).get('code'))
    s,d=A('teacher','GET','/admin/llm/routes'); say('TC29 teacher GET routes',s)
    say('TC29 CHAT chain đổi?', json.dumps(route_of('CHAT')['chain'])!=before, 'version',route_of('CHAT')['version'],'==',v)
    # TC-OC-30: PUT lỗi
    import uuid as U
    for name,chain,ver in (('model_id không tồn tại',[str(U.uuid4())],v),('chain rỗng',[],v),('version cũ',[oc],v-1)):
        s,d=A('admin','PUT','/admin/llm/routes',{'task':'CHAT','chain':chain,'version':ver}); say('TC30',name,s,(d or {}).get('code'),str((d or {}).get('details'))[:100])
    s,d,dt=call('CHAT',course); say('TC30 CHAT vẫn chạy sau PUT lỗi:',s,d.get('provider'),d.get('text'),dt,'s')
    # TC-OC-26: đổi tuyến → fake → OpenCode Go
    t0=time.time()
    s,d,tp=put('CHAT',[fake]); say('PUT CHAT→QC fake',s); s2,d2,dt=call('CHAT',course); say('  chat ngay sau PUT: provider',d2.get('provider'),'sau',round(time.time()-tp,2),'s kể từ PUT')
    s,d,tp=put('CHAT',[oc]); say('PUT CHAT→OpenCode Go',s); s2,d2,dt=call('CHAT',course); say('  chat ngay sau PUT: provider',d2.get('provider'),d2.get('model'),repr(d2.get('text')),'sau',round(time.time()-tp,2),'s kể từ PUT; call',dt,'s')
    say('StartedAt giữa chừng: gateway',started('ep-qc-gateway-1')==gw0,'worker',started('ep-qc-worker-1')==wk0)
    # TC-OC-27: về 9router
    s,d,tp=put('CHAT',[r9]); say('PUT CHAT→QC 9router',s); s2,d2,dt=call('CHAT',course); say('  chat ngay sau PUT: HTTP',s2,str(d2)[:160],'sau',round(time.time()-tp,2),'s kể từ PUT; call',dt,'s')
    s,d,tp=put('CHAT',[fake]); s2,d2,dt=call('CHAT',course); say('  PUT→QC fake lại: provider',d2.get('provider'))
    say('AUDIT theo lớp QCOC1:'); say(audit(course,t0))
finally:
    for task,ids in orig.items():
        r=route_of(task)
        if [c['model_id'] for c in r['chain']]!=ids:
            s,d,_=put(task,ids); say('RESTORE',task,s)
    cur={r['task']:[c['model_id'] for c in r['chain']] for r in routes()['items'] if r['task']!='EMBEDDING'}
    say('RESTORED đúng?',cur==orig)
    subprocess.run("docker exec ep-qc-redis-1 sh -c \"redis-cli --scan --pattern 'ep:llm:cb:*' | xargs -r redis-cli del\"",shell=True,capture_output=True)
    os.rename(LK+'/owner',LK+'/owner.done'); os.remove(LK+'/owner.done'); os.rmdir(LK); say('UNLOCK qc3',datetime.datetime.now().isoformat())
