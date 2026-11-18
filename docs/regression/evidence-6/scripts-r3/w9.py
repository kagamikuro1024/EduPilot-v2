from lib import *
import datetime as dt,re,subprocess
C1=psql("select id from courses where class_code='761987'"); C2=psql("select id from courses where class_code='761988'")
now=dt.datetime.now(dt.timezone.utc); iso=lambda x:x.strftime('%Y-%m-%dT%H:%M:%SZ')
F=iso(now-dt.timedelta(days=45)); Tt=iso(now+dt.timedelta(days=16))
# ---- 02: reminder_log
cols=psql("select string_agg(column_name,',' order by ordinal_position) from information_schema.columns where table_name='reminder_log'")
en=psql("select enum_range(null::reminder_source)")
fk=psql("select string_agg(conname,',') from pg_constraint where conrelid='reminder_log'::regclass and contype='f'")
uq=psql("select pg_get_constraintdef(oid) from pg_constraint where conrelid='reminder_log'::regclass and contype='u'")
ck=psql("select pg_get_constraintdef(oid) from pg_constraint where conrelid='reminder_log'::regclass and contype='c' and conname like '%kind%'")
idx=psql("select count(*) from pg_indexes where tablename='reminder_log' and indexdef like '%(starts_at)%'")
uid=psql("select id from users where email='sv.gioi@edupilot.local'")
ins=f"insert into reminder_log(user_id,course_id,source_type,source_id,starts_at,kind) values('{uid}','{C1}','CALENDAR_EVENT',gen_random_uuid(),now(),'T24H');"
x=sqlx("create temp table _r as select gen_random_uuid() sid; "+f"insert into reminder_log(user_id,course_id,source_type,source_id,starts_at,kind) select '{uid}','{C1}','CALENDAR_EVENT',sid,'2030-01-01','T24H' from _r; insert into reminder_log(user_id,course_id,source_type,source_id,starts_at,kind) select '{uid}','{C1}','CALENDAR_EVENT',sid,'2030-01-01','T24H' from _r;")
k=sqlx(f"insert into reminder_log(user_id,course_id,source_type,source_id,starts_at,kind) values('{uid}','{C1}','CALENDAR_EVENT',gen_random_uuid(),now(),'T48H');")
rec('TC-P8-03-02',x[0]=='23505' and k[0]=='23514' and 'source_id' not in fk and idx=='1' and 'user' in fk and 'course' in fk,f'cột={cols}; enum reminder_source={en} (nay 4 giá trị — ASSIGNMENT thêm ở sprint 9 proposals #3; 3 giá trị gốc vẫn có đủ); FK={fk} (source_id không FK); UNIQUE={uq}; chèn trùng → {x[0]}; kind≠T24H → {k[0]}; chỉ mục (starts_at)={idx}')
# ---- 03: ics_token ràng buộc
a=sqlx("update users set ics_token='token-tho-khong-bam' where email='sv.gioi@edupilot.local';")
b=sqlx("update users set ics_token=repeat('a',64) where email='sv.gioi@edupilot.local';")
c=sqlx("update users set ics_token=null where email='sv.gioi@edupilot.local';")
pr=subprocess.run("grep -nE '00011 +calendar|00009 +calendar' /Users/kuro/Documents/TA_Agent_qc3p/docs/PROGRESS.md | head -3",shell=True,capture_output=True,text=True).stdout.strip()
rec('TC-P8-03-03',a[0]=='23514' and 'users_ics_token_hash_chk' in a[1] and b[0]=='OK' and c[0]=='OK',f'token thô → {a[0]} ({a[1][-60:]}); 64 hex → {b[0]}; NULL → {c[0]}; PROGRESS.md ghi migration calendar: {pr!r} (số đã đổi 00009→00011 theo proposals #13)')
# ---- 05: weekly_exam trong lịch
s,d=A('teacher','GET',f'/courses/{C1}/calendar?from={F}&to={Tt}&limit=100'); items=d.get('items',[]) if s==200 else []
wex=[i for i in items if i['source']=='weekly_exam']
db={r.split('|')[0]:r.split('|') for r in psql(f"select id,status,to_char(opens_at at time zone 'UTC','YYYY-MM-DD\"T\"HH24:MI:SS'),to_char(closes_at at time zone 'UTC','YYYY-MM-DD\"T\"HH24:MI:SS') from exams where course_id='{C1}'").splitlines()}
bad=[]
for i in wex:
    eid=i['id'].split(':')[-1]; r=db.get(eid)
    if not r: bad.append(('thiếu trong DB',i['id'])); continue
    if r[1]=='DRAFT': bad.append(('DRAFT lộ',eid))
    if not (i['starts_at'].startswith(r[2]) and (i['ends_at'] or '').startswith(r[3])): bad.append(('lệch giờ',eid,i['starts_at'],r[2]))
