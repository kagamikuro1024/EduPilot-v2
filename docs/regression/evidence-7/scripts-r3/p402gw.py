from lib3 import *
mkusers('qc3t6','TEACHER',1)
H2=mkcourse('QC3I','qc3t6',['qc3ta'],'QKQ3JJK'); C7['I']=H2; json.dump(C7,open('/tmp/qc3e/c7.json','w'))
T='qc3t6'; tu=uid(T); Q=f'/courses/{H2}'
for i in range(1,11):
    u='qc3s%02d'%i
    if not psql(f"select 1 from enrollments where course_id='{H2}' and user_id='{uid(u)}'"): A(u,'POST','/courses/join',{'code':'QKQ3JJK'})
for u in ['qc3s%02d'%i for i in range(1,11)]+[T]: login(u)
def msg(i): return mkchat('qc3s%02d'%i,H2)[2]
def escB(i,am): return rawB('POST',f'/chat/messages/{am}/escalate',login('qc3s%02d'%i),port=18090)
def nf(c): return len([f for f in c.frames if f[2]=='notification'])
R={}
# 07a: SSE ở B, escalate qua A (Caddy → gateway-1)
cB=SSEConn2(T,18090); time.sleep(1); n0=nf(cB); t0=time.time(); s,d=esc('qc3s01',msg(1)); ok=cB.wait(lambda f:len([x for x in f if x[2]=='notification'])>n0,8); ra=(s,round(time.time()-t0,2) if ok else None); cB.close()
# 07b: SSE ở A (Caddy), escalate trúng B
cA=SSEConn(T); time.sleep(1); n0=nf(cA); t0=time.time(); sb,db=escB(2,msg(2)); ok=cA.wait(lambda f:len([x for x in f if x[2]=='notification'])>n0,8); rb=(sb,round(time.time()-t0,2) if ok else None)
rec('TC-P4-02-07',ra[0]==201 and ra[1] is not None and ra[1]<=5 and rb[0]==201 and rb[1] is not None and rb[1]<=5,f'hai gateway dùng chung Redis/Postgres (gateway-1 qua Caddy, gateway phụ :18090): SSE ở B + escalate trúng A → B nhận khung notification sau {ra[1]} s (escalate {ra[0]}); SSE ở A + escalate trúng B ({rb[0]}) → A nhận sau {rb[1]} s (≤5 s cả hai chiều)'); cA.close()
# 08: dừng B khi SSE đang mở; nối lại A với Last-Event-ID
cB=SSEConn2(T,18090); time.sleep(1); 
sx,_=escB(3,msg(3)); cB.wait(lambda f:len([x for x in f if x[2]=='notification'])>=1,8)
last=[x for x in cB.frames if x[1]][-1][1]; got_before=nf(cB)
subprocess.run(['docker','stop','-t','1','ep-r-gateway-qc3b'],capture_output=True)
mids=[msg(i) for i in (4,5,6)]
for i,am in zip((4,5,6),mids): esc('qc3s%02d'%i,am); time.sleep(3.2)
time.sleep(3)
cA2=SSEConn(T,last=last); time.sleep(3)
got=[x for x in cA2.frames if x[2]=='notification']; ids=[x[1] for x in got]
rec('TC-P4-02-08',len(got)==3 and len(set(ids))==3 and ids==sorted(ids) and all(i>last for i in ids),f'dừng gateway B khi SSE đang mở (đã nhận {got_before} khung, Last-Event-ID={last}); trong lúc B tắt tạo 3 ticket qua A; nối lại vào A với Last-Event-ID → nhận {len(got)} khung notification ids={ids}, đúng thứ tự, không trùng, không mất (bộ đệm Redis dùng chung, gateway không giữ trạng thái)')
cA2.close()
subprocess.run(['docker','start','ep-r-gateway-qc3b'],capture_output=True)
save('p402gw')
