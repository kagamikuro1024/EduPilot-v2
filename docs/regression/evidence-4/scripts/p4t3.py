import os; os.environ['SP']='4-t3'
from smkT import *
import hashlib,concurrent.futures as cf
from collections import Counter
B2='http://127.0.0.1:18241/api/v1'
def ck(h): return (h.get('Set-Cookie') or h.get('set-cookie') or '').split(';')[0]
def cv(c): return c.split('=',1)[1] if '=' in c else ''
def sha(x): return hashlib.sha256(x.encode()).hexdigest()
def session_of(em,cookie):
    return psql(f"select id from auth_sessions where refresh_hash='{sha(cv(cookie))}'")
em,tk=stu('s202')   # SV đã xác minh
uid_=uid(em)
# ---- P202-02
resetrl(); n0=psql(f"select count(*) from auth_sessions where user_id='{uid_}'"); l0=psql(f"select coalesce(last_login_at::text,'null') from users where id='{uid_}'")
s,j,h=call('POST','/auth/login',None,{'email':em,'password':NPW}); c1=ck(h); seen(cv(c1))
n1=psql(f"select count(*) from auth_sessions where user_id='{uid_}'"); l1=psql(f"select last_login_at::text from users where id='{uid_}'")
rec('TC-P202-02','P' if s==200 and int(n1)==int(n0)+1 and l1!=l0 else 'F',f'login {s}: auth_sessions {n0}→{n1}; last_login_at {l0[:19]}→{l1[:19]}')
# ---- P202-08 băm refresh khi xoay
old=cv(c1); resetrl(); s,j,h=call('POST','/auth/refresh',None,{},hdr={'Cookie':c1}); c2=ck(h); seen(cv(c2))
row=psql(f"select (refresh_hash='{sha(cv(c2))}'),(prev_refresh_hash='{sha(old)}'),rotated_at is not null,last_used_at>created_at from auth_sessions where user_id='{uid_}' and refresh_hash='{sha(cv(c2))}'")
plain=psql(f"select count(*) from auth_sessions where refresh_hash='{old}' or prev_refresh_hash='{old}' or refresh_hash='{cv(c2)}'")
dump=sh(f"docker exec ep-r-postgres-1 pg_dump -U edupilot -d edupilot --data-only -t auth_sessions 2>/dev/null | grep -c -e '{old}' -e '{cv(c2)}'").strip()
lg_=sum(subprocess.run(['docker','logs',n],capture_output=True,text=True).stdout.count(old)+subprocess.run(['docker','logs',n],capture_output=True,text=True).stderr.count(old) for n in ('ep-r-gateway-1','ep-r-caddy-1'))
rec('TC-P202-08','P' if s==200 and row=='t|t|t|t' and plain=='0' and dump=='0' and lg_==0 else 'F',f'refresh {s}; DB: hash mới/prev=sha256(cũ)/rotated_at/last_used = {row}; bản rõ ở auth_sessions={plain}, pg_dump bảng={dump}, log gateway+caddy={lg_}')
# ---- P202-09 trượt + trần tuyệt đối (đồng hồ test = chỉnh DB, ghi rõ)
sid=session_of(em,c2)
r=psql(f"select round(extract(epoch from expires_at-now())/3600),round(extract(epoch from absolute_expires_at-now())/3600) from auth_sessions where id='{sid}'")
resetrl(); s,j,h=call('POST','/auth/refresh',None,{},hdr={'Cookie':c2}); c3=ck(h)
r2=psql(f"select round(extract(epoch from expires_at-now())/3600),round(extract(epoch from absolute_expires_at-now())/3600),expires_at<=absolute_expires_at from auth_sessions where id='{sid}'")
# trần: đặt absolute = now+1h (expires đồng thời ≤ absolute) → refresh kế tiếp: expires_at bị kẹp ≤ absolute
psql(f"update auth_sessions set absolute_expires_at=now()+interval '1 hour', expires_at=now()+interval '30 minutes' where id='{sid}'")
resetrl(); s4,j4,h4=call('POST','/auth/refresh',None,{},hdr={'Cookie':c3}); c4=ck(h4)
r3=psql(f"select round(extract(epoch from expires_at-now())/60),round(extract(epoch from absolute_expires_at-now())/60),expires_at<=absolute_expires_at from auth_sessions where id='{sid}'")
psql(f"update auth_sessions set absolute_expires_at=now()-interval '1 second', expires_at=now()-interval '2 second' where id='{sid}'")
resetrl(); s5,j5,_=call('POST','/auth/refresh',None,{},hdr={'Cookie':c4 or c3})
rec('TC-P202-09','P' if s==200 and r2.endswith('|t') and s4==200 and r3.endswith('|t') and s5==401 and j5.get('code') in('TOKEN_INVALID','SESSION_REVOKED','TOKEN_EXPIRED') else 'F',f'refresh trượt expires_at +{r2.split("|")[0]}h (mặc định 336h), trần tuyệt đối {r2.split("|")[1]}h (720h); kẹp trần: absolute=now+1h → sau refresh expires={r3.split("|")[0]} phút ≤ absolute={r3.split("|")[1]} phút; hết trần (chỉnh DB, ghi rõ) → refresh {s5} {j5.get("code")}')
# ---- P202-12/13/14 dùng lại refresh
def fresh(label):
    e,_=stu(label); resetrl(); s,j,h=call('POST','/auth/login',None,{'email':e,'password':NPW}); return e,j['access_token'],ck(h)
