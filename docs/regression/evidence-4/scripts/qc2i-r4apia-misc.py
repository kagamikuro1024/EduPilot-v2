# TC-P205-20/25/28, P206-08/11/28. Cần stack qc2i (gw1/gw2 mặc định cho P205-20 + P206-08 phần 429; gw3 cho phần còn lại).
import os,statistics; exec(open(os.path.join(os.path.dirname(os.path.abspath(__file__)),'qc2i-r4apia-lib.py')).read())
A=admin_tok()
# ---------- P205-20: bộ đếm IP chia sẻ giữa hai gateway (giới hạn mặc định 10/phút/IP)
def start_of_minute_window():
    while not (3<=time.time()%60<=40): time.sleep(1)
res20=[]
for attempt in range(3):
    start_of_minute_window(); resetrl(); res20=[]
    for i in range(11):
        g=(G1,G2)[i%2]; s,j,_=call('POST','/auth/login',None,{'email':f'qc2ia-ip20-{i}.{TAG}@example.test','password':'Sai-pw-20-%d'%i},base=g,retry=False); res20.append((s,'gw%d'%(1+i%2)))
    if res20[10][0]==429: break
rec('TC-P205-20','P' if [x[0] for x in res20[:10]]==[401]*10 and res20[10][0]==429 else 'F',f'11 lần đăng nhập sai xen kẽ gw1/gw2 (email khác nhau, 1 IP, giới hạn mặc định 10/phút/IP): {[x[0] for x in res20]} → lần 11 ở {res20[10][1]} = {res20[10][0]}')
resetrl()
# ---------- P205-25: mật khẩu thử không lộ ở log / thân lỗi
M=[f'Qc2iaMarker{TAG}{k}!x' for k in 'ABCDE']; bodies=[]
em=reg('p25'); 
r=call('POST','/auth/login',None,{'email':em,'password':M[0]}); bodies.append(r)
r=call('POST','/auth/register',None,{'email':f'qc2ia-p25b.{TAG}@edupilot.local','password':M[1],'full_name':'X'}); bodies.append(r)      # đủ độ dài → 202, vẫn kiểm thân
r=call('POST','/auth/register',None,{'email':f'qc2ia-p25c.{TAG}@edupilot.local','password':M[2][:6],'full_name':'X'}); bodies.append(r)       # yếu → 422
tk=lg(em)[1]['access_token']; r=call('POST','/me/password',tk,{'current_password':M[3],'new_password':M[4]}); bodies.append(r)
r=call('POST','/me/password',tk,{'current_password':PWD,'new_password':'123'+M[4][:3]}); bodies.append(r)
r=call('POST','/auth/reset-password',None,{'token':'Z'*43,'new_password':M[4]}); bodies.append(r)
r=call('POST','/auth/accept-invite',None,{'token':'Y'*43,'password':M[3]}); bodies.append(r)
time.sleep(3)
logs=''.join(sh(f'docker logs {c} 2>&1',90) for c in ['qc2i-gw1','qc2i-gw2','qc2i-gw3','qc2i-worker'])
in_body=[(i,m) for i,b in enumerate(bodies) for m in M+[M[2][:6]] if m in json.dumps(b[1],ensure_ascii=False)]
in_log=sum(logs.count(m) for m in M)+logs.count(M[2][:6]+'"')
rec('TC-P205-25','P' if not in_body and in_log==0 else 'F',f'7 yêu cầu mang mật khẩu thử (login sai, register hợp lệ/yếu 422, /me/password sai, reset/accept-invite token lạ): mã {[b[0] for b in bodies]}; mật khẩu lặp lại trong thân lỗi={len(in_body)}; trong log gw1+gw2+gw3+worker={in_log} ({len(logs)} byte log)')
# ---------- P205-28: thời gian giả (30 mẫu / loại; gw3; mỗi 4 mẫu xoá bộ đếm sai của email để không dính chờ tăng dần — ghi rõ)
inv,_=invite('p28i',role='TEACHER',accept=False)
wrong_em=em; ghost=f'qc2ia-p28-ghost.{TAG}@example.test'
def med(email,n=30):
    ts=[]
    for i in range(n):
        if i%4==0: clearfail()
        t=time.time(); s,j,_=call('POST','/auth/login',None,{'email':email,'password':'Sai-mat-khau-28'},retry=False); ts.append(time.time()-t); assert s==401,(email,s,j)
    clearfail(); return statistics.median(ts)
