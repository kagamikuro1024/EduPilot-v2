from lib3 import *
F_=C7['F']; A_=C7['A']
def mailrow(to,tpl,payload):
    mid=psql(f"insert into mail_outbox(to_addr,template,payload,status) values('{to}','{tpl}','{payload}','QUEUED') returning id").split('\n')[0]
    psql(f"insert into outbox(topic,payload) values('mail.send','{{\"mail_id\":\"{mid}\"}}')"); return mid
def waitdead(mid,timeout=30):
    for _ in range(timeout*2):
        r=psql(f"select status||'/'||attempts||'/'||coalesce(last_error,'-') from mail_outbox where id='{mid}'")
        if not r.startswith('QUEUED'): return r
        time.sleep(0.5)
    return r
t,am=tick('qc3s09',A_)
pl=psql("select payload::text from mail_outbox where template='escalation_answered' and status='SENT' order by created_at desc limit 1"); m21=mailrow('a"b@x.vn','escalation_answered',pl); r21=waitdead(m21)
rec('TC-P4-04-21',r21.startswith('DEAD/1/') and 'bad_recipient' in r21,f'địa chỉ người nhận không hợp lệ → {r21} (DEAD ngay, attempts=1, không thử lại). SMTP 550 thật không dựng được trên Mailpit: lớp lỗi vĩnh viễn 5xx được dev test TestClassify; ở đây dùng bad_recipient')
tu=uid('qc3t4'); K=f'ep:mail:ticket_new:{tu}:{F_}'
def subj(): return len([x for x in mpq('to:qc3t4@edupilot.local',40) if x['Subject']=='Có câu hỏi mới cần giảng viên'])
clearopen('qc3t4',F_)
rds('DEL',K); a0=subj(); tick('qc3s11',F_); time.sleep(8); a1=subj()
now=int(time.time()*1000); rds('SET',K,str(now-(14*60+50)*1000),'EX','960'); tick('qc3s12',F_); time.sleep(8); a2=subj()
rds('SET',K,str(int(time.time()*1000)-(15*60+1)*1000),'EX','960'); tick('qc3s13',F_); time.sleep(8); a3=subj()
rec('TC-P4-04-16',a1==a0+1 and a2==a1 and a3==a2+1,f'cửa sổ trượt 15 phút (mốc là giá trị khoá Redis ms; TTL khoá 948 s ≤ 16 phút): ticket đầu → {a1-a0} thư; mốc cách 14 m 50 s (dành 9 s cho độ trễ xử lý; 14 m 59 s sát biên bị vượt do độ trễ worker) + ticket → thêm {a2-a1} thư; mốc cách 15 m 01 s + ticket → thêm {a3-a2} thư (mong 1/0/1)')
# 17 sẽ chạy cùng cửa sổ tắt Redis
save('p404c')
