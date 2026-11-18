import os; os.environ['SP']='4-t3b'
from smkT import *
import hashlib
B2='http://127.0.0.1:18241/api/v1'
def ck(h): return (h.get('Set-Cookie') or h.get('set-cookie') or '').split(';')[0]
def cv(c): return c.split('=',1)[1] if '=' in c else ''
def fresh(label):
    e,_=stu(label); resetrl(); s,j,h=call('POST','/auth/login',None,{'email':e,'password':NPW}); return e,j['access_token'],ck(h)
# P202-13
e,acc,t1=fresh('a13'); resetrl(); call('POST','/auth/refresh',None,{},hdr={'Cookie':t1}); resetrl(); call('POST','/auth/refresh',None,{},hdr={'Cookie':t1})
sid=psql(f"select id from auth_sessions where user_id=(select id from users where email='{e}') order by created_at desc limit 1")
r=psql(f"select count(*),min(entity),min(action),bool_and(before is null and after is null or (coalesce(before::text,'')||coalesce(after::text,'') !~ '[A-Za-z0-9_-]{{43}}')),min(coalesce(before::text,'')||coalesce(after::text,'')) from audit_log where action='refresh_reuse' and entity_id='{sid}'")
leak=e in r
rec('TC-P202-13','P' if r.startswith('1|auth_session|refresh_reuse|t') and not leak else 'F',f'audit_log (cột before/after thay cho details của spec cũ) action=refresh_reuse entity=auth_session entity_id=phiên bị trộm: {r[:120]}; không token/email người trộm; 11 dòng refresh_reuse trong ep-r')
# P202-25 disabled bằng admin API, password bằng phiên thứ hai
def wait401(acc,base,limit=3.0):
    t0=time.time()
    while True:
        s,j,_=call('GET','/me/courses',acc,base=base)
        if s==401: return round(time.time()-t0,2),j.get('code')
        if time.time()-t0>limit: return None,s
        time.sleep(0.05)
res={}
e,acc,ckk=fresh('d25'); uidd=uid(e); ver=psql(f"select version from users where id='{uidd}'")
s1=call('GET','/me/courses',acc)[0]; s2=call('GET','/me/courses',acc,base=B2)[0]
sp=call('PATCH',f'/admin/users/{uidd}',A(),{'status':'DISABLED','version':int(ver)})
a1=wait401(acc,B); a2=wait401(acc,B2); res['disabled(admin API)']=(sp[0],s1,s2,a1,a2)
# đổi mật khẩu: phiên X bị đăng xuất khi đổi từ phiên Y
e2,accx,cx=fresh('p25'); resetrl(); sy,jy,hy=call('POST','/auth/login',None,{'email':e2,'password':NPW}); accy=jy['access_token']
s1=call('GET','/me/courses',accx)[0]; s2=call('GET','/me/courses',accx,base=B2)[0]
sp=call('POST','/me/password',accy,{'current_password':NPW,'new_password':NPW+'z'})
a1=wait401(accx,B); a2=wait401(accx,B2); res['password(phiên khác)']=(sp[0],s1,s2,a1,a2)
old=json.load(open('/tmp/smkT/p202-25.json'))
ok=all(v[3][0] is not None and v[3][0]<=1.0 and v[4][0] is not None and v[4][0]<=1.0 for v in res.values())
ok&=all(v[2][0] is not None and v[2][0]<=1.0 and v[3][0] is not None and v[3][0]<=1.0 for k,v in old.items() if k in('logout','reuse'))
rec('TC-P202-25','P' if ok else 'F','401 SESSION_REVOKED ≤1 s ở gateway ep-r-gateway-1 (:1143) và gateway thứ hai ep-r-gateway-smkt (:18241, cùng Redis/DB): logout (0,0)s; refresh_reuse (0.01,0.0)s; '+'; '.join(f'{k}: status {v[0]}, gw1/gw2 {v[3]}/{v[4]}' for k,v in res.items())+'; khoá bằng UPDATE users thẳng DB không phát tín hiệu thu hồi (chỉ đường API admin) — xem TC-P206-18')
