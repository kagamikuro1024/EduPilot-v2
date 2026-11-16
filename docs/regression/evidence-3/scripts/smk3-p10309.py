# TC-P103-09: nhà có rpm_limit/tpm_limit = null trên gateway KHÔNG đặt LLM_DEFAULT_RPM/TPM (smk3-gw4 :18204, env-file đã lọc) → mặc định 60 / 100.000
from smk3iso import *
IB['gw4']='http://localhost:18204/api/v1'
V='http://host.docker.internal:9751/v1'
def reset9751(): urllib.request.urlopen(urllib.request.Request('http://localhost:9751/__reset',data=b'{}',method='POST'),timeout=10)
P=imkprov('SmkI-N9',V,['m-ok'],gw='gw4'); assert P['rpm_limit'] is None and P['tpm_limit'] is None,(P['rpm_limit'],P['tpm_limit'])
iput('CHAT',[mid(P,'m-ok')],{'retries':0},gw='gw4'); time.sleep(2); reset9751()
print('env LLM_DEFAULT_* trong container:',subprocess.run('docker exec smk3-gw4 printenv | grep -c LLM_DEFAULT',shell=True,capture_output=True,text=True).stdout.strip())
stop=[False]
def worker(): 
    while not stop[0]: ichat('CHAT',gw='gw4')
ths=[threading.Thread(target=worker) for i in range(20)]
t0=time.time(); [t.start() for t in ths]; marks=[]
for k in range(1,11): time.sleep(1-((time.time()-t0)%1)); marks.append(stub(9751)['n'])
stop[0]=True; [t.join() for t in ths]
print('stub tích luỹ mỗi giây:',marks,'tại 10 s =',marks[-1],'(null→60 rpm: ≤ 16)')
k='ep:llm:rl:%s:'+P['id']
print('rpm bucket',ird('hgetall',k%'rpm').replace('\n',' '),'| tpm bucket',ird('hgetall',k%'tpm').replace('\n',' '))
tp_=float(ird('hget',k%'tpm','tokens'))
print('KQ','P' if marks[-1]<=16 and tp_<=10000 else 'F',marks[-1],tp_)
