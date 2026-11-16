# TC-P102-25: gửi 50 lời gọi rồi SIGTERM gateway ngay => cả 50 dòng audit có mặt trước khi tiến trình thoát
from smk3iso import *
ipsql("delete from llm_audit where trace_id like '8%'")
def one(i): return ichat(gw='gw3',hdr=tp(f'8{i:031d}'))[0]
with cf.ThreadPoolExecutor(10) as ex: rs=list(ex.map(one,range(50)))
t0=time.time(); r=subprocess.run(['docker','stop','-t','20','smk3-gw3'],capture_output=True,text=True); el=time.time()-t0
n=ipsql("select count(*) from llm_audit where trace_id like '8%'")
code=subprocess.run(['docker','inspect','smk3-gw3','--format','{{.State.ExitCode}}'],capture_output=True,text=True).stdout.strip()
print('calls',{c:rs.count(c) for c in set(rs)},'docker stop %.1fs exit=%s rows=%s'%(el,code,n))
