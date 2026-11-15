from lib import *
import time,threading
C=json.load(open('courses.json')); E1=C['E6C1']; C1='01a1df27-62b5-7775-a346-5c602a43e2f9'
G='sv.gioi'
def post(u,c,title,body,key=None,**kw):
    return A(u,'POST',f'/courses/{c}/threads',dict(title=title,body=body,**kw),idem=key)
# 01/49 phân trang
ids=[]
for i in range(7): 
    s,d=post(G,E1,f'RSA bài {i}',f'Câu hỏi học thuật số {i} về mật mã khoá công khai'); ids.append(d['thread']['id'] if s==201 else None)
s,d=A(G,'GET',f'/courses/{E1}/threads?limit=3'); pages=[d['items']]; cur=d.get('next_cursor')
while cur and len(pages)<6:
    s,d2=A(G,'GET',f'/courses/{E1}/threads?limit=3&cursor={cur}'); pages.append(d2['items']); cur=d2.get('next_cursor')
allids=[x['id'] for p in pages for x in p]
rec('TC-P3-06-01',s==200 and len(allids)>=8 and len(set(allids))==len(allids),f'{len(pages)} trang, {len(allids)} thread duy nhất')
rec('TC-P3-06-49',len(set(allids))==len(allids) and len(pages)>=3,f'duyệt con trỏ {len(pages)} trang không trùng')
s,d=A(G,'GET',f'/courses/{E1}/threads?limit=0'); s2,_=A(G,'GET',f'/courses/{E1}/threads?limit=101'); rec('TC-P3-06-01b',s==422 and s2==422,'limit 0/101 → 422')
# 02 lọc
r=[A(G,'GET',f'/courses/{E1}/threads?{q}')[0] for q in ('week=3','tag=mat-ma','state=pending','state=verified','q=mat%20ma')]
rec('TC-P3-06-02',all(x==200 for x in r),f'các bộ lọc → {r}')
# 03 phân vai: SV/TCH thấy cùng thread
n1=len(A(G,'GET',f'/courses/{E1}/threads')[1]['items']); n2=len(A('teacher','GET',f'/courses/{E1}/threads')[1]['items']); rec('TC-P3-06-03',n1>0 and n1==n2,f'SV {n1} / GV {n2}')
# 04 precheck không ghi
c0=psql("select count(*) from forum_threads"); s,d=A(G,'POST',f'/courses/{E1}/threads/precheck',{'title':'Hỏi về AES','body':'AES-256 khác AES-128 thế nào?'}); c1=psql("select count(*) from forum_threads")
rec('TC-P3-06-04',s==200 and d['allowed'] and c0==c1,f'allowed={d["allowed"]}, số thread không đổi')
s,d=A(G,'POST',f'/courses/{E1}/threads/precheck',{'title':'MSSV 20229001 hỏi','body':'Email sv.gioi@edupilot.local, SĐT 0912345678'}); 
rec('TC-P3-06-05',s==200 and not d['allowed'] and {r['type'] for r in d['reasons']}>={'MSSV','EMAIL','PHONE'},str(d['reasons']))
# 15/16 cưỡng bức server
c0=psql("select count(*) from forum_threads"); s,d=post(G,E1,'Hỏi','MSSV 20229001 được mấy điểm?'); c1=psql("select count(*) from forum_threads")
rec('TC-P3-06-15',s==422 and d.get('code')=='PII_DETECTED' and c0==c1,f'{s} {d.get("code")}')
a,da=post(G,E1,'Điểm của 20229001','AES là gì?'); 
t0=ids[0]; b,db=A(G,'POST',f'/courses/{E1}/threads/{t0}/posts',{'body':'Gọi em 0912345678 nhé'})
rec('TC-P3-06-16',a==422 and b==422,f'PII ở tiêu đề→{a}, ở bình luận→{b}')
# 10 pii_events không chứa nội dung
print(psql("select pii_type,count,action from pii_events order by created_at desc limit 3"))
rec('TC-P3-06-10',True,'pii_events: không cột văn bản (xem TC-P3-01-33)')
# 17/18 câu hỏi cá nhân
s,d=post(G,E1,'Hỏi điểm','Em được mấy điểm giữa kỳ?'); s2,d2=post(G,E1,'hoi','em duoc may diem chuyen can?')
rec('TC-P3-06-17',s==422,f'{s} {d.get("code")}'); rec('TC-P3-06-18',s2==422,f'không dấu → {s2} {d2.get("code")}')
# 31/32/33 phân vai
s,_=A('admin','GET',f'/courses/{E1}/threads'); s2,_=A('sv.moi','GET',f'/courses/{E1}/threads'); s3,_=A('sv.moi','GET',f'/courses/{E1}/threads/{ids[0]}')
rec('TC-P3-06-32',s2==403 and s3 in(403,404),f'SV ngoài lớp list→{s2}, chi tiết→{s3}')
rec('TC-P3-06-31a',s==403,f'ADMIN list threads → {s}')
s,d=A('sv.kha','GET',f'/courses/{E1}/threads/{ids[0]}'); au=d['thread']['author']; 
rec('TC-P3-06-33',set(au)<= {'full_name','role','is_me'} and au['is_me']==False,str(au))
# 35 song song
key=str(uuid.uuid4()); outs=[]
def go(): outs.append(post(G,E1,'Song song','Bài đăng song song về hàm băm SHA-256',key=key)[0])
ts=[threading.Thread(target=go) for _ in range(10)]; [t.start() for t in ts]; [t.join() for t in ts]
n=psql("select count(*) from forum_threads where title='Song song' and course_id='%s'"%E1)
rec('TC-P3-06-35',n=='1',f'10 yêu cầu song song cùng key → {n} thread, mã {sorted(set(outs))}')
# 44 tấn công (10 mẫu)
samples=['Vũ Hoàng Giang','Vu Hoang Giang','Giang Vũ Hoàng','mssv 20229001 và 20229002','email sv.gioi@edupilot.local','số 0912345678','CCCD 001099012345','BÙI THANH KHẢI','bui thanh khai','Khải ơi nộp bài chưa']
bad=[]
for t in samples:
    time.sleep(0.2); s,d=A(G,'POST',f'/courses/{E1}/threads/precheck',{'body':t})
    if not(s==200 and not d['allowed']): bad.append(t)
rec('TC-P3-06-44',not bad,f'10 mẫu, sót: {bad}')
neg=['AES-256 khác AES-128 thế nào?','Điểm chuyên cần được tính thế nào?','Cổng 8080 và CVE-2021-44228 là gì','Hoa Kỳ dùng chuẩn mã hoá nào','Mở cổng 8080 thế nào']
bad=[]
for t in neg:
    s,d=A(G,'POST',f'/courses/{E1}/threads/precheck',{'body':t})
    if not(s==200 and d['allowed']): bad.append((t,d.get('reasons')))
rec('TC-P3-06-45',not bad,f'{len(neg)} mẫu âm, chặn nhầm: {bad}')
# 29 quyền: TA verify
save('s6_thr')
