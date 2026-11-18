from w3lib import *
import fx,importlib.util,subprocess,re
spec=importlib.util.spec_from_file_location('p604','/Users/kuro/Documents/TA_Agent_qc3p/docs/sprints/6/qc/scripts/p604-onbehalf.py'); P4=importlib.util.module_from_spec(spec); spec.loader.exec_module(P4)
def clr(): subprocess.run(['docker','exec','ep-r-redis-1','sh','-c',"for p in 'ep:emb:*' 'ep:ans:*'; do redis-cli --scan --pattern $p | xargs -r redis-cli del; done"],capture_output=True)
def keys(p): return set(rds('--scan','--pattern',p).split())
def intent(r): return r['row'].split('|')[0]
def ngen(r): return len(nogen(r['payload']))
def nemb(r): return len([i for i in r['payload'] if i['task']=='EMBEDDING'])
USERS=['sv.gioi','sv.kha','sv04']

sh=lambda c,t=900: subprocess.run(c,shell=True,capture_output=True,text=True,timeout=t)
