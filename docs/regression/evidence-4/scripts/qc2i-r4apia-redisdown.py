# TC-P202-26, P205-16, P205-21: Redis chết (CHỈ qc2i-redis riêng, không động ep-r). Một cửa sổ chết ~2–3 phút.
import os; exec(open(os.path.join(os.path.dirname(os.path.abspath(__file__)),'qc2i-r4apia-lib.py')).read())
WRONG='Sai-Mat-Khau-Qc2ia-1'
def dbrow(em): return psql(f"select failed_logins, coalesce(locked_until::text,'NULL') from users where email='{em}'")
u=reg('d26'); r16=reg('d16'); ok16=reg('d16ok'); s,j,h=lg(u); A1=j['access_token']; ck1=ck(h)
pre=[call('GET','/me/courses',A1,base=g,retry=False)[0] for g in (G1,G2)]
t_stop=time.time(); sh('docker stop qc2i-redis',120); ts_stop=time.strftime('%Y-%m-%dT%H:%M:%S',time.gmtime(t_stop)).replace('T','T')
try:
    # --- P202-26: token hợp lệ chưa thu hồi vẫn được chấp nhận ~70 s ở cả hai gateway
    t_end=time.time()+70; res=[]; lat=[]
    while time.time()<t_end:
        for g in (G1,G2):
            t=time.time(); r=call('GET','/me/courses',A1,base=g,retry=False); lat.append(time.time()-t); res.append(r[0])
        time.sleep(2)
    hz=[call('GET','/healthz',None,base=g,retry=False)[0] for g in ('http://127.0.0.1:18221','http://127.0.0.1:18222')]
    # --- P205-16: 10 lần sai cho tài khoản thật
    w16=[call('POST','/auth/login',None,{'email':r16,'password':WRONG},base=G1,retry=False)[0] for _ in range(10)]
    row16=dbrow(r16); after=call('POST','/auth/login',None,{'email':r16,'password':PWD},base=G1,retry=False)
    ghost=[call('POST','/auth/login',None,{'email':f'qc2ia-d16-{i}.{TAG}@example.test','password':WRONG},base=G1,retry=False) for i in range(6)]
    okl=call('POST','/auth/login',None,{'email':ok16,'password':PWD},base=G1,retry=False)
    # --- P205-21: 15 lần sai nhanh (gateway giới hạn mặc định 10/phút/IP → Redis chết: mở cửa)
    w21=[call('POST','/auth/login',None,{'email':f'qc2ia-d21-{i}.{TAG}@example.test','password':WRONG},base=G2,retry=False)[0] for i in range(15)]
    logs={c:sh(f'docker logs --since {int(t_stop)-1} {c} 2>&1',90) for c in ('qc2i-gw1','qc2i-gw2')}
finally:
    sh('docker start qc2i-redis',120); t_start=time.time()
# log: số dòng error "Redis không với tới" theo cửa sổ 30 s mỗi gateway
def errtimes(txt):
    out=[]
    for l in txt.splitlines():
        try: d=json.loads(l)
        except Exception: continue
        if d.get('level')=='ERROR' or d.get('level')=='error': out.append((d.get('time',''),d.get('msg','')[:60]))
    return out
def maxper30(ts):
    from datetime import datetime
    T=[datetime.fromisoformat(t.replace('Z','+00:00')).timestamp() for t,_ in ts if t]
    return max([sum(1 for y in T if x<=y<x+30) for x in T] or [0])
errs={c:errtimes(t) for c,t in logs.items()}; worst={c:maxper30(e) for c,e in errs.items()}; msgs={c:Counter(m for _,m in e) for c,e in errs.items()}
print('outage',round(t_start-t_stop),'s; error log',{c:len(e) for c,e in errs.items()},worst,msgs)
# --- sau khi Redis về
time.sleep(5)
for _ in range(20):
    if rds('ping')=='PONG': break
    time.sleep(2)
s2,j2,h2=lg(u); A2=j2['access_token']; ck2=ck(h2)
r_pre=[call('GET','/me/courses',A2,base=g,retry=False)[0] for g in (G1,G2)]
t=time.time(); call('POST','/auth/logout',None,{},hdr={'Cookie':ck2},base=G1); rv={}
for n,g in (('gw1',G1),('gw2',G2)):
    while True:
        r=call('GET','/me/courses',A2,base=g,retry=False); d=time.time()-t
        if r[0]==401 or d>3: rv[n]=(r[0],r[1].get('code'),round(d,2)); break
locked_after=call('POST','/auth/login',None,{'email':r16,'password':PWD},base=G1,retry=False)
# throttle trở lại: email mới 5 sai → lần 6 bị chờ
resetrl(); ee=f'qc2ia-d16-back.{TAG}@example.test'; back=[call('POST','/auth/login',None,{'email':ee,'password':WRONG},base=G1,retry=False)[0] for _ in range(6)]
rec('TC-P202-26','P' if pre==[200,200] and set(res)=={200} and hz==[200,200] and all(w<=1 for w in worst.values()) and all(v[0]==401 and v[1]=='SESSION_REVOKED' and v[2]<=1.0 for v in rv.values()) and r_pre==[200,200] else 'F',f'Redis dừng {round(t_start-t_stop)}s: {len(res)} lần gọi API với access hợp lệ (gw1+gw2) → {dict(Counter(res))}, trễ trung vị {sorted(lat)[len(lat)//2]*1000:.0f} ms, tối đa {max(lat)*1000:.0f} ms; /healthz {hz}; log ERROR trong cửa sổ 30 s tối đa {worst} (tổng {({c:len(e) for c,e in errs.items()})}; nội dung {({c:dict(m) for c,m in msgs.items()})}); Redis về → mốc thu hồi: logout rồi access cũ → {rv}')
ok16b=w16==[401]*10 and row16.startswith('10|') and not row16.endswith('NULL') and after[0]==429 and all(g[0]==401 for g in ghost) and okl[0]==200 and all(w<=1 for w in worst.values()) and locked_after[0]==429 and back[:5]==[401]*5 and back[5]==429
rec('TC-P205-16','P' if ok16b else 'F',f'Redis chết: 10 lần sai tài khoản thật → {Counter(w16)}, DB {row16}; đăng nhập đúng ngay sau → {after[0]} {after[1].get("code")}; email không tồn tại 6 lần sai → {[g[0] for g in ghost]} (không bị chờ — chấp nhận); tài khoản chưa khoá đăng nhập đúng → {okl[0]}; log ERROR/30 s ≤ {max(worst.values())}; Redis về: tài khoản vẫn khoá (DB) → {locked_after[0]}; email mới 6 lần sai → {back} (chờ tăng dần hoạt động lại)')
rec('TC-P205-21','P' if w21==[401]*15 else 'F',f'Redis chết, 15 đăng nhập sai nhanh từ 1 IP (gw2 giới hạn mặc định 10/phút/IP): {dict(Counter(w21))} → giới hạn IP MỞ CỬA (fail-open) đúng thiết kế; khoá theo tài khoản vẫn dựa DB (TC-P205-16: failed_logins=10, locked_until có)')
