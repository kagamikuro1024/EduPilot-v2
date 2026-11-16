# P102-42/43/44 + P104-37: nạp lại nóng giữa hai gateway (smk3-gw1:18201, smk3-gw2:18202), DB+Redis cô lập
from smk3iso import *
V1='http://host.docker.internal:9751/v1'; V2='http://host.docker.internal:9752/v1'
SM=['m-ok','m-400','m-401','m-big','m-canary','m-ra2','m-ra30','m-slow10','m-503','m-500','m-midkill','m-prefail','m-slow']
T1=imkprov('SmkI-T1',V1,SM); T2=imkprov('SmkI-T2',V2,SM)
only=sys.argv[1:] or ['42','43','44','37']
def call_loop(stopat,res,gws=('gw1','gw2'),every=0.05):
    i=0
    while time.time()<stopat:
        gw=gws[i%2]; i+=1; t0=time.time(); r=ichat('CHAT',gw=gw); res.append((gw,t0,time.time(),r[0],r[1].get('provider')))
        time.sleep(every)
if '42' in only or '37' in only:
    iput('CHAT',[mid(T1,'m-ok')]); time.sleep(1.5)
    res=[]; stopat=time.time()+8
    ths=[threading.Thread(target=call_loop,args=(stopat,res)) for _ in range(3)]
    [t.start() for t in ths]; time.sleep(3)
    r=iput('CHAT',[mid(T2,'m-ok')]); tput=time.time()
    [t.join() for t in ths]
    errs=[x for x in res if x[3]!=200]
    old_after=[x for x in res if x[1]>tput and x[4]=='SmkI-T1']
    new_first={gw:min([x[2]-tput for x in res if x[0]==gw and x[4]=='SmkI-T2'] or [None]) for gw in ('gw1','gw2')}
    last_old_start={gw:max([x[1]-tput for x in res if x[0]==gw and x[4]=='SmkI-T1' and x[1]>tput] or [-9]) for gw in ('gw1','gw2')}
    n=len(res); print('P102-42/P104-37a: PUT',r[0],'tổng lời gọi',n,'~%.0f req/s'%(n/8),'lỗi',len(errs),'; sau PUT còn dùng nhà cũ (bắt đầu sau PUT):',len(old_after),'; lần bắt đầu muộn nhất dùng cấu hình cũ sau PUT (s):',last_old_start)
    print(' provider theo gateway:',{gw:sorted(set(x[4] for x in res if x[0]==gw)) for gw in ('gw1','gw2')})
    print(' audit theo gateway/provider:',ipsql("select provider,count(*) from llm_audit where created_at>now()-interval '60 seconds' and task='CHAT' group by 1"))
if '43' in only:
    iput('CHAT',[mid(T1,'m-ok')]); time.sleep(1.5)
    ipsql("alter table llm_models rename to llm_models_x")
    ird('PUBLISH','ep:llm:reload','x'); time.sleep(2)
    outs=[ichat('CHAT',gw=g) for g in ('gw1','gw2')]
    lg=[l for g in ('smk3-gw1','smk3-gw2') for l in gwlogs('20s',g) if '"level":"ERROR"' in l or ('nạp' in l and 'WARN' in l)]
    ipsql("alter table llm_models_x rename to llm_models")
    print('P102-43 sau khi hỏng cấu hình+reload: chat',[(o[0],o[1].get('provider')) for o in outs],'| log lỗi:',[l[:230] for l in lg][:2])
    print(' readyz',lib._call('GET','/readyz',base='http://localhost:18201/api/v1')[:2])
    ird('PUBLISH','ep:llm:reload','x'); time.sleep(2); print(' sau khôi phục',ichat('CHAT')[1].get('provider'))
if '44' in only:
    iput('CHAT',[mid(T1,'m-ok')]); time.sleep(1.5)
    assert ichat('CHAT',gw='gw2')[1]['provider']=='SmkI-T1'
    ipsql(f"update llm_task_routes set model_id='{mid(T2,'m-ok')}' where task='CHAT'")  # sửa DB trực tiếp, KHÔNG PUBLISH
    t0=time.time(); got=None
    while time.time()-t0<80:
        p=ichat('CHAT',gw='gw2')[1].get('provider')
        if p=='SmkI-T2': got=time.time()-t0; break
        time.sleep(1)
    print('P102-44 sửa DB không PUBLISH → gateway 2 nạp sau',None if got is None else round(got,1),'s')
if '37' in only:
    iput('CHAT',[mid(T1,'m-ok')]); time.sleep(1.5)
    subprocess.run(['docker','stop','smk3-redis'],capture_output=True)
    t0=time.time(); r=iput('CHAT',[mid(T2,'m-ok')]); print('P104-37b: PUT route khi Redis dừng ->',r[0],str(r[1])[:150],'%.1fs'%(time.time()-t0))
    subprocess.run(['docker','start','smk3-redis'],capture_output=True); time.sleep(3)
    t0=time.time(); got=None
    while time.time()-t0<80:
        p=ichat('CHAT',gw='gw2')[1].get('provider')
        if p=='SmkI-T2': got=time.time()-t0; break
        time.sleep(1)
    print(' gateway 2 dùng cấu hình mới sau khi Redis về:',None if got is None else round(got,1),'s')
