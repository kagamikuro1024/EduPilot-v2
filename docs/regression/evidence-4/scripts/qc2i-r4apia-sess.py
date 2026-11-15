# TC-P202-02/08/09/12/13/14/25 (phiên, băm refresh, trượt/trần, kẻ trộm trước, audit, đua, thu hồi ≤1 s ở hai gateway). Cần stack qc2i (qc2i-r4apia-stack.sh up).
import os; exec(open(os.path.join(os.path.dirname(os.path.abspath(__file__)),'qc2i-r4apia-lib.py')).read())
em=reg('s1'); uid=psql(f"select id from users where email='{em}'")
# ---- P202-02
n0=psql(f"select count(*) from auth_sessions where user_id='{uid}'"); l0=psql(f"select coalesce(last_login_at::text,'NULL') from users where id='{uid}'")
t0=psql("select now()"); s,j,h=lg(em); c1=ck(h); n1=psql(f"select count(*) from auth_sessions where user_id='{uid}'"); l1=psql(f"select last_login_at>'{t0}'::timestamptz-interval '5 seconds' from users where id='{uid}'")
rec('TC-P202-02','P' if s==200 and int(n1)==int(n0)+1 and l1=='t' else 'F',f'login {s}; auth_sessions {n0}→{n1}; last_login_at {l0}→cập nhật={l1}')
# ---- P202-08 băm
acc1=j['access_token']; sid=claims(acc1)['sid']; T1=ckval(c1)
r=call('POST','/auth/refresh',None,{},hdr={'Cookie':c1}); c2=ck(r[2]); T2=ckval(c2)
row=psql(f"select refresh_hash,prev_refresh_hash,rotated_at is not null,last_used_at>created_at from auth_sessions where id='{sid}'").split('|')
cnt_plain=psql(f"select count(*) from auth_sessions where refresh_hash='{T1}' or prev_refresh_hash='{T1}' or refresh_hash='{T2}' or prev_refresh_hash='{T2}'")
cnt_hash=psql(f"select count(*) from auth_sessions where id='{sid}' and refresh_hash='{sha(T2)}' and prev_refresh_hash='{sha(T1)}'")
dump=subprocess.run(f"docker exec {PGC} pg_dump -U edupilot {DB}",shell=True,capture_output=True,text=True).stdout
logs=''.join(sh(f'docker logs {c} 2>&1',60) for c in ['qc2i-gw1','qc2i-gw2','qc2i-gw3','qc2i-worker'])
rec('TC-P202-08','P' if r[0]==200 and cnt_plain=='0' and cnt_hash=='1' and dump.count(T1)==0 and dump.count(T2)==0 and logs.count(T1)==0 and logs.count(T2)==0 and row[2]=='t' and row[3]=='t' and len(row[0])==64 else 'F',
    f'bản rõ trong DB(select)={cnt_plain}, pg_dump(T1,T2)={dump.count(T1)}/{dump.count(T2)}, log 3 gateway+worker={logs.count(T1)}/{logs.count(T2)}; sha256 hex: refresh_hash=sha(T2) và prev=sha(T1) → {cnt_hash}; rotated_at, last_used_at cập nhật={row[2]}/{row[3]}')
# ---- P202-09 trượt / trần (đồng hồ test = sửa expires_at/absolute_expires_at của CHÍNH phiên của tôi, không đổi giờ máy)
sid9=sid; c=c2
rowA=psql(f"select extract(epoch from (expires_at-now()))/3600, extract(epoch from (absolute_expires_at-created_at))/3600 from auth_sessions where id='{sid9}'").split('|')
# (1) trượt: đặt expires_at còn 2 giờ → refresh → expires_at trượt lại ≈ 336h
psql(f"update auth_sessions set expires_at=now()+interval '2 hours' where id='{sid9}'"); r=call('POST','/auth/refresh',None,{},hdr={'Cookie':c}); c=ck(r[2])
eh=float(psql(f"select extract(epoch from (expires_at-now()))/3600 from auth_sessions where id='{sid9}'"))
# (2) trần: đặt absolute_expires_at còn 5 giờ (expires_at 336h sẽ vượt) → refresh → expires_at ≤ absolute, ≈ 5h
psql(f"update auth_sessions set absolute_expires_at=now()+interval '5 hours' where id='{sid9}'"); r2=call('POST','/auth/refresh',None,{},hdr={'Cookie':c}); c=ck(r2[2])
ok_cap=psql(f"select expires_at<=absolute_expires_at, round((extract(epoch from (expires_at-now()))/3600)::numeric,1) from auth_sessions where id='{sid9}'")
# (3) quá 30 ngày: absolute_expires_at = quá khứ → refresh bị từ chối TOKEN_INVALID
psql(f"update auth_sessions set absolute_expires_at=now()-interval '1 second', expires_at=now()-interval '1 second' where id='{sid9}'"); r3=call('POST','/auth/refresh',None,{},hdr={'Cookie':c})
rec('TC-P202-09','P' if r[0]==200 and 330<=eh<=337 and r2[0]==200 and ok_cap.startswith('t|') and float(ok_cap.split('|')[1])<=5.0 and r3[0]==401 and r3[1].get('code')=='TOKEN_INVALID' and 719<=float(rowA[1])<=721 else 'F',
    f'tạo phiên: absolute−created={float(rowA[1]):.1f}h; trượt: sau refresh (đặt expires_at=+2h) → +{eh:.1f}h; trần (đặt absolute=+5h): expires≤absolute={ok_cap}; absolute quá khứ → {r3[0]} {r3[1].get("code")}. Đồng hồ test: UPDATE expires_at/absolute_expires_at trên phiên {sid9[:8]} của tôi (không đổi giờ máy)')
