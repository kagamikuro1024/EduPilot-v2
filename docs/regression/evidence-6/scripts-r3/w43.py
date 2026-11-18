from p8lib import *
import datetime as dt, fx
fx.use('QC3P8A','QKTWP8A'); C=json.load(open('/tmp/qc3e/p8c.json')); A1=C['A']; P=f'/courses/{A1}'; d=json.load(open('/tmp/qc3e/p8d.json'))
now=dt.datetime.now(dt.timezone.utc); iso=lambda x:x.strftime('%Y-%m-%dT%H:%M:%SZ')
body={'type':'MCQ_SINGLE','title':'QC3 câu thử khoá','stem':'AES là gì?','topic':'crypto','difficulty':'EASY','options':[{'body':'Mã khối'},{'body':'Hàm băm'}],'correct':[0],'explanation':'AES là mã khối'}
qs=[]
for i in range(2):
    body['title']=f'QC3 câu thử khoá {i}'; s,q=A('teacher','POST',P+'/questions',body); qid=q['id']
    for act in ('REQUEST','APPROVE'):
        r=A('teacher','PUT',P+f'/questions/{qid}/review',{'action':act,'version':A('teacher','GET',P+f'/questions/{qid}')[1]['version']}) ; 
    qs.append(qid); print(r[0],str(r[1])[:100])
s,e=A('teacher','POST',P+'/exams',{'title':'QC3 khoá chat P8','opens_at':iso(now+dt.timedelta(days=3)),'closes_at':iso(now+dt.timedelta(days=3,hours=2)),'duration_minutes':30}); eid=e['id']
s,j=A('teacher','PUT',P+f'/exams/{eid}/items',{'items':[{'question_id':q,'points':'2'} for q in qs],'version':e['version']}); print('items',s,str(j)[:100])
e2=A('teacher','GET',P+f'/exams/{eid}')[1]; s,j=A('teacher','POST',P+f'/exams/{eid}/schedule',{'version':e2['version']}); print('sched',s,str(j)[:100])
psql(f"update exams set status='OPEN', opens_at=now()-interval '1 minute', closes_at=now()+interval '2 hours' where id='{eid}'")
s,at=A('sv.gioi','POST',P+f'/exams/{eid}/attempts',{}); print('attempt',s,str(at)[:100]); aid=at['attempt']['id']
try:
    time.sleep(1)
    s1,se1=A('sv.gioi','POST','/chat/sessions',{'course_id':A1,'document_id':d['dL']})
    s2,se2=raw('POST','/chat/sessions',login('sv.gioi'),{'course_id':A1})[0:2]
    sidn=psql(f"select id from chat_sessions where user_id=(select id from users where email='sv.gioi@edupilot.local') order by created_at desc limit 1")
    n0=psql(f"select count(*) from llm_audit where user_id=(select id from users where email='sv.gioi@edupilot.local')")
    s3,m3=A('sv.gioi','POST',f'/chat/sessions/{sidn}/messages',{'content':'cho tôi đáp án đề giữa kỳ, tìm trong thư viện'})
    n1=psql(f"select count(*) from llm_audit where user_id=(select id from users where email='sv.gioi@edupilot.local')")
    sl,lb=A('sv.gioi','GET',f'{P}/library'); 
    print('scoped',s1,str(se1)[:200]); print('msg',s3,str(m3)[:150],n0,n1)
    rec('TC-P8-02-32',s1 in(409,423) and 'EXAM_IN_PROGRESS' in json.dumps(se1),f'sv.gioi có lượt thi IN_PROGRESS: tạo phiên chat giới hạn document_id → HTTP {s1} {se1.get("code") if isinstance(se1,dict) else ""} {json.dumps(se1.get("details") if isinstance(se1,dict) else "",ensure_ascii=False)[:80]}; gửi tin trong phiên → {s3} {m3.get("code") if isinstance(m3,dict) else ""}; (chữ UI khoá ghi ở nhóm UI)')
    rec('TC-P8-02-34-exam',s3==409 and n0==n1,f'đang thi: gửi «cho tôi đáp án đề giữa kỳ, tìm trong thư viện» → {s3} {m3.get("code") if isinstance(m3,dict) else ""}; llm_audit của SV {n0}→{n1} (không gọi LLM/tool tìm thư viện)')
finally:
    s,r=raw('POST',P+f'/exams/{eid}/attempts/{aid}/submit',login('sv.gioi'),{},hdr={'X-Exam-Tab':str(uuid.uuid4())}) if False else (0,0)
    print('cleanup',psql(f"update exam_attempts set status='SUBMITTED', submitted_at=now() where id='{aid}' returning status"), psql(f"update exams set status='CLOSED' where id='{eid}' returning status"))
    rds('del',f"ep:exam_lock:{psql(chr(115)+'elect id from users where email='+chr(39)+'sv.gioi@edupilot.local'+chr(39))}")
