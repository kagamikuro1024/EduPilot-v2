# TC-P103-08 (rpm_limit=60, 2 gateway, 10 s) / TC-P103-09 (cột null → mặc định LLM_DEFAULT_RPM=60, trên gateway không đặt biến)
from smk3iso import *
mode=sys.argv[1]
V1='http://host.docker.internal:9751/v1'; V2='http://host.docker.internal:9752/v1'
if mode=='08':
    T1=imkprov('SmkI-T1',V1,['m-ok']); gws=['gw1','gw2']; prov=T1
else:
    T2=imkprov('SmkI-T2',V2,['m-ok']); gws=['gw3']; prov=T2
    s,j,_=iprov_set('SmkI-T2',rpm_limit=None,tpm_limit=None); print('T2 rpm/tpm null',s)
iput('CHAT',[mid(prov,'m-ok')],{'retries':0}); time.sleep(1.5); stubreset()
stop=[False]; res=[]; lk=threading.Lock()
def worker(gw):
    while not stop[0]:
        r=ichat('CHAT',gw=gw)
        with lk: res.append((time.time(),r[0],r[1].get('code'),r[2]))
ths=[threading.Thread(target=worker,args=(gws[i%len(gws)],)) for i in range(20)]
t0=time.time(); [t.start() for t in ths]; time.sleep(10); n_stub=stub(9751)['n']+(stub(9752)['n'] if False else 0); stop[0]=True; snap=len(res); [t.join() for t in ths]
print('stub nhận trong 10 s:',stub(9751)['n'] if mode=='08' else stub(9752)['n'],'(đếm chung hai cổng)','| trả về trong cửa sổ:',snap)
from collections import Counter; print('mã phản hồi:',Counter((r[1],r[2]) for r in res))
