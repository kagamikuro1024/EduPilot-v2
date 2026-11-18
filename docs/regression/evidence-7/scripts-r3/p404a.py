from lib3 import *
A_=C7['A']; F_=C7['F']; Q=f'/courses/{A_}'; T='qc3t'; TA='qc3ta'
def mails(addr,sub=None):
    q=f'to:{addr}'+(f' subject:"{sub}"' if sub else ''); return mpq(q,50)
def wait_mail(addr,sub=None,n=1,timeout=25):
    t0=time.time()
    while time.time()-t0<timeout:
        m=mails(addr,sub)
        if len(m)>=n: return m
        time.sleep(1)
    return mails(addr,sub)
def body(mid): m=mp('/api/v1/message/'+mid); return (m.get('Text') or '')+(m.get('HTML') or ''),m
def answer_mail(u,q='Câu hỏi thử mail',extra=None):
    t,am=tick(u,q=q) if False else (None,None)
    _,_,am=mkchat(u,A_,q=q); s,d=esc(u,am); t=d['ticket']['id']; A(T,'POST',f'{Q}/tickets/{t}/claim')
    b={'body':'Chào em, thầy trả lời câu hỏi của em.'}; 
    if extra: b.update(extra)
    s,d=A(T,'POST',f'{Q}/tickets/{t}/answer',b); assert s==200,(s,d); return t
