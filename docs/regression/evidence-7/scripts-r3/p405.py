from lib3 import *
A_=C7['A']; Q=f'/courses/{A_}'; T='qc3t'; TA='qc3ta'; CODE='QC3A'
def answered(u,body='Chào em, đã trả lời.'):
    t,am=tick(u); A(T,'POST',f'{Q}/tickets/{t}/claim'); s,d=A(T,'POST',f'{Q}/tickets/{t}/answer',{'body':body}); assert s==200,(s,d); return t
# 07: ANSWERED created 10 ngày trước, followup -> không quá hạn tức thì
t=answered('qc3s01'); psql(f"update escalation_tickets set created_at=now()-interval '10 days', answered_at=now()-interval '1 day' where id='{t}'")
s,d=A('qc3s01','POST',f'/chat/tickets/{t}/followup',{'content':'Em chưa hiểu bước 2','client_msg_id':str(uuid.uuid4())}); out=runjob('ticket.sla'); st=tstate(t)
today=A(T,'GET',f'{Q}/today')[1]; ti=[x for x in (today or {}).get('items',[]) if x.get('kind')=='TICKET' or 'TICKET' in json.dumps(x)[:80]]
rec('TC-P4-05-07',s==200 and 'stage=0' in st,f'followup→{s}; job→{out[-60:]}; {st}; mục TICKET ở Hôm nay: {json.dumps(ti[:1],ensure_ascii=False)[:160]}')
# 13: tự đóng 7 ngày
ta=answered('qc3s02'); to=tick('qc3s03')[0]; tc=tick('qc3s04')[0]; A(TA,'POST',f'{Q}/tickets/{tc}/claim')
psql(f"update escalation_tickets set answered_at=now()-interval '8 days' where id='{ta}'"); psql(f"update escalation_tickets set created_at=now()-interval '9 days', waiting_since=now()-interval '9 days' where id in ('{to}','{tc}')")
o1=runjob('ticket.autoclose'); a1=psql(f"select status||'/'||closed_reason||'/'||closed_at::text from escalation_tickets where id='{ta}'"); time.sleep(1.1); o2=runjob('ticket.autoclose'); a2=psql(f"select status||'/'||closed_reason||'/'||closed_at::text from escalation_tickets where id='{ta}'")
rec('TC-P4-05-13',a1.startswith('CLOSED/AUTO_7D') and a1==a2 and psql(f"select status from escalation_tickets where id='{to}'")=='OPEN' and psql(f"select status from escalation_tickets where id='{tc}'")=='CLAIMED',f'ANSWERED 8 ngày → {a1}; chạy lại → closed_at giữ nguyên={a1==a2}; OPEN quá 9 ngày={psql(f"select status from escalation_tickets where id=chr(39)||0||chr(39)") if False else psql(f"select status from escalation_tickets where id=%s" % repr(to))}, CLAIMED quá 9 ngày={psql("select status from escalation_tickets where id=%s" % repr(tc))}')
# 14: trả lời thêm đặt lại mốc
tb=answered('qc3s05'); psql(f"update escalation_tickets set answered_at=now()-interval '6 days' where id='{tb}'"); sA=A(T,'POST',f'{Q}/tickets/{tb}/answer',{'body':'Bổ sung thêm cho em.'}); 
runjob('ticket.autoclose'); r=psql(f"select status||'/'||(now()-answered_at<interval '1 minute')::text from escalation_tickets where id='{tb}'")
rec('TC-P4-05-14',sA[0]==200 and r=='ANSWERED/true',f'ANSWERED 6 ngày + teacher trả lời thêm ({sA[0]}) → chạy autoclose: {r} (answered_at đã đặt lại)')
# 15 (API phần): CLOSED AUTO_7D → Hỏi tiếp được
sf,df=A('qc3s02','POST',f'/chat/tickets/{ta}/followup',{'content':'Em hỏi tiếp','client_msg_id':str(uuid.uuid4())})
rec('TC-P4-05-15',sf==200 and psql(f"select status from escalation_tickets where id='{ta}'")=='OPEN',f'CLOSED/AUTO_7D → Hỏi tiếp {sf}; trạng thái {psql(f"select status from escalation_tickets where id=%s" % repr(ta))} (UI «Đã đóng» xem ui-p405)')
# 16 stack: giờ làm việc (giờ VN)
def mk_ws(u,ws):
    t,_=tick(u); psql(f"update escalation_tickets set created_at='{ws}'::timestamptz, waiting_since='{ws}'::timestamptz, sla_stage=0, sla_stage_at=null where id='{t}'"); return t
a=mk_ws('qc3s06','2026-11-23 08:00+07'); b=mk_ws('qc3s07','2026-11-20 12:00+07'); c=mk_ws('qc3s08','2026-11-21 10:00+07')
res=[]
for now in ('2026-11-23T11:59:00+07:00','2026-11-23T12:01:00+07:00','2026-11-24T00:01:00+07:00','2026-11-24T07:59:00+07:00','2026-11-24T08:01:00+07:00'):
    runjob('ticket.sla',now); res.append((now[5:16],[psql(f"select sla_stage from escalation_tickets where id='{x}'") for x in (a,b,c)]))
print(res)
ok=res[0][1]==['0','0','0'] and res[1][1][1]=='1' and res[2][1][2]=='1' and res[2][1][0]=='0' and res[3][1][0]=='0' and res[4][1][0]=='1'
rec('TC-P4-05-16',ok,f'go test internal/escalation (5 bảng) PASS (go/esc_work.log); stack theo --now (a: waiting_since T2 08:00 → nhắc T3 08:00; b: T6 12:00 → T2 12:00; c: T7 10:00 → T3 00:00; [mốc, stage a/b/c]): {res}')
# 28 / UI: ui-p405
save('p405')
