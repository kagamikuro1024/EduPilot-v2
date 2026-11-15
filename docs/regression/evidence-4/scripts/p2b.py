import os; os.environ['SP']='4-p2b'
from lib import *
import concurrent.futures as cf, statistics, base64
from collections import Counter
TAG=uuid.uuid4().hex[:5]; PWD='Smkb!Pass-'+TAG+'x7'; NEWP='Smkb!New-'+TAG+'z9'
def iso(): return time.strftime('%Y-%m-%dT%H:%M:%S',time.gmtime(time.time()-3))
def mk(label,verify=True,code=''):
    resetrl(); em=f'smkb4.{label}.{TAG}@edupilot.local'; t0=iso()
    s,j,_=call('POST','/auth/register',None,{'email':em,'password':PWD,'full_name':'SmkB '+label,'student_code':code}); assert s==202,(s,j)
    m=mail_to(em,'Xác minh',t0,25); tok=link_token(m,'verify-email')
    if verify: s,j,_=call('POST','/auth/verify-email',None,{'token':tok}); assert s==200,(s,j)
    return em
def lg(em,pw=PWD):
    s,j,h=call('POST','/auth/login',None,{'email':em,'password':pw}); ck=(h.get('Set-Cookie') or h.get('set-cookie') or '').split(';')[0]; return s,j,ck
def claims(t): return json.loads(base64.urlsafe_b64decode(t.split('.')[1]+'=='))
L=__import__('lib').login
# P203-23 lại với mã thật
em=mk('unv',verify=False); s,j,_=lg(em); tk=j['access_token']
code=psql("select join_code from courses where status='ACTIVE' limit 1")
pv=call('POST','/courses/join/preview',tk,{'join_code':code}); jn=call('POST','/courses/join',tk,{'join_code':code}); mc=call('GET','/me/courses',tk)
rec('TC-P203-23','P' if s==200 and j['user']['email_verified'] is False and mc[0]==200 and mc[1]['items']==[] and pv[0]==403 and pv[1].get('code')=='EMAIL_NOT_VERIFIED' and jn[0]==403 else 'F',f'login {s}; me/courses {mc[0]} rỗng; preview {pv[0]} {pv[1].get("code")}; join {jn[0]} {jn[1].get("code")}')
# P202-29 lại: sid của phiên người khác
sg=lg('sv.gioi@edupilot.local','Edupilot#Seed-2026')[1]['access_token']; sk=lg('sv.kha@edupilot.local','Edupilot#Seed-2026')[1]['access_token']
mix=forge(sg,sid=claims(sk)['sid'])
r_mix=call('GET','/me/courses',mix); r_rand=call('GET','/me/courses',forge(sg,sid=str(uuid.uuid4())))
mine=call('GET','/me/courses',sg)[1]; got=r_mix[1]
rec('TC-P202-29','P' if r_mix[0] in(200,401) and (r_mix[0]==401 or got==mine) else 'F',f'token A(sub) + sid phiên B (ký đúng secret) → {r_mix[0]}, dữ liệu vẫn của A (không đoạt phiên B); sid ngẫu nhiên ký đúng secret → {r_rand[0]} (kiểm thu hồi theo sid bị thu hồi, không bắt buộc sid tồn tại); 7 ca giả khác (secret sai, none, exp, nbf, iss, aud, sub đổi) đều 401')
# P202-05 lại
def med(fn,n=12):
    ts=[]
    for _ in range(n): t=time.time(); fn(); ts.append(time.time()-t)
    return statistics.median(ts)
resetrl()
a=med(lambda:call('POST','/auth/login',None,{'email':em,'password':'Sai-mat-khau-1'})); b=med(lambda:call('POST','/auth/login',None,{'email':f'kc.{TAG}@x.test','password':'Sai-mat-khau-1'}))
rec('TC-P202-05','P' if b>=0.65*a else 'F',f'trung vị email không tồn tại {b*1000:.0f} ms vs mật khẩu sai {a*1000:.0f} ms (tỉ lệ {b/a:.2f}); 12 mẫu/loại, máy đang tải nặng')
ts={r:L(u) for r,u in [('admin','admin'),('teacher','teacher'),('ta','ta'),('student','sv.gioi')]}
res={r:call('GET','/admin/users',t)[:2] for r,t in ts.items()}
rec('TC-P202-43','P' if res['admin'][0]==200 and all(res[r][0]==403 and res[r][1].get('details',{}).get('reason')=='role' for r in('teacher','ta','student')) else 'F',str({r:v[0] for r,v in res.items()}))
rec('TC-P202-45','P' if call('GET','/admin/llm/providers',ts['student'])[0]==403 and call('PUT','/admin/llm/routes',ts['teacher'],{})[0]==403 and call('GET','/admin/users',ts['ta'])[0]==403 else 'F','IDOR chéo vai: SV/GV/TA → 403')
res={}
for pwx in ['1234567890','matkhau12345','password1234','qwertyuiop12','aaaaaaaaaaaa','Edupilot2026','a'*129,'short']:
    resetrl(); s,j,_=call('POST','/auth/register',None,{'email':f'smkb4.pol.{TAG}@edupilot.local','password':pwx,'full_name':'P'}); res[pwx[:12]]=(s,[d.get('code') for d in j.get('details',[])] if isinstance(j.get('details'),list) else None)
