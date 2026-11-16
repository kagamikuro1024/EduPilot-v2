"""QC2 smoke sprint 5 (SmkM) — dữ liệu lớn trong lớp R5C4: 5.000 SV (bản ghi QC, xoá sau bằng s20h_bulk.py clean), 1 bài MIXED với 1.000 lượt GRADED + 1.000 bản nộp code biến thể (PE07-36, PE08-38 cuộn 1.000 dòng, PE08-44 CSV 5.000/5.001 dòng).
Chạy: python3 s20h_bulk.py setup | clean"""
import os, sys, random, hashlib; os.environ['SUITE'] = 'BULK'
from r5lib import *
part = sys.argv[1]; C4 = db("select id from courses where class_code='R5C4'"); E4 = f'/courses/{C4}/exams'; Q4 = f'/courses/{C4}/questions'
def psql(sql_text): return subprocess.run(['docker', 'exec', '-i', 'ep-r-postgres-1', 'psql', '-U', 'edupilot', '-d', 'edupilot', '-v', 'ON_ERROR_STOP=1', '-q'], input=sql_text, capture_output=True, text=True)
OPS = ['+', '-', '*', '^', '|', '&', '%', '<<', '>>']
def prog(rng):
    n = rng.randint(22, 34); body = []
    for _ in range(n):
        k = rng.randint(0, 9); o1 = rng.choice(OPS); o2 = rng.choice(OPS); a = rng.randint(1, 9)
        body.append([f'for(int i=0;i<n;i++) s{o1}=v[i]{o2}{a};', f'if(s{o1}{a}>{o2 and a}) {{ s=s{o2}{a}; }} else {{ s=s{o1}1; }}', f'while(s>{a} && n>0) {{ s=s/2{o1}1; n--; }}', f'for(int i=0;i<n;i+=2) for(int j=0;j<n;j++) t=t{o1}v[j]{o2}i;',
                     f'switch(s%{a+1}) {{ case 0: t{o1}=1; break; case 1: t{o2}=2; break; default: t=s{o1}t; }}', f's=(s{o1}t){o2}{a};', f'if(t{o1}s) t{o2}=s; else s{o2}=t;', f'do {{ s{o1}=1; }} while(s<{a});', f'for(int i=n-1;i>=0;i--) {{ t{o1}=v[i]; if(t>s) break; }}', f'v[{a}]{o1}=s{o2}t;'][k])
    return '#include <stdio.h>\nlong long v[64];\nint main(){long long s=0,t=0;int n=10;\n' + '\n'.join(body) + '\nprintf("%lld\\n",s+t);return 0;}\n'
if part == 'setup':
    N = 5000; users_sql = f"""insert into users (email, full_name, role, status, student_code, email_verified_at, password_hash)
select 'qc2m5-bulk-' || lpad(g::text, 5, '0') || '@edupilot.local', 'QC Bulk ' || g, 'STUDENT', 'ACTIVE', 'QCB' || lpad(g::text, 5, '0'), now(), (select password_hash from users where email='r5s01@edupilot.local') from generate_series(1, {N}) g;
insert into enrollments (course_id, user_id, role_in_course, status, joined_via, student_code_snapshot)
select '{C4}', id, 'STUDENT', 'ACTIVE', 'ADMIN', student_code from users where email like 'qc2m5-bulk-%@edupilot.local' and not exists (select 1 from enrollments e where e.user_id = users.id and e.course_id = '{C4}');"""
    if db("select count(*) from users where email like 'qc2m5-bulk-%'") == '0':
        r = psql(users_sql); print('users', r.returncode, r.stderr[:300])
    code_q = mkcode(95, base=Q4); mcq = [mkq(950 + i, 'MCQ_SINGLE', 4, (1,), base=Q4) for i in range(2)]
    e, items = mkexam(open_s=10, win=420, dur=2, items=mcq + [code_q], base_e=E4, base_q=Q4, title='R5M lớn 1000 lượt'); time.sleep(13)
    iids = db(f"select id from exam_items where exam_id='{e}' order by position").split('\n'); opens = db(f"select opens_at from exams where id='{e}'")
    ins = []; rng = random.Random(20261116); subs = []
    bul = db("select id from users where email like 'qc2m5-bulk-%' order by email limit 1000").split('\n')
    for i, uid in enumerate(bul):
        sc = round(rng.uniform(0, 10), 2); bd = json.dumps([dict(item_id=iids[0], earned='1' if rng.random() < .5 else '0', max='1.00', points='1.00'), dict(item_id=iids[1], earned='1' if rng.random() < .5 else '0', max='1.00', points='1.00'), dict(item_id=iids[2], earned='8', max='8.00', points='8.00', code=dict(compile_ok=True, hidden=dict(passed=2, total=2), samples=[]))])
        aid = str(uuid.uuid4()); ins.append(f"insert into exam_attempts (id, course_id, exam_id, student_id, status, started_at, deadline_at, submitted_at, submit_reason, auto_score, breakdown, graded_at) values ('{aid}', '{C4}', '{e}', '{uid}', 'GRADED', now(), now() + interval '2 minutes', now(), 'MANUAL', {sc}, '{bd}', now());")
        src = prog(rng); ins.append(f"insert into code_submissions (course_id, exam_id, attempt_id, item_id, problem_id, student_id, kind, language, source, source_sha256, status, verdict, tests_version, compile_ok, passed_weight, total_weight) values ('{C4}', '{e}', '{aid}', '{iids[2]}', '{code_q}', '{uid}', 'SUBMIT', 'cpp17', $s${src}$s$, '{hashlib.sha256(src.encode()).hexdigest()}', 'DONE', 'AC', 1, true, 2, 2);")
    r = psql('\n'.join(ins)); print('attempts+subs', r.returncode, r.stderr[:300])
    json.dump(dict(exam=e, code_q=code_q, c4=C4, n_att=len(bul)), open('/tmp/qc2L/bulk.json', 'w')); print(e, db(f"select count(*) from exam_attempts where exam_id='{e}'"), db(f"select count(*) from code_submissions where exam_id='{e}'"))
if part == 'clean':
    bk = json.load(open('/tmp/qc2L/bulk.json')) if os.path.exists('/tmp/qc2L/bulk.json') else {}
    print('xoá bản ghi QC bulk: xem hàm clean trong tệp (chạy cuối phiên)')