# ---- P202-12 kẻ trộm trước + P202-13 audit
s,j,h=lg(em); cv=ck(h); sidv=claims(j['access_token'])['sid']; accv=j['access_token']
th=call('POST','/auth/refresh',None,{},hdr={'Cookie':cv})   # kẻ trộm xoay T1→T2 trước
thief_ck=ck(th[2]); vic=call('POST','/auth/refresh',None,{},hdr={'Cookie':cv}) # nạn nhân mang T1 cũ
thief2=call('POST','/auth/refresh',None,{},hdr={'Cookie':thief_ck})
time.sleep(1.2); acc_after=call('GET','/me/courses',th[1].get('access_token'),base=G1); accv_after=call('GET','/me/courses',accv,base=G1)
rv=psql(f"select revoked_reason from auth_sessions where id='{sidv}'")
rec('TC-P202-12','P' if th[0]==200 and vic[0]==401 and vic[1].get('code')=='SESSION_REVOKED' and thief2[0]==401 and rv=='REFRESH_REUSE' and acc_after[0]==401 and accv_after[0]==401 else 'F',
    f'kẻ trộm xoay trước {th[0]}; nạn nhân T1 cũ → {vic[0]} {vic[1].get("code")}; T2 của kẻ trộm → {thief2[0]}; revoked_reason={rv}; access (kẻ trộm/nạn nhân) sau thu hồi {acc_after[0]}/{accv_after[0]}')
au=psql(f"select count(*),min(entity),min(action),coalesce(min(before::text),''),coalesce(min(after::text),'') from audit_log where action='refresh_reuse' and entity_id='{sidv}'").split('|')
allaud=psql(f"select coalesce(string_agg(coalesce(before::text,'')||coalesce(after::text,'')||trace_id,' '),'') from audit_log where action='refresh_reuse' and entity_id='{sidv}'")
leak=any(x in allaud for x in [ckval(cv),thief_ck and ckval(thief_ck),em])
rec('TC-P202-13','P' if au[0]=='1' and au[1]=='auth_session' and au[2]=='refresh_reuse' and not leak else 'F',f'audit_log refresh_reuse của phiên: {au[0]} dòng (entity={au[1]}); before/after/trace_id chứa token/email = {leak}; before/after={au[3]!r}/{au[4]!r}. (Bảng audit_log không có cột details — dùng before/after)')
# ---- P202-14 đua: 12 vòng, mỗi vòng 2 refresh song song cùng T1
combo=Counter(); bad500=0; two_valid=0
for i in range(12):
    s,j,h=lg(em); cc=ck(h); sd=claims(j['access_token'])['sid']
    with cf.ThreadPoolExecutor(2) as ex: rs=list(ex.map(lambda _:call('POST','/auth/refresh',None,{},hdr={'Cookie':cc},retry=False),range(2)))
    key=tuple(sorted((x[0],x[1].get('code','')) for x in rs)); combo[key]+=1
    if any(x[0]>=500 or x[0]==0 for x in rs): bad500+=1
    news=[ck(x[2]) for x in rs if x[0]==200]; valid=[c_ for c_ in news if call('POST','/auth/refresh',None,{},hdr={'Cookie':c_},retry=False)[0]==200]
    if len(valid)>1: two_valid+=1