clearopen(T,A_)
# 04: phát lại outbox -> 1 thư
t=answer_mail('qc3s10'); m=wait_mail('qc3s10@edupilot.local','Giảng viên đã trả lời')
pl=psql(f"select payload::text from outbox where topic='escalation.answered' and payload->>'ticket_id'='{t}' limit 1"); psql(f"insert into outbox(topic,payload) values('escalation.answered','{pl}')"); time.sleep(10)
m2=mails('qc3s10@edupilot.local','Giảng viên đã trả lời'); dk=psql(f"select count(*) from mail_outbox where to_addr='qc3s10@edupilot.local' and template='escalation_answered'")
rec('TC-P4-04-04',len(m)==1 and len(m2)==1 and dk=='1',f'trả lời → {len(m)} thư; phát lại outbox escalation.answered ({pl[:40]}…): Mailpit {len(m2)} thư, mail_outbox {dk} dòng (dedupe escalation_answered:<answer_message_id>)')
# 11: Message-ID tất định; 03/24: tới đúng email xác minh
mo=psql(f"select id from mail_outbox where to_addr='qc3s10@edupilot.local' and template='escalation_answered' order by created_at desc limit 1")
full=mp('/api/v1/message/'+m2[0]['ID']); mid=full.get('MessageID'); dom=psql("select 1") and None
rec('TC-P4-04-11',mid is not None and mid.startswith(mo+'@'),f'Mailpit MessageID={mid} so với mail_outbox.id={mo}: dạng <id@domain MAIL_FROM>')
# 06 không rò chéo
ta=answer_mail('qc3s11','CANARY-A1 hỏi về học phần'); tb=answer_mail('qc3s12','CANARY-B2 hỏi về lịch thi')
ma=wait_mail('qc3s11@edupilot.local'); mb=wait_mail('qc3s12@edupilot.local')
ba=body(ma[0]['ID'])[0]; bb=body(mb[0]['ID'])[0]
rec('TC-P4-04-06',len(ma)>=1 and len(mb)>=1 and 'CANARY-B2' not in ba and 'CANARY-A1' not in bb,f'thư A chứa A1={"CANARY-A1" in ba} B2={"CANARY-B2" in ba}; thư B chứa B2={"CANARY-B2" in bb} A1={"CANARY-A1" in bb}; chéo=0')
# 24: To đúng email; body.to bị bỏ qua
t24=answer_mail('qc3s13',extra={'to':'evil@x.vn','cc':'evil@x.vn'}); m24=wait_mail('qc3s13@edupilot.local'); ev=mails('evil@x.vn')
tos=[x.get('To') for x in m24]
rec('TC-P4-04-24',len(m24)==1 and len(ev)==0 and 'qc3s13@edupilot.local' in json.dumps(tos),f'To={[ [y["Address"] for y in x["To"]] for x in m24]}; thư gửi evil@x.vn={len(ev)} (client truyền to/cc trong body answer bị bỏ qua)')
# 22: SV tắt notify_answer_by_mail
s,d=A('qc3s14','GET','/me/settings'); s1,d1=A('qc3s14','PUT','/me/settings',{'notify_answer_by_mail':False,'version':d['version']})
t22=answer_mail('qc3s14'); time.sleep(10); m22=mails('qc3s14@edupilot.local'); nn=nrows('qc3s14',"and type='TICKET_ANSWERED'")
rec('TC-P4-04-22',s1==200 and len(m22)==0 and int(nn)>=1,f'PUT notify_answer_by_mail=false → {s1}; sau khi trả lời: thư={len(m22)}, chuông TICKET_ANSWERED={nn}')
# 18: tiêu đề không chứa tên
allm=[]
for u in ('qc3s10','qc3s11','qc3s12','qc3s13'): allm+=mails(f'{u}@edupilot.local')
subs=[x['Subject'] for x in allm]; bad=[x for x in subs if re.search(r'QC3 qc3|qc3s\d|Giảng viên QC3',x)]
rec('TC-P4-04-18',bool(subs) and not bad,f'{len(subs)} thư test: tiêu đề {sorted(set(subs))}; chứa tên={bad}')
# 08/13/25: thư nhắc
clearopen(T,A_); tr,_=tick('qc3s15',q='CANARY-R1 nội dung riêng'); print(seedage('QC3A',tr,100)); out=runjob('ticket.sla'); time.sleep(8)
st=tstate(tr)
mt=wait_mail('qc3t@edupilot.local','Có câu hỏi chờ giảng viên quá 24 giờ làm việc'); mta=wait_mail('qc3ta@edupilot.local','Có câu hỏi chờ giảng viên quá 24 giờ làm việc')
adm=mpq('to:admin@edupilot.local subject:"quá 24 giờ"'); oth=mpq('to:r5t@edupilot.local subject:"quá 24 giờ"')
bt=body(mt[-1]['ID'])[0] if mt else ''
mssv=psql("select student_code from users where email='qc3s15@edupilot.local'")
leak=[k for k in ('QC3 qc3s15','CANARY-R1',mssv,'qc3s15@') if k in bt]
rec('TC-P4-04-13',bool(mt) and bool(mta) and 'stage=1' in st or 'stage=2' in st,f'seed-age 100h + run-job: {st}; thư tới qc3t={len(mt)}, qc3ta={len(mta)}; tiêu đề "{mt[-1]["Subject"] if mt else None}"')
rec('TC-P4-04-08',bool(mt) and not leak and '/inbox?course=' in bt and 'QC3A' in bt,f'thân thư nhắc: có /inbox?course=…={"/inbox?course=" in bt}, mã lớp QC3A={"QC3A" in bt}; rò tên/nội dung/MSSV/email={leak}')
rec('TC-P4-04-25',bool(mt) and bool(mta) and not adm and not oth,f'thư nhắc lớp QC3A: TEACHER {len(mt)}, TA {len(mta)}; ADMIN {len(adm)}; teacher lớp khác {len(oth)}')
# 14 thread_reply
s,d=A('qc3s16','POST',f'{Q}/threads',{'title':'Thảo luận QC3 mail','body':'Nội dung thảo luận thử cho kiểm thử thư.','tags':[]}); th=d['thread']['id']
s2,d2=A('qc3s17','POST',f'{Q}/threads/{th}/posts',{'body':'Mình xin góp một ý kiến cho bài này.'}); mtr=wait_mail('qc3s16@edupilot.local','Câu hỏi của bạn trên Threads có trả lời mới')
bb=body(mtr[0]['ID'])[0] if mtr else ''
rec('TC-P4-04-14',s==201 and s2 in(200,201) and len(mtr)>=1 and 'Thảo luận QC3 mail' in bb,f'đăng thread {s}, bình luận {s2}; thư thread_reply tới tác giả: {len(mtr)}; thân chứa tiêu đề thread={"Thảo luận QC3 mail" in bb}, liên kết={"/threads" in bb}; (nhánh VERIFY/CORRECT của câu trả lời AI: xem TC-P4-04-14b ở nhóm threads)')
save('p404a')
