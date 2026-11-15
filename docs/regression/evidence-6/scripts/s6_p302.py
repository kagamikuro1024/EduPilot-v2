from lib import *
import time
C=json.load(open('courses.json')); E1=C['E6C1']; E2=C['E6C2']
cnt={}
def pc(u,c,t,title=None):
    # tự giãn: ≤ 50 yêu cầu/phút/người
    now=time.time(); L=[x for x in cnt.get(u,[]) if now-x<60]
    if len(L)>=50: time.sleep(61-(now-L[0])); L=[]
    L.append(time.time()); cnt[u]=L
    b={'body':t}; 
    if title: b['title']=title
    for i in range(3):
        s,d=A(u,'POST',f'/courses/{c}/threads/precheck',b)
        if s!=429: return s,d
        time.sleep(30)
    return s,d
def blocked(u,c,t,typ): 
    s,d=pc(u,c,t); return s==200 and not d['allowed'] and any(r['type']==typ for r in d['reasons']),d
def allowed(u,c,t):
    s,d=pc(u,c,t); return s==200 and d['allowed'],d
def chk(tc,items):
    bad=[(t,str(d)[:100]) for ok,d,t in items if not ok]
    rec(tc,not bad,'; '.join(f'{b[0][:30]}→{b[1]}' for b in bad[:2]) or f'{len(items)} mẫu đúng')
def cases(tc,u,c,lst,typ):
    chk(tc,[(*blocked(u,c,t,typ),t) for t in lst])
G='sv.gioi'
cases('TC-P3-02-01',G,E1,["Bài lab 3 của 20221234 nộp chưa?","Mã B21DCAT123 có trong danh sách không?"],'MSSV')
cases('TC-P3-02-03',G,E1,["MSSV 123456 là ai","mã số sinh viên 1234567890123 ạ","mssv: abc123xyz"],'MSSV')
cases('TC-P3-02-04',G,E1,["nam.nt@edupilot.local","abc.xyz+tag@gmail.com","sv@cntt.hust.edu.vn"],'EMAIL')
cases('TC-P3-02-05',G,E1,["gọi em 0912345678","+84 912 345 678","0912.345.678","09 1234 5678"],'PHONE')
cases('TC-P3-02-06',G,E1,["CCCD 001203004567","001 203 004 567"],'CCCD')
chk('TC-P3-02-09',[(*allowed(G,E1,"Gọi văn phòng khoa 0241234567"),'x')])
cases('TC-P3-02-11',G,E1,["Bùi Thanh Khải được mấy điểm lab?"],'NAME')
cases('TC-P3-02-12',G,E1,["bui thanh khai lam bai nay chua"],'NAME')
cases('TC-P3-02-13',G,E1,["BÙI THANH KHẢI"],'NAME')
cases('TC-P3-02-14',G,E1,["Khải Bùi Thanh nộp bài chưa","Khải Bùi có đi học không"],'NAME')
cases('TC-P3-02-15',G,E1,["Bùi Khải"],'NAME')
# 19: theo lớp: C2 cùng chuỗi → cho qua
chk('TC-P3-02-19',[(*blocked(G,E1,"Bùi Thanh Khải nộp chưa?",'NAME'),'c1'),(*allowed('sv04',E2,"Bùi Thanh Khải nộp chưa?"),'c2 không chặn')])
# 20 cache redis
print('ep:roster keys',rds('--scan','--pattern','ep:roster:*'))
ex=rds('EXISTS','ep:roster:'+E1); ttl=rds('TTL','ep:roster:'+E1); typ=rds('TYPE','ep:roster:'+E1)
rec('TC-P3-02-20',ex=='1' and int(ttl)>0,f'exists={ex} ttl={ttl} type={typ}')
# 21: vô hiệu hoá khi thêm thành viên
t0=time.time(); s,d=A('sv04','POST','/courses/join',{'code':'QKTWXZH'}); 
dt=None
for i in range(20):
    if rds('EXISTS','ep:roster:'+E1)=='0': dt=time.time()-t0; break
    time.sleep(0.5)
ok,d=blocked(G,E1,"Vũ Minh Long có nộp bài chưa",'NAME')
rec('TC-P3-02-21',dt is not None and ok,f'cache xoá sau {dt}s; tên SV mới bị chặn={ok}')
# 22 mời ra
uid=psql("select id from users where email='sv04@edupilot.local'")
s,d=A('teacher','DELETE',f'/courses/{E1}/members/{uid}'); time.sleep(3)
ok,d=allowed(G,E1,"Vũ Minh Long có nộp bài chưa")
rec('TC-P3-02-22',s in(200,204) and ok,f'DELETE {s}; sau mời ra → allowed={ok}')
# 24 staff
chk('TC-P3-02-24',[(*allowed(G,E1,"Lê Thu Hà là giảng viên lớp này đúng không"),'gv'),(*allowed(G,E1,"Phạm Quốc Bảo là trợ giảng"),'ta')])
# 26 redact
s,d=pc(G,E1,"Bùi Thanh Khải 20229002 bui.khai@edupilot.local 0912345678")
rec('TC-P3-02-26',s==200 and not d['allowed'] and all(x not in d['redacted_text'] for x in ['Khải','20229002','@edupilot','0912345678']),d['redacted_text'] if s==200 else str(s))
s,d2=pc(G,E1,d['redacted_text']); rec('TC-P3-02-27',s==200 and d2['redacted_text']==d['redacted_text'],'redact idempotent')
# 58-61
s,d=A('sv.moi','POST',f'/courses/{E1}/threads/precheck',{'body':'test'}); rec('TC-P3-02-58',s==403,f'SV ngoài lớp precheck → {s}')
s,d=A('admin','POST',f'/courses/{E1}/threads/precheck',{'body':'test'}); rec('TC-P3-02-59',s==403,f'ADMIN precheck → {s}')
s1,_=A('ta','POST','/chat/sessions',{'course_id':E1}); s2,_=A('teacher','POST','/chat/sessions',{'course_id':E1})
rec('TC-P3-02-61',s1==403 and s2==403,f'TA {s1}, TCH {s2}')
# 56 regex dos
import time as T
for nm,b in (('a1',"a1"*4000),('par',"("*8000)):
    t=T.time(); s,d=pc(G,E1,b); print(nm,s,round(T.time()-t,2))
t=T.time(); s,d=pc(G,E1,"a1"*4000); t1=T.time()-t; s2,_=pc(G,E1,"("*8000); t2=T.time()-t
rec('TC-P3-02-56',s==200 and t1<2,f'precheck 8000 ký tự a1: {t1:.2f}s ({s})')
# 55 p95 4000 chars
ts=[]
for i in range(30):
    a=T.time(); pc(G if i<25 else 'sv.kha',E1,("Thuật toán RSA dựa trên bài toán nào? "*100)[:4000]); ts.append(T.time()-a)
ts.sort(); rec('TC-P3-02-55',ts[int(len(ts)*.95)-1]<0.5,f'30 lượt 4.000 ký tự: p95={ts[int(len(ts)*.95)-1]*1000:.0f} ms')
save('s6_p302')
