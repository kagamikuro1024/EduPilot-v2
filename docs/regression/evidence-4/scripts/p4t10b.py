import os,subprocess; exec(open(os.path.join(os.path.dirname(os.path.abspath(__file__)),'qc2i-r4apia-lib.py')).read())
from datetime import datetime
OUT='/tmp/qc2-i/res/p4t10.tsv'
CTL=os.path.expanduser('~/.smkt-ctl')
def dsh(c): return subprocess.run(c,shell=True,capture_output=True,text=True).stdout.strip()
dsh("docker rm -f qc2i-fakesmtp qc2i-wf >/dev/null 2>&1; docker stop qc2i-worker >/dev/null 2>&1")
open(CTL+'/mode','w').write('drop'); dsh(f"rm -f {CTL}/log")
dsh(f"docker run -d --name qc2i-fakesmtp --network ep-r_default -v {CTL}/fakesmtp:/fakesmtp:ro -v {CTL}:/ctl -e CTL=/ctl alpine:latest /fakesmtp"); time.sleep(2)
psql("delete from mail_outbox where status='QUEUED'"); psql("delete from outbox where topic='mail.send' and dispatched_at is null")
dsh("docker run -d --name qc2i-wf --network ep-r_default --env-file /tmp/qc2i-r4apia.env -e SMTP_HOST=qc2i-fakesmtp -e INSTANCE_ID=qc2i-wf -v $HOME/Documents/TA_Agent_qc2/seed/demo:/seed/demo:ro edupilot-worker-test"); time.sleep(6)
dsh(f"rm -f {CTL}/log"); em=f'qc2ia-m19b.{TAG}@edupilot.local'; resetrl(); t0=time.time(); s,j,_=call('POST','/auth/register',None,{'email':em,'password':PWD,'full_name':'m19'}); assert s==202
e=time.time()+120; row=''
while time.time()<e:
    row=psql(f"select status,attempts,coalesce(last_error,'') from mail_outbox where to_addr='{em}'")
    if row.startswith('DEAD'): break
    time.sleep(0.3)
td=time.time()-t0
st=[]
for l in open(CTL+'/log').read().splitlines():
    if 'DROP' in l: st.append(datetime.strptime(l.split(' ')[0][:26],'%Y-%m-%dT%H:%M:%S.%f').timestamp())
gaps=[round(b-a,1) for a,b in zip(st,st[1:])]
mid=psql(f"select id from mail_outbox where to_addr='{em}'"); ob=psql(f"select attempts,coalesce(dead_at::text,''),coalesce(last_error,'') from outbox where topic='mail.send' and payload->>'mail_id'='{mid}'")
dead=rds('xlen','outbox.dispatch.dead'); lastd=rds('xrevrange','outbox.dispatch.dead','+','-','count','1')
ok=len(st)==4 and 0.7<=gaps[0]<=1.3 and 3.5<=gaps[1]<=6.5 and 21<=gaps[2]<=39 and row.startswith('DEAD|4|') and ob.split('|')[1]!=''
rec('TC-P201-19','P' if ok else 'F',f'SMTP giả qc2i-fakesmtp ở chế độ drop (kết nối bị ngắt) thay Mailpit; đăng ký → 4 kết nối ở các mốc cách nhau {gaps}s (kỳ vọng 1 s / 5 s / 30 s ±30 %: 0,7–1,3 / 3,5–6,5 / 21–39); DEAD sau {td:.0f}s; mail_outbox {row}; outbox attempts|dead_at|last_error={ob}; stream outbox.dispatch.dead dài {dead}, mục mới nhất chứa outbox_id={mid in lastd or "outbox" in lastd}; worker qc2i-wf riêng, DB qc2i_auth')
dsh("docker rm -f qc2i-wf qc2i-fakesmtp >/dev/null 2>&1; docker start qc2i-worker >/dev/null 2>&1")
