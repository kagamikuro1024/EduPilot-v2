# TC-P4-02-14 / DEF-R7-02 chạy lại sau 425396a7 (có e4b82a05): lớp QC3G, GV qc3t5, SV qc3s01..05
from lib3 import *
import datetime as dt
G=C7['G']; T='qc3t5'; Q=f'/courses/{G}'; tu=uid(T)
clearopen(T,G); time.sleep(2)
def rows(): return psql(f"select id||'|'||coalesce(read_at::text,'-')||'|'||title||'|'||updated_at::text from notifications where user_id='{tu}' and type='TICKET_NEW' and course_id='{G}' order by created_at")
def nt_(i):
    u='qc3s%02d'%i; _,_,am=mkchat(u,G,q=f'Câu hỏi r7-14 từ {u}'); s,d=esc(u,am); assert s==201,(s,d); return d['ticket']['id']
T0=time.strftime('%H:%M:%S')
c=SSEConn(T); time.sleep(1); assert c.status==200,c.status
out=[]
def snap(tag): r=rows().split('\n')[-1]; print(tag,r); out.append((tag,r)); return r.split('|')
# vòng A (kịch bản DEF-R7-02): ticket → xử lý hết (n=0) → ticket mới
t1=nt_(1); time.sleep(4); a1=snap('A1 ticket 1 (n=1)')
A(T,'POST',f'{Q}/tickets/{t1}/claim'); A(T,'POST',f'{Q}/tickets/{t1}/dismiss',{'reason':'OUT_OF_SCOPE'}); time.sleep(4); a2=snap('A2 dismiss (n=0)')
n0=len(c.notif()); t2=nt_(2); time.sleep(6); a3=snap('A3 ticket mới (n=1)'); fa=len(c.notif())-n0
okA=a3[0]==a1[0] and a3[1]=='-' and a3[2].startswith('1 ') and a2[1]!='-' and fa>=1
# vòng B: GV đọc dòng (link) rồi xử lý hết, ticket mới lần nữa (n lại =1 ≤ số cũ)
link=psql(f"select link from notifications where id='{a3[0]}'"); A(T,'POST','/notifications/read',{'link':link}); time.sleep(1); b0=snap('B0 GV đọc (n=1)')
A(T,'POST',f'{Q}/tickets/{t2}/claim'); A(T,'POST',f'{Q}/tickets/{t2}/dismiss',{'reason':'OUT_OF_SCOPE'}); time.sleep(4); b1=snap('B1 dismiss (n=0)')
n1=len(c.notif()); t3=nt_(3); time.sleep(6); b2=snap('B2 ticket mới (n=1)'); fb=len(c.notif())-n1
okB=b0[1]!='-' and b2[1]=='-' and b2[2].startswith('1 ') and fb>=1
# vòng C: hai ticket cùng lúc rồi n=2 → tiêu đề đếm lại
n2=len(c.notif()); t4=nt_(4); time.sleep(5); c1=snap('C1 thêm ticket (n=2)')
print('khung SSE notification:',[ (x[1],x[3][:90]) for x in c.notif()][-4:])
c.close(); clearopen(T,G)
rec('TC-P4-02-14',okA and okB,f'lớp QC3G (GV qc3t5, {T0}): A) ticket → xử lý hết (dòng tự đọc «{a2[2]}», read_at đặt) → ticket mới ⇒ cùng dòng {a3[0][:8]} đưa lại chưa đọc (read_at={a3[1]}), tiêu đề «{a3[2]}», khung SSE notification mới={fa}; B) GV đọc dòng, xử lý hết, ticket mới ⇒ read_at={b2[1]}, «{b2[2]}», khung SSE mới={fb}; C) thêm ticket ⇒ «{c1[2]}»')
