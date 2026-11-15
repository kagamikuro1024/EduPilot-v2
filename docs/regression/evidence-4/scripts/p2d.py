import os; os.environ['SP']='4-p2d'
from lib import *
from collections import Counter
TAG=uuid.uuid4().hex[:5]
# P203-23 lại với field đúng
resetrl()
def mk(label):
    resetrl(); em=f'smkb4.{label}.{TAG}@edupilot.local'; t0=time.strftime('%Y-%m-%dT%H:%M:%S',time.gmtime(time.time()-3))
    s,j,_=call('POST','/auth/register',None,{'email':em,'password':'Smkb!Pass-'+TAG+'x7','full_name':'SmkB','student_code':''}); assert s==202
    return em
em=mk('unv2'); s,j,h=call('POST','/auth/login',None,{'email':em,'password':'Smkb!Pass-'+TAG+'x7'}); tk=j['access_token']
code=psql("select join_code from courses where status='ACTIVE' limit 1")
pv=call('POST','/courses/join/preview',tk,{'code':code}); jn=call('POST','/courses/join',tk,{'code':code}); mc=call('GET','/me/courses',tk)
rec('TC-P203-23','P' if j['user']['email_verified'] is False and mc[0]==200 and mc[1]['items']==[] and pv[0]==403 and pv[1].get('code')=='EMAIL_NOT_VERIFIED' and jn[0]==403 else 'F',f'login 200 email_verified=False; me/courses rỗng; preview {pv[0]} {pv[1].get("code")}; join {jn[0]} {jn[1].get("code")}')
# preview rate limit: cách ly
time.sleep(2); resetrl(); r=[call('POST','/auth/tokens/preview',None,{'kind':'RESET_PASSWORD','token':'A'*43})[0] for _ in range(25)]
rec('TC-P204-12','P' if 429 in r else 'F',f'25 lần preview token lạ từ 1 IP: {dict(Counter(r))} (kỳ vọng 429 sau 20/phút/IP; không trúng token)') 
resetrl(); co=[]
for i in range(12): co.append(call('POST','/auth/login',None,{'email':f'smkb4.l{i}.{TAG}@x.test','password':'Sai-pw-%d'%i})[0])
rec('TC-P205-18','P' if co[:10].count(401)==10 and 429 in co[10:] else 'F',f'login 10/phút/IP (cô lập): {co}')
resetrl(); r=[call('POST','/auth/verify-email',None,{'token':'B'*43})[0] for _ in range(25)]
rec('TC-P203-18','P' if 429 in r and 200 not in r else 'F',f'25 token đoán verify-email từ 1 IP: {dict(Counter(r))}')
resetrl(); r=[call('POST','/auth/forgot-password',None,{'email':f'smkb4.f{i}.{TAG}@x.test'})[0] for i in range(7)]
rec('TC-P205-18b','P' if 429 in r else 'F',f'forgot-password 5/giờ/IP: {r}')
