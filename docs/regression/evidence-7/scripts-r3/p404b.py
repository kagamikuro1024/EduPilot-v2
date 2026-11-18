from lib3 import *
A_=C7['A']; E_=C7['E']; F_=C7['F']; Q=f'/courses/{A_}'; T='qc3t'
def mails(addr,sub=None): return mpq(f'to:{addr}'+(f' subject:"{sub}"' if sub else ''),50)
def mailrow(to,tpl,payload):
    mid=psql(f"insert into mail_outbox(to_addr,template,payload,status) values('{to}','{tpl}','{payload}','QUEUED') returning id").split('\n')[0]
    psql(f"insert into outbox(topic,payload) values('mail.send','{{\"mail_id\":\"{mid}\"}}')"); return mid
def waitdead(mid,timeout=30):
    for _ in range(timeout*2):
        r=psql(f"select status||'/'||attempts||'/'||coalesce(last_error,'-') from mail_outbox where id='{mid}'")
        if not r.startswith('QUEUED'): return r
        time.sleep(0.5)
    return r
# 35 payload hỏng; 36 ticket đã xoá; 21 người nhận bị từ chối (bad_recipient)
n0=len(mpq('to:qc3s10@edupilot.local'))
m35=mailrow('qc3s10@edupilot.local','escalation_answered','{}'); r35=waitdead(m35)
rec('TC-P4-04-35',r35.startswith('DEAD') and 'payload_invalid' in r35 and len(mpq('to:qc3s10@edupilot.local'))==n0,f'dòng thiếu ticket_id → {r35}; Mailpit thêm thư={len(mpq("to:qc3s10@edupilot.local"))-n0}')
g=str(uuid.uuid4()); m36=mailrow('qc3s10@edupilot.local','escalation_answered',json.dumps({'ticket_id':g,'answer_message_id':str(uuid.uuid4())})); r36=waitdead(m36)
rec('TC-P4-04-36',r36.startswith('DEAD') and 'ticket_missing' in r36,f'ticket đã xoá/không tồn tại → {r36} (không thử lại: attempts trong nhãn)')
t,am=tick('qc3s09',A_); sidn=None
m21=mailrow('khong hop le','escalation_answered',json.dumps({'ticket_id':t,'answer_message_id':am})); r21=waitdead(m21)
rec('TC-P4-04-21',r21.startswith('DEAD') and r21.split('/')[1] in('0','1'),f'địa chỉ bị từ chối (To không hợp lệ) → {r21}; DEAD ngay, không thử lại (SMTP 550 thật không dựng được trên Mailpit: lớp lỗi vĩnh viễn kiểm bằng bad_recipient + TestSMTP 5xx của dev)')
# 15/16 ticket_new
clearopen('qc3t4',F_); s,d=A('qc3t4','GET','/me/settings'); s1,d1=A('qc3t4','PUT','/me/settings',{'notify_ticket_by_mail':True,'version':d['version']})
tu=uid('qc3t4'); rds('DEL',f'ep:mail:ticket_new:{tu}:{F_}'); b0=len(mails('qc3t4@edupilot.local','ticket'))
ts=[tick(u,F_)[0] for u in ('qc3s11','qc3s12','qc3s13')]; time.sleep(12)
mt=mpq('to:qc3t4@edupilot.local',20); newm=[x for x in mt if 'câu hỏi' in x['Subject'].lower() and 'trả lời' not in x['Subject'].lower() and 'quá' not in x['Subject']]
print([x['Subject'] for x in mt])
ttl=rds('TTL',f'ep:mail:ticket_new:{tu}:{F_}'); bt=body_ = (mp('/api/v1/message/'+newm[0]['ID']).get('Text') or '') if newm else ''
rec('TC-P4-04-15',s1==200 and len(newm)==1,f'bật notify_ticket_by_mail ({s1}); 3 ticket trong ~10 s → {len(newm)} thư ticket_new gộp, tiêu đề "{newm[0]["Subject"] if newm else None}"; thân: {bt[:100]!r}; khoá cửa sổ TTL={ttl}s')
leak=[k for k in ('QC3 qc3s','CANARY') if k in bt]
# 16 cửa sổ trượt
rec('TC-P4-04-16',0<int(ttl)<=900 and int(ttl)>=840,f'khoá ep:mail:ticket_new:{{uid}}:{{course}} TTL={ttl}s ≈ 15 phút sau thư đầu (840–900 s)')
tin=tick('qc3s14',F_)[0]; time.sleep(8); w1=len([x for x in mpq('to:qc3t4@edupilot.local',30) if x['ID'] in {y['ID'] for y in newm}]); 
cnt_in=len([x for x in mpq('to:qc3t4@edupilot.local',30) if 'chờ' in x['Subject'] or 'đang chờ' in x['Subject']])
rds('EXPIRE',f'ep:mail:ticket_new:{tu}:{F_}','1'); time.sleep(2.5); tout=tick('qc3s15',F_)[0]; time.sleep(10)
allm=[x for x in mpq('to:qc3t4@edupilot.local',30) if 'ticket' in x['Subject'].lower() or 'đang chờ' in x['Subject'] or 'câu hỏi' in x['Subject'].lower()]
subj=[x['Subject'] for x in mpq('to:qc3t4@edupilot.local',30)]
n_after=len([x for x in subj if x==newm[0]['Subject']]) if newm else 0
rec('TC-P4-04-16',True,'') if False else None
R16=(cnt_in,n_after,subj)
print(R16)
rec('TC-P4-04-16',0<int(ttl)<=900 and int(ttl)>=840 and n_after==2,f'TTL khoá={ttl}s; ticket trong cửa sổ → vẫn {len(newm)} thư; hết khoá (EXPIRE 1 ≙ >15 phút) + ticket mới → tổng thư ticket_new = {n_after} (mong 2)')
# 23 mặc định tắt
tu3=uid('qc3t3'); m0=len(mpq('to:qc3t3@edupilot.local',30)); tick('qc3s05',E_); time.sleep(10); m1=len(mpq('to:qc3t3@edupilot.local',30)); fl=psql(f"select notify_ticket_by_mail from user_settings where user_id='{tu3}'")
rec('TC-P4-04-23',m1==m0,f'teacher mới (notify_ticket_by_mail={fl or "mặc định false"}) nhận ticket → thư mới={m1-m0}')
# E2 không có email xác minh
psql("update users set email_verified_at=null where email='qc3s28@edupilot.local'")
_,_,am=mkchat('qc3s28',A_); s,d=esc('qc3s28',am); te=d['ticket']['id']; A(T,'POST',f'{Q}/tickets/{te}/claim'); sa,_=A(T,'POST',f'{Q}/tickets/{te}/answer',{'body':'Chào em, thầy trả lời.'}); time.sleep(8)
row=psql("select status||'/'||attempts||'/'||coalesce(last_error,'-') from mail_outbox where to_addr='qc3s28@edupilot.local' order by created_at desc limit 1"); mm=len(mails('qc3s28@edupilot.local'))
psql("update users set email_verified_at=now() where email='qc3s28@edupilot.local'")
rec('TC-P4-04-E2',sa==200 and mm==0,f'SV chưa xác minh email: answer {sa}; mail_outbox: {row or "không có dòng"}; Mailpit {mm} thư (ghi nhận thực tế)')
# P1
adm=psql("select count(*) from mail_outbox where to_addr='admin@edupilot.local' and template in ('escalation_answered','ticket_reminder','ticket_new','thread_reply','escalation_stalled')")
st=psql("select count(*) from mail_outbox where to_addr in ('qc3t@edupilot.local','qc3ta@edupilot.local','qc3t2@edupilot.local','qc3t3@edupilot.local','qc3t4@edupilot.local') and template='escalation_answered'")
sub_adm=[x['Subject'] for x in mpq('to:admin@edupilot.local',50)]
rec('TC-P4-04-P1',adm=='0' and st=='0',f'mail_outbox escalation_answered tới GV/TA={st}; mail ticket tới ADMIN={adm} (COURSE_TICKETS_STALLED chỉ thông báo); tiêu đề thư tới admin: {sorted(set(sub_adm))[:5]}')
# 30 log
g=subprocess.run("docker logs ep-r-gateway-1 --since 40m 2>&1 | grep -c '@edupilot.local'; docker logs ep-r-worker-1 --since 40m 2>&1 | grep -c '@edupilot.local'",shell=True,capture_output=True,text=True).stdout.split()
ex=subprocess.run("docker logs ep-r-worker-1 --since 40m 2>&1 | grep '@edupilot.local' | head -2 | cut -c1-220; docker logs ep-r-gateway-1 --since 40m 2>&1 | grep '@edupilot.local' | head -2 | cut -c1-220",shell=True,capture_output=True,text=True).stdout
rec('TC-P4-04-30',g==['0','0'],f'log 40 phút qua (gateway/worker) chứa @edupilot.local: {g}; mẫu: {ex[:300]!r}')
save('p404b')
