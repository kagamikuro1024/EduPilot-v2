from lib import *
import time,threading
C1='01a1df27-62b5-7775-a346-5c602a43e2f9'
K='sv.kha'; tk=login(K); uidK=psql("select id from users where email='sv.kha@edupilot.local'")
def newsess(u=K,c=C1):
    s,d=A(u,'POST','/chat/sessions',{'course_id':c,'title':'qc-e6b'}); assert s==201,(s,d); return d['id']
sid=newsess()
# 03/04/05: sự kiện đầu < 300 ms và loại khung (câu COURSE_QA thật)
t0=time.time(); res={}
def run():
    res['r']=sse('POST',f'/chat/sessions/{sid}/messages',tk,{'content':'Quy chế thi cuối kỳ nói gì về tài liệu được mang vào phòng thi?'})
th=threading.Thread(target=run); th.start()
seen=[]
for i in range(40):
    r=psql(f"select role||':'||stream_status||':'||coalesce(length(partial_content),-1) from chat_messages where session_id='{sid}' order by created_at,role desc")
    seen.append(r.replace('\n',' | ')); time.sleep(0.4)
    if not th.is_alive(): break
th.join(); st,ev=res['r']
first=ev[0][2] if ev else None
types=[e for e,*_ in ev]
rec('TC-P3-05-03',st==200 and first is not None and first<0.3,f'status={st}, khung đầu {first}s, các khung={sorted(set(types))}')
rec('TC-P3-05-04',types[0]=='status' and types[-1] in('done','error') and 'token' in types,f'chuỗi khung: {types[:3]}…{types[-1]}, {len(types)} khung')
print('db thấy',seen[:6])
rec('TC-P3-05-05',any('ASSISTANT:STREAMING' in x or 'USER:' in x for x in seen),'; '.join(seen[:3]))
done=[json.loads(d) for e,d,*_ in ev if e=='done']
mid=json.loads([d for e,d,*_ in ev if e=='status'][0])['message_id']
# 10: nối lại sau done (không Last-Event-ID) → trả lại khung done
st2,ev2=sse('GET',f'/chat/messages/{mid}/stream',tk,maxsec=30)
rec('TC-P3-05-10',st2==200 and any(e=='done' for e,*_ in ev2),f'GET stream sau done → {st2}, {len(ev2)} khung')
# 43 feedback + phân quyền
s,d=A(K,'PUT',f'/chat/messages/{mid}/feedback',{'value':'HELPFUL'}); s2,d2=A('sv.gioi','PUT',f'/chat/messages/{mid}/feedback',{'value':'HELPFUL'})
# mid là USER message? lấy assistant
amid=psql(f"select id from chat_messages where session_id='{sid}' and role='ASSISTANT' order by created_at limit 1")
s,d=A(K,'PUT',f'/chat/messages/{amid}/feedback',{'value':'HELPFUL'}); s2,d2=A('sv.gioi','PUT',f'/chat/messages/{amid}/feedback',{'value':'HELPFUL'})
rec('TC-P3-05-43',s in(200,204) and s2 in(403,404),f'chủ phiên {s}, SV khác {s2}')
# 15/16 cancel
s,_=A(K,'POST',f'/chat/messages/{amid}/cancel'); s2,_=A(K,'POST',f'/chat/messages/{amid}/cancel'); s3,_=A('sv.gioi','POST',f'/chat/messages/{amid}/cancel')
rec('TC-P3-05-15',s in(200,204) and s2 in(200,204),f'cancel 2 lần → {s},{s2}'); rec('TC-P3-05-16',s3 in(403,404),f'SV khác cancel → {s3}')
# 28-30 idempotency (câu theo luật, không gọi LLM)
sid2=newsess(); key=str(uuid.uuid4())
r1=sse('POST',f'/chat/sessions/{sid2}/messages',tk,{'content':'Em được mấy điểm giữa kỳ?'},idem=key); time.sleep(1)
r2=sse('POST',f'/chat/sessions/{sid2}/messages',tk,{'content':'Em được mấy điểm giữa kỳ?'},idem=key)
n=psql(f"select count(*) from chat_messages where session_id='{sid2}' and role='USER'")
rec('TC-P3-05-28',r1[0]==200 and r2[0]==200 and n=='1',f'2 lần cùng key → {r1[0]}/{r2[0]}, hàng USER={n}')
time.sleep(3)
r3=sse('POST',f'/chat/sessions/{sid2}/messages',tk,{'content':'Em được mấy điểm giữa kỳ?'}); n=psql(f"select count(*) from chat_messages where session_id='{sid2}' and role='USER'")
rec('TC-P3-05-29',r3[0]==200 and n=='2',f'key khác → hàng USER={n}')
sid3=newsess(); key=str(uuid.uuid4()); outs=[]
def go(): outs.append(sse('POST',f'/chat/sessions/{sid3}/messages',tk,{'content':'song song'},idem=key)[0])
ts=[threading.Thread(target=go) for _ in range(10)]
[t.start() for t in ts]; [t.join() for t in ts]; time.sleep(2)
n=psql(f"select count(*) from chat_messages where session_id='{sid3}' and role='USER'")
rec('TC-P3-05-30',n=='1',f'10 yêu cầu song song cùng key → hàng USER={n}, mã={sorted(set(outs))}')
time.sleep(40)  # hạ rate
sid4=newsess()
s,d=A(K,'POST',f'/chat/sessions/{sid4}/messages',{'content':'   '}); s2,_=A(K,'POST',f'/chat/sessions/{sid4}/messages',{'content':''})
rec('TC-P3-05-31',s==422 and s2==422,f'rỗng/khoảng trắng → {s}/{s2}')
s,d=A(K,'POST',f'/chat/sessions/{sid4}/messages',{'content':'a'*4100}); rec('TC-P3-05-32',s==422,f'4100 ký tự → {s} {str(d)[:60]}')
# 34 (a) phiên đã xoá (b) phiên người khác (d) không auth (e) thiếu Idempotency-Key
sx=newsess(); s,_=A(K,'DELETE',f'/chat/sessions/{sx}'); a,_=A(K,'POST',f'/chat/sessions/{sx}/messages',{'content':'hi'})
b,_=A('sv.gioi','POST',f'/chat/sessions/{sid4}/messages',{'content':'hi'}); c,_=j('POST',f'/chat/sessions/{sid4}/messages',None,{'content':'hi'})
rec('TC-P3-05-34',a in(404,409,410) and b==404 and c==401,f'(a) phiên đã xoá→{a} (b) khác chủ→{b} (d) không auth→{c}')
# 46 delete/restore
s,_=A(K,'DELETE',f'/chat/sessions/{sx}'); s2,_=A(K,'POST',f'/chat/sessions/{sx}/restore'); s3,d3=A(K,'GET',f'/chat/sessions/{sx}/messages')
rec('TC-P3-05-46',s2 in(200,204) and s3==200,f'delete lặp→{s}, restore→{s2}, đọc lại→{s3}')
# 45 list & pagination
s,d=A(K,'GET',f'/chat/sessions?course_id={C1}&limit=2'); s2,d2=A(K,'GET',f'/chat/sessions?course_id={C1}&limit=0'); s3,d3=A(K,'GET',f'/chat/sessions?course_id={C1}&limit=101')
rec('TC-P3-05-45',s==200 and len(d['items'])<=2 and d.get('next_cursor') and s2==422 and s3==422,f'limit=2→{s} {len(d["items"])} mục, cursor={bool(d.get("next_cursor"))}; limit 0/101→{s2}/{s3}')
# 47/48/49 phân quyền
r=[A(u,'GET',f'/chat/sessions?course_id={C1}')[0] for u in ('ta','teacher','admin')]; r2=[A(u,'GET',f'/chat/sessions/{sid4}/messages')[0] for u in ('ta','teacher','admin')]
rec('TC-P3-05-47',all(x==403 for x in r) and all(x in(403,404) for x in r2),f'danh sách TA/TCH/ADM {r}, tin {r2}')
rs=[A('sv.gioi',m,p,b)[0] for m,p,b in [('GET',f'/chat/sessions/{sid4}/messages',None),('POST',f'/chat/messages/{amid}/cancel',None),('POST',f'/chat/messages/{amid}/retry',None),('PUT',f'/chat/messages/{amid}/feedback',{'value':'HELPFUL'}),('POST',f'/chat/messages/{amid}/escalate',None)]]
rec('TC-P3-05-48',all(x==404 for x in rs),f'SV A trên phiên SV B → {rs}')
s,_=A('sv.moi','GET',f'/chat/sessions?course_id={C1}'); s2,_=A('sv.moi','POST','/chat/sessions/from-draft',{'course_id':C1,'draft':'x'})
rec('TC-P3-05-49',s==403 and s2 in(403,404,422),f'SV ngoài lớp: list→{s}, from-draft→{s2}')
print('done')
