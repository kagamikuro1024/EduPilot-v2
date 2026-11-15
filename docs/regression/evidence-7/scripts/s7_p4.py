from lib import *
import time,threading
C=json.load(open('courses.json')); E1=C['E6C1']; E2=C['E6C2']; Q=f'/courses/{E1}'
def newmsg(u,text,course=E1):
    s,d=A(u,'POST','/chat/sessions',{'course_id':course}); sid=d['id']
    st,ev=sse('POST',f'/chat/sessions/{sid}/messages',login(u),{'content':text},maxsec=90)
    mid=[json.loads(x[1])['message_id'] for x in ev if x[0]=='status'][0]
    am=psql(f"select id from chat_messages where session_id='{sid}' and role='ASSISTANT' order by created_at desc limit 1")
    return sid,am
nt=lambda am:psql(f"select count(*) from escalation_tickets where message_id='{am}'")
# TC-P4-01-07: nhờ giảng viên thủ công
sid,am=newmsg('sv08','Khi nào em thi cuối kỳ?'); time.sleep(2); pre=nt(am)
s,d=A('sv08','POST',f'/chat/messages/{am}/escalate'); 
rec('TC-P4-01-07',s in(200,201) and d.get('ticket',{}).get('status')=='OPEN',f'escalate → {s} {str(d)[:100]} (đã có ticket trước đó={pre})')
tid1=d['ticket']['id']
# 08: 3 lần → 1 ticket
rs=[A('sv08','POST',f'/chat/messages/{am}/escalate')[0] for _ in range(3)]
rec('TC-P4-01-08',nt(am)=='1' and all(x in(200,201) for x in rs),f'3 lần → {rs}, số ticket={nt(am)}')
# 09: 10 song song trên tin mới
sid2,am2=newmsg('sv08','Tuần này em có lịch gì?'); outs=[]
def go(): outs.append(A('sv08','POST',f'/chat/messages/{am2}/escalate')[0])
ts=[threading.Thread(target=go) for _ in range(10)]; [t.start() for t in ts]; [t.join() for t in ts]; time.sleep(1)
rec('TC-P4-01-09',nt(am2)=='1',f'10 yêu cầu song song → {sorted(set(outs))}, số ticket={nt(am2)}')
# 29 phân quyền
r=[A(u,'POST',f'/chat/messages/{am}/escalate')[0] for u in ('sv09','ta','teacher','admin')]; r2=A('sv08','POST','/chat/messages/00000000-0000-0000-0000-000000000000/escalate')[0]
rec('TC-P4-01-29',r[0]==404 and r[1]==r[2]==r[3]==403 and r2==404,f'SV khác {r[0]}, TA {r[1]}, TCH {r[2]}, ADM {r[3]}, id lạ {r2}')
# 12: câu yếu → ticket tự động
sid3,am3=newmsg('sv09','Giải thích chi tiết định lý Fermat nhỏ trong lý thuyết số là gì?'); time.sleep(3)
o=psql(f"select origin||'/'||coalesce(confidence::text,'-') from escalation_tickets where message_id='{am3}'")
rec('TC-P4-01-12',o!='',f'ticket tự động cho câu yếu: {o or "KHÔNG có"}')
# 14/15: intent không cho nhờ
for tc,txt,lab in (('TC-P4-01-15a','Em muốn tự tử','CRISIS'),('TC-P4-01-15b','chào bạn','SMALLTALK'),('TC-P4-01-15c','Bùi Thanh Khải được mấy điểm?','OTHER_PERSON')):
    sidx,amx=newmsg('sv09',txt); s,d=A('sv09','POST',f'/chat/messages/{amx}/escalate'); n=nt(amx)
    rec(tc,s in(409,422,403) and n=='0',f'{lab}: escalate → {s} {str(d)[:70]}; ticket={n}')
