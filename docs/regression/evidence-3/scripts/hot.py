import os,sys; os.environ['SP']='3-hot'
exec(open('p10x.py').read().split("LOCK='/tmp/qc2-lock-global'")[0])
LOCK='/tmp/qc2-lock-global'
for _ in range(300):
    try: os.mkdir(LOCK); break
    except FileExistsError: time.sleep(4)
open('/tmp/qc2-global-changes.log','a').write(datetime.datetime.now().strftime('%H:%M:%S')+" SmkB: LOCK hot-reload CHAT→stub (≤2 phút)\n")
try:
    ok1=[mid(S1,'m-ok')]
    r=put('CHAT',ok1); print('PUT',r[0],r[2].get('Etag'))
    g=routes()['CHAT']; print('GET chain',[ (c['provider_name'],c['model']) for c in g['chain']],g['version'])
    for i in range(6):
        t=time.time(); s,j,ms,_=chat(); print(i,s,j.get('provider'),j.get('model'),ms, round(time.time()-t,2)); time.sleep(1)
finally:
    restore(); os.rmdir(LOCK)
    open('/tmp/qc2-global-changes.log','a').write(datetime.datetime.now().strftime('%H:%M:%S')+" SmkB: UNLOCK hot-reload, hoàn lại\n")
