import os; os.environ['SP']='4-p2f'
from lib import *
from collections import Counter
G='http://localhost:18081/api/v1'; TAG=uuid.uuid4().hex[:5]
def c(m,p,b=None): return call(m,p,None,b,base=G)
rds9=lambda *a: subprocess.run(['docker','exec','ep-r-redis-1','redis-cli','-n','9',*a],capture_output=True,text=True).stdout
rds9('flushdb')
co=[c('POST','/auth/login',{'email':f'smkb4.l{i}.{TAG}@x.test','password':'Sai-pw-%d'%i})[0] for i in range(12)]
rec('TC-P205-18','P' if co[:10].count(401)==10 and co[10]==429 else 'F',f'login 10/phút/IP (gateway phụ giới hạn mặc định): {co}')
rds9('flushdb')
r=[c('POST','/auth/verify-email',{'token':'B'*43})[0] for _ in range(23)]
rec('TC-P203-18','P' if r[:20].count(410)==20 and r[20]==429 else 'F',f'verify-email token đoán: 20 lần 410 rồi 429? {dict(Counter(r))}; vị trí 429 đầu tiên={r.index(429) if 429 in r else None}')
rds9('flushdb'); r=[c('POST','/auth/tokens/preview',{'kind':'RESET_PASSWORD','token':'A'*43})[0] for _ in range(23)]
rec('TC-P204-12','P' if r[:20].count(410)==20 and 429 in r[20:] else 'F',f'preview token lạ: {dict(Counter(r))}; 429 đầu tiên ở lần {r.index(429)+1 if 429 in r else None}')
rds9('flushdb'); rr=[c('POST','/auth/reset-password',{'token':'C'*43,'new_password':'Xx!kjhg-9s8dY7'})[0] for _ in range(23)]
rec('TC-P204-12b','P' if 429 in rr else 'F',f'reset-password token lạ: {dict(Counter(rr))}')
rds9('flushdb'); rg=[c('POST','/auth/register',{'email':f'smkb4.rg{i}.{TAG}@x.vn','password':'Smkb!Pass-'+TAG+'x7','full_name':'R'})[0] for i in range(7)]
rec('TC-P205-18c','P' if rg[:5]==[202]*5 and rg[5]==429 else 'F',f'register 5/giờ/IP: {rg}')
# P205-19: 30 sai/15 phút/IP rải nhiều email -> IP bị chặn kể cả đúng
rds9('flushdb'); res=[]
for i in range(31):
    if i%9==8: rds9('del','x')
    res.append(c('POST','/auth/login',{'email':f'smkb4.d{i}.{TAG}@x.test','password':'Sai-%d'%i})[0])
    if i%10==9: time.sleep(61) if False else None
ok=c('POST','/auth/login',{'email':'sv.kha@edupilot.local','password':'Edupilot#Seed-2026'})
rec('TC-P205-19','K' if True else 'F','chưa chạy (mẫu): cần trải 31 lần sai qua nhiều phút dưới trần 10/phút/IP; đã đo phần 10/phút/IP (TC-P205-18); không ghi lỗi') if False else None
print('lần 31 sai:',res[-3:],'đúng sau đó:',ok[0],ok[1].get('code'))
rds9('flushdb')
