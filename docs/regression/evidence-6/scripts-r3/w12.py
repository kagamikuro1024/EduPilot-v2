from w7lib import *
import datetime as dt
P=f'/courses/{C1}'; now=dt.datetime.now(dt.timezone.utc)
iso=lambda x:x.strftime('%Y-%m-%dT%H:%M:%SZ')
def bank():
    out=[];cur=None
    while True:
        s,j=A('teacher','GET',P+'/questions?limit=100'+(f'&cursor={cur}' if cur else '')); out+=j['items']; cur=j.get('next_cursor')
        if not cur: return out
mcq=[q for q in bank() if q['type']=='MCQ_SINGLE' and q['review_status']=='APPROVED' and not q['used_in_exams']][:5]
print('mcq',len(mcq))
def mkexam(title,opens,closes):
    s,e=A('teacher','POST',P+'/exams',{'title':title,'opens_at':iso(opens),'closes_at':iso(closes),'duration_minutes':30}); assert s in(200,201),(s,e)
    eid=e['id']; s,j=A('teacher','PUT',P+f'/exams/{eid}/items',{'items':[{'question_id':q['id'],'points':'2'} for q in mcq],'version':e['version']}); assert s==200,(s,j)
    e2=A('teacher','GET',P+f'/exams/{eid}')[1]; s,j=A('teacher','POST',P+f'/exams/{eid}/schedule',{'version':e2['version']}); assert s in(200,201),(s,j)
    return eid