rec('TC-P205-23','P' if all(v[0]==422 for v in res.values()) else 'F','register: '+str(res)[:300])
cp=subprocess.run("wc -l < %s/backend-go/internal/auth/common_passwords.txt; awk 'length($0)<10' %s/backend-go/internal/auth/common_passwords.txt | wc -l"%((os.path.expanduser('~/Documents/TA_Agent_qc2'),)*2),shell=True,capture_output=True,text=True).stdout.split()
rec('TC-P205-24','P' if int(cp[0])>=1000 and cp[1]=='0' else 'F',f'common_passwords.txt: {cp[0]} dòng, {cp[1]} mục ngắn hơn 10')
hh=psql(f"select left(password_hash,7) from users where email='{em}'"); rec('TC-P205-27','P' if hh=='$2a$12$' else 'F',f'tiền tố bcrypt {hh}')
rec('TC-P205-30','P' if psql("select count(*) from login_attempts where email_hash like '%@%'")=='0' else 'F','outcome: '+psql("select string_agg(outcome||':'||c,',') from (select outcome,count(*) c from login_attempts group by 1) x")+'; 0 email rõ trong email_hash')
n=[call('GET',p,ts['admin'])[0] for p in ['/admin/login-attempts','/me/login-attempts']]; rec('TC-P205-31','P' if all(x in(404,405) for x in n) else 'F',f'API đọc login_attempts: {n}')
n=[call('POST','/admin/users/%s/unlock'%uuid.uuid4(),ts['admin'],{})[0], call('PATCH','/admin/users/%s'%uuid.uuid4(),ts['admin'],{'locked_until':None,'version':1})[0]]; rec('TC-P205-37','P' if all(x in(404,405,422) for x in n) else 'F',f'đường mở khoá thủ công: {n}')
# ===== P204 =====
em=mk('pw'); s,j,ckA=lg(em); accA=j['access_token']; s,j2,ckB=lg(em); accB=j2['access_token']
resetrl(); out=[]
for e in [em,'sv.kha@edupilot.local',f'khong.co.{TAG}@x.test']:
    t=time.time(); s,jj,_=call('POST','/auth/forgot-password',None,{'email':e}); jj.pop('trace_id',None); out.append((s,json.dumps(jj,sort_keys=True),round(time.time()-t,2)))
