# TC-P201-19 (duyệt lại): đo 3 lần mốc thử lại outbox thư với SMTP giả ngắt kết nối (chế độ drop), worker riêng qc2i-wf; mặc định poll 500 ms
import os,subprocess; exec(open(os.path.join(os.path.dirname(os.path.abspath(__file__)),'qc2i-r4apia-lib.py')).read())
from datetime import datetime
CTL=os.path.expanduser('~/.smkt-ctl')
def dsh(c): return subprocess.run(c,shell=True,capture_output=True,text=True).stdout.strip()
runs=[]
try:
    dsh("docker rm -f qc2i-fakesmtp qc2i-wf >/dev/null 2>&1; docker stop qc2i-worker >/dev/null 2>&1")
    open(CTL+'/mode','w').write('drop')
    dsh(f"docker run -d --name qc2i-fakesmtp --network ep-r_default -v {CTL}/fakesmtp:/fakesmtp:ro -v {CTL}:/ctl -e CTL=/ctl alpine:latest /fakesmtp"); time.sleep(2)
    psql("delete from mail_outbox where status='QUEUED'"); psql("delete from outbox where topic='mail.send' and dispatched_at is null")
    dsh("docker run -d --name qc2i-wf --network ep-r_default --env-file /tmp/qc2i-r4apia.env -e SMTP_HOST=qc2i-fakesmtp -e INSTANCE_ID=qc2i-wf -v $HOME/Documents/TA_Agent_qc2/seed/demo:/seed/demo:ro edupilot-worker-test"); time.sleep(6)
    for i in range(3):
        dsh(f"rm -f {CTL}/log"); em=f'qc2ia-m19c{i}.{TAG}@edupilot.local'; resetrl(); t0=time.time(); s,j,_=call('POST','/auth/register',None,{'email':em,'password':PWD,'full_name':'m19'}); assert s==202
        e=time.time()+120; row=''
        while time.time()<e:
            row=psql(f"select status,attempts from mail_outbox where to_addr='{em}'")
            if row.startswith('DEAD'): break
            time.sleep(0.3)
        st=[]
        for l in open(CTL+'/log').read().splitlines():
            if 'DROP' in l: st.append(datetime.strptime(l.split(' ')[0][:26],'%Y-%m-%dT%H:%M:%S.%f').timestamp())
        runs.append(([round(b-a,2) for a,b in zip(st,st[1:])],len(st),row))
        print(runs[-1],flush=True)
finally:
    dsh("docker rm -f qc2i-wf qc2i-fakesmtp >/dev/null 2>&1; docker start qc2i-worker >/dev/null 2>&1")
import json; print(json.dumps(runs))
