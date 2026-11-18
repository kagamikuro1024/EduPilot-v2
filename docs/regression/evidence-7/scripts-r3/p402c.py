from lib3 import *
mkusers('qc3t3','TEACHER',1)
E_=mkcourse('QC3E','qc3t3',['qc3ta'],'QKQ3EEE'); C7['E']=E_; json.dump(C7,open('/tmp/qc3e/c7.json','w'))
T='qc3t3'; Q=f'/courses/{E_}'; tu=uid(T)
for i in range(1,11):
    u='qc3s%02d'%i
    if not psql(f"select 1 from enrollments where course_id='{E_}' and user_id='{uid(u)}'"): A(u,'POST','/courses/join',{'code':'QKQ3EEE'})
def nt_(i):
    u='qc3s%02d'%i; _,_,am=mkchat(u,E_); s,d=esc(u,am); assert s==201,(s,d); return d['ticket']['id']
def rows(): return psql(f"select id||'|'||coalesce(read_at::text,'-')||'|'||title from notifications where user_id='{tu}' and type='TICKET_NEW' and course_id='{E_}' order by created_at")
# 09
c=SSEConn(T); time.sleep(1); t0=nt_(1); c.wait(lambda f:any(x[2]=='notification' for x in f),8)
E0=[x for x in c.frames if x[1]][-1][1]; c.close(); time.sleep(2)
for i in (2,3,4): nt_(i)
time.sleep(10)
c2=SSEConn(T,last=E0); time.sleep(3); got=[x for x in c2.frames if x[2]=='notification']; ids=[x[1] for x in got]
rec('TC-P4-02-09',len(got)==3 and len(set(ids))==3 and ids==sorted(ids),f'Last-Event-ID={E0}: nhận {len(got)} khung notification (3 ticket sinh khi ngắt 10 s) ids={ids}; các khung={[x[2] for x in c2.frames]}')
c2.close(); time.sleep(2)
# 10
bufk=f'ep:sse:buf:{tu}'; ex=rds('EXISTS',bufk); rds('DEL',bufk)
c3=SSEConn(T,last=E0); time.sleep(2)
rs=[x for x in c3.frames if x[2]=='resync']; ua=A(T,'GET','/notifications')[1]; ud=nrows(T,'and read_at is null')
rec('TC-P4-02-10',bool(rs) and 'buffer_exceeded' in (rs[0][3] or '') and str(ua.get('unread_count'))==ud,f'khoá {bufk[:20]}… tồn tại={ex}, xoá; nối Last-Event-ID cũ → {[(x[2],x[3]) for x in c3.frames][:3]}; unread API={ua.get("unread_count")} DB={ud}')
c3.close(); time.sleep(2)
# 11
c4=SSEConn(T,last='abc'); time.sleep(2); rs=[x for x in c4.frames if x[2]=='resync']
rec('TC-P4-02-11',c4.status==200 and bool(rs) and 'invalid_last_event_id' in (rs[0][3] or ''),f'status={c4.status}, khung={[(x[2],x[3]) for x in c4.frames][:3]}'); c4.close(); time.sleep(2)
# 22
cs=[SSEConn(T),SSEConn(T)]; time.sleep(0.5); c5=SSEConn(T,retry429=False)
rec('TC-P4-02-22',[x.status for x in cs]==[200,200] and c5.status==429 and 'SSE_LIMIT_REACHED' in c5.body,f'2 kết nối {[x.status for x in cs]}, thứ ba → {c5.status} {c5.body[:90]}')
for x in cs: x.close()
time.sleep(3)
# 16: đọc, rồi số giảm không đặt lại read_at và không phát SSE
r=rows().split('\n'); r1=r[-1].split('|'); print(r)
link=psql(f"select link from notifications where id='{r1[0]}'"); A(T,'POST','/notifications/read',{'link':link}); time.sleep(1)
rb=psql(f"select read_at::text from notifications where id='{r1[0]}'"); ce=SSEConn(T); time.sleep(1); nf0=len(ce.notif())
tid=psql(f"select id from escalation_tickets where course_id='{E_}' and status='OPEN' limit 1"); A(T,'POST',f'{Q}/tickets/{tid}/claim'); time.sleep(5)
rc=psql(f"select read_at::text||'/'||title from notifications where id='{r1[0]}'")
rec('TC-P4-02-16',bool(rb) and rc.startswith(rb) and len(ce.notif())==nf0,f'số 4→3 sau khi đọc: read_at trước {rb}, sau {rc}; khung notification mới={len(ce.notif())-nf0}')
# 15: xử lý hết → tự đọc
for t in psql(f"select id from escalation_tickets where course_id='{E_}' and status in ('OPEN','CLAIMED')").split('\n'):
    if t: A(T,'POST',f'{Q}/tickets/{t}/claim'); A(T,'POST',f'{Q}/tickets/{t}/dismiss',{'reason':'OUT_OF_SCOPE'})
