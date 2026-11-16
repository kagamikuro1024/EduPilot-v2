# TC-P103-05: 200 CHAT fake 200–400 ms vào MỖI gateway cùng lúc; lấy mẫu ZCARD ep:llm:inflight:<provider> mỗi ~3 ms
from smk3iso import *
F=mkfake(); print(iput('CHAT',[mid(F,'fake-chat')])[0]); time.sleep(1.5)
key='ep:llm:inflight:'+F['id']; peak=[0]; stop=[False]; samples=[0]
def sampler():
    r=RC()
    while not stop[0]:
        n=r.cmd('ZCARD',key); samples[0]+=1
        if n>peak[0]: peak[0]=n
        time.sleep(0.003)
th=threading.Thread(target=sampler); th.start()
res=[]; lk=threading.Lock()
def one(gw,i):
    r=ichat('CHAT',f'q{i}',gw=gw)
    with lk: res.append((gw,r[0],r[1].get('code'),r[2]))
t0=time.time()
with cf.ThreadPoolExecutor(400) as ex: list(ex.map(lambda a:one(*a),[('gw1' if i%2==0 else 'gw2',i) for i in range(400)]))
el=time.time()-t0; time.sleep(0.5); stop[0]=True; th.join()
from collections import Counter
print('400 yêu cầu (200 mỗi gateway) trong %.1fs'%el,Counter((r[1],r[2]) for r in res),'đỉnh ZCARD',peak[0],'số mẫu',samples[0])
time.sleep(2); r=RC(); print('ZCARD sau khi xong:',r.cmd('ZCARD',key))
lat=[r[3] for r in res if r[1]==200]; print('độ trễ 200: min %d median %d max %d ms'%(min(lat),sorted(lat)[len(lat)//2],max(lat)))
print(ipsql("select status,count(*) from llm_audit where provider='SmkI-F' and created_at>now()-interval '5 minutes' group by 1"))
