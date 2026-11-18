from lib3 import *
import statistics
mkusers('qc3t2','TEACHER',1)
D=mkcourse('QC3D','qc3t2',['qc3ta'],'QKQ3DDD'); C7['D']=D; json.dump(C7,open('/tmp/qc3e/c7.json','w'))
T='qc3t2'; Q=f'/courses/{D}'; tu=uid(T)
for i in range(1,21):
    if not psql(f"select 1 from enrollments where course_id='{D}' and user_id='{uid('qc3s%02d'%i)}'"): A('qc3s%02d'%i,'POST','/courses/join',{'code':'QKQ3DDD'})
def nticket(i):
    u='qc3s%02d'%i; _,_,am=mkchat(u,D); s,d=esc(u,am); assert s==201,(s,d); return d['ticket']['id']
def tn_rows(): return psql(f"select id||'|'||coalesce(read_at::text,'-')||'|'||title from notifications where user_id='{tu}' and type='TICKET_NEW' and course_id='{D}' order by created_at")
tids=[r for r in psql(f"select id from escalation_tickets where course_id='{D}'").split('\n') if r]
# 09 đọc bù Last-Event-ID
c=SSEConn(T); time.sleep(1); n0=len(c.notif())
_=A('qc3s21' if False else 'qc3s01','GET','/me/today')
base=tn_rows()
# tạo khung E0 (một ticket) rồi ngắt
dummy=nticket_extra=None
def free_student():
    for i in range(1,21):
        u='qc3s%02d'%i
        if int(psql(f"select count(*) from escalation_tickets where student_id='{uid(u)}' and course_id='{D}' and status in ('OPEN','CLAIMED')"))<5: return i
# dọn: đóng hết ticket cũ để trần 5 không cản
for tid in tids: A(T,'POST',f'{Q}/tickets/{tid}/dismiss',{'reason':'OUT_OF_SCOPE'})
time.sleep(2)
ev0=len(c.frames); i0=free_student(); nticket(i0); c.wait(lambda f:len([x for x in f if x[2]=='notification'])>n0,8)
print('c.status',c.status,[(f[2],f[1]) for f in c.frames],n0,i0); E0=[x for x in c.frames if x[1]][-1][1]; c.close(); time.sleep(0.5)
for i in (2,3,4): nticket(i)
time.sleep(10)
c2=SSEConn(T,last=E0); time.sleep(3)
got=[x for x in c2.frames if x[2]=='notification']; ids=[x[1] for x in got]
rec('TC-P4-02-09',len(got)==3 and len(set(ids))==3 and ids==sorted(ids),f'Last-Event-ID={E0}: nhận {len(got)} khung notification ids={ids}, các khung={[x[2] for x in c2.frames]}')
c2.close()
# 10 bộ đệm quá hạn
c=SSEConn(T); time.sleep(1); c.close()
keys=rds('--scan','--pattern',f'*{tu}*'); print('redis keys',keys)
bufk=f'ep:sse:buf:{tu}'; ex=rds('EXISTS',bufk); rds('DEL',bufk)
c3=SSEConn(T,last=E0); time.sleep(2)
rs=[x for x in c3.frames if x[2]=='resync']; unread_api=A(T,'GET','/notifications')[1]; unread_db=nrows(T,'and read_at is null')
rec('TC-P4-02-10',bool(rs) and 'buffer_exceeded' in (rs[0][3] or '') and str(unread_api.get('unread_count'))==unread_db,f'buf tồn tại={ex}, xoá; nối Last-Event-ID cũ → khung {[(x[2],x[3]) for x in c3.frames][:3]}; unread API={unread_api.get("unread_count")} DB={unread_db}')
c3.close()
# 11 id sai
c4=SSEConn(T,last='abc'); time.sleep(2)
rs=[x for x in c4.frames if x[2]=='resync']
rec('TC-P4-02-11',c4.status==200 and bool(rs) and 'invalid_last_event_id' in (rs[0][3] or ''),f'status={c4.status}, khung={[(x[2],x[3]) for x in c4.frames][:3]}'); c4.close()
# 22 tab thứ ba
cs=[SSEConn(T),SSEConn(T)]; time.sleep(0.5); c5=SSEConn(T,retry429=False)
rec('TC-P4-02-22',[x.status for x in cs]==[200,200] and c5.status==429 and 'SSE_LIMIT_REACHED' in c5.body,f'2 kết nối {[x.status for x in cs]}, thứ ba → {c5.status} {c5.body[:80]}')
for x in cs: x.close()
# 16/14/15: dọn → 3 ticket → đọc → nhận 1 → mới → xử lý hết
for r in psql(f"select id from escalation_tickets where course_id='{D}' and status in ('OPEN','CLAIMED')").split('\n'):
    if r: A(T,'POST',f'{Q}/tickets/{r}/dismiss',{'reason':'OUT_OF_SCOPE'})
