from lib3 import *
A_=C7['A']; Q=f'/courses/{A_}'; T='qc3t'
def dk(*a): return subprocess.run(['docker',*a],capture_output=True,text=True)
def worker_up():
    for _ in range(60):
        st=dk('inspect','ep-r-worker-1','--format','{{.State.Status}} {{.State.Health.Status}}').stdout.strip()
        if st=='running healthy': return True
        if st.startswith('exited'): dk('start','ep-r-worker-1')
        time.sleep(2)
    return False
def mk(u,q='Câu hỏi kiểm tra kill worker'):
    _,_,am=mkchat(u,A_,q=q); s,d=esc(u,am); t=d['ticket']['id']; A(T,'POST',f'{Q}/tickets/{t}/claim'); return t
clearopen(T,A_)
for u in ('qc3s09','qc3s10'): psql(f"update user_settings set notify_answer_by_mail=true where user_id='{uid(u)}'")
t9=mk('qc3s09'); t10=mk('qc3s10'); [login(u) for u in ('qc3s09','qc3s10','qc3t')]
E9=f'{uid("qc3s09")}'; R={}
assert lockg('qc3','Sprint7 cửa sổ kill worker (P4-04-09/10): pause mailpit / trigger pg_sleep, SIGKILL worker; ≤ 5 phút')
try:
    # 09
    dk('pause','ep-r-mailpit-1'); s,_=A(T,'POST',f'{Q}/tickets/{t9}/answer',{'body':'Chào em, thư thử kill trước SMTP.'}); R['a9']=s
    time.sleep(3); R['st9a']=psql("select status||'/'||attempts from mail_outbox where to_addr='qc3s09@edupilot.local' and template='escalation_answered' order by created_at desc limit 1")
    k=dk('kill','-s','KILL','ep-r-worker-1'); R['kill9']=k.returncode; dk('unpause','ep-r-mailpit-1'); R['up9']=worker_up()
    for _ in range(60):
        r=psql("select status from mail_outbox where to_addr='qc3s09@edupilot.local' and template='escalation_answered' order by created_at desc limit 1")
        if r=='SENT': break
        time.sleep(1)
    R['st9b']=r; time.sleep(2); R['mp9']=len(mpq('to:qc3s09@edupilot.local subject:"Giảng viên đã trả lời"'))
    # 10
    psql("create or replace function qc3_slow() returns trigger as $$ begin if NEW.status='SENT' and OLD.status<>'SENT' and NEW.to_addr='qc3s10@edupilot.local' then perform pg_sleep(12); end if; return NEW; end $$ language plpgsql")
    psql("create trigger qc3_slow_t before update on mail_outbox for each row execute function qc3_slow()")
    s,_=A(T,'POST',f'{Q}/tickets/{t10}/answer',{'body':'Chào em, thư thử kill sau SMTP.'}); R['a10']=s
    t0=time.time(); got=False
    while time.time()-t0<30:
        if len(mpq('to:qc3s10@edupilot.local subject:"Giảng viên đã trả lời"'))>=1: got=True; break
        time.sleep(0.3)
    R['mp10_before']=got; R['st10a']=psql("select status from mail_outbox where to_addr='qc3s10@edupilot.local' and template='escalation_answered' order by created_at desc limit 1")
    k=dk('kill','-s','KILL','ep-r-worker-1'); R['kill10']=k.returncode
    psql("drop trigger qc3_slow_t on mail_outbox"); psql("drop function qc3_slow()")
    R['up10']=worker_up()
    for _ in range(60):
        r=psql("select status from mail_outbox where to_addr='qc3s10@edupilot.local' and template='escalation_answered' order by created_at desc limit 1")
        if r=='SENT': break
        time.sleep(1)
    R['st10b']=r; time.sleep(3)
    ms=mpq('to:qc3s10@edupilot.local subject:"Giảng viên đã trả lời"'); R['mp10']=len(ms)
    mo=psql("select id from mail_outbox where to_addr='qc3s10@edupilot.local' and template='escalation_answered' order by created_at desc limit 1")
    R['mid10']=mp('/api/v1/message/'+ms[0]['ID']).get('MessageID')==mo+'@edupilot.local' if ms else None
finally:
    dk('unpause','ep-r-mailpit-1')
    psql("drop trigger if exists qc3_slow_t on mail_outbox"); psql("drop function if exists qc3_slow()")
    worker_up()
    unlockg('qc3','worker chạy lại, mailpit unpause, trigger đã gỡ')
print(R); json.dump(R,open('/tmp/qc3e/w_kill.json','w'),ensure_ascii=False)
rec('TC-P4-04-09',R['a9']==200 and R['st9b']=='SENT' and R['mp9']==1 and R['up9'],f'pause Mailpit → answer ({R["a9"]}) → worker đang kẹt gửi (dòng {R["st9a"]}) → SIGKILL worker (rc {R["kill9"]}), unpause, worker tự chạy lại ({R["up9"]}): mail_outbox → {R["st9b"]}, Mailpit {R["mp9"]} thư tới qc3s09 (đúng 1), không mất thư [điểm 1: trước khi ghi SENT; trạng thái PROCESSING không có (khoá dòng trong giao dịch)]')
rec('TC-P4-04-10',R['mp10_before'] and R['st10a']=='QUEUED' and R['mp10']==1 and R['st10b']=='SENT' and R['mid10'],f'trigger tạm làm bước ghi SENT chậm 12 s; Mailpit đã nhận thư trong khi dòng còn {R["st10a"]} → SIGKILL worker → worker lên lại: Mailpit có đúng {R["mp10"]} thư mang Message-ID=<mail_outbox.id@…> (khớp={R["mid10"]}, MP_IGNORE_DUPLICATE_IDS), mail_outbox={R["st10b"]}; trigger đã gỡ')
save('w_kill')
