import sys,subprocess,json,time,os
sys.path.insert(0,'/tmp/qc3e')
from lib import *
FP='01a1df2d-a8ed-7389-b1d8-d6805e510bed'
def setrpm(v):
    ps=A('admin','GET','/admin/llm/providers')[1]['items']; p=[x for x in ps if x['id']==FP][0]
    if (p.get('rpm_limit') or None)==v: return
    A('admin','PUT','/admin/llm/providers/'+FP,{'type':'fake','name':p['name'],'enabled':True,'rpm_limit':v,'version':p['version']})
cases=sys.argv[1:]; delay=int(os.environ.get('DELAY','500'))
subprocess.run(['python3','/tmp/qc3e/win.py','on'],capture_output=True)
try:
    for c in cases:
        subprocess.run(['docker','exec','ep-r-redis-1','sh','-c',"redis-cli --scan --pattern 'ep:ans:*' | xargs -r redis-cli del"],capture_output=True)
        A('admin','POST','/_test/llm/fake',{'stream_delay_ms':delay,'error_rate':0})
        if os.environ.get('RPM1') and c in os.environ['RPM1'].split(','): setrpm(1)
        r=subprocess.run(['node','/tmp/qc3e/ui/'+os.environ.get('SCRIPT','ui1.mjs'),c],capture_output=True,text=True,cwd='/tmp/qc3e/ui',env={**os.environ,'NODE_TLS_REJECT_UNAUTHORIZED':'0'},timeout=300)
        print(r.stdout[-1500:],r.stderr[-300:])
finally:
    setrpm(None)
    A('admin','POST','/_test/llm/fake',{'stream_delay_ms':20,'error_rate':0})
    subprocess.run(['python3','/tmp/qc3e/win.py','off'],capture_output=True)