time.sleep(2)
t3=[nticket(i) for i in (5,6,7)]; time.sleep(3)
rows=tn_rows().split('\n'); r1=rows[-1].split('|'); print('rows',rows)
link=psql(f"select link from notifications where id='{r1[0]}'")
s,d=A(T,'POST','/notifications/read',{'link':link}); time.sleep(1)
r1b=psql(f"select read_at::text from notifications where id='{r1[0]}'")
ce=SSEConn(T); time.sleep(1); nf0=len(ce.notif())
A(T,'POST',f'{Q}/tickets/{t3[0]}/claim'); time.sleep(5)
r1c=psql(f"select read_at::text||'/'||title from notifications where id='{r1[0]}'")
rec('TC-P4-02-16',r1b!='' and r1c.startswith(r1b) and len(ce.notif())==nf0,f'đã đọc lúc {r1b}; sau claim (3→2): {r1c}; khung notification mới={len(ce.notif())-nf0}')
tn=nticket(8); time.sleep(3); rows2=tn_rows().split('\n'); newr=rows2[-1].split('|')
rec('TC-P4-02-14',newr[0]!=r1[0] and newr[1]=='-',f'sau khi đọc, ticket mới → dòng mới id khác={newr[0]!=r1[0]}, read_at={newr[1]}, title={newr[2]}')
for t in t3[1:]+[tn,t3[0]]: A(T,'POST',f'{Q}/tickets/{t}/claim'); 
for t in t3+[tn]: A(T,'POST',f'{Q}/tickets/{t}/dismiss',{'reason':'OUT_OF_SCOPE'})
time.sleep(3); last=tn_rows().split('\n')[-1].split('|')
rec('TC-P4-02-15',last[1]!='-',f'hết ticket chờ → dòng mới nhất read_at={last[1]} title={last[2]}'); ce.close()
# 20 idempotent: phát lại outbox
tid=nticket(9); time.sleep(3); n_before=psql(f"select count(*) from notifications where user_id='{tu}' and type='TICKET_NEW' and course_id='{D}'")
cc=SSEConn(T); time.sleep(1); f0=len(cc.notif())
payload=psql(f"select payload::text from outbox where topic='escalation.created' and payload->>'ticket_id'='{tid}' limit 1")
psql(f"insert into outbox(topic,payload) values('escalation.created','{payload}')"); time.sleep(8)
n_after=psql(f"select count(*) from notifications where user_id='{tu}' and type='TICKET_NEW' and course_id='{D}'")
rec('TC-P4-02-20',n_before==n_after and len(cc.notif())==f0,f'phát lại outbox escalation.created (payload ticket {tid[:8]}): số dòng thông báo {n_before}→{n_after}, khung SSE mới={len(cc.notif())-f0}')
cc.close()
# E1
s1,d1=A(T,'POST','/notifications/read',{'link':''}); 
sv_link=psql(f"select link from notifications where user_id='{tu}' and type='TICKET_NEW' order by created_at desc limit 1")
b=nrows('qc3s01',"and read_at is null"); s2,d2=A('qc3s01','POST','/notifications/read',{'link':sv_link}); 
tt=psql(f"select count(*) from notifications where user_id='{tu}' and read_at is null")
rec('TC-P4-02-E1',s1==422 and s2==200 and (d2 or {}).get('updated')==0,f'link rỗng → {s1}; link của người khác → {s2} {d2}; thông báo người kia không đổi (chưa đọc {tt})')
# P1
adm=psql(f"select count(*) from notifications where user_id='{uid('admin')}' and type='TICKET_NEW'"); stu=psql(f"select count(*) from notifications where user_id='{uid('qc3s09')}' and type='TICKET_NEW'")
rec('TC-P4-02-P1',adm=='0' and stu=='0',f'SV nhận TICKET_NEW={stu}; ADMIN nhận TICKET_NEW={adm}')
save('p402')
