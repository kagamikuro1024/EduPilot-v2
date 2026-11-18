"""Cửa sổ fake-LLM: khoá global, CHAT/CLASSIFY → QC fake (+ EMBEDDING → fake-embed). python3 win.py on|off"""
from lib import *
import sys,datetime,subprocess
LOCK='/tmp/qc2-lock-global'; LOG='/tmp/qc2-global-changes.log'; BK='/tmp/qc3e/route-backup.json'
FP='01a1df2d-a8ed-7389-b1d8-d6805e510bed'; FM='01a1df2d-a8ee-70d6-91cd-be284dbb4097'
def log(m): open(LOG,'a').write(f"{datetime.datetime.now():%H:%M:%S} qc3: {m}\n")
def routes(): return {r['task']:r for r in A('admin','GET','/admin/llm/routes')[1]['items']}
def put(task,chain,ver):
    return A('admin','PUT','/admin/llm/routes',{'task':task,'chain':chain,'version':ver})
def on(tasks=('CHAT','CLASSIFY')):
    for i in range(80):
        r=subprocess.run(['mkdir',LOCK],capture_output=True)
        if r.returncode==0: break
        time.sleep(15)
    else: raise SystemExit('không lấy được khoá')
    log(f"LOCK — cửa sổ fake-LLM sprint 6 (CHAT/CLASSIFY → QC fake); ≤ 10 phút")
    rt=routes(); json.dump({t:{'chain':[c['model_id'] for c in rt[t]['chain']],'version':rt[t]['version']} for t in tasks},open(BK,'w'))
    for t in tasks:
        s,d=put(t,[FM],rt[t]['version']); print(t,s,str(d)[:80])
def off():
    bk=json.load(open(BK)); rt=routes()
    for t,v in bk.items():
        s,d=put(t,v['chain'],rt[t]['version']); print('restore',t,s)
    A('admin','POST','/_test/llm/fake',{'error_rate':0,'latency_min_ms':0,'latency_max_ms':0,'stream_delay_ms':20})
    log("UNLOCK — hoàn route CHAT/CLASSIFY về QC 9router, fake về mặc định")
    subprocess.run(['rmdir',LOCK])
if __name__=='__main__':
    on() if sys.argv[1]=='on' else off()
