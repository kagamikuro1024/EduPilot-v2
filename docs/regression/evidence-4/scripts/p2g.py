import os; os.environ['SP']='4-p2g'
from lib import *
G='http://localhost:18081/api/v1'; TAG=uuid.uuid4().hex[:5]
def c(m,p,b=None): return call(m,p,None,b,base=G)
subprocess.run(['docker','exec','ep-r-redis-1','redis-cli','-n','9','flushdb'],capture_output=True)
res=[]
for minute in range(4):
    t=time.time()
    for i in range(10):
        res.append(c('POST','/auth/login',{'email':f'smkb4.e{minute}{i}.{TAG}@x.test','password':'Sai-%d'%i})[0])
    if minute<3: time.sleep(max(0,61-(time.time()-t)))
good=c('POST','/auth/login',{'email':'sv.kha@edupilot.local','password':'Edupilot#Seed-2026'})
rec('TC-P205-19','P' if res[:30].count(401)==30 and all(x==429 for x in res[30:]) and good[0]==429 else 'F',f'40 lần sai rải 4 phút (10/phút) từ 1 IP cho 40 email khác nhau: {res.count(401)}×401 rồi {res.count(429)}×429; đăng nhập ĐÚNG sau đó → {good[0]} {good[1].get("code")}')