draft=[e for e,r in db.items() if r[1]=='DRAFT']
absent=all(f'weekly_exam:{e}' not in [i['id'] for i in items] for e in draft)
rec('TC-P8-03-05',s==200 and not bad and absent and len(draft)>=1 and len(wex)>=1,f'GET lịch (GV) {s}: {len(wex)} mục weekly_exam, {len(db)} bài thi trong DB ({len(draft)} DRAFT, không xuất hiện={absent}); lệch giờ/lộ: {bad}')
# ---- 08: SV chỉ ở lớp 1
only1=psql(f"select u.email from enrollments e join users u on u.id=e.user_id where e.course_id='{C1}' and e.role_in_course='STUDENT' and e.status='ACTIVE' and e.user_id not in (select user_id from enrollments where course_id='{C2}') order by 1 limit 1").split('@')[0]
s1,d1=A(only1,'GET',f'/courses/{C1}/calendar?from={F}&to={Tt}&limit=100'); s2,d2=A(only1,'GET',f'/courses/{C2}/calendar?from={F}&to={Tt}')
t2=set(i['title'] for i in A('teacher','GET',f'/courses/{C2}/calendar?from={F}&to={Tt}&limit=100')[1].get('items',[])); t1=[i['title'] for i in d1.get('items',[])]
rec('TC-P8-03-08',s1==200 and s2==403 and not (set(t1)&(t2-set(i['title'] for i in A('teacher','GET',f'/courses/{C1}/calendar?from={F}&to={Tt}&limit=100')[1].get('items',[])))),f'{only1} (chỉ lớp 1): lịch lớp 1 → {s1} ({len(t1)} mục), lịch lớp 2 → {s2}; không mục nào thuộc riêng lớp 2')
# ---- 16: không lộ điểm/câu hỏi/id lượt thi/tên email
s,d=A('sv.gioi','GET',f'/courses/{C1}/calendar?from={F}&to={Tt}&limit=100'); blob=json.dumps(d.get('items',[]),ensure_ascii=False)
hit=re.findall(r'(?i)score|điểm|question|câu hỏi|answer|đáp án|attempt_id|sv\.|@edupilot',blob)
rec('TC-P8-03-16',s==200 and not hit and len(d['items'])>0,f'{len(d["items"])} mục; khoá cấm tìm thấy: {hit[:5]}')
# ---- 15: personal_state theo người, href, editable
exs=[i for i in d['items'] if i['source']=='weekly_exam']
def view(u): 
    s,x=A(u,'GET',f'/courses/{C1}/calendar?from={F}&to={Tt}&limit=100'); return {i['id']:(i.get('personal_state'),i.get('href'),i.get('editable'),i['source']) for i in x['items']}
va,vb,vt=view('sv.gioi'),view('sv.kha'),view('teacher')
states_a={v[0] for k,v in va.items() if v[3]=='weekly_exam'}; states_t={v[0] for k,v in vt.items() if v[3]=='weekly_exam'}
hrefs_sv={v[1][:7] for k,v in va.items() if v[3]=='weekly_exam'}; hrefs_t={v[1][:7] for k,v in vt.items() if v[3]=='weekly_exam'}
edit_ok=all((v[2] is True)==(v[3]=='calendar_event') for v in vt.values()) and not any(v[2] for v in va.values())
rec('TC-P8-03-15',states_t=={None} and edit_ok and all('/take' in v[1] for v in va.values() if v[3]=='weekly_exam') and all('/take' not in v[1] for v in vt.values() if v[3]=='weekly_exam') and len(states_a)>=1,f'SV A: personal_state={sorted(map(str,states_a))}; SV B: {sorted({str(v[0]) for v in vb.values() if v[3]=="weekly_exam"})}; Staff: {states_t}; href SV=/exams/{{id}}/take, Staff=/exams/{{id}}; editable chỉ calendar_event+Staff: {edit_ok}')
json.dump(R,open('/tmp/qc3e/w9.json','w'),ensure_ascii=False)