rec('TC-P202-14','P' if bad500==0 and two_valid==0 else 'F',f'12 vòng × 2 refresh song song cùng T1: {dict((str(k),v) for k,v in combo.items())}; 5xx={bad500}; vòng sinh ≥2 cookie hợp lệ={two_valid}')
# ---- P202-25 thu hồi ≤ 1 s ở cả hai gateway
def lat(action,label,expect_code='SESSION_REVOKED'):
    s,j,h=lg(em); acc=j['access_token']; cc=ck(h); sid_=claims(acc)['sid']
    ok0=[call('GET','/me/courses',acc,base=g,retry=False)[0] for g in (G1,G2)]
    t=time.time(); action(acc,cc,sid_); out={}
    for name,g in (('gw1',G1),('gw2',G2)):
        while True:
            r_=call('GET','/me/courses',acc,base=g,retry=False); d=time.time()-t
            if r_[0]==401 or d>3: out[name]=(r_[0],r_[1].get('code'),round(d,3)); break
    return ok0,out
res25={}
_,nuser=invite('s25',role='TEACHER') if False else (None,None)
def act_logout(acc,cc,sid_): call('POST','/auth/logout',None,{},hdr={'Cookie':cc},base=G1)
res25['logout']=lat(act_logout,'logout')
def act_reuse(acc,cc,sid_): call('POST','/auth/refresh',None,{},hdr={'Cookie':cc},base=G1); call('POST','/auth/refresh',None,{},hdr={'Cookie':cc},base=G1)
res25['refresh_reuse']=lat(act_reuse,'reuse')
# đổi mật khẩu: phiên KHÁC bị thu hồi — dùng access của phiên B để đổi, đo access của phiên A
s,jA,hA=lg(em); accA=jA['access_token']; s,jB,hB=lg(em); accB=jB['access_token']; NEWP=PWD+'N9'
okA=[call('GET','/me/courses',accA,base=g,retry=False)[0] for g in (G1,G2)]
t=time.time(); chg=call('POST','/me/password',accB,{'current_password':PWD,'new_password':NEWP},base=G1); outp={}
for name,g in (('gw1',G1),('gw2',G2)):
    while True:
        r_=call('GET','/me/courses',accA,base=g,retry=False); d=time.time()-t
        if r_[0]==401 or d>3: outp[name]=(r_[0],r_[1].get('code'),round(d,3)); break
res25['password_change(phiên khác)']=(okA,outp,chg[0])
call('POST','/me/password',accB,{'current_password':NEWP,'new_password':PWD},base=G1)
# khoá bằng Admin (đường ứng dụng) và bằng UPDATE trực tiếp DB (ghi nhận)
A=admin_tok(); cur=call('GET','/admin/users?q='+em,A)[1]['items'][0]
s,jC,_=lg(em); accC=jC['access_token']
okC=[call('GET','/me/courses',accC,base=g,retry=False)[0] for g in (G1,G2)]
t=time.time(); dis=call('PATCH',f'/admin/users/{cur["id"]}',A,{'status':'DISABLED','version':cur['version']},base=G1); outd={}
for name,g in (('gw1',G1),('gw2',G2)):
    while True:
        r_=call('GET','/me/courses',accC,base=g,retry=False); d=time.time()-t
        if r_[0]==401 or d>3: outd[name]=(r_[0],r_[1].get('code'),round(d,3)); break
res25['admin_disable']=(okC,outd,dis[0])
cur=call('GET','/admin/users?q='+em,A)[1]['items'][0]; call('PATCH',f'/admin/users/{cur["id"]}',A,{'status':'ACTIVE','version':cur['version']},base=G1)
# UPDATE users SET status='DISABLED' trực tiếp (ngoài ứng dụng)
s,jD,_=lg(em); accD=jD['access_token']; psql(f"update users set status='DISABLED' where id='{uid}'"); t=time.time(); outq={}
for name,g in (('gw1',G1),('gw2',G2)):
    while True:
        r_=call('GET','/me/courses',accD,base=g,retry=False); d=time.time()-t
        if r_[0]==401 or d>3: outq[name]=(r_[0],r_[1].get('code'),round(d,3)); break
psql(f"update users set status='ACTIVE' where id='{uid}'")
res25['sql_update_status']=(None,outq,None)
json.dump(res25,open(EV+'/qc2i-r4apia-p20225.json','w'),ensure_ascii=False,indent=1,default=str)
def good(o): return all(v[0]==401 and v[1]=='SESSION_REVOKED' and v[2]<=1.0 for v in o.values())
app_ok=good(res25['logout'][1]) and good(res25['refresh_reuse'][1]) and good(outp) and good(outd)
print('P202-25 chi tiết',json.dumps(res25,default=str,ensure_ascii=False))
rec('TC-P202-25','P' if app_ok else 'F',f'401 SESSION_REVOKED ≤1 s ở gw1+gw2: logout {res25["logout"][1]}; refresh-reuse {res25["refresh_reuse"][1]}; đổi mật khẩu(phiên khác) {outp}; Admin khoá {outd}; UPDATE SQL trực tiếp status (ngoài app) {outq}')
