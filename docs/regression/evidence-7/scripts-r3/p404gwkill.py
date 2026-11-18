from lib3 import *
H=C7['H']; Q=f'/courses/{H}'
def up():
    for _ in range(40):
        try:
            c=http.client.HTTPConnection('localhost',18090,timeout=2); c.request('GET','/api/v1/readyz'); r=c.getresponse(); r.read(); 
            if r.status==200: return True
        except Exception: pass
        time.sleep(0.5)
    return False
subprocess.run(['docker','start','ep-r-gateway-qc3b'],capture_output=True); assert up()
clearopen('qc3t',H)
studs=['qc3s%02d'%i for i in range(1,31)]; toks={u:login(u) for u in studs}
msgs=[]
for r in range(20):
    for j in range(5):
        u=studs[(r*5+j)%30]; msgs.append((u,mkchat(u,H)[2]))
print('messages',len(msgs))
# mỗi sinh viên tối đa 5 ticket mở -> 100 ticket / 30 SV = ≤4 mỗi người
psql("create or replace function qc3_slow4() returns trigger as $$ begin if NEW.course_id='%s' then perform pg_sleep(0.4); end if; return NEW; end $$ language plpgsql" % H)
psql("create trigger qc3_slow4_t after insert on escalation_tickets for each row execute function qc3_slow4()")
t_start=psql("select now()"); res=[]; 
for r in range(20):
    batch=msgs[r*5:(r+1)*5]; out=[]
    def go(u,am): 
        out.append(rawB('POST',f'/chat/messages/{am}/escalate',toks[u],port=18090)[0])
    ths=[threading.Thread(target=go,args=b) for b in batch]; [t.start() for t in ths]
    time.sleep(0.12+0.05*(r%4)); subprocess.run(['docker','kill','-s','KILL','ep-r-gateway-qc3b'],capture_output=True)
    [t.join() for t in ths]; res.append(sorted(out)); subprocess.run(['docker','start','ep-r-gateway-qc3b'],capture_output=True); up()
psql("drop trigger if exists qc3_slow4_t on escalation_tickets"); psql("drop function if exists qc3_slow4()")
time.sleep(5)
orph_t=psql(f"select count(*) from escalation_tickets t where t.course_id='{H}' and t.created_at>'{t_start}' and not exists (select 1 from outbox o where o.topic='escalation.created' and o.payload->>'ticket_id'=t.id::text)")
orph_o=psql(f"select count(*) from outbox o where o.topic='escalation.created' and o.payload->>'course_id'='{H}' and o.created_at>'{t_start}' and not exists (select 1 from escalation_tickets t where t.id::text=o.payload->>'ticket_id')")
nt_=psql(f"select count(*) from escalation_tickets t where t.course_id='{H}' and t.created_at>'{t_start}'"); no=psql(f"select count(*) from outbox o where o.topic='escalation.created' and o.payload->>'course_id'='{H}' and o.created_at>'{t_start}'")
ok201=sum(x.count(201) for x in res); dead=sum(x.count(0) for x in res)
rec('TC-P4-04-12',orph_t=='0' and orph_o=='0' and nt_==no,f'20 vòng: mỗi vòng 5 yêu cầu escalate song song tới gateway phụ rồi SIGKILL gateway sau 120–270 ms, khi trigger tạm (pg_sleep 0,4 s sau INSERT ticket, trước COMMIT) đang giữ giao dịch (gateway-1 giữ nguyên); {ok201} yêu cầu 201, {dead} bị cắt; ticket tạo trong cửa sổ={nt_}, outbox escalation.created={no}; ticket thiếu dòng outbox={orph_t}; dòng outbox mồ côi (không ticket)={orph_o}')
save('p404gwkill')
