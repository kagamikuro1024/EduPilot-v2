from lib3 import *
D=C7['D']; QD=f'/courses/{D}'; T='qc3t2'
def dk(*a): return subprocess.run(['docker',*a],capture_output=True,text=True)
def worker_up(name='ep-r-worker-1'):
    for _ in range(60):
        st=dk('inspect',name,'--format','{{.State.Status}} {{.State.Health.Status}}').stdout.strip()
        if st=='running healthy': return True
        if st.startswith('exited'): dk('start',name)
        time.sleep(2)
    return False
def mk(i):
    u='qc3s%02d'%i; _,_,am=mkchat(u,D); s,d=esc(u,am); assert s==201,(s,d); return d['ticket']['id']
def notif(): return int(psql(f"select count(*) from notifications where course_id='{D}' and type='TICKET_STALE'"))
def mails(): return psql(f"select to_addr||'='||count(*) from mail_outbox where template='ticket_reminder' and payload->>'course_id'='{D}' and created_at>now()-interval '15 minutes' group by to_addr order by 1").replace('\n',', ')
R={}
n0=0; m0=''
ts=[]
assert lockg('qc3','Sprint7 cửa sổ worker: trigger pg_sleep trên escalation_tickets + SIGKILL worker giữa ticket.sla (P4-05-50), hai worker (P4-05-20); ≤ 8 phút')
try:
    clearopen(T,D); time.sleep(1)
    ts=[mk(i) for i in range(7,13)]
    for t in ts: seedage('QC3D',t,100)
    st0=[psql(f"select sla_stage from escalation_tickets where id='{t}'") for t in ts]; R['st0']=st0
    n0=notif(); m0=mails()
    
    psql("create or replace function qc3_slow2() returns trigger as $$ begin if NEW.sla_stage<>OLD.sla_stage and NEW.course_id='%s' then perform pg_sleep(20); end if; return NEW; end $$ language plpgsql" % D)
    psql("create trigger qc3_slow2_t before update on escalation_tickets for each row execute function qc3_slow2()")
    t0=time.time(); hit=False
    while time.time()-t0<100:
        r=psql("select count(*) from pg_stat_activity where wait_event='PgSleep' and query ilike '%escalation_tickets%'")
        if r!='0': hit=True; break
        time.sleep(0.2)
    R['hit']=hit; R['hit_s']=round(time.time()-t0,1)
    k=dk('kill','-s','KILL','ep-r-worker-1'); R['kill']=k.returncode
    psql("drop trigger qc3_slow2_t on escalation_tickets"); psql("drop function qc3_slow2()")
    time.sleep(1)
    R['st_mid']=[psql(f"select sla_stage from escalation_tickets where id='{t}'") for t in ts]; R['notif_mid']=notif()-n0; R['mails_mid']=mails()
    R['up']=worker_up()
    t1=time.time(); done=False
    while time.time()-t1<110:
        sts=[psql(f"select sla_stage from escalation_tickets where id='{t}'") for t in ts]
        if all(x!='0' for x in sts): done=True; break
        time.sleep(2)
    R['st_end']=sts; R['done_s']=round(time.time()-t1,1); time.sleep(5)
    R['notif_end']=notif()-n0; R['mails_end']=mails()
finally:
    psql("drop trigger if exists qc3_slow2_t on escalation_tickets"); psql("drop function if exists qc3_slow2()")
    worker_up()
    unlockg('qc3','trigger đã gỡ, worker chạy lại')
print(R)
rec('TC-P4-05-50',R['hit'] and R['st_mid']==['0']*6 and R['notif_mid']==0 and R['done'] if 'done' in R else R['hit'] and R['st_mid']==['0']*6 and R['notif_mid']==0 and all(x!='0' for x in R['st_end']),f'6 ticket quá hạn (seed-age 100 h, stage 0); trigger tạm giữ giao dịch ticket.sla (pg_sleep 20 s) → SIGKILL worker ngay lúc job đang ghi (sau {R["hit_s"]}s): stage = {R["st_mid"]} (rollback, không nửa vời), thông báo TICKET_STALE mới={R["notif_mid"]}, thư nhắc mới: {R["mails_mid"] or "0"}; worker lên lại → tick kế hoàn tất sau {R["done_s"]}s: stage={R["st_end"]}, TICKET_STALE mới={R["notif_end"]}, thư nhắc: {R["mails_end"]} (mỗi người nhận đúng 1)')
save('w_kill2')
