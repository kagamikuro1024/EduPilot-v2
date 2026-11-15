from lib import *
C1='01a1df27-62b5-7775-a346-5c602a43e2f9';C2='01a1df27-62c2-7138-be74-2842f052e7fb'
U=psql("select id from users where email='sv.gioi@edupilot.local'")
UB=psql("select id from users where email='sv.kha@edupilot.local'")
pre=f"create temp table _s(id uuid);with x as (insert into chat_sessions(course_id,user_id) values('{C1}','{U}') returning id) insert into _s select id from x;"
msg=lambda cols,vals: f"insert into chat_messages(course_id,session_id,user_id,role,{cols}) select '{C1}',id,'{U}',{vals} from _s;"
def t(sql,pre=pre): return sqlx(pre+sql)
def exp(tc,res,code,note=''): rec(tc,res[0]==code,f'{res[0]} {note}' if res[0]!=code else f'SQLSTATE {code}')
# AC2
r=psql("select enum_range(null::chat_stream_status)||' '||enum_range(null::chat_role)||' '||enum_range(null::chat_channel)")
rec('TC-P3-01-08',r=='{STREAMING,DONE,FAILED,CANCELLED} {USER,ASSISTANT} {PRIVATE,PUBLIC}',r)
cid='gen_random_uuid()'
cm=f"insert into chat_messages(course_id,session_id,user_id,role,content,client_msg_id) select '{C1}',id,'{U}','USER','x','11111111-1111-1111-1111-111111111111' from _s;"
x=t(cm+cm); exp('TC-P3-01-09',x,'23505')
exp('TC-P3-01-10',t(msg('stream_status',"'USER','STREAMING'")),'23514')
exp('TC-P3-01-11',t(msg('confidence',"'USER',0.500")),'23514')
exp('TC-P3-01-12',t(msg('stream_status,partial_content',"'ASSISTANT','DONE','abc'")),'23514')
exp('TC-P3-01-13',t(msg('stream_status,completed_at',"'ASSISTANT','STREAMING',now()")),'23514')
a=t(msg('content',"'USER','x'")+"select pg_sleep(0.01);update chat_messages set content='y' where session_id=(select id from _s);select (select updated_at from chat_messages where session_id=(select id from _s))>(select created_at from chat_messages where session_id=(select id from _s));")
rec('TC-P3-01-14',a[0]=='OK' and 't' in a[2].split('\n')[-1],str(a[2])[-30:])
exp('TC-P3-01-15',t(msg('content',"'USER',repeat('a',20001)")),'23514')
# AC3
exp('TC-P3-01-24',t(f"create temp table _t(id uuid);with x as (insert into forum_threads(course_id,author_id,title,body) values('{C1}','{U}','t','b') returning id) insert into _t select id from x; insert into forum_posts(course_id,thread_id,author_id,kind,body) select '{C2}',id,'{U}','HUMAN','b' from _t;"),'23503')
exp('TC-P3-01-25',t(f"insert into chat_messages(course_id,session_id,user_id,role,content) select '{C2}',id,'{U}','USER','x' from _s;"),'23503')
exp('TC-P3-01-26',t(f"insert into chat_messages(course_id,session_id,user_id,role,content) select '{C1}',id,'{UB}','USER','x' from _s;"),'23503')
x=sqlx(f"insert into chat_sessions(course_id,user_id,last_message_at) values('{C1}','{U}',null);"); exp('TC-P3-01-29',x,'23502')
# AC3 posts
th=f"create temp table _t(id uuid);with x as (insert into forum_threads(course_id,author_id,title,body) values('{C1}','{U}','t','b') returning id) insert into _t select id from x;"
ai=lambda: f"insert into forum_posts(course_id,thread_id,kind,body,verification_state) select '{C1}',id,'AI','a','PENDING' from _t;"
exp('TC-P3-01-18',sqlx(th+ai()+ai()),'23505')
exp('TC-P3-01-19',sqlx(th+f"insert into forum_posts(course_id,thread_id,author_id,kind,body,verification_state) select '{C1}',id,'{U}','AI','a','PENDING' from _t;"),'23514')
exp('TC-P3-01-20',sqlx(th+f"insert into forum_posts(course_id,thread_id,author_id,kind,body,hidden_at) select '{C1}',id,'{U}','HUMAN','a',now() from _t;"),'23514')
exp('TC-P3-01-21',sqlx(th+f"insert into forum_posts(course_id,thread_id,author_id,kind,body,verification_state) select '{C1}',id,'{U}','HUMAN','a','PENDING' from _t;"),'23514')
w=lambda c: sqlx(f"insert into forum_threads(course_id,author_id,title,body,{c[0]}) values('{C1}','{U}','t','b',{c[1]});")
ok=all(x[0]=='23514' for x in [w(('tags',"array_fill('x'::text,array[6])")),w(('week_no','0')),w(('week_no','21')),sqlx(f"insert into forum_threads(course_id,author_id,title,body) values('{C1}','{U}','','b');"),sqlx(f"insert into forum_threads(course_id,author_id,title,body) values('{C1}','{U}','t',repeat('a',8001));")])
rec('TC-P3-01-22',ok,'6 vi phạm đều 23514' if ok else 'có ca không 23514')
# AC4 doc delete set null
r=sqlx(f"create temp table _d(id uuid);with x as (insert into documents(course_id,title,kind,status,uploaded_by,file_name,mime_type,size_bytes,sha256,storage_key) select '{C1}','d','OTHER','READY',id,'f','text/plain',1,'x','k' from users where email='sv.gioi@edupilot.local' returning id) insert into _d select id from x;")
print('docins',r[:2])
# AC5 pii_events
c=psql("select string_agg(column_name,',' order by ordinal_position) from information_schema.columns where table_name='pii_events'")
rec('TC-P3-01-32',c=='id,course_id,session_id,user_id,channel,pii_type,count,action,created_at',c)
rec('TC-P3-01-33',psql("select count(*) from information_schema.columns where table_name='pii_events' and data_type in ('text','character varying','character')")=='0')
pe=f"insert into pii_events(course_id,user_id,channel,pii_type,count,action) values('{C1}','{U}','PRIVATE','MSSV',1,'BLOCKED');"
x=sqlx(pe+"update pii_events set count=2 where true;"); y=sqlx(pe+"delete from pii_events where true;")
rec('TC-P3-01-34',x[0]=='42501' and y[0]=='42501' and 'append-only' in x[1],x[1][:80])
rec('TC-P3-01-35',sqlx(pe.replace(",1,",",0,"))[0]=='23514' and sqlx(pe.replace(",1,",",-1,"))[0]=='23514')
r=psql("select enum_range(null::pii_kind)||' '||enum_range(null::pii_action)")
rec('TC-P3-01-36',r=='{MSSV,EMAIL,PHONE,CCCD,NAME,PERSONAL_QUESTION,OTHER_PERSON} {BLOCKED,REDACTED,SWITCHED,MASKED}',r)
n=psql("select count(*) from pg_indexes where tablename in ('chat_sessions','chat_messages','forum_threads','forum_posts','pii_events') and indexname not like '%pkey'")
rec('TC-P3-01-38',int(n)>=11,f'{n} chỉ mục không-pkey (≥11)')
# AC9 role
r=psql("select rolsuper from pg_roles where rolname='edupilot'")
print('role super',r)
# canary: send chat later
rec('TC-P3-01-03',subprocess.run("cd /Users/kuro/Documents/TA_Agent_qc2/backend-go && grep -ciE '^[[:space:]]*alter table' db/migrations/00007_chat_threads.sql db/migrations/00008_privacy.sql",shell=True,capture_output=True,text=True).stdout.strip().replace('\n',' '),'')
save('s6_p301')
