# TC-P102-24: khoá bảng llm_audit 12 s (chỉ chặn ghi audit) + 1.150 lời gọi nhanh => đệm 1.000 đầy, bỏ dòng cũ nhất, bộ đếm tăng, không OOM
from smk3iso import *
S1=imkprov('SmkI-S1',U1,MODELS); iput('CHAT',[mid(S1,'m-ok')]); ipsql("delete from llm_audit"); stubreset()
st0=ic('GET','/_test/llm/stats')[1]['audit']
def locker(): subprocess.run(['docker','exec','-i','smk3-pg','psql','-U','edupilot','-d','edupilot','-c',"begin; lock table llm_audit in access exclusive mode; select pg_sleep(16); commit;"],capture_output=True,text=True,timeout=60)
th=threading.Thread(target=locker); th.start(); time.sleep(1)
t0=time.time()
def one(i): return ichat(hdr=tp(f'7{i:031d}'))[0]
with cf.ThreadPoolExecutor(10) as ex: rs=list(ex.map(one,range(1700)))
el=time.time()-t0
st1=ic('GET','/_test/llm/stats')[1]['audit']
print('calls',len(rs),{c:rs.count(c) for c in set(rs)},'%.1fs'%el,st0,'->',st1)
th.join(); time.sleep(12)
n=ipsql("select count(*) from llm_audit where trace_id like '7%'"); print('rows trong DB sau khi nhả khoá',n)
print([l[:200] for l in gwlogs('120s') if 'llm_audit' in l][:3])
st2=ic('GET','/_test/llm/stats')[1]['audit']; print(st2)
