from smk3iso import *
V1='http://host.docker.internal:9751/v1'
T1=imkprov('SmkI-T1',V1,['m-ok']); iput('CHAT',[mid(T1,'m-ok')],{'retries':0}); time.sleep(65)  # xả token bucket (60 s)
stubreset(); stop=[False]; res=[]; lk=threading.Lock()
def worker(gw):
    while not stop[0]:
        r=ichat('CHAT',gw=gw)
        with lk: res.append((time.time(),r[0],r[1].get('code')))
ths=[threading.Thread(target=worker,args=(['gw1','gw2'][i%2],)) for i in range(20)]
t0=time.time(); [t.start() for t in ths]
for k in range(1,13):
    time.sleep(1-((time.time()-t0)%1)); print('t=%2d s stub tích luỹ=%d'%(round(time.time()-t0),stub(9751)['n']))
stop[0]=True; [t.join() for t in ths]