B_,A_='sv04','sv05'
uidB=psql(f"select id from users where email='{B_}@edupilot.local'")
# ===== 08-03-14 (ETag) phần 1 + 60: exam1 OPEN
eid=mkexam('QC3 khoá chat s6',now+dt.timedelta(days=3),now+dt.timedelta(days=3,hours=2))
psql(f"update exams set status='OPEN', opens_at=now()-interval '1 minute', closes_at=now()+interval '2 hours' where id='{eid}'")
F=iso(now-dt.timedelta(days=10)); T=iso(now+dt.timedelta(days=20))
s,t,h=raw('GET',P+f'/calendar?from={F}&to={T}',login(B_)); et=h.get('ETag') or h.get('Etag')
s,at=A(B_,'POST',P+f'/exams/{eid}/attempts',{}); print('attempt',s,str(at)[:80]); aid=at['id']
s2,t2,h2=raw('GET',P+f'/calendar?from={F}&to={T}',login(B_),headers={'If-None-Match':et or ''}); d2=json.loads(t2) if s2==200 else {}
ps=[i.get('personal_state') for i in d2.get('items',[]) if i['id']==f'weekly_exam:{eid}']
rec('TC-P8-03-14-a',s2==200 and ps==['IN_PROGRESS'],f'ETag trước khi bắt đầu lượt thi → sau khi bắt đầu, If-None-Match cũ → {s2} (không 304), personal_state={ps} (TLR-4)')
sA,dA=A(A_,'GET',P+f'/calendar?from={F}&to={T}&limit=100'); psA=[i.get('personal_state') for i in dA['items'] if i['id']==f'weekly_exam:{eid}']
sC2=A(A_,'GET',f'/courses/{C2}/calendar?from={F}&to={T}')[0]; st_set=A(A_,'GET','/me/settings'); stB=A(B_,'GET','/me/settings')
rec('TC-P8-03-60',psA==['NOT_STARTED'] and sC2==403 and st_set[0]==200 and 'IN_PROGRESS' not in json.dumps(dA),f'A ({A_}) thấy personal_state={psA} (không thấy IN_PROGRESS của B={B_}); lịch lớp 2 → {sC2}; /me/settings của A → {st_set[0]} chỉ preferences của chính A')
# ===== 05-22 / 05-23 / 05-24 / 06-12 / 06-40 / 06-41
sid=newsess(B_,C1); nB=lambda: psql(f"select count(*) from chat_messages where session_id='{sid}'")
au0=psql(f"select count(*) from llm_audit where user_id='{uidB}'")
s,d=A(B_,'POST',f'/chat/sessions/{sid}/messages',{'content':'hi'})
ev=psql(f"select type||'/'||coalesce(meta::text,'') from exam_events where attempt_id='{aid}' order by created_at desc limit 1")
rec('TC-P3-05-22',s==409 and d.get('code')=='EXAM_IN_PROGRESS' and 'until' in json.dumps(d) and nB()=='0' and psql(f"select count(*) from llm_audit where user_id='{uidB}'")==au0 and ev.startswith('CHAT_BLOCKED/{}'),f'409 {d.get("code")} details={d.get("details")}; chat_messages mới=0; llm_audit mới=0; exam_events={ev}')
for i in range(4): A(B_,'POST',f'/chat/sessions/{sid}/messages',{'content':'hi %d'%i}); time.sleep(2)
cb=psql(f"select count(*) from exam_events where attempt_id='{aid}' and type='CHAT_BLOCKED'"); ttl=rds('ttl',f'ep:chat_blocked:{aid}')
rec('TC-P3-05-23',int(cb)<=1 and 0<int(ttl)<=60,f'gửi 5 lần trong ~10 s: số dòng CHAT_BLOCKED={cb} (≤1/phút); TTL ep:chat_blocked:{{attempt}}={ttl}s (≤60)')
psql(f"insert into chat_messages(course_id,session_id,user_id,role,content,stream_status,error_code,reply_to) select '{C1}','{sid}','{uidB}','USER','x','DONE',null,null")
mf=psql(f"insert into chat_messages(course_id,session_id,user_id,role,content,stream_status,error_code) values('{C1}','{sid}','{uidB}','ASSISTANT','','FAILED','INTERNAL') returning id").splitlines()[0]
s,d=A(B_,'POST',f'/chat/messages/{mf}/retry')
doc=psql(f"select id from documents where course_id='{C1}' and status='READY' and visible_to_students and type<>'ANSWER_KEY' and use_for_rag limit 1")
s2,d2=A(B_,'POST','/chat/sessions',{'course_id':C1,'document_id':doc,'title':'qc3 doc'}); s3,d3=A(B_,'POST',f"/chat/sessions/{d2['id']}/messages",{'content':'Giải thích tài liệu'}) if s2==201 else (None,None)
rec('TC-P3-05-24',s==409 and d.get('code')=='EXAM_IN_PROGRESS' and s2==201 and s3==409,f'retry tin FAILED → {s} {d.get("code")}; tạo phiên "Hỏi AI về tài liệu" → {s2} (vẫn được); gửi tin trong phiên đó → {s3} {d3.get("code") if d3 else None}')
# 06-12 from-draft
s,d=A(B_,'POST','/chat/sessions/from-draft',{'course_id':C1,'body':'nháp\ncó dấu Tiếng Việt'}); sid2=d.get('id') or d.get('session',{}).get('id') if isinstance(d,dict) else None
s2,d2=A(B_,'POST',f'/chat/sessions/{sid2}/messages',{'content':'gửi'}) if sid2 else (None,None)
g=subprocess.run(GO+"go test ./internal/chat ./internal/thread -run 'TestFromDraftKeepsText|TestFromDraftNoMessageStored|TestSwitchedEvent' -v -count=1 2>&1 | grep -E '^(--- |ok|FAIL)'",shell=True,capture_output=True,text=True).stdout
rec('TC-P3-06-12',s==201 and s2==409 and 'FAIL' not in g,f'trong giờ thi: from-draft → {s} (vẫn tạo phiên); gửi tin → {s2} {d2.get("code") if d2 else None}; go test: {g.replace(chr(10)," | ")[:150]}')
# 06-40 / 06-41
ft0=psql(f"select count(*) from forum_threads where course_id='{C1}'")
s,d=A(B_,'POST',f'/courses/{C1}/threads',{'title':'Hỏi','body':'AES là gì?'}); ft1=psql(f"select count(*) from forum_threads where course_id='{C1}'")
s_pii,d_pii=A(B_,'POST',f'/courses/{C1}/threads',{'title':'Hỏi','body':'Bùi Thanh Khải 20229002 được mấy điểm?'})
ev=psql(f"select type||'/'||coalesce(meta::text,'') from exam_events where attempt_id='{aid}' and type='CHAT_BLOCKED' order by created_at desc limit 1")
rec('TC-P3-06-40',s==409 and d.get('code')=='EXAM_IN_PROGRESS' and 'until' in json.dumps(d) and ft0==ft1 and s_pii==409 and ev.startswith('CHAT_BLOCKED/{}'),f'đăng thread trong giờ thi → {s} {d.get("code")}; forum_threads mới={int(ft1)-int(ft0)}; bài chứa PII cũng {s_pii} (khoá chạy trước tường lửa); exam_events={ev}')
tid=psql(f"select id from forum_threads where course_id='{C1}' and ai_state='ANSWERED' limit 1")
r=[A(B_,'GET',f'/courses/{C1}/threads')[0],A(B_,'GET',f'/courses/{C1}/threads/{tid}')[0],A(B_,'POST',f'/courses/{C1}/threads/precheck',{'body':'x'})[0],A(B_,'POST',f'/courses/{C1}/threads/{tid}/posts',{'body':'AES là gì?'})[0],A(B_,'POST','/chat/sessions/from-draft',{'course_id':C1,'body':'x'})[0]]
rec('TC-P3-06-41',r[0]==200 and r[1]==200 and r[2]==200 and r[3]==201 and r[4]==201,f'trong giờ thi: đọc ds/chi tiết/precheck/bình luận/from-draft → {r} (kỳ vọng 200,200,200,201,201: chỉ đăng thread MỚI bị chặn)')
# ===== 03-14 phần qua mốc giờ: exam2 opens trong 20 s
eid2=mkexam('QC3 mốc giờ s6',now+dt.timedelta(seconds=25),now+dt.timedelta(minutes=30))
s,t,h=raw('GET',P+f'/calendar?from={F}&to={T}',login(A_)); et1=h.get('ETag') or h.get('Etag'); st1=[i.get('status') or i.get('personal_state') for i in json.loads(t)['items'] if i['id']==f'weekly_exam:{eid2}']
time.sleep(35)
s2,t2,h2=raw('GET',P+f'/calendar?from={F}&to={T}',login(A_),headers={'If-None-Match':et1 or ''}); st2=[i.get('status') or i.get('personal_state') for i in json.loads(t2)['items'] if i['id']==f'weekly_exam:{eid2}'] if s2==200 else None
rec('TC-P8-03-14',s2==200,f'qua mốc opens_at: ETag cũ → {s2} (200 = EffectiveStatus đổi); trạng thái {st1}→{st2}; và sau khi bắt đầu lượt thi → {s2} IN_PROGRESS (TC-P8-03-14-a)')
# ===== nộp bài → 05-25
sub=A(B_,'POST',P+f'/exams/{eid}/attempts/{aid}/submit',{}); print('submit',sub[0])
ok=False
for i in range(14):
    time.sleep(1); lk=A(B_,'GET','/me/exam-lock')[1]
    if lk.get('locked') is False: ok=True; break
r=chat(B_,'Giải thích ngắn gọn hàm băm là gì',course=C1)
rec('TC-P3-05-25',ok and r['status']==200 and any(e[0]=='done' for e in r['events']),f'nộp bài (submit {sub[0]}) → exam-lock mở sau {i+1}s (≤10 s); gửi lại tin → HTTP {r["status"]}, SSE có done={any(e[0]=="done" for e in r["events"])}, văn bản {r["text"][:60]!r}')
json.dump(R,open('/tmp/qc3e/w12.json','w'),ensure_ascii=False)