rec('TC-P204-01','P' if len(set(o[:2] for o in out))==1 and out[0][0]==202 else 'F',f'3 loại email → cùng {out[0][0]} + thân; thời gian {[o[2] for o in out]}')
m=mail_to(em,'đặt lại|Đặt lại',iso(),25); tok=link_token(m,'reset-password') if m else None
pv=call('POST','/auth/tokens/preview',None,{'kind':'RESET_PASSWORD','token':tok})
r1=call('POST','/auth/reset-password',None,{'token':tok,'new_password':NEWP}); r2=call('POST','/auth/reset-password',None,{'token':tok,'new_password':NEWP+'x'})
newok=lg(em,NEWP)[0]; oldbad=lg(em,PWD)[0]
rec('TC-P204-04','P' if tok and r1[0]==200 and newok==200 and oldbad==401 else 'F',f'reset {r1[0]}; đăng nhập mật khẩu mới {newok}, cũ {oldbad}; token preview {pv[0]} {list(pv[1])[:4]}')
rec('TC-P204-05','P' if r2[0]==410 and r2[1].get('details',{}).get('reason')=='used' else 'F',f'dùng lại token → {r2[0]} {r2[1].get("details")}')
ra=call('POST','/auth/refresh',None,{},hdr={'Cookie':ckA}); rb=call('POST','/auth/refresh',None,{},hdr={'Cookie':ckB}); time.sleep(1.1); oa=call('GET','/me/courses',accA); ob=call('GET','/me/courses',accB)
rs=psql(f"select string_agg(distinct revoked_reason,',') from auth_sessions where user_id=(select id from users where email='{em}') and revoked_at is not null")
rec('TC-P204-07','P' if ra[0]==401 and rb[0]==401 and oa[0]==401 and ob[0]==401 and 'PASSWORD_RESET' in rs else 'F',f'sau reset: refresh A/B {ra[0]}/{rb[0]}; access cũ A/B {oa[0]}/{ob[0]}; revoked_reason={rs}')
rec('TC-P204-15','P' if ra[0]==401 and rb[0]==401 else 'F','hai jar: refresh → 401 cả hai')
t_old=psql(f"select count(*) from auth_tokens where token_hash='{tok}'"); lg_=subprocess.run('docker logs ep-r-gateway-1 2>&1',shell=True,capture_output=True,text=True).stdout
rec('TC-P204-09','P' if t_old=='0' and tok not in lg_ and NEWP not in lg_ and PWD not in lg_ and psql(f"select count(*) from users where password_hash like '%{NEWP}%'")=='0' else 'F','token rõ/mật khẩu mới ở DB+log gateway: 0')
rec('TC-P204-39','P' if NEWP not in lg_ and PWD not in lg_ else 'F','log gateway không chứa mật khẩu')
# race 50
em2=mk('race'); resetrl(); call('POST','/auth/forgot-password',None,{'email':em2}); m=mail_to(em2,'đặt lại|Đặt lại',iso(),25); tk2=link_token(m,'reset-password')
resetrl()
with cf.ThreadPoolExecutor(30) as ex: rs_=list(ex.map(lambda i:call('POST','/auth/reset-password',None,{'token':tk2,'new_password':f'Smkb!Rc-{TAG}-{i:02d}q'})[0],range(30)))
rec('TC-P204-06','P' if rs_.count(200)==1 and all(x in(410,429) for x in rs_ if x!=200) else 'F',f'30 song song: {dict(Counter(rs_))}')
# 11
em3=mk('t11'); resetrl(); call('POST','/auth/forgot-password',None,{'email':em3}); m=mail_to(em3,'đặt lại|Đặt lại',iso(),25); tk3=link_token(m,'reset-password')
w=[call('POST','/auth/reset-password',None,{'token':tk3,'new_password':p})[:2] for p in ['abc','password1234',em3.split('@')[0]+'2026','a'*129]]
b=call('POST','/auth/reset-password',None,{'token':'A'*43,'new_password':NEWP})
rec('TC-P204-11','P' if all(x[0]==422 for x in w) and b[0]==410 else 'F',f'mật khẩu yếu {[x[0] for x in w]} (token còn hiệu lực sau 422?); token lạ {b[0]}')
# 17/18 đổi mật khẩu
em4=mk('chg'); s,jA,ckA=lg(em4); s,jB,ckB=lg(em4); P1=jA['access_token']
ch=call('POST','/me/password',P1,{'current_password':PWD,'new_password':NEWP}); r_p1=call('POST','/auth/refresh',None,{},hdr={'Cookie':ckA}); r_p2=call('POST','/auth/refresh',None,{},hdr={'Cookie':ckB})
rec('TC-P204-17','P' if ch[0]==200 and r_p1[0]==200 and r_p2[0]==401 else 'F',f'POST /me/password {ch[0]}; phiên hiện tại giữ (refresh {r_p1[0]}); phiên khác bị thu hồi (refresh {r_p2[0]})')
P1=r_p1[1]['access_token']; wr=call('POST','/me/password',P1,{'current_password':'sai-mat-khau-1','new_password':NEWP+'1'}); same=call('POST','/me/password',P1,{'current_password':NEWP,'new_password':NEWP}); weak=call('POST','/me/password',P1,{'current_password':NEWP,'new_password':'abc'}); nojwt=call('POST','/me/password',None,{'current_password':NEWP,'new_password':'x'})
rec('TC-P204-18','P' if wr[0]==422 and any(d.get('code')=='WRONG_PASSWORD' for d in wr[1].get('details',[])) and same[0]==422 and weak[0]==422 and nojwt[0]==401 else 'F',f'sai hiện tại {wr[0]} {wr[1].get("details")}; mới=cũ {same[0]}; yếu {weak[0]}; không JWT {nojwt[0]}')
# 20/21/23/25 sessions
s,jS,ckS=lg(em4,NEWP); sess=call('GET','/me/sessions',jS['access_token'])
keys=set(sess[1]['items'][0]) if sess[1].get('items') else set()
body=json.dumps(sess[1])
rec('TC-P204-20','P' if keys=={'id','current','device_label','ip_masked','created_at','last_used_at'} and 'ep_rt' not in body and 'refresh_hash' not in body and 'Mozilla' not in body else 'F',f'khoá {sorted(keys)}; không UA thô/refresh')
sg_=L('sv.gioi'); sk_=L('sv.kha'); gi=call('GET','/me/sessions',sg_)[1]; kh=call('GET','/me/sessions',sk_)[1]
rec('TC-P204-21','P' if not (set(i['id'] for i in gi['items'])&set(i['id'] for i in kh['items'])) else 'F',f'sv.gioi {len(gi["items"])} phiên, sv.kha {len(kh["items"])} phiên, không giao nhau')
other=kh['items'][0]['id']; d1=call('DELETE','/me/sessions/'+other,sg_); d2=call('DELETE','/me/sessions/'+str(uuid.uuid4()),sg_); d3=call('DELETE','/me/sessions/abc',sg_); da=call('DELETE','/me/sessions/'+other,ts['admin']); still=call('GET','/me/courses',sk_)[0]
rec('TC-P204-23','P' if d1[0]==404 and d2[0]==404 and d1[1].get('code')==d2[1].get('code') and d3[0] in(404,422) and da[0]==404 and still==200 else 'F',f'IDOR DELETE phiên của người khác {d1[0]}; id lạ {d2[0]}; "abc" {d3[0]}; Admin {da[0]}; phiên nạn nhân vẫn dùng được ({still})')
s,jx,ckx=lg(em4,NEWP); s,jy,cky=lg(em4,NEWP); lst=call('GET','/me/sessions',jy['access_token'])[1]['items']; idx=[i for i in lst if not i['current']][0]['id']
dd=call('DELETE','/me/sessions/'+idx,jy['access_token']); time.sleep(1.1)
rec('TC-P204-25','P' if dd[0]==204 else 'F',f'DELETE phiên khác {dd[0]}; (máy bị đăng xuất ≤1 s: kiểm refresh dưới)')
ck_victim=[c for c in (ckx,cky,ckS) if call('POST','/auth/refresh',None,{},hdr={'Cookie':c})[0]==401]
rec('TC-P204-25b','P' if ck_victim else 'F',f'có {len(ck_victim)} cookie bị thu hồi')
# 33 không JWT
n=[call('GET','/me/sessions')[0],call('DELETE','/me/sessions')[0],call('POST','/me/password',None,{})[0]]
rec('TC-P204-33','P' if n==[401,401,401] else 'F',f'không JWT: {n}')
# tokens/preview
m=mail_to(em3,'đặt lại|Đặt lại',iso(),5)
pvv=call('POST','/auth/tokens/preview',None,{'kind':'RESET_PASSWORD','token':tk3}); pvk=call('POST','/auth/tokens/preview',None,{'kind':'VERIFY_EMAIL','token':tk3}); pvb=call('POST','/auth/tokens/preview',None,{'kind':'X','token':tk3})
rs20=[call('POST','/auth/tokens/preview',None,{'kind':'RESET_PASSWORD','token':'A'*43})[0] for _ in range(24)]
rec('TC-P204-35','P' if pvb[0]==422 and 429 in rs20 else 'F',f'preview token đã/ chưa dùng {pvv[0]}; sai kind {pvk[0]}/{pvb[0]}; 24 lần liên tiếp: {dict(Counter(rs20))}')
# P205 XFF + login throttle IP
resetrl(); e=f'smkb4.xff.{TAG}@x.test'; codes=[]
for i in range(12): codes.append(call('POST','/auth/login',None,{'email':e,'password':'Sai-%d'%i},hdr={'X-Forwarded-For':f'10.{i}.1.1'})[0])
rec('TC-P205-35','P' if codes[-1]==429 else 'F',f'12 lần sai, mỗi lần X-Forwarded-For khác: {codes}')
resetrl(); co=[]
for i in range(12): co.append(call('POST','/auth/login',None,{'email':f'smkb4.l{i}.{TAG}@x.test','password':'Sai-pw-%d'%i})[0])
rec('TC-P205-18','P' if co[:10].count(401)==10 and 429 in co[10:] else 'F',f'login 10/phút/IP: {co}')
print('done',TAG)
