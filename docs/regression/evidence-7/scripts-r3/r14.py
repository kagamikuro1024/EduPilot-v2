from lib3 import *
mkusers('qc3t5','TEACHER',1)
G=mkcourse('QC3G','qc3t5',['qc3ta'],'QKQ3GGG'); C7['G']=G; json.dump(C7,open('/tmp/qc3e/c7.json','w'))
T='qc3t5'; Q=f'/courses/{G}'
for i in range(1,6):
    u='qc3s%02d'%i
    if not psql(f"select 1 from enrollments where course_id='{G}' and user_id='{uid(u)}'"): A(u,'POST','/courses/join',{'code':'QKQ3GGG'})
def rows(): return psql(f"select id||'|'||coalesce(read_at::text,'-')||'|'||title from notifications where user_id='{uid(T)}' and type='TICKET_NEW' and course_id='{G}' order by created_at")
def nt_(i):
    u='qc3s%02d'%i; _,_,am=mkchat(u,G); s,d=esc(u,am); assert s==201,(s,d); return d['ticket']['id']
c=SSEConn(T); time.sleep(1)
t1=nt_(1); time.sleep(4); r1=rows(); print('1',r1)
id1=r1.split('|')[0]; A(T,'POST',f'{Q}/tickets/{t1}/claim'); A(T,'POST',f'{Q}/tickets/{t1}/dismiss',{'reason':'OUT_OF_SCOPE'}); time.sleep(4); r2=rows(); print('2 (n=0)',r2)
n0=len(c.notif()); t2=nt_(2); time.sleep(6); r3=rows(); print('3 (ticket mới)',r3,'khung mới',len(c.notif())-n0)
p=r3.split('|')
rec('TC-P4-02-14',p[1]=='-' and len(c.notif())>n0 and p[2].startswith('1 '),f'ticket1→đã xử lý (dòng tự đọc) → ticket mới: dòng {p[0][:8]} (cũ {id1[:8]}) read_at={p[1]} title="{p[2]}", khung SSE mới={len(c.notif())-n0} (ep-r @1409b04d gồm fix R7-02 e4b82a05)')
save('r14')
