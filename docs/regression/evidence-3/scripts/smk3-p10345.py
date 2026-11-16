# TC-P103-45: dừng Redis (smk3-redis) giữa lúc bơm 10 req/s vào smk3-gw1 (≈ 28 s): đếm thành công theo giây, log error, llm_audit, quay lại toàn cục không restart
from smk3iso import *
V='http://host.docker.internal:9751/v1'
P=imkprov('SmkI-N9',V,['m-ok']); iput('CHAT',[mid(P,'m-ok')],{'retries':0}); time.sleep(2)
ird('del','ep:llm:rl:rpm:'+P['id'],'ep:llm:rl:tpm:'+P['id'])
n0=int(ipsql("select count(*) from llm_audit where task='CHAT'") or 0)
res=[]; lk=threading.Lock(); pool=cf.ThreadPoolExecutor(40)
def one(sec):
    r=ichat('CHAT'); 
    with lk: res.append((sec,r[0],r[2]))
t0=time.time(); ev=[]
for sec in range(30):
    for _ in range(10): pool.submit(one,sec)
    if sec==6: subprocess.Popen(['docker','stop','smk3-redis'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL); ev.append(('stop',sec))
    if sec==18: subprocess.Popen(['docker','start','smk3-redis'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL); ev.append(('start',sec))
    time.sleep(max(0,t0+sec+1-time.time()))
pool.shutdown(wait=True); time.sleep(4)
from collections import Counter,defaultdict
by=defaultdict(Counter)
for sec,s,ms in res: by[sec][s]+=1
print('sự kiện',ev); print('theo giây (mã→số):',{k:dict(v) for k,v in sorted(by.items())})
print('tổng',len(res),Counter(s for _,s,_ in res),'độ trễ tối đa ms',max(ms for *_,ms in res))
n1=int(ipsql("select count(*) from llm_audit where task='CHAT'")); print('llm_audit CHAT +',n1-n0,'(đã gửi',len(res),')')
lg=[l for l in gwlogs('120s') if '"level":"ERROR"' in l or '"level":"WARN"' in l]
print('log WARN/ERROR gw1 trong 120 s:',len(lg)); [print('  ',l[:230]) for l in lg[:12]]
# Redis trở lại: ZCARD có lại (không restart gateway)
time.sleep(1); stubl=[ichat('CHAT') for _ in range(3)]; print('sau khi Redis về: 3 lời gọi',[r[0] for r in stubl],'khoá bucket',ird('exists','ep:llm:rl:rpm:'+P['id']))
# đo ZCARD trong lúc có lời gọi chậm: dùng nhà ngủ m-slow1 không cần; chỉ kiểm khoá inflight được tạo khi lời gọi chạy
print('ZCARD sau cùng',ird('zcard','ep:llm:inflight:'+P['id']))
