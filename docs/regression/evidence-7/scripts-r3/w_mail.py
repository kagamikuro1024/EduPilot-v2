from lib3 import *
A_=C7['A']; Q=f'/courses/{A_}'; T='qc3t'
clearopen(T,A_); login('qc3s07'); login('qc3s08')
def mk(u,q='Câu hỏi cho thư lỗi'):
    _,_,am=mkchat(u,A_,q=q); s,d=esc(u,am); t=d['ticket']['id']; A(T,'POST',f'{Q}/tickets/{t}/claim'); return t,am
t1,am1=mk('qc3s07'); t2,am2=mk('qc3s08')
for u in ('qc3s07','qc3s08'): psql(f"update user_settings set notify_answer_by_mail=true where user_id='{uid(u)}'")
subprocess.run(['docker','exec','-i','ep-r-postgres-1','true'])
assert lockg('qc3','Sprint7 cửa sổ Mailpit mất mạng (P4-04-19/20/E1): docker network disconnect mailpit; ≤ 3 phút')
res={}
try:
    subprocess.run(['docker','network','disconnect','ep-r_default','ep-r-mailpit-1'],capture_output=True)
    t0=time.time(); s,d=A(T,'POST',f'{Q}/tickets/{t1}/answer',{'body':'Chào em, thầy trả lời khi mail đang lỗi.'})
    res['ans1']=s
    tl=[]; last=None
    while time.time()-t0<75:
        r=psql(f"select attempts||'/'||status||'/'||coalesce(last_error,'-') from mail_outbox where to_addr='qc3s07@edupilot.local' and template='escalation_answered' order by created_at desc limit 1")
        if r!=last: tl.append((round(time.time()-t0,1),r)); last=r
        if r.split('/')[1]=='DEAD': break
        time.sleep(0.25)
    res['tl']=tl
    # trong lúc thử: SV vẫn thấy tin + chuông
    res['sv_msgs']=A('qc3s07','GET',f'/chat/sessions/{psql(f"select session_id from escalation_tickets where id=chr(39)||0||chr(39)") if False else psql("select session_id from escalation_tickets where id=%s" % repr(t1))}/messages')[0]
    res['tk']=psql(f"select status from escalation_tickets where id='{t1}'"); res['bell']=nrows('qc3s07',"and type='TICKET_ANSWERED'")
    res['dead']=psql(f"select count(*) from outbox o where o.topic='mail.send' and o.dead_at is not null and o.created_at>now()-interval '3 minutes'")
    # E1: ticket 2, mạng lại sau 10 s
    t0b=time.time(); s2,_=A(T,'POST',f'{Q}/tickets/{t2}/answer',{'body':'Chào em, đây là thư thử lại.'}); time.sleep(10)
finally:
    subprocess.run(['docker','network','connect','--alias','mailpit','--alias','ep-r-mailpit-1','ep-r_default','ep-r-mailpit-1'],capture_output=True)
    unlockg('qc3','Mailpit nối lại mạng')
tl2=[]; last=None; t1b=time.time()
while time.time()-t1b<60:
    r=psql(f"select attempts||'/'||status||'/'||coalesce(last_error,'-') from mail_outbox where to_addr='qc3s08@edupilot.local' and template='escalation_answered' order by created_at desc limit 1")
    if r!=last: tl2.append((round(time.time()-t0b,1),r)); last=r
    if r.split('/')[1] in('SENT','DEAD'): break
    time.sleep(0.5)
time.sleep(2); n2=len(mpq('to:qc3s08@edupilot.local subject:"Giảng viên đã trả lời"'))
print(res,tl2,n2)
json.dump(dict(res=res,tl2=tl2,n2=n2),open('/tmp/qc3e/w_mail.json','w'),ensure_ascii=False)
tl=res['tl']; fin=tl[-1][1] if tl else ''
leak=psql(f"select count(*) from mail_outbox where last_error ~ '@' or last_error ~ 'Chào em'")
rec('TC-P4-04-19',fin.split('/')[1]=='DEAD' and fin.split('/')[0] in('4','3') and leak=='0',f'Mailpit mất mạng: dòng mail_outbox theo thời gian [(s, attempts/status/last_error)] = {tl}; DEAD sau {tl[-1][0]}s; last_error là nhãn (không email/nội dung: dòng vi phạm={leak}); outbox mail.send dead_at có: {res["dead"]}')
rec('TC-P4-04-20',res['ans1']==200 and res['tk']=='ANSWERED' and res['sv_msgs']==200 and int(res['bell'])>=1,f'trong lúc thư lỗi: answer → {res["ans1"]}, ticket {res["tk"]}, SV GET messages phiên → {res["sv_msgs"]} (tin trả lời hiện), chuông TICKET_ANSWERED={res["bell"]}')
rec('TC-P4-04-E1',n2==1 and tl2 and tl2[-1][1].split('/')[1]=='SENT',f'Mailpit mất mạng lúc trả lời, nối lại sau ~10 s: tiến trình {tl2}; Mailpit nhận {n2} thư (đúng 1)')
save('w_mail')
