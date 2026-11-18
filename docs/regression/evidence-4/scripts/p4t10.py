# P201 nhóm thư (D) trên stack qc2i: SMTP giả + worker riêng. Không đụng ep-r.
import os,subprocess; exec(open(os.path.join(os.path.dirname(os.path.abspath(__file__)),'qc2i-r4apia-lib.py')).read())
OUT='/tmp/qc2-i/res/p4t10.tsv'
CTL=os.path.expanduser('~/.smkt-ctl'); os.makedirs(CTL,exist_ok=True)
def dsh(c): return subprocess.run(c,shell=True,capture_output=True,text=True).stdout.strip()
def setmode(m): open(CTL+'/mode','w').write(m)
def flog(): return open(CTL+'/log').read() if os.path.exists(CTL+'/log') else ''
def conns_for(t0):  # các kết nối SMTP giả từ thời điểm t0 (epoch)
    out=[]
    for l in flog().splitlines():
        ts=l.split(' ')[0]
        try: e=time.mktime(time.strptime(ts[:19],'%Y-%m-%dT%H:%M:%S'))+float('0'+ts[19:ts.index('+') if '+' in ts[19:] else len(ts)].replace('Z','')) if False else None
        except: e=None
        out.append(l)
    return out
def wk(name,host):
    dsh(f"docker rm -f {name} >/dev/null 2>&1; docker run -d --name {name} --network ep-r_default --env-file /tmp/qc2i-r4apia.env -e SMTP_HOST={host} -e INSTANCE_ID={name} -v $HOME/Documents/TA_Agent_qc2/seed/demo:/seed/demo:ro edupilot-worker-test")
def mail_row(em): return psql(f"select id,status,attempts,coalesce(last_error,''),coalesce(sent_at::text,'') from mail_outbox where to_addr='{em}' order by created_at desc limit 1")
def outbox_row(mid): return psql(f"select attempts,coalesce(dead_at::text,''),coalesce(last_error,'') from outbox where topic='mail.send' and payload->>'mail_id'='{mid}'")
def newuser(label):
    em=f'qc2ia-{label}.{TAG}@edupilot.local'; resetrl(); s,j,_=call('POST','/auth/register',None,{'email':em,'password':PWD,'full_name':'QC2IA '+label}); assert s==202,(s,j); return em
def wait(cond,t=120,step=0.5):
    e=time.time()+t
    while time.time()<e:
        v=cond()
        if v: return v
        time.sleep(step)
    return None
# ---- chuẩn bị: dừng worker thường; bật SMTP giả
dsh("docker rm -f qc2i-fakesmtp >/dev/null 2>&1; docker stop qc2i-worker >/dev/null 2>&1; docker rm -f qc2i-wf >/dev/null 2>&1")
dsh(f"docker run -d --name qc2i-fakesmtp --network ep-r_default -v {CTL}/fakesmtp:/fakesmtp:ro -v {CTL}:/ctl -e CTL=/ctl alpine:latest /fakesmtp")
setmode('ok'); time.sleep(2)
# ---- P201-15: commit → đúng 1 mail_outbox + 1 outbox mail.send {mail_id}; (không worker nào chạy)
em15=newuser('m15'); time.sleep(1.5)
rows=psql(f"select count(*) from mail_outbox where to_addr='{em15}'"); mid=psql(f"select id from mail_outbox where to_addr='{em15}'")
ob=psql(f"select count(*),min(payload::text) from outbox where topic='mail.send' and payload->>'mail_id'='{mid}'")
# rollback: đăng ký với mật khẩu yếu (422) không để lại dòng nào
n0=psql("select count(*) from mail_outbox"); o0=psql("select count(*) from outbox where topic='mail.send'"); resetrl(); sw=call('POST','/auth/register',None,{'email':f'qc2ia-w15.{TAG}@edupilot.local','password':'123','full_name':'x'}); n1=psql("select count(*) from mail_outbox"); o1=psql("select count(*) from outbox where topic='mail.send'")
rec('TC-P201-15','P' if rows=='1' and ob.startswith('1|') and json.loads(ob.split('|',1)[1])=={'mail_id':mid} and sw[0]==422 and n0==n1 and o0==o1 else 'F',f'commit (đăng ký hợp lệ): mail_outbox=1 (QUEUED) + outbox topic mail.send payload={ob.split("|",1)[1]} (đúng {{mail_id}}); rollback (đăng ký mật khẩu yếu → {sw[0]}): mail_outbox {n0}→{n1}, outbox mail.send {o0}→{o1} (không thêm); rollback do lỗi sau khi xếp thư: go test TestEnqueueRollback (xanh)')
# ---- P201-19/20: SMTP chết (drop) → thử 1 s, 5 s, 30 s rồi DEAD
setmode('drop'); wk('qc2i-wf','qc2i-fakesmtp'); time.sleep(5)
dsh(f"rm -f {CTL}/log"); 
em19=newuser('m19'); t0=time.time()
def dead(): 
    r=mail_row(em19); return r if '|DEAD|' in r else None
r19=wait(dead,150); td=time.time()-t0
lines=[l for l in flog().splitlines() if 'DROP' in l and em19 or 'DROP' in l]
stamps=[]
from datetime import datetime
for l in lines:
    ts=l.split(' ')[0]; ts=ts[:26]+'Z' if len(ts)>26 else ts
    try: stamps.append(datetime.strptime(ts.replace('Z','')[:26],'%Y-%m-%dT%H:%M:%S.%f').timestamp())
    except Exception: pass