# 12: kẻ trộm xoay trước
e12,acc12,t1=fresh('r12'); resetrl(); sa,ja,ha=call('POST','/auth/refresh',None,{},hdr={'Cookie':t1}); t2=ck(ha)
resetrl(); sb,jb,_=call('POST','/auth/refresh',None,{},hdr={'Cookie':t1})   # nạn nhân mang T1 đã bị xoay
sid12=psql(f"select revoked_reason from auth_sessions where user_id=(select id from users where email='{e12}') order by created_at desc limit 1")
resetrl(); sc,jc_,_=call('POST','/auth/refresh',None,{},hdr={'Cookie':t2})
# thứ tự ngược: nạn nhân xoay trước, kẻ trộm mang T1
e12b,acc12b,u1=fresh('r12b'); resetrl(); call('POST','/auth/refresh',None,{},hdr={'Cookie':u1}); resetrl(); sd,jd,_=call('POST','/auth/refresh',None,{},hdr={'Cookie':u1})
rsn=psql(f"select revoked_reason from auth_sessions where user_id=(select id from users where email='{e12b}') order by created_at desc limit 1")
rec('TC-P202-12','P' if sa==200 and sb==401 and sid12=='REFRESH_REUSE' and sc==401 and sd==401 and rsn=='REFRESH_REUSE' else 'F',f'kẻ trộm xoay trước: T1→T2 {sa}; nạn nhân T1 {sb} {jb.get("code")}; cả phiên revoked_reason={sid12}; T2 sau đó {sc}; thứ tự ngược: T1 tái dùng {sd}, reason={rsn}')
t0=time.time(); sx,jx,_=call('GET','/me/courses',acc12); dt=time.time()-t0
a=psql(f"select count(*),min(details::text),min(entity),min(action) from audit_log where action='refresh_reuse' and entity_id in (select id::text from auth_sessions where user_id=(select id from users where email='{e12}'))")
a2=psql(f"select details::text from audit_log where action='refresh_reuse' order by created_at desc limit 1")
leak=(cv(t1) in a2) or (cv(t2) in a2) or (e12 in a2)
rec('TC-P202-13','P' if a.startswith('1|') and 'auth_session' in a and not leak else 'F',f'audit_log refresh_reuse cho phiên: {a[:160]}; details không chứa token/email={not leak}')
# 14: đua hai refresh cùng T1
e14,acc14,x1=fresh('r14'); resetrl()
def rf(_): 
    s,j,h=call('POST','/auth/refresh',None,{},hdr={'Cookie':x1}); return s,j.get('code'),ck(h)
with cf.ThreadPoolExecutor(2) as ex: rs=list(ex.map(rf,range(2)))
ok=[r for r in rs if r[0]==200]; valid=[r for r in rs if r[0]==200 and r[2]]
sess=psql(f"select count(*) filter (where revoked_at is null),count(*) from auth_sessions where user_id=(select id from users where email='{e14}')")
rec('TC-P202-14','P' if all(r[0]!=500 and r[0]!=0 for r in rs) and len(valid)<=1 else 'F',f'2 refresh song song cùng T1: {[(r[0],r[1]) for r in rs]}; số cookie hợp lệ sinh ra={len(valid)}; phiên chưa thu hồi/tổng={sess}; không 500')
# ---- P202-25 thu hồi ≤ 1 s ở cả hai gateway
def wait401(acc,base,limit=3.0):
    t0=time.time()
    while True:
        s,j,_=call('GET','/me/courses',acc,base=base,retry=False) if False else call('GET','/me/courses',acc,base=base)
        if s==401: return round(time.time()-t0,2),j.get('code')
        if time.time()-t0>limit: return None,s
        time.sleep(0.05)
res={}
for trig in ['logout','password','disabled','reuse']:
    e,acc,cookie=fresh('t25'+trig)
    s1=call('GET','/me/courses',acc)[0]; s2=call('GET','/me/courses',acc,base=B2)[0]
    t0=time.time()
    if trig=='logout': call('POST','/auth/logout',None,{},hdr={'Cookie':cookie})
    elif trig=='password': call('POST','/me/password',acc,{'current_password':NPW,'new_password':NPW+'z'}); 
    elif trig=='disabled': psql(f"update users set status='DISABLED' where email='{e}'")
    else:
        resetrl(); call('POST','/auth/refresh',None,{},hdr={'Cookie':cookie}); resetrl(); call('POST','/auth/refresh',None,{},hdr={'Cookie':cookie})
    a1=wait401(acc,B); a2=wait401(acc,B2)
    res[trig]=(s1,s2,a1,a2)
    if trig=='password' and a1[0] is not None: pass   # POST /me/password giữ phiên hiện tại: dùng phiên KHÁC để đo
ok25=all(v[0]==200 and v[1]==200 and v[2][0] is not None and v[2][0]<=1.0 and v[3][0] is not None and v[3][0]<=1.0 for k,v in res.items() if k!='password')
rec('TC-P202-25','P' if ok25 else 'F',f'(s1,s2,gw1,gw2) logout/disabled/reuse: '+'; '.join(f'{k}={v}' for k,v in res.items() if k!='password')+' (gateway thứ hai = ep-r-gateway-smkt:18241 cùng Redis/DB ep-r; đổi mật khẩu giữ phiên gọi → đo ở TC-P204-17/25 đã P)')
json.dump(res,open('/tmp/smkT/p202-25.json','w'),default=str)
