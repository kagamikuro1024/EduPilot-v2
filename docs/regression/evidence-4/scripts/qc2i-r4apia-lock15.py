# TC-P205-09: đo thật 15 phút, KHÔNG rút gọn. Chạy nền: nohup python3 qc2i-r4apia-lock15.py &
import os; exec(open(os.path.join(os.path.dirname(os.path.abspath(__file__)),'qc2i-r4apia-lib.py')).read())
WRONG='Sai-Mat-Khau-Qc2ia-1'; em=reg('r9')
n=0
while True:
    s,j,h=call('POST','/auth/login',None,{'email':em,'password':WRONG},retry=False)
    if s==401: n+=1
    elif s==429:
        ra=int(j.get('retry_after') or 1)
        if ra>=800: break
        time.sleep(ra+0.3)
    if n>=10:
        s,j,h=call('POST','/auth/login',None,{'email':em,'password':WRONG},retry=False); break
T=time.time()   # thời điểm nhận thân 429 khoá đầu tiên (xấp xỉ mốc khoá, trễ ≤ 1 s)
lu=psql(f"select extract(epoch from locked_until) from users where email='{em}'"); lock_end=float(lu)
print('lock lúc',T,'locked_until',lock_end,'Δ',lock_end-T,'retry_after',j.get('retry_after'),flush=True)
def at(offset):
    d=T+offset-time.time()
    if d>0: time.sleep(d)
    s,j,h=lg(em); return s,j.get('code'),j.get('retry_after'),round(time.time()-T,1)
r1=at(60); r2=at(14*60+30); r3=at(15*60+5)
dtm=lock_end-T
ok=abs(dtm-900)<=10 and r1[0]==429 and abs((r1[2] or 0)-(900-r1[3]))<=10 and r2[0]==429 and abs((r2[2] or 0)-(900-r2[3]))<=10 and r3[0]==200
rec('TC-P205-09','P' if ok else 'F',f'khoá đo thật (không rút gọn): locked_until−mốc khoá = {dtm:.1f}s (≈900±10); +1:00 → {r1}; +14:30 → {r2} (kỳ vọng 429, retry_after ≈30); +15:05 → {r3} (kỳ vọng 200)')
