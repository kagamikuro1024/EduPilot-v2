# TC-P207-06: EXPLAIN (FORMAT JSON) 6 truy vấn lớp trên 20.000 dòng giả mỗi bảng lớn (DB tạm smkt2_exp trong ep-r-postgres-1, xoá sau khi đo)
import os,subprocess,json
exec(open(os.path.join(os.path.dirname(os.path.abspath(__file__)),'smkt2lib.py')).read())
D='smkt2_exp'
def pg(db,q,i=False):
    r=subprocess.run(['docker','exec','-i','ep-r-postgres-1','psql','-U','edupilot','-d',db,'-At','-v','ON_ERROR_STOP=1','-c',q],capture_output=True,text=True)
    if r.returncode: raise Exception(r.stderr[:600])
    return r.stdout.strip()
pg('postgres',f'drop database if exists {D} with (force)'); pg('postgres',f'create database {D}')
try:
    out=subprocess.run(f'docker run --rm --network ep-r_default -e DATABASE_URL="postgres://edupilot:edupilot-dev@postgres:5432/{D}?sslmode=disable" edupilot-gateway:latest migrate up 2>&1 | tail -2',shell=True,capture_output=True,text=True).stdout
    print('migrate:',out.strip()[:200])
    # dữ liệu giả
    pg(D,"""insert into users(email,password_hash,full_name,role,status,email_verified_at)
select 'u'||g||'@x.test','x','U '||g,'STUDENT','ACTIVE',now() from generate_series(1,20000) g""")
    pg(D,"""insert into courses(subject_code,class_code,name,semester,join_code,created_by)
select 'INT1006','CC'||g,'Lop '||g,'2026-2027-HK1',
 translate(substr(md5(g::text),1,7),'0123456789abcdef','ABCDEFGHJKMNPQRS'), (select id from users limit 1) from generate_series(1,500) g""")
    n=int(pg(D,"select count(distinct join_code) from courses")); assert n==500,n
    pg(D,"""insert into enrollments(course_id,user_id,role_in_course,status,joined_via)
select c.id,u.id,'STUDENT',(array['ACTIVE','PENDING'])[1+(u.rn%2)]::enrollment_status,'CODE'
from (select id,row_number() over (order by id) rn from courses) cc join courses c on c.id=cc.id
join (select id,row_number() over (order by id) rn from users) u on u.rn between ((cc.rn-1)*37%20000)+1 and ((cc.rn-1)*37%20000)+40 limit 20000""")
    pg(D,"""insert into notifications(user_id,course_id,type,title,body,link,created_at)
select (select id from users offset (g%20000) limit 1),null,'X_TEST','t '||g,'b','/x',now()-(g||' seconds')::interval from generate_series(1,20000) g""")
    pg(D,"insert into documents(course_id,title) select id,'doc '||id from courses")
    pg(D,"""insert into content_chunks(document_id,course_ids,ord,text)
select d.id, case when g%5=0 then array[d.course_id,(select id from courses offset (g%500) limit 1)] else array[d.course_id] end, g, 'chunk '||g
from (select id,course_id,row_number() over () rn from documents) d, generate_series(1,40) g""")
    pg(D,"analyze")
    cnt={t:int(pg(D,f'select count(*) from {t}')) for t in('users','courses','enrollments','notifications','documents','content_chunks')}
    cid=pg(D,"select course_id from enrollments group by 1 order by count(*) desc limit 1"); uid=pg(D,"select user_id from enrollments where status='ACTIVE' limit 1"); jc=pg(D,"select join_code from courses limit 1")
    cnt['enrollments_courseX']=int(pg(D,f"select count(*) from enrollments where course_id='{cid}'"))
    Q={'GetMembership (course_id,user_id)':f"select role_in_course,status from enrollments where course_id='{cid}' and user_id='{uid}'",
    'ListMyCourses (user_id,status)':f"select c.id,e.role_in_course from enrollments e join courses c on c.id=e.course_id join users u on u.id=e.user_id where e.user_id='{uid}' and (e.status='ACTIVE' or (e.status='PENDING' and e.role_in_course='STUDENT')) order by e.created_at desc,e.id desc limit 21",
    'ListMembers (course_id,status,…)':f"select e.user_id,u.full_name from enrollments e join users u on u.id=e.user_id where e.course_id='{cid}' and e.status in ('ACTIVE','PENDING') order by e.status_changed_at desc, e.user_id limit 31",
    'CountCourseStudents (course_id)':f"select count(*) filter (where status='ACTIVE') from enrollments where course_id='{cid}' and role_in_course='STUDENT'",
    'GetCourseByJoinCode (join_code)':f"select * from courses where join_code='{jc}'",
    'ChunksForCourse (course_ids @> ARRAY)':f"select id,document_id,text,audience from content_chunks where course_ids @> array['{cid}'::uuid] order by id limit 50",
    'ListNotifications (user_id,created_at desc,id desc)':f"select id,type,title from notifications where user_id='{uid}' order by created_at desc,id desc limit 21",
    'CountUnreadNotifications (user_id)':f"select count(*) from notifications where user_id='{uid}' and read_at is null"}
    big={t for t,c in cnt.items() if c>1000}
    res=[];bad=[]
    def walk(n,acc):
        acc.append((n['Node Type'],n.get('Relation Name'),n.get('Index Name')))
        for c in n.get('Plans',[]): walk(c,acc)
    for k,q in Q.items():
        pg(D,'set enable_seqscan=on')
        plan=json.loads(pg(D,'explain (format json) '+q))[0]['Plan']; acc=[]; walk(plan,acc)
        seq=[(r) for t,r,i in acc if t=='Seq Scan' and r in big]
        idx=[f'{t}:{i}' for t,r,i in acc if i]
        res.append(f'{k} → {", ".join(idx) or "?"}'+(f' [SEQ SCAN {seq}]' if seq else ''))
        if seq or not idx: bad.append(k)
        if 'ChunksForCourse' in k and not any(t=='Bitmap Index Scan' and 'gin' in (i or '') for t,r,i in acc): bad.append(k+' không Bitmap Index Scan GIN')
    open(os.path.expanduser('~/Documents/TA_Agent_qc2/docs/regression/evidence-4/logs/smkT2/p20706-explain.txt'),'w').write('\n'.join(f'{k}\n{json.dumps(json.loads(pg(D,"explain (format json) "+q)),ensure_ascii=False)}\n' for k,q in Q.items()))
    rec('TC-P207-06','P' if not bad else 'F',f'EXPLAIN (FORMAT JSON) trên DB tạm smkt2_exp (migrate tới bản mới nhất, analyze), số dòng {cnt}: '+'; '.join(res)+'; không Seq Scan ở bảng > 1.000 dòng, chunk dùng Bitmap Index Scan content_chunks_course_ids_gin'+(f'; LỖI {bad}' if bad else '')+'; kế hoạch đầy đủ logs/smkT2/p20706-explain.txt; smkt2-p207-06.py')
finally:
    pg('postgres',f'drop database if exists {D} with (force)')