gaps=[round(b-a,1) for a,b in zip(stamps,stamps[1:])]
mid19=psql(f"select id from mail_outbox where to_addr='{em19}'"); ob19=outbox_row(mid19); dl=rds('xlen','outbox.dispatch.dead'); 
okgaps=len(gaps)==3 and 0.7<=gaps[0]<=1.3+1.5 and 3.5<=gaps[1]<=6.5+1.5 and 21<=gaps[2]<=39+1.5
rec('TC-P201-19','P' if r19 and okgaps and ob19.split('|')[1] else 'F',f'SMTP giả (qc2i-fakesmtp, chế độ drop) thay Mailpit: 4 kết nối, khoảng cách giữa các lần thử {gaps}s (mốc 1 s / 5 s / 30 s ±30 %, đo bằng thời điểm kết nối ở log SMTP giả); DEAD sau {td:.0f}s kể từ lúc xếp; mail_outbox {r19}; outbox attempts|dead_at={ob19[:70]}; stream outbox.dispatch.dead dài {dl}')
le=r19.split('|')[3] if r19 else ''
cnt=psql("select count(*) from mail_outbox where last_error ~* '@|token='")
rec('TC-P201-20','P' if r19 and 0<len(le)<=1000 and '@' not in le and 'token=' not in le and cnt=='0' else 'F',f'last_error dòng DEAD dài {len(le)} (≤1000): {le[:80]!r}; không chứa "@"/"token="/nội dung thư; select count(*) from mail_outbox where last_error ~* \'@|token=\' = {cnt}')
# ---- P201-21: hồi phục trước lần thử cuối
dsh(f"rm -f {CTL}/log"); em21=newuser('m21'); setmode('drop'); time.sleep(8)   # ≥ 2 lần đầu hỏng (1 s, 6 s)
setmode('ok'); r21=wait(lambda: (lambda r: r if '|SENT|' in r else None)(mail_row(em21)),90)
acc=flog().count('ACCEPTED')
rec('TC-P201-21','P' if r21 and acc==1 else 'F',f'SMTP drop 8 s rồi bật lại (trước lần thử cuối ở ≈36 s): mail_outbox {r21}; SMTP giả nhận ACCEPTED {acc} lần (đúng 1 thư, không gửi đôi)')
# ---- P201-29: 550 → DEAD ngay lần đầu; 451 → thử lại; mẫu lạ → DEAD ngay
setmode('550'); dsh(f"rm -f {CTL}/log"); em29=newuser('m29'); r29=wait(lambda: (lambda r: r if '|DEAD|' in r else None)(mail_row(em29)),30); c550=flog().count('RCPT->550')
lg29=dsh("docker logs qc2i-wf 2>&1 | grep -c '"+em29+"'")
setmode('451'); dsh(f"rm -f {CTL}/log"); em451=newuser('m451'); time.sleep(20); r451a=mail_row(em451); c451=flog().count('RCPT->451'); r451=wait(lambda: (lambda r: r if '|DEAD|' in r else None)(mail_row(em451)),90); c451b=flog().count('RCPT->451')
# mẫu không tồn tại / email không hợp lệ
setmode('ok'); mid_bad=psql(f"insert into mail_outbox(to_addr,template,payload) values('qc2ia-tpl.{TAG}@example.test','mau_khong_ton_tai','{{}}') returning id").split('\n')[0]
psql(f"insert into outbox(topic,payload) values('mail.send',jsonb_build_object('mail_id','{mid_bad}'))"); rbad=wait(lambda: (lambda r: r if '|DEAD|' in r else None)(psql(f"select id,status,attempts,coalesce(last_error,'') from mail_outbox where id='{mid_bad}'")),40)
rec('TC-P201-29','P' if r29 and r29.split('|')[2]=='1' and c550==1 and r451 and c451>=2 and rbad and rbad.split('|')[2] in('0','1') and lg29=='0' else 'F',f'SMTP giả 550: DEAD ngay lần đầu (attempts={r29.split("|")[2] if r29 else None}, {c550} kết nối), last_error={r29.split("|")[3][:60] if r29 else None!r}, log worker chứa email={lg29} lần; 451: sau 20 s attempts đã thử {c451} lần (thử lại như AC6), cuối cùng {r451.split("|")[1] if r451 else None} sau {c451b} kết nối; mẫu không tồn tại → {rbad}')
# ---- P201-23: giao hai lần cùng mail.send → 1 thư
setmode('ok'); dsh(f"rm -f {CTL}/log"); em23=newuser('m23'); r23=wait(lambda: (lambda r: r if '|SENT|' in r else None)(mail_row(em23)),40); mid23=psql(f"select id from mail_outbox where to_addr='{em23}'"); oid=psql(f"select id from outbox where topic='mail.send' and payload->>'mail_id'='{mid23}'")
for _ in range(3): rds('xadd','outbox.dispatch','*','outbox_id',oid,'topic','mail.send')
time.sleep(8); acc23=flog().count('ACCEPTED')
dk=psql(f"select count(*) from mail_outbox where dedupe_key is not null and to_addr='{em23}'")
rec('TC-P201-23','P' if r23 and acc23==1 else 'F',f'mail đã SENT ({r23}); XADD lại 3 lần cùng outbox_id vào outbox.dispatch → SMTP giả vẫn ACCEPTED {acc23} lần (khoá dòng mail_outbox.id + SENT chặn); dedupe_key chặn xếp hai thư cùng sự kiện: go test TestDedupeKey (xanh); kill -9 worker giữa lúc gửi + XAUTOCLAIM chưa làm (cần worker thứ hai; TC cho phép chọn XADD lại)')
# ---- dọn: trả stack
dsh("docker rm -f qc2i-wf qc2i-fakesmtp >/dev/null 2>&1; docker start qc2i-worker >/dev/null 2>&1")
print('xong')
