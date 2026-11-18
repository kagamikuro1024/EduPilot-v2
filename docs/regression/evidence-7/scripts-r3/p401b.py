from lib3 import *
A_=C7['A']; B_=C7['B']; Q=f'/courses/{A_}'
# 30 (sửa): REMOVED (tin của chính họ) → 403; PENDING / ngoài lớp (tin người khác) → 404 theo #15
_,_,amR=mkchat('qc3s20',A_); rs=A('qc3t','DELETE',f'{Q}/members/{uid("qc3s20")}'); r_rem=esc('qc3s20',amR)
_,_,amX=mkchat('qc3s13',A_)
psql(f"update enrollments set status='PENDING' where course_id='{A_}' and user_id='{uid('qc3s19')}'")
r_pen=esc('qc3s19',amX); r_out=esc('qc3s30',amX)
st20=psql(f"select status from enrollments where course_id='{A_}' and user_id='{uid('qc3s20')}'")
psql(f"update enrollments set status='ACTIVE' where course_id='{A_}' and user_id='{uid('qc3s19')}'")
rec('TC-P4-01-30',r_rem[0]==403 and r_pen[0]==404 and r_out[0]==404 and nt(amR)=='0' and nt(amX)=='0',f'REMOVED (remove→{rs}, enrol={st20}) tin của mình → {r_rem[0]}; PENDING gọi tin người khác → {r_pen[0]}; ngoài lớp → {r_out[0]} (spec #15: 403/404/404)')
# 34
_,um,_=mkchat('qc3s08',A_)
r1=esc('qc3s08',um)
_,_,amS=mkchat('qc3s08',A_,stream='STREAMING'); r2=esc('qc3s08',amS)
_,_,amC=mkchat('qc3s08',A_,intent='CRISIS'); r3=esc('qc3s08',amC)
_,_,amO=mkchat('qc3s08',A_,intent='OTHER_PERSON'); r4=esc('qc3s08',amO)
_,_,amSm=mkchat('qc3s08',A_,intent='SMALLTALK'); r5=esc('qc3s08',amSm)
_,_,amF=mkchat('qc3s08',A_,stream='FAILED'); r6=esc('qc3s08',amF)
out=[(r[0],r[1].get('code')) for r in (r1,r2,r3,r4,r5,r6)]
cnt=[nt(x) for x in (um,amS,amC,amO,amSm,amF)]
rec('TC-P4-01-34',all(o==(422,'MESSAGE_NOT_ESCALATABLE') for o in out) and set(cnt)=={'0'},f'USER/STREAMING/CRISIS/OTHER_PERSON/SMALLTALK/FAILED → {out}; ticket={cnt}')
# 36
_,_,am=mkchat('qc3s09',A_); k=str(uuid.uuid4())
a1=A('qc3s09','POST',f'/chat/messages/{am}/escalate',idem=k); a2=A('qc3s09','POST',f'/chat/messages/{am}/escalate',idem=k)
_,_,am2=mkchat('qc3s09',A_); s,h,hd=raw('POST',f'/chat/messages/{am2}/escalate',login('qc3s09'),None,idem='')  # POST luôn gán key; thử không key
req=urllib.request.Request(B+f'/chat/messages/{am2}/escalate',data=None,method='POST',headers={'authorization':'Bearer '+login('qc3s09'),'content-type':'application/json'})
try: r=urllib.request.urlopen(req,context=ctx); nk=r.status
except urllib.error.HTTPError as e: nk=e.code
rec('TC-P4-01-36',a1[0] in(200,201) and a2[0] in(200,201) and a1[1]['ticket']['id']==a2[1]['ticket']['id'] and nt(am)=='1' and nk in(200,201),f'cùng key: {a1[0]}/{a2[0]} cùng id, count={nt(am)}; không Idempotency-Key → {nk} (count={nt(am2)})')
# E2 phiên xoá mềm sau khi nhờ
sid,um,am=mkchat('qc3s10',A_); s,d=esc('qc3s10',am); tid=d['ticket']['id']
sd,_=A('qc3s10','DELETE',f'/chat/sessions/{sid}'); sg,dg=A('qc3t','GET',f'{Q}/tickets/{tid}')
sess=psql(f"select session_id from escalation_tickets where id='{tid}'")
rec('TC-P4-01-E2',sd in(200,204) and sg==200 and sess==sid,f'xoá phiên→{sd}; GV mở chi tiết → {sg}; ticket.session_id giữ nguyên={sess==sid}; keys={list(dg)[:6] if isinstance(dg,dict) else dg}')
# P1
sid,um,am=mkchat('qc3s11',A_); s,d=esc('qc3s11',am); tid=d['ticket']['id']
r=A('qc3s12','GET',f'/chat/sessions/{sid}/messages'); r2=A('qc3s12','GET',f'{Q}/tickets/{tid}'); r3=A('qc3s12','GET','/chat/sessions')
rec('TC-P4-01-P1',r[0]==404 and r2[0]==403 and tid not in json.dumps(r3[1]),f'SVB GET messages phiên SVA → {r[0]}; GET ticket → {r2[0]}; danh sách phiên của SVB có ticket SVA={tid in json.dumps(r3[1])}')
save('p401')
