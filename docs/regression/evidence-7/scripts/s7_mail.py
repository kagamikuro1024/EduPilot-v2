from lib import *
import urllib.request,time,re
C=json.load(open('courses.json')); E1=C['E6C1']; Q=f'/courses/{E1}'
mp=lambda p: json.loads(urllib.request.urlopen('http://localhost:8725'+p,timeout=20).read())
tix=json.load(open('s7_tix.json'))
# P4-04-01/03: thư trả lời tới sv08
msgs=mp('/api/v1/search?query='+urllib.request.quote('to:sv08@edupilot.local subject:"Giảng viên đã trả lời"')+'&limit=5')['messages']
rec('TC-P4-04-01',len(msgs)>=1,f'Mailpit có {len(msgs)} thư «Giảng viên đã trả lời» tới sv08')
m=mp('/api/v1/message/'+msgs[0]['ID']); body=(m.get('Text') or '')+(m.get('HTML') or '')
leak=[x for x in ['chương trình học','Chào em, kỳ thi cuối kỳ do phòng đào tạo','CANARY','20229','0912345678'] if x in body]
rec('TC-P4-04-03',not leak and '/chat' in body or 'localhost' in body,f'thư chứa link; rò nội dung trả lời/PII: {leak}; độ dài {len(body)}')
# timing: so created_at outbox và gửi
r=psql("select extract(epoch from (updated_at-created_at))::numeric(6,2) from mail_outbox where template='escalation_answered' order by created_at desc limit 5"); rec('TC-P4-04-02',all(float(x)<60 for x in r.split('\n')),f'độ trễ outbox→SENT (s): {r.replace(chr(10)," ")}')
# 05/07 payload
pl=psql("select distinct jsonb_object_keys(payload) from mail_outbox where template='escalation_answered'"); rec('TC-P4-04-07',set(pl.split('\n'))<= {'ticket_id','answer_message_id'},f'khoá payload: {pl.replace(chr(10),",")}')
rec('TC-P4-04-05',set(pl.split('\n'))<= {'ticket_id','answer_message_id'},'payload escalation chỉ chứa id (không nội dung)')
# 26-28 settings
s,d=A('teacher','GET','/me/settings'); v=d['version']; s1,d1=A('teacher','PUT','/me/settings',{'notify_ticket_by_mail':True,'version':v}); s2,d2=A('teacher','PUT','/me/settings',{'notify_ticket_by_mail':False,'version':v})
rec('TC-P4-04-28',s1==200 and s2 in(409,412),f'GV đặt notify_ticket_by_mail → {s1}; version cũ → {s2}')
s3,d3=A('sv08','PUT','/me/settings',{'notify_ticket_by_mail':True,'version':1}); rec('TC-P4-04-27',s3 in(403,422),f'SV đặt notify_ticket_by_mail → {s3} {str(d3)[:60]}')
s,d=A('teacher','GET','/me/settings'); A('teacher','PUT','/me/settings',{'notify_ticket_by_mail':False,'version':d['version']})
# 29 route outbox
n=subprocess.run("grep -c 'outbox' /Users/kuro/Documents/TA_Agent_qc2/backend-go/api/openapi.yaml",shell=True,capture_output=True,text=True).stdout.strip(); print('openapi outbox',n)
rec('TC-P4-04-29',True,f'openapi: {n} lần nhắc "outbox" (đếm; kiểm route bên dưới)') if False else None
paths=subprocess.run("grep -n '^  /api/v1/.*outbox' /Users/kuro/Documents/TA_Agent_qc2/backend-go/api/openapi.yaml",shell=True,capture_output=True,text=True).stdout.strip()
rec('TC-P4-04-29',paths=='',f'route chứa "outbox": {paths or "không có"}')
# 31/32 failed-mail
s,d=A('admin','GET','/admin/observability/failed-mail'); r=[A(u,'GET','/admin/observability/failed-mail')[0] for u in ('teacher','ta','sv08')]
rec('TC-P4-04-31',s==200,f'ADMIN failed-mail → {s} {str(d)[:80]}'); rec('TC-P4-04-32',all(x==403 for x in r),f'GV/TA/SV → {r}')
# P4-02 notifications
s,d=A('teacher','GET','/notifications?limit=3'); rec('TC-P4-02-28',s==200 and len(d['items'])<=3,f'danh sách chuông {s}, {len(d["items"])} mục, unread? keys={[k for k in d if k!="items"]}')
tk=[x for x in d['items']] 
nid=psql("select id from notifications where user_id=(select id from users where email='teacher@edupilot.local') and read_at is null order by created_at desc limit 1")
r=A('sv08','POST',f'/notifications/{nid}/read'); r2=A('teacher','POST',f'/notifications/{nid}/read'); r3=A('teacher','POST',f'/notifications/{nid}/read')
rec('TC-P4-02-21',r[0]==404 and r2[0] in(200,204) and r3[0] in(200,204),f'SV đọc thông báo GV → {r[0]}; GV đọc → {r2[0]}; lặp → {r3[0]}')
r=[A('teacher','POST','/notifications/read',{'link':''})[0]]; rec('TC-P4-02-E1',r[0]==422,f'link rỗng → {r[0]}')
r2=A('teacher','POST','/notifications/read',{'link':'/inbox?course='+E1})[0]; rec('TC-P4-02-E1b',r2 in(200,204),f'đọc theo link → {r2}')
# 19 không PII
n=psql("select count(*) from notifications where type like 'TICKET%' and (title||body) ~* '20229|@edupilot|09[0-9]{8}'"); rec('TC-P4-02-19',n=='0',f'thông báo ticket chứa PII: {n}')
# 17 số khớp
a=psql(f"select count(*) from escalation_tickets where course_id='{E1}' and status='OPEN'"); 
t=psql("select title from notifications where type='TICKET_NEW' and course_id='%s' order by created_at desc limit 1"%E1); print(a,t)
