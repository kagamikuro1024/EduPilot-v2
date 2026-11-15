# TC-P205-10/11/13: đồng nhất email thật/không tồn tại; thư account_locked chỉ chủ thật; đặt lại bộ đếm. Cần stack qc2i.
import os,threading; exec(open(os.path.join(os.path.dirname(os.path.abspath(__file__)),'qc2i-r4apia-lib.py')).read())
WRONG='Sai-Mat-Khau-Qc2ia-1'
def dbrow(em): return psql(f"select failed_logins, coalesce(locked_until::text,'NULL') from users where email='{em}'")
def shape(s,j,h):
    jj=dict(j); jj.pop('trace_id',None)
    hh={k.lower():v for k,v in h.items() if k.lower() in ('content-type','retry-after','cache-control','www-authenticate')}
    return (s,json.dumps(jj,sort_keys=True),json.dumps(hh,sort_keys=True))
# ---- P205-10/11
real=reg('r10'); ghost=f'qc2ia-khong-co-1.{TAG}@example.test'; bar=threading.Barrier(2)
rows=[]; m0=mail_count(real)
def one(em):
    bar.wait(); s,j,h=call('POST','/auth/login',None,{'email':em,'password':WRONG},retry=False); return s,j,h
with cf.ThreadPoolExecutor(2) as ex:
    for i in range(12):
        fa=ex.submit(one,real); fb=ex.submit(one,ghost); a=fa.result(); b=fb.result(); rows.append((i+1,shape(*a),shape(*b),a,b))
diff=[(i,sa[:2],sb[:2]) for i,sa,sb,a,b in rows if sa[0]!=sb[0] or sa[1].replace(real,'X')!=sb[1].replace(ghost,'X') or sa[2]!=sb[2]]
pat=[(i,sa[0],json.loads(sa[1]).get('code'),json.loads(sa[1]).get('retry_after')) for i,sa,sb,a,b in rows]
retry_ok=all(abs((json.loads(sa[1]).get('retry_after') or 0)-(json.loads(sb[1]).get('retry_after') or 0))<=1 for i,sa,sb,a,b in rows)
idnone=not any('user_id' in sa[1] or 'user_id' in sb[1] for i,sa,sb,a,b in rows)
# tiếp: tiếp tục tới khoá thật cho cả hai, song song, theo retry_after
def drive(em):
    n=0; end=time.time()+220; last=None
    while time.time()<end:
        s,j,h=call('POST','/auth/login',None,{'email':em,'password':WRONG},retry=False)
        if s==401: n+=1
        elif s==429:
            ra=int(j.get('retry_after') or 1); last=(s,j,h)
            if ra>=800: break
            time.sleep(ra+0.3)
    return n,last
with cf.ThreadPoolExecutor(2) as ex:
    fa=ex.submit(drive,real); fb=ex.submit(drive,ghost); (na,la),(nb,lb)=fa.result(),fb.result()
time.sleep(8)
lockshape_a=shape(*la) if la else None; lockshape_b=shape(*lb) if lb else None
same_lock=lockshape_a and lockshape_b and lockshape_a[0]==lockshape_b[0] and json.loads(lockshape_a[1]).get('code')==json.loads(lockshape_b[1]).get('code') and abs(json.loads(lockshape_a[1]).get('retry_after',0)-json.loads(lockshape_b[1]).get('retry_after',0))<=2
mails_real=[m['Subject'] for m in mails(real)]; mails_ghost=mail_count(ghost)
rec('TC-P205-10','P' if not diff and retry_ok and idnone and same_lock else 'F',f'12 lần song song (thật vs không tồn tại): khác khuôn ở {len(diff)} bước {diff[:2]}; mẫu mã/retry_after (bước,mã,code,retry) {pat[:12]}; chờ/khoá tới 10: thật {na} lần sai, không tồn tại {nb}; 429 khoá cùng khuôn={bool(same_lock)}: {lockshape_a and lockshape_a[1][:120]} | {lockshape_b and lockshape_b[1][:120]}; không user_id={idnone}')
rec('TC-P205-11','P' if mails_ghost==0 and sum('khoá' in s for s in mails_real)==1 else 'F',f'sau 10 lần sai: email không tồn tại nhận {mails_ghost} thư; email thật nhận thư {mails_real} (account_locked đúng 1)')
# ---- P205-13 đặt lại bộ đếm
em=reg('r13'); res13={}
def fails(n): return [call('POST','/auth/login',None,{'email':em,'password':WRONG},retry=False)[0] for _ in range(n)]
a=fails(4); d1=dbrow(em); ok1=lg(em)[0]; d2=dbrow(em); r_key=rds('hget','ep:auth:fail:'+sha(em)[:32],'n')
b=fails(4); d3=dbrow(em)
# 3 sai, "đợi >30 phút" rút gọn: kéo mốc `last` của bộ đếm Redis lùi 31 phút (GHI RÕ), 1 lần sai nữa
lg(em); fails(3); d4=dbrow(em); k='ep:auth:fail:'+sha(em)[:32]; last=int(rds('hget',k,'last')); rds('hset',k,'last',str(last-31*60*1000)); fails(1); d5=dbrow(em); n_key=rds('hget',k,'n')
ok=a==[401]*4 and ok1==200 and d2.startswith('0|') and r_key in('','0') and d3.startswith('4|') and d4.startswith('3|') and d5.startswith('1|') and n_key=='1'
rec('TC-P205-13','P' if ok else 'F',f'4 sai → failed_logins {d1}; 1 đúng → {d2} (khoá Redis n={r_key!r}); 4 sai nữa → {d3} (không cộng dồn); 3 sai → {d4}; rút gọn: kéo `last` Redis lùi 31 phút (HSET) rồi 1 sai → {d5}, Redis n={n_key}')