time.sleep(4); last=rows().split('\n')[-1].split('|')
rec('TC-P4-02-15',last[1]!='-',f'xử lý hết → dòng read_at={last[1]} title={last[2]}')
# 14 F (xem p402c_def.txt)
rec('TC-P4-02-14',False,'F (DEF-R7-02): sau khi số về 0 (dòng tự đã đọc, tiêu đề giữ "1 câu hỏi…"), ticket mới (n=1) không nâng dòng/không chưa đọc/không khung SSE vì so n > leadingInt(title) cũ; xem evidence-7/scripts-r3/p402c_def.txt'+'; (SRS 4.4.3 cũng nói dòng giữ id và đặt read_at=NULL khi số tăng — id KHÁC trong TC là lời văn, không tính)')
# 20 replay outbox
t9=nt_(9); time.sleep(3); n_b=psql(f"select count(*)||'/'||coalesce(max(title),'') from notifications where user_id='{tu}' and type='TICKET_NEW' and course_id='{E_}'")
cc=SSEConn(T); time.sleep(1); f0=len(cc.notif())
payload=psql(f"select payload::text from outbox where topic='escalation.created' and payload->>'ticket_id'='{t9}' limit 1")
psql(f"insert into outbox(topic,payload) values('escalation.created','{payload}')"); time.sleep(8)
n_a=psql(f"select count(*)||'/'||coalesce(max(title),'') from notifications where user_id='{tu}' and type='TICKET_NEW' and course_id='{E_}'")
rec('TC-P4-02-20',n_b==n_a and len(cc.notif())==f0,f'phát lại outbox escalation.created (ticket {t9[:8]}): thông báo {n_b} → {n_a}; khung SSE mới={len(cc.notif())-f0}'); cc.close()
# E1
s1,d1=A(T,'POST','/notifications/read',{'link':''}); lk=psql(f"select link from notifications where user_id='{tu}' and type='TICKET_NEW' limit 1")
ub=psql(f"select count(*) from notifications where user_id='{tu}' and read_at is null"); s2,d2=A('qc3s01','POST','/notifications/read',{'link':lk}); ua=psql(f"select count(*) from notifications where user_id='{tu}' and read_at is null")
rec('TC-P4-02-E1',s1==422 and s2==200 and (d2 or {}).get('updated')==0 and ub==ua,f'link rỗng → {s1}; link của người khác → {s2} {d2}; chưa đọc của GV {ub}→{ua}')
adm=psql(f"select count(*) from notifications where user_id='{uid('admin')}' and type='TICKET_NEW'"); stu=psql(f"select count(*) from notifications where user_id in (select id from users where email like 'qc3s%') and type='TICKET_NEW'")
rec('TC-P4-02-P1',adm=='0' and stu=='0',f'dòng TICKET_NEW của ADMIN={adm}, của sinh viên QC3={stu}')
save('p402c')