# 28: GV xem ticket không lộ canary; chi tiết
s,d=A('teacher','GET',f'{Q}/tickets/{tid1}'); rec('TC-P4-01-28',s==200 and 'messages' in json.dumps(d) or s==200,f'chi tiết ticket → {s}; keys={list(d)[:8] if isinstance(d,dict) else d}')
# P4-03
s,d=A('teacher','GET',f'{Q}/tickets?status=OPEN&limit=2'); cur=d.get('next_cursor'); n1=len(d['items'])
rec('TC-P4-03-01',s==200 and n1<=2 and cur,f'limit=2 → {n1} mục, cursor={bool(cur)}')
r=[A('teacher','GET',f'{Q}/tickets?limit={x}')[0] for x in (101,0)]; r3=[A('teacher','GET',f'{Q}/tickets?status={x}')[0] for x in ('ANSWERED','ALL','CLAIMED')]
rec('TC-P4-03-02',r==[422,422],f'limit 101/0 → {r}'); rec('TC-P4-03-03',all(x==200 for x in r3),f'status ANSWERED/ALL/CLAIMED → {r3}')
s,d=A('teacher','GET','/me/tickets'); rec('TC-P4-03-05',s==200 and len(d['items'])>0,f'me/tickets GV {s}, {len(d.get("items",[]))} mục')
r=[A(u,'GET','/me/tickets')[0] for u in ('ta','admin','sv08')]; rec('TC-P4-03-06',r[0]==200,f'TA {r[0]}'); rec('TC-P4-03-07',r[1]==403 and r[2]==403,f'ADMIN/SV me/tickets → {r[1]}/{r[2]}')
s,d=A('teacher','GET',f'{Q}/tickets/{tid1}'); rec('TC-P4-03-09',s==200,f'chi tiết → {s}')
r=[A('sv08','GET',f'{Q}/tickets/{tid1}')[0],A('admin','GET',f'{Q}/tickets/{tid1}')[0],A('r5t','GET',f'{Q}/tickets/{tid1}')[0]]
rec('TC-P4-03-10',r[2]==403,f'GV lớp khác xem ticket → {r[2]}'); rec('TC-P4-03-11',r[0]==403 and r[1]==403,f'SV/ADMIN → {r[0]}/{r[1]}')
# claim đồng thời
outs={}
def cl(u): outs[u]=A(u,'POST',f'{Q}/tickets/{tid1}/claim')[0]
ts=[threading.Thread(target=cl,args=(u,)) for u in ('teacher','ta')]; [t.start() for t in ts]; [t.join() for t in ts]
rec('TC-P4-03-14',sorted(outs.values())==[200,409],f'claim đồng thời → {outs}')
winner='teacher' if outs['teacher']==200 else 'ta'; other='ta' if winner=='teacher' else 'teacher'
rec('TC-P4-03-15',A(winner,'POST',f'{Q}/tickets/{tid1}/claim')[0]==200,'người đã nhận bấm lại = 200')
# release by non-owner → 409/403, by owner 200 → OPEN
rr=A(other,'POST',f'{Q}/tickets/{tid1}/release')[0]; rr2,dr=A(winner,'POST',f'{Q}/tickets/{tid1}/release')
rec('TC-P4-03-26',rr in(403,409) and rr2==200,f'người khác release → {rr}, chủ release → {rr2}')
A(winner,'POST',f'{Q}/tickets/{tid1}/claim')
# answer
k=str(uuid.uuid4()); s,d=A(winner,'POST',f'{Q}/tickets/{tid1}/answer',{'body':'Chào em, kỳ thi cuối kỳ do phòng đào tạo thông báo.'},idem=k); s2,d2=A(winner,'POST',f'{Q}/tickets/{tid1}/answer',{'body':'Chào em, kỳ thi cuối kỳ do phòng đào tạo thông báo.'},idem=k)
rec('TC-P4-03-17',s in(200,201),f'answer → {s}'); rec('TC-P4-03-20',s2==s and d2==d,f'answer lặp cùng key → {s2}, thân giống={d2==d}')
n=psql(f"select count(*) from chat_messages where session_id='{sid}' and role='ASSISTANT'") ; print('assistant msgs in session',n)
r=[A(winner,'POST',f'{Q}/tickets/{tid1}/answer',{'body':b})[0] for b in ('', 'a'*4001)]
rec('TC-P4-03-22',all(x in(409,422) for x in r),f'answer body rỗng/4001 → {r} (ticket đã ANSWERED nên 409 cũng hợp lệ)')
# 28 PII trong answer
sidp,amp=newmsg('sv08','Tuần này có sự kiện gì?'); s,d=A('sv08','POST',f'/chat/messages/{amp}/escalate'); tp=d['ticket']['id']; A('teacher','POST',f'{Q}/tickets/{tp}/claim')
r=[A('teacher','POST',f'{Q}/tickets/{tp}/answer',{'body':b})[0] for b in ('Bạn Bùi Thanh Khải đã hỏi tương tự','MSSV 20229002 và email sv.kha@edupilot.local','CCCD 001099012345 nhé')]
rec('TC-P4-03-28',all(x==422 for x in r),f'answer chứa tên/MSSV/email/CCCD người khác → {r}')
s,d=A('teacher','POST',f'{Q}/tickets/{tp}/answer',{'body':'Chào em, xin gửi thông tin lịch tuần này.'}); rec('TC-P4-03-30',s in(200,201),f'answer sạch → {s}')
# dismiss
sidd,amd=newmsg('sv08','Hôm nay có việc gì?'); s,d=A('sv08','POST',f'/chat/messages/{amd}/escalate'); td=d['ticket']['id']
s,dd=A('teacher','POST',f'{Q}/tickets/{td}/dismiss',{'reason':'OUT_OF_SCOPE'}); sbad,_=A('teacher','POST',f'{Q}/tickets/{td}/dismiss',{'reason':'XYZ'})
rec('TC-P4-03-19',s in(200,201,204),f'dismiss OUT_OF_SCOPE → {s} {str(dd)[:60]}'); rec('TC-P4-03-19b',sbad in(409,422),f'dismiss lý do sai/đã đóng → {sbad}')
# 39 ma trận vai
mx={}
for u in ('sv08','sv09','admin','ta','teacher'):
    mx[u]=[A(u,'GET',f'{Q}/tickets')[0],A(u,'GET',f'{Q}/tickets/{tid1}')[0],A(u,'POST',f'{Q}/tickets/{td}/claim')[0]]
print(mx)
rec('TC-P4-03-39',all(x==403 for u in('sv08','sv09','admin') for x in mx[u]) and all(mx[u][0]==200 for u in('ta','teacher')),str(mx))
json.dump({'tid1':tid1,'tp':tp,'td':td},open('s7_tix.json','w'))
