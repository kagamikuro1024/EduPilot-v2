# P207 các ca D (schema / ràng buộc / chỉ mục) chạy trên DB riêng qc2i_auth (ep-r-postgres-1), KHÔNG đụng DB `edupilot` của ep-r.
import os,sys; sys.path.insert(0,os.path.dirname(os.path.abspath(__file__)))
from smkT import rec,sh
import subprocess,json,re,uuid
def pq(q,db='qc2i_auth'):
    r=subprocess.run(['docker','exec','-i','ep-r-postgres-1','psql','-U','edupilot','-d',db,'-At','-F','|','-c',q],capture_output=True,text=True); return r.stdout.strip()
def tryins(sql):
    """chạy 1 câu trong transaction + ROLLBACK, trả SQLSTATE hoặc 'OK'"""
    q="do $$ begin "+sql.replace('$$','')+"; raise notice 'RES OK'; exception when others then raise notice 'RES %', sqlstate; end $$"
    r=subprocess.run(['docker','exec','-i','ep-r-postgres-1','psql','-U','edupilot','-d','qc2i_auth','-At','-c','begin;\n'+q+';\nrollback;'],capture_output=True,text=True)
    m=re.search(r'RES (\w+)',r.stderr+r.stdout); return m.group(1) if m else 'NO:'+(r.stderr[:80])
# ---- P207-01/02 schema
tabs=pq("select string_agg(table_name,',' order by table_name) from information_schema.tables where table_schema='public' and table_name in ('courses','enrollments','class_sessions','notifications','user_settings','documents','content_chunks','document_courses')")
n8=len(tabs.split(',')) if tabs else 0
def cols(t): return pq(f"select string_agg(column_name||':'||udt_name,',' order by ordinal_position) from information_schema.columns where table_name='{t}' and table_schema='public'")
dc=cols('documents'); cc=cols('content_chunks')
need_d=['blob_key','visible_to_students','use_for_rag','category','week_no','download_count']
miss_d=[c for c in need_d if c+':' not in dc]
emb=pq("select format_type(atttypid,atttypmod) from pg_attribute where attrelid='content_chunks'::regclass and attname='embedding'")
ci=pq("select udt_name from information_schema.columns where table_name='content_chunks' and column_name='course_ids'")
aud=pq("select count(*) from information_schema.columns where table_name='content_chunks' and column_name='audience'")
# chỉ mục phức hợp có course_id không đứng đầu
bad_idx=[]
for l in pq("select tablename||'|'||indexdef from pg_indexes where schemaname='public' and tablename in ('courses','enrollments','class_sessions','notifications','user_settings','documents','content_chunks','document_courses')").split('\n'):
    t,d=l.split('|',1); m=re.search(r'USING btree \((.*?)\)',d)
    if m:
        cl=[x.strip() for x in m.group(1).split(',')]
        if 'course_id' in cl and len(cl)>1 and cl[0]!='course_id': bad_idx.append(l[:100])
gin=pq("select indexdef from pg_indexes where tablename='content_chunks' and indexdef ilike '%gin%course_ids%'")
ext=pq("select extname from pg_extension where extname='vector'")
hist=sh("cd ~/Documents/TA_Agent_qc2 && git log --format=%h --diff-filter=M -- backend-go/db/migrations/00001_pg_platform.sql backend-go/db/migrations/00002_llm.sql | wc -l").strip()
rec('TC-P207-01','P' if n8==8 and not miss_d and emb=='vector(1536)' and ci=='_uuid' and aud=='1' and not bad_idx and hist=='0' else 'F',f'8/8 bảng lớp ({tabs}); documents có đủ {need_d} (thiếu {miss_d}); content_chunks.course_ids={ci}(uuid[]), audience có, embedding={emb}; chỉ mục phức hợp có course_id không đứng đầu: {bad_idx or "không"}; 00001/00002 không bị sửa (git log --diff-filter=M = {hist} commit) [DB qc2i_auth migrate tới 00027; so cột theo danh sách TC thay cho expected-course.tsv]')
rec('TC-P207-02','P' if len(gin.split('\n'))==1 and 'gin' in gin.lower() and ext=='vector' else 'F',f'pg_indexes content_chunks GIN course_ids: {gin[:110]}; extension {ext}')
# ---- P207-04 INSERT sai
adm=pq("select id from users where email='qc2ia-admin@edupilot.local'"); u1=pq("select id from users where email like 'qc2ia-bulk1@%' limit 1"); u2=pq("select id from users where email like 'qc2ia-bulk2@%' limit 1")
mkc=lambda cc,jc,extra='',cols='',vals='': f"insert into courses(subject_code,class_code,name,semester,join_code,created_by{cols}) values('INT1006','{cc}','L','2026-2027-HK1','{jc}','{adm}'{vals})"
base=pq("select count(*) from courses")
cases=[]
def case(name,sql,exp): cases.append((name,tryins(sql),exp))
case('join_code có 0','insert into courses(subject_code,class_code,name,semester,join_code,created_by) values(\'INT1006\',\'QT01\',\'L\',\'2026-2027-HK1\',\'ABCDEF0\',\''+adm+'\')','23514')
for nm,jc in [('join_code có O','ABCDEFO'),('join_code có 1','ABCDEF1'),('join_code có I','ABCDEFI'),('join_code có L','ABCDEFL'),('join_code 6 ký tự','ABCDEF'),('join_code 8 ký tự','ABCDEFGH'),('join_code chữ thường','abcdefg')]:
    case(nm,mkc('QT01',jc),'23514')
