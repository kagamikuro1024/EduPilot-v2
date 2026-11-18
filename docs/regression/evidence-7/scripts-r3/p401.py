from lib3 import *
A_=C7['A']; B_=C7['B']; Q=f'/courses/{A_}'
def n5(u,course=A_,k=5):
    return [mkchat(u,course)[2] for _ in range(k)]
# 10
sid,um,am=mkchat('qc3s01',A_); s,d=esc('qc3s01',am); t=d['ticket']['id']
s2,d2=A('qc3t','POST',f'{Q}/tickets/{t}/dismiss',{'reason':'OUT_OF_SCOPE'}); s3,d3=esc('qc3s01',am)
rec('TC-P4-01-10',s3==200 and d3['already'] is True and d3['ticket']['status']=='CLOSED',f'dismiss→{s2}; escalate lại → {s3} already={d3.get("already")} status={d3["ticket"]["status"] if isinstance(d3,dict) and "ticket" in d3 else d3}')
# 11
ms=n5('qc3s02'); r=[esc('qc3s02',m)[0] for m in ms]; s,d=esc('qc3s02',ms[0])
rec('TC-P4-01-11',r==[201]*5 and s==200 and d['already'],f'5 lần đầu {r}; bấm lại tin đầu → {s} already={d.get("already")}')
# 25
ms=n5('qc3s03'); r=[esc('qc3s03',m)[0] for m in ms]; ex=mkchat('qc3s03',A_)[2]; s,d=esc('qc3s03',ex)
rec('TC-P4-01-25',r==[201]*5 and s==409 and d.get('code')=='TICKET_LIMIT_REACHED' and nt(ex)=='0',f'6th → {s} {d.get("code")} count={nt(ex)}')
# 27
ms=n5('qc3s04'); [esc('qc3s04',m) for m in ms]
# (a) lớp 2: s04 không ở B -> dùng s02 (5 ở A) trong B
_,_,amb=mkchat('qc3s02',B_); a=esc('qc3s02',amb)[0]
# (b) svb ở lớp 1
_,_,amc=mkchat('qc3s05',A_); b=esc('qc3s05',amc)[0]
# (c) trả lời 1 ticket của s04 rồi nhờ tin thứ 6
t1=tk(ms[0]).split('/')[0]; A('qc3t','POST',f'{Q}/tickets/{t1}/claim'); sa,_=A('qc3t','POST',f'{Q}/tickets/{t1}/answer',{'body':'Chào em, đã xử lý.'})
x6=mkchat('qc3s04',A_)[2]; c=esc('qc3s04',x6)[0]
rec('TC-P4-01-27',a==201 and b==201 and c==201,f'(a) s02 lớp B {a}; (b) s05 lớp A {b}; (c) answer={sa}, nhờ tin 6 → {c}')
# 30
res={}
for u,mode in (('qc3s19','PENDING'),('qc3s20','REMOVED'),('qc3s18','DELETE')):
    _,_,am=mkchat(u,A_)
    if mode=='DELETE': psql(f"delete from enrollments where course_id='{A_}' and user_id='{uid(u)}'")
    else: psql(f"update enrollments set status='{mode}' where course_id='{A_}' and user_id='{uid(u)}'")
    res[mode]=(esc(u,am)[0],nt(am))
psql(f"update enrollments set status='ACTIVE' where course_id='{A_}' and user_id in ('{uid('qc3s19')}','{uid('qc3s20')}')")
rec('TC-P4-01-30',all(v[0]==403 and v[1]=='0' for v in res.values()),f'PENDING/REMOVED/ngoài lớp → {res}')
# 31 archived
CC=mkcourse('QC3C','qc3t',['qc3ta'],'QKQ3CCC')
s,d=A('qc3s06','POST','/courses/join',{'code':'QKQ3CCC'}); _,_,am=mkchat('qc3s06',CC)
C7['C']=CC; json.dump(C7,open('/tmp/qc3e/c7.json','w'))
sa,da=A('admin','POST',f'/admin/courses/{CC}/archive',{}); s,d=esc('qc3s06',am)
rec('TC-P4-01-31',s==409 and d.get('code')=='COURSE_ARCHIVED' and nt(am)=='0',f'archive→{sa}; escalate → {s} {d.get("code")}')
# 33 sinh viên không thấy nội bộ
sid,um,am=mkchat('qc3s07',A_); s,d=esc('qc3s07',am); tid=d['ticket']['id']
sid2,um2,am2=mkchat('qc3s07',A_,lowc=True,conf=0.2); psql(f"insert into escalation_tickets(course_id,origin,source_type,source_id,message_id,session_id,student_id,confidence) values('{A_}','AUTO_LOW_CONFIDENCE','chat_message','{am2}','{am2}','{sid2}','{uid('qc3s07')}',0.2)")
A('qc3t','POST',f'{Q}/tickets/{tid}/claim')
bag=[]
for p in (f'/chat/sessions/{sid}/messages',f'/chat/sessions/{sid2}/messages','/chat/sessions',f'/chat/sessions/{sid}'):
    s,d=A('qc3s07','GET',p); bag.append((p,s,json.dumps(d)))
def keys(o,acc):
    if isinstance(o,dict):
        for k,v in o.items(): acc.add(k); keys(v,acc)
    elif isinstance(o,list):
        for x in o: keys(x,acc)
    return acc
bad={'confidence','claimed_by','origin','sla_stage','answered_by'}
allk=set(); 
for p,s,t in bag:
    try: keys(json.loads(t),allk)
    except Exception: pass
# ticket sub-object keys
tk_keys=set()
for p,s,t in bag:
    try:
        d=json.loads(t)
        for m in (d.get('items') or d.get('messages') or []):
            if isinstance(m,dict) and m.get('ticket'): tk_keys|=set(m['ticket'].keys())
    except Exception: pass
leak=bad&allk
rec('TC-P4-01-33',not leak and bool(tk_keys),f'status={[b[1] for b in bag]}; khoá ticket thấy: {sorted(tk_keys)}; khoá cấm lộ: {sorted(leak)}')
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
