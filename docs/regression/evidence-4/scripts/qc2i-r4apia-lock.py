# TC-P205-07/10/11/13/14/36/39 + P204-08: chuỗi chờ tăng dần / khoá 15 phút / đặt lại mở khoá (gw3 giới hạn IP nới; bộ đếm sai theo email vẫn là logic thật).
# Mỗi nhánh chạy song song một luồng với tài khoản riêng. Cần stack qc2i + Admin thứ hai qc2ia-admin2 (stack.sh up2).
import os,threading; exec(open(os.path.join(os.path.dirname(os.path.abspath(__file__)),'qc2i-r4apia-lib.py')).read())
WRONG='Sai-Mat-Khau-Qc2ia-1'
def wrong(em,pw=WRONG,base=None):
    s,j,h=call('POST','/auth/login',None,{'email':em,'password':pw},base=base,retry=False); return s,j,h
def lock_account(em,maxsec=200):
    """sai liên tiếp tới khi bị khoá: tôn trọng retry_after của 429; trả (số 401, mốc khoá lúc, thân 429 cuối)."""
    n=0; t_end=time.time()+maxsec; last=None
    while time.time()<t_end:
        s,j,h=wrong(em)
        if s==401: n+=1
        elif s==429:
            last=(j,h); ra=int(j.get('retry_after') or j.get('details',{}).get('retry_after') or h.get('Retry-After') or 1)
            if ra>=800: return n,time.time(),last
            time.sleep(ra+0.3)
        else: raise Exception(f'lock: {s} {j}')
        if n>=10:
            s2,j2,h2=wrong(em); return n,time.time(),(j2,h2)
    raise Exception('không khoá được')
def dbrow(em): return psql(f"select failed_logins, coalesce(locked_until::text,'NULL'), status, email_verified_at is not null from users where email='{em}'")
out={}
def run(name,fn):
    def w():
        try: fn()
        except Exception as e: out[name]=('ERR',repr(e)); print(name,'ERR',repr(e),flush=True)
    t=threading.Thread(target=w); t.start(); return t
ths=[]
# ---- P204-08 (+đặt lại): tài khoản CHƯA xác minh, khoá rồi reset
def t204_08():
    em=reg('r8',verify=False); n,tl,_=lock_account(em); before=dbrow(em); t0=iso()
    s,j,_=call('POST','/auth/forgot-password',None,{'email':em}); m=mail_to(em,'Đặt lại',t0,60); tok=link_token(m,'reset-password')
    NEWP=PWD+'R8'; r=call('POST','/auth/reset-password',None,{'token':tok,'new_password':NEWP}); after=dbrow(em); pc=mail_to(em,'đã được đổi',t0,60); lg_=lg(em,NEWP)
    a=after.split('|')
    ok=before.split('|')[0]=='10' and before.split('|')[1]!='NULL' and r[0]==200 and a[0]=='0' and a[1]=='NULL' and a[2]=='ACTIVE' and a[3]=='t' and pc is not None and lg_[0]==200
    rec('TC-P204-08','P' if ok else 'F',f'trước: failed_logins|locked_until|status|verified = {before}; reset {r[0]}; sau = {after}; thư password_changed={bool(pc)}; đăng nhập mk mới {lg_[0]}')
ths.append(run('204-08',t204_08))
# ---- P205-14 (+P205-39): khoá → nạn nhân bị 429 → đặt lại mở ngay
def t205_14_39():
    em=reg('r14'); n,tl,last=lock_account(em); row=dbrow(em)
    victim=lg(em)  # nạn nhân (mật khẩu ĐÚNG) bị chặn
    ra=victim[1].get('retry_after') or victim[1].get('details',{}).get('retry_after')
    lm=mail_count(em)
    ok39=n>=10 and victim[0]==429 and (ra is None or 600<=int(ra)<=900)
    t0=iso(); call('POST','/auth/forgot-password',None,{'email':em}); m=mail_to(em,'Đặt lại',t0,60); tok=link_token(m,'reset-password'); NEWP=PWD+'R14'
    r=call('POST','/auth/reset-password',None,{'token':tok,'new_password':NEWP}); after=dbrow(em); l2=lg(em,NEWP)
    rec('TC-P205-39','P' if ok39 else 'F',f'RỦI RO ĐÃ DUYỆT (QUESTIONS Q3): kẻ xấu sai {n} lần → nạn nhân đăng nhập ĐÚNG → {victim[0]} {victim[1].get("code")} retry_after={ra}; DB {row}; lối thoát = đặt lại mật khẩu (TC-14)')
    rec('TC-P205-14','P' if r[0]==200 and l2[0]==200 and after.split('|')[1]=='NULL' and after.split('|')[0]=='0' else 'F',f'đang khoá ({row}) → đặt lại qua thư {r[0]}; đăng nhập mk mới {l2[0]}; sau: {after}')
ths.append(run('205-14',t205_14_39))
# ---- P205-07: hết khoá (đặt locked_until=now()-1s + xoá khoá Redis tương ứng — GHI RÕ)
def t205_07():
    em=reg('r7'); n,tl,_=lock_account(em); row=dbrow(em)
    psql(f"update users set locked_until=now()-interval '1 second' where email='{em}'")
    eh=sha(em); rds('hset','ep:auth:fail:'+eh[:32],'lock','1')  # mốc khoá Redis → quá khứ (1 ms epoch)
    s,j,_=lg(em); after=dbrow(em)
    rec('TC-P205-07','P' if s==200 and after.startswith('0|') else 'F',f'khoá thật ({row}) → ĐẶT locked_until=now()-1s (DB) và hset lock=1 (Redis, mốc quá khứ) → đăng nhập đúng {s}; sau: {after}. (Hết khoá thật 15 phút đo ở TC-P205-09)')
ths.append(run('205-07',t205_07))
# ---- P205-36: Admin (qc2ia-admin2) cũng bị chờ/khoá
def t205_36():
    em='qc2ia-admin2@edupilot.local'; n,tl,last=lock_account(em); row=dbrow(em)
    s,j,_=call('POST','/auth/login',None,{'email':em,'password':'Qc2ia!Admin-Pw-2026x'},retry=False)
    rec('TC-P205-36','P' if n>=10 and s==429 else 'F',f'ADMIN (vai ADMIN, tài khoản bootstrap `gateway admin create`) sai {n} lần: bị chờ rồi khoá; đăng nhập ĐÚNG sau đó → {s} {j.get("code")}; DB {row}')
ths.append(run('205-36',t205_36))
for t in ths: t.join()
print(out)
