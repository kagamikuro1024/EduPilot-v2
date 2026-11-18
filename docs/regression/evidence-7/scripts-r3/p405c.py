from lib3 import *
A_=C7['A']; Q=f'/courses/{A_}'; B_=C7['B']; T='qc3t'; TA='qc3ta'
# P4-05-P1: TA/GV lớp khác (r5ta) knowledge/dismiss ticket lớp A → 404/403; STALLED chỉ ADMIN
clearopen(T,A_); t,_=tick('qc3s20' if False else 'qc3s21',A_); A(T,'POST',f'{Q}/tickets/{t}/claim'); A(T,'POST',f'{Q}/tickets/{t}/answer',{'body':'Chào em, đã trả lời.'})
x1=A('r5ta','POST',f'{Q}/tickets/{t}/knowledge',{'question':'Hỏi thử?','answer':'Trả lời thử.'})[0]; x2=A('r5ta','POST',f'{Q}/tickets/{t}/dismiss',{'reason':'OUT_OF_SCOPE'})[0]; x3=A('r5t','POST',f'{Q}/tickets/{t}/knowledge',{'question':'Hỏi thử?','answer':'Trả lời thử.'})[0]
adm=psql("select count(*) from notifications where type='COURSE_TICKETS_STALLED' and user_id=(select id from users where email='admin@edupilot.local')"); tch=psql("select count(*) from notifications where type='COURSE_TICKETS_STALLED' and user_id in (select id from users where email in ('qc3t@edupilot.local','qc3t2@edupilot.local','qc3t3@edupilot.local','qc3t4@edupilot.local','qc3t5@edupilot.local','qc3t6@edupilot.local','qc3ta@edupilot.local'))")
rec('TC-P4-05-P1',x1 in(403,404) and x2 in(403,404) and x3 in(403,404) and adm!='0' and tch=='0',f'TA lớp khác (r5ta) knowledge/dismiss ticket lớp QC3A → {x1}/{x2}; GV lớp khác (r5t) knowledge → {x3}; thông báo COURSE_TICKETS_STALLED: ADMIN {adm} dòng, GV/TA QC3 {tch} dòng (chỉ ADMIN nhận)')
# P4-05-E2: autoclose khi DB chậm
clearopen(T,A_); to,_=tick('qc3s22',A_); ta2,_=tick('qc3s23',A_); A(T,'POST',f'{Q}/tickets/{ta2}/claim'); A(T,'POST',f'{Q}/tickets/{ta2}/answer',{'body':'Chào em, trả lời.'})
psql(f"update escalation_tickets set created_at=now()-interval '9 days',waiting_since=now()-interval '9 days' where id='{to}'"); psql(f"update escalation_tickets set answered_at=now()-interval '8 days' where id='{ta2}'")
psql("create or replace function qc3_slow3() returns trigger as $$ begin if NEW.status='CLOSED' and OLD.status<>'CLOSED' and NEW.course_id='%s' then perform pg_sleep(3); end if; return NEW; end $$ language plpgsql" % A_)
psql("create trigger qc3_slow3_t before update on escalation_tickets for each row execute function qc3_slow3()")
try:
    t0=time.time(); out=runjob('ticket.autoclose'); el=time.time()-t0
finally:
    psql("drop trigger if exists qc3_slow3_t on escalation_tickets"); psql("drop function if exists qc3_slow3()")
rec('TC-P4-05-E2',tstate(to).startswith('OPEN') and tstate(ta2).startswith('CLOSED/AUTO_7D') and el>=2.5,f'trigger tạm làm UPDATE đóng ticket chậm 3 s (DB chậm); run-job ticket.autoclose ({el:.1f} s: {out[-50:]}): ticket OPEN quá 9 ngày → {tstate(to)} (không đóng nhầm); ticket ANSWERED 8 ngày → {tstate(ta2)}; trigger đã gỡ')
# P4-06-E1/E2
sid=A('qc3s24','POST',f'{Q}/threads',{'title':'Thảo luận QCE bình luận','body':'Nội dung thảo luận để thử bình luận.','tags':[]})[1]['thread']['id']
s,c=A('qc3s24','POST',f'{Q}/threads/{sid}/posts',{'body':'Bình luận sạch ban đầu của mình QCE1.'}); cid=c.get('id') or (c.get('post') or {}).get('id'); print(s,str(c)[:200])
sp,dp=A('qc3s24','PUT',f'{Q}/posts/{cid}',{'body':'Gọi mình số 0933777418 nhé','version':1}); body=psql(f"select body from forum_posts where id='{cid}'")
rec('TC-P4-06-E1',sp==422 and dp.get('code')=='PII_DETECTED' and 'QCE1' in body,f'PUT bình luận của mình kèm SĐT → {sp} {dp.get("code")}; bản lưu vẫn «{body[:40]}» (không lưu)')
s2,c2=A('qc3s25','POST',f'{Q}/threads/{sid}/posts',{'body':'Bình luận sẽ bị xoá QCE2.'}); cid2=c2.get('id') or (c2.get('post') or {}).get('id')
sd=A('qc3s25','DELETE',f'{Q}/posts/{cid2}')[0]; sr,dr=A('qc3s26','POST',f'{Q}/posts/{cid2}/report',{'reason':'SPAM'})
rec('TC-P4-06-E2',sd in(200,204) and sr==404,f'xoá bình luận (chủ) → {sd}; SV khác báo cáo bình luận đã xoá → {sr} {dr.get("code") if isinstance(dr,dict) else dr}')
save('p405c')