case('semester=2026',"insert into courses(subject_code,class_code,name,semester,join_code,created_by) values('INT1006','QT02','L','2026','ABCDEFG','"+adm+"')",'23514')
case('ARCHIVED thiếu archived_at',"insert into courses(subject_code,class_code,name,semester,join_code,created_by,status,join_enabled) values('INT1006','QT03','L','2026-2027-HK1','ABCDEFG','"+adm+"','ARCHIVED',false)",'23514')
case('archived_at khi ACTIVE',"insert into courses(subject_code,class_code,name,semester,join_code,created_by,archived_at) values('INT1006','QT04','L','2026-2027-HK1','ABCDEFG','"+adm+"',now())",'23514')
case('ARCHIVED join_enabled=true',"insert into courses(subject_code,class_code,name,semester,join_code,created_by,status,archived_at,join_enabled) values('INT1006','QT05','L','2026-2027-HK1','ABCDEFG','"+adm+"','ARCHIVED',now(),true)",'23514')
for nm,cp in [('capacity=0',0),('capacity=1001',1001)]:
    case(nm,"insert into courses(subject_code,class_code,name,semester,join_code,created_by,capacity) values('INT1006','QT06','L','2026-2027-HK1','ABCDEFG','"+adm+f"',{cp})",'23514')
case('allowed_email_domain=x',"insert into courses(subject_code,class_code,name,semester,join_code,created_by,allowed_email_domain) values('INT1006','QT07','L','2026-2027-HK1','ABCDEFG','"+adm+"','x')",'23514')
# trùng: cần dòng gốc trong cùng transaction
def dup(name,first,second,exp): cases.append((name,tryins(first+'; '+second),exp))
c0="insert into courses(id,subject_code,class_code,name,semester,join_code,created_by) values('00000000-0000-7000-8000-0000000000a1','INT1006','QT10','L','2026-2027-HK1','ABCDEFG','"+adm+"')"
dup('class_code trùng (hoa/thường)',c0,"insert into courses(subject_code,class_code,name,semester,join_code,created_by) values('INT1006','qt10','L','2026-2027-HK1','HJKMNPQ','"+adm+"')",'23505')
dup('join_code trùng',c0,"insert into courses(subject_code,class_code,name,semester,join_code,created_by) values('INT1006','QT11','L','2026-2027-HK1','ABCDEFG','"+adm+"')",'23505')
en=lambda uid,role='STUDENT',st='ACTIVE',extra='':f"insert into enrollments(course_id,user_id,role_in_course,status,joined_via{extra.split('|')[0] if extra else ''}) values('00000000-0000-7000-8000-0000000000a1','{uid}','{role}','{st}','ADMIN'{extra.split('|')[1] if extra else ''})"
dup('enrollments trùng (course,user)',c0+'; '+en(u1),en(u1),'23505')
dup('hai GV ACTIVE cùng lớp',c0+'; '+en(u1,'TEACHER'),en(u2,'TEACHER'),'23505')
dup('student_code_snapshot cho vai TA',c0,en(u1,'TA','ACTIVE',',student_code_snapshot|,\'AB123456\''),'23514')
dup('REMOVED thiếu removed_at',c0,en(u1,'STUDENT','REMOVED'),'23514')
dup('ends_at<=starts_at',c0,"insert into class_sessions(course_id,session_no,starts_at,ends_at) values('00000000-0000-7000-8000-0000000000a1',1,now(),now())",'23514')
cs="insert into class_sessions(course_id,session_no,starts_at,ends_at) values('00000000-0000-7000-8000-0000000000a1',%d,'2026-12-01 08:00+07','2026-12-01 10:00+07')"
dup('buổi trùng session_no',c0+'; '+cs%1,"insert into class_sessions(course_id,session_no,starts_at,ends_at) values('00000000-0000-7000-8000-0000000000a1',1,'2026-12-02 08:00+07','2026-12-02 10:00+07')",'23505')
dup('buổi trùng starts_at',c0+'; '+cs%1,cs%2,'23505')
good=tryins("insert into courses(subject_code,class_code,name,semester,join_code,created_by) values('INT1006','QTOK','L','2026-2027-HK1','ABCDEFG','"+adm+"'); insert into courses(subject_code,class_code,name,semester,join_code,created_by) values('INT1006','QTOK2','L','2026-2027-HK1','HJKMNPQ','"+adm+"')")
bad=[(n,r,e) for n,r,e in cases if r!=e]
after=pq("select count(*) from courses")
rec('TC-P207-04','P' if not bad and good=='OK' and base==after and len(cases)>=20 else 'F',f'{len(cases)} INSERT sai (join_code 0/O/1/I/L/6/8 ký tự/chữ thường, semester, ARCHIVED⇄archived_at, ARCHIVED+join_enabled, capacity 0/1001, domain x, trùng class_code/join_code, enrollments trùng, 2 GV ACTIVE, snapshot cho TA, REMOVED thiếu removed_at, ends<=starts, trùng buổi) mỗi ca trong transaction ROLLBACK: lệch {bad or "không"}; ca lành (join_code=ABCDEFG, semester=2026-2027-HK1) → {good}; courses trước/sau {base}/{after} [DB qc2i_auth]')
json.dump(cases,open('/tmp/smkT/p207-04.json','w'),ensure_ascii=False)
