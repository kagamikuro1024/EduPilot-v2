from lib import *
C1='01a1df27-62b5-7775-a346-5c602a43e2f9'
U=psql("select id from users where email='sv.gioi@edupilot.local'")
r=psql("select enum_range(null::chat_stream_status)::text||' '||enum_range(null::chat_role)::text||' '||enum_range(null::chat_channel)::text")
rec('TC-P3-01-08',r=='{STREAMING,DONE,FAILED,CANCELLED} {USER,ASSISTANT} {PRIVATE,PUBLIC}',r)
r=psql("select enum_range(null::pii_kind)::text||' '||enum_range(null::pii_action)::text")
rec('TC-P3-01-36',r=='{MSSV,EMAIL,PHONE,CCCD,NAME,PERSONAL_QUESTION,OTHER_PERSON} {BLOCKED,REDACTED,SWITCHED,MASKED}',r)
pre=f"create temp table _s(id uuid);with x as (insert into chat_sessions(course_id,user_id) values('{C1}','{U}') returning id) insert into _s select id from x;"
a=sqlx(pre+f"insert into chat_messages(course_id,session_id,user_id,role,content) select '{C1}',id,'{U}','USER','x' from _s;select pg_sleep(0.05);update chat_messages set content='y' where session_id=(select id from _s);select (select updated_at from chat_messages where session_id=(select id from _s))>(select created_at from chat_messages where session_id=(select id from _s));")
rec('TC-P3-01-14',a[0]=='OK' and '\nt\n' in '\n'+a[2]+'\n','updated_at tăng sau update (trigger)')
r=sqlx(f"create temp table _d(id uuid);with x as (insert into documents(course_id,title,uploaded_by) select '{C1}','qc-d',id from users where email='sv.gioi@edupilot.local' returning id) insert into _d select id from x;insert into chat_sessions(course_id,user_id,document_id) select '{C1}','{U}',id from _d;delete from documents where id=(select id from _d);select document_id is null from chat_sessions where user_id='{U}' order by created_at desc limit 1;")
rec('TC-P3-01-28',r[0]=='OK' and 't' in r[2].split('\n'),'delete ok, document_id→NULL' if r[0]=='OK' else r[1])
r=sqlx(f"insert into forum_threads(course_id,author_id,title,body) values('{C1}','{U}','t','b');select 1;")
t=sqlx(f"create temp table _t(id uuid);with x as (insert into forum_threads(course_id,author_id,title,body) values('{C1}','{U}','t','b') returning id) insert into _t select id from x;select pg_sleep(0.05);update forum_threads set title='u' where id=(select id from _t);select updated_at>created_at from forum_threads where id=(select id from _t);")
rec('TC-P3-01-30',t[0]=='OK' and 't' in t[2].split('\n'),'updated_at tăng')
import subprocess
o=subprocess.run("cd /Users/kuro/Documents/TA_Agent_qc2/backend-go && grep -ciE '^[[:space:]]*alter table' db/migrations/00007_chat_threads.sql db/migrations/00008_privacy.sql",shell=True,capture_output=True,text=True).stdout.split()
rec('TC-P3-01-03',all(x.endswith(':0') for x in o),' '.join(o))
rec('TC-P3-01-49',psql("select rolsuper from pg_roles where rolname='edupilot'")=='f','vai ứng dụng edupilot rolsuper?'+psql("select rolsuper from pg_roles where rolname='edupilot'"))
save('s6_p301b')