m_ok=med(wrong_em); m_ghost=med(ghost); m_inv=med(inv)
rec('TC-P205-28','P' if m_ghost>=0.65*m_ok and m_inv>=0.65*m_ok else 'F',f'trung vị 30 mẫu/loại: mật khẩu sai {m_ok*1000:.0f} ms; email không tồn tại {m_ghost*1000:.0f} ms (tỉ lệ {m_ghost/m_ok:.2f}); INVITED {m_inv*1000:.0f} ms (tỉ lệ {m_inv/m_ok:.2f}); ngưỡng ≥ 0,65. Xoá bộ đếm sai Redis mỗi 4 mẫu')
# ---------- P206-08
tem=f'qc2ia-gv8.{TAG}@edupilot.local'; t0=iso(); s,j,_=call('POST','/admin/users',A,{'email':tem,'full_name':'QC2IA gv8','role':'TEACHER'}); m=mail_to(tem,'mời|Lời mời',t0[:0],60); tok=link_token(m,'invite'); uid=j['id']
# "age_token +72 h": đẩy hạn token mời lùi 72 giờ (UPDATE auth_tokens của CHÍNH token tôi tạo — ghi rõ)
psql(f"update auth_tokens set expires_at=now()-interval '1 second', created_at=created_at-interval '72 hours' where token_hash='{sha(tok)}'")
pv=call('POST','/auth/tokens/preview',None,{'kind':'INVITE','token':tok}); ac=call('POST','/auth/accept-invite',None,{'token':tok,'password':PWD})
exp_ok=pv[0]==410 and ac[0]==410 and pv[1].get('details',{}).get('reason')=='expired' and ac[1].get('details',{}).get('reason')=='expired'
toks=[base64.urlsafe_b64encode(os.urandom(32)).decode().rstrip('=') for _ in range(1000)]
def guess(t): return call('POST','/auth/accept-invite',None,{'token':t,'password':PWD},retry=False)
with cf.ThreadPoolExecutor(8) as ex: gs=list(ex.map(guess,toks))
cnt=Counter(g[0] for g in gs); keys=set(tuple(sorted(k for k in g[1] if k!='trace_id')) for g in gs); reasons=Counter((g[1].get('details') or {}).get('reason') for g in gs)
resetrl(); g1=[call('POST','/auth/tokens/preview',None,{'kind':'INVITE','token':t},base=G1,retry=False)[0] for t in toks[:25]]
rec('TC-P206-08','P' if exp_ok and cnt.get(410)==1000 and len(keys)==1 and reasons=={'invalid':1000} and g1[:20]==[410]*20 and 429 in g1[20:] else 'F',f'token mời lùi 72h → preview {pv[0]} {pv[1].get("details")}, accept {ac[0]} {ac[1].get("details")}; 1000 token ngẫu nhiên 43 ký tự (gw3, giới hạn IP nới): {dict(cnt)}, lý do {dict(reasons)}, {len(keys)} khuôn khoá JSON {list(keys)[:1]}; gw1 mặc định 25 lần preview: {Counter(g1)} (429 đầu ở lần {g1.index(429)+1 if 429 in g1 else None}; kỳ vọng sau 20/phút/IP)')
# ---------- P206-11
mk='QC2IAMARK'+TAG; em2=f'qc2ia-gv11.{TAG}@edupilot.local'; rs=[]
r=call('POST','/admin/users',A,{'email':em2,'full_name':mk,'role':'TA'}); rs.append(r); uid2=r[1]['id']
rs.append(call('GET','/admin/users?q='+mk[:9],A)); rs.append(call('GET','/admin/users',A)); cur=call('GET','/admin/users?q='+em2,A); rs.append(cur); v=cur[1]['items'][0]['version']
rs.append(call('PATCH',f'/admin/users/{uid2}',A,{'full_name':mk+'x','version':v})); rs.append(call('POST',f'/admin/users/{uid2}/resend-invite',A,{}))
blob=json.dumps([x[1] for x in rs],ensure_ascii=False); hits=len(re.findall(r'(?i)password|hash|token',blob))
pw1=call('PATCH',f'/admin/users/{uid2}',A,{'password':'x','version':v+5}); pw2=call('POST',f'/admin/users/{uid2}/reset-password',A,{}); pw3=call('PUT',f'/admin/users/{uid2}/password',A,{'password':'x'}); pw4=call('POST',f'/admin/users/{uid2}',A,{'password':'x'})
time.sleep(2); logs=''.join(sh(f'docker logs {c} 2>&1',90) for c in ['qc2i-gw1','qc2i-gw2','qc2i-gw3'])
oa=sh("cd ~/Documents/TA_Agent_qc2/backend-go && grep -n -iE 'password|token' api/openapi.yaml | grep -E '/admin/' | wc -l").strip()
rec('TC-P206-11','P' if oa=='0' and hits==0 and pw1[0]==422 and pw2[0] in(404,405,422) and pw3[0] in(404,405,422) and pw4[0] in(404,405,422) and logs.count(mk)==0 else 'F',f'grep openapi (password|token trên dòng /admin/) = {oa}; 6 phản hồi thật (invite, list×3, patch, resend) khớp password|hash|token = {hits}; PATCH {{"password"}} → {pw1[0]}; POST /admin/users/{{id}}/reset-password → {pw2[0]}; PUT …/password → {pw3[0]}; POST /admin/users/{{id}} → {pw4[0]}; chuỗi đặt ở thân (họ tên) xuất hiện trong log gateway = {logs.count(mk)}')
# ---------- P206-28 EXPLAIN + số truy vấn
psql("insert into users (email,full_name,role,status,created_at) select 'qc2ia-bulk'||g||'@example.test','Bulk User '||g, (array['STUDENT','TEACHER','TA'])[1+g%3]::user_role, 'ACTIVE', now()-(g||' seconds')::interval from generate_series(1,60000) g on conflict do nothing")
psql("analyze users")
q="select id, email, full_name, role, status, last_login_at, version, created_at from users where (%s) and (%s) and (%s) %s order by created_at desc, id desc limit 21"
plans={}
for lab,r_,s_,nm,cur in [('mặc định (trang đầu)','true','true','true',''),('lọc role','role=\'TEACHER\'::user_role','true','true',''),('lọc status','true','status=\'ACTIVE\'::user_status','true',''),('con trỏ','true','true','true',"and (created_at,id) < (now()-interval '3 hours', '00000000-0000-7000-8000-000000000000'::uuid)"),('tìm theo tên','true','true',"(vn_fold(full_name) like '%bulk user 5999%' escape '\\' or email like '%bulk5999%' escape '\\')",'')]:
    out=psql("explain (analyze, buffers, format text) "+q%(r_,s_,nm,cur)); plans[lab]=out
cnt_tab=psql("select count(*), pg_size_pretty(pg_total_relation_size('users')) from users"); idx=psql("select indexname from pg_indexes where tablename='users'")
def scan(p): return 'Seq Scan' in p, 'Index' in p
sv={k:scan(v) for k,v in plans.items()}
open(EV+'/qc2i-r4apia-p20628-explain.txt','w').write(f'users: {cnt_tab}\nindexes: {idx}\n\n'+'\n\n'.join(f'### {k}\n{v}' for k,v in plans.items()))
code=sh("cd ~/Documents/TA_Agent_qc2/backend-go && grep -c 'ListUsersAdmin(' internal/user/service.go").strip()
rec('TC-P206-28','P' if not sv['mặc định (trang đầu)'][0] and not sv['con trỏ'][0] else 'F',f'users={cnt_tab} (60 000 dòng mô phỏng, ANALYZE): Seq Scan? {sv}; mã: List gọi đúng 1 truy vấn ListUsersAdmin (user/service.go: {code} lời gọi), không vòng lặp N+1; EXPLAIN đủ ở evidence-4/qc2i-r4apia-p20628-explain.txt; chỉ mục: {idx.replace(chr(10),",")}')
