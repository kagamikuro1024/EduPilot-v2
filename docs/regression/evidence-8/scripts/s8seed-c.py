exec(open('s8seed.py').read())
env = dict(os.environ, API_URL=B, MAILPIT_URL='http://127.0.0.1:18233', SEED_DEFAULT_PASSWORD=PW, SEED_ADMIN_CMD='docker exec -i qc2s-gw /gateway admin create')
uid = lambda e: psql(f"select id from users where email='{e}@edupilot.local'")
tt = tok('teacher'); B_ = tok('sv.kha')
# 37
jv = psql("select joined_via||'|'||n from (select joined_via, count(*) n from enrollments e join courses c on c.id=e.course_id where c.class_code='761987' and e.role_in_course='STUDENT' and e.status='ACTIVE' group by joined_via) t order by 1")
ent = psql("select count(distinct e.student_id) from grade_entries e where e.source='IMPORT'"); jvi = psql("select count(*) from grade_entries ge join enrollments en on en.user_id=ge.student_id and en.course_id=ge.course_id where ge.source='IMPORT' and en.joined_via in ('ROSTER','ADMIN')")
print(jv, ent, jvi)
C('TC-P6-SEED-37', ent == '30' and jvi == '60', f'lớp 1 ACTIVE theo joined_via: {jv.replace(chr(10), " ; ")}; sinh viên có điểm IMPORT = {ent} (30); ô điểm IMPORT của người có joined_via ROSTER/ADMIN = {jvi}/60 (nhập XLSX khớp theo student_code_snapshot của hàng ROSTER/ADMIN)')
# 32 sweep + canary
CAN = 'CANARY-GRADE-7Q'; bid = uid('sv.kha'); tid = uid('teacher')
kinds = psql("select string_agg(distinct kind::text, ',') from student_observations") or 'NOTE'
obs_id = psql(f"insert into student_observations(course_id, student_id, author_id, kind, body) select '{C1}', '{bid}', '{tid}', coalesce((select kind from student_observations limit 1), 'NOTE'), '{CAN} ghi chu quan sat' returning id")
import yaml
spec = yaml.safe_load(open(f'{REPO}/backend-go/api/openapi.yaml'))
paths = [p for p, v in spec['paths'].items() if 'get' in v and re.sub(r'\{id\}', '', p).count('{') == 0 and (p.startswith('/api/v1/me') or p.startswith('/api/v1/courses/{id}/'))]
allowed = []; res32 = {}; hits = []; keysbad = []; five = []
FORB = ['hp_raw', 'breakdown', 'observations', 'study_time', 'risk']
for p in paths:
    url = p.replace('/api/v1', '').replace('{id}', C1); s_, j_ = req('GET', url, B_); res32[p] = s_
    if s_ >= 500: five.append((p, s_))
    blob = json.dumps(j_, ensure_ascii=False)
    if CAN in blob or '7.77' in blob or '7,77' in blob: hits.append(p)
    if s_ == 200 and any(f'"{k}"' in blob for k in FORB): keysbad.append((p, [k for k in FORB if f'"{k}"' in blob])) if p != '/api/v1/me/settings' else allowed.append((p, 'study_time = trạng thái đồng ý theo dõi của chính SV (acked, khoá theo lớp), không phải dữ liệu thời gian học'))
ok200 = [p for p, c in res32.items() if c == 200]
psql(f"delete from student_observations where id='{obs_id}'")
C('TC-P6-SEED-32', not hits and not five and not keysbad, f'ghi chú quan sát của B mang «{CAN}» (SQL fixture, đã xoá sau quét); quét {len(paths)} route GET của sinh viên (me/* và courses/{{id}}/*) bằng token B: {len(ok200)} trả 200, {sum(1 for c in res32.values() if c == 403)} trả 403, 5xx {len(five)}; chuỗi CANARY hoặc 7,77 xuất hiện ở: {hits}; khoá cấm (hp_raw, breakdown, observations, study_time, risk) ở phản hồi 200: {keysbad}; ngoại lệ có chủ đích: {allowed} (cùng cách kết luận với TC-P6-09-09 sprint 8)')
