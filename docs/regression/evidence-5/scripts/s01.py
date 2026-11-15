import os; os.environ['SUITE']='PE-01'
from r5lib import *
import subprocess
R = '/Users/kuro/Documents/TA_Agent_qc2'
def sh(c, **k): return subprocess.run(c, shell=True, capture_output=True, text=True, cwd=k.get('cwd', R)).stdout.strip()
# TC01
n = db("select count(*) from information_schema.tables where table_name in ('question_bank','question_options','code_problems','code_testcases','exams','exam_items','exam_attempts','exam_answers','code_drafts','code_submissions','exam_events','similarity_reports','exam_appeals')")
ver = db("select max(version_id) from goose_db_version where is_applied")
chk('TC-PE01-01', n == '13', f'13 bảng PE có đủ; phiên bản migration hiện tại {ver} (gốc "5": sau sprint 5 có thêm migration — lệch dự kiến)')
# TC03 chỉ mục course_id đầu
rows = db("select tablename||'|'||indexname||'|'||indexdef from pg_indexes where tablename in ('question_bank','question_options','code_problems','code_testcases','exams','exam_items','exam_attempts','exam_answers','code_drafts','code_submissions','exam_events','similarity_reports','exam_appeals') and indexdef not like '%UNIQUE%' and indexname not like '%pkey'").split('\n')
withc = {}
for r in rows:
    t, i, d = r.split('|', 2); cols = d.split('(', 1)[1]
    withc.setdefault(t, []).append(cols.startswith('course_id'))
hascid = {t: db(f"select count(*) from information_schema.columns where table_name='{t}' and column_name='course_id'") == '1' for t in withc}
bad = [t for t, v in withc.items() if hascid[t] and not any(v)]
chk('TC-PE01-03', not bad, f'bảng có course_id đều ≥1 chỉ mục bắt đầu course_id; ngoại lệ ghi: {bad}; bảng con không course_id: {[t for t in withc if not hascid[t]]}')
# TC04 migrations cũ
mods = sh("git log --oneline -- backend-go/db/migrations/00001* backend-go/db/migrations/00002* backend-go/db/migrations/00003* backend-go/db/migrations/00004* | head -20")
alter = sh("grep -ciE 'alter table (users|courses|enrollments|auth_|mail_|llm_|jobs|outbox|audit_log)' backend-go/db/migrations/00006_weekly_exam*.sql backend-go/db/migrations/00005*.sql 2>/dev/null | tr '\\n' ' '")
rec('TC-PE01-04', 'K', f'LỖI THỜI: migration PE nay là 00006_weekly_exam (không còn 00005); 4 migration đầu có {len(mods.splitlines())} commit sau sprint 4; ALTER bảng cũ trong 00005/00006: {alter}')
# TC02 p501-schema
out = sh("PSQL='docker exec -i ep-r-postgres-1 psql -U edupilot -d edupilot -tAF|' python3 docs/regression/evidence-5/scripts/p501-schema.py docs/specs/FEAT-weekly-exam/SRS.md 2>&1 | tail -15")
open(R + '/docs/regression/evidence-5/logs/p501-schema.log', 'w').write(out); print(out[-900:])
# TC12/14: ma trận quyền
eid, items = mkexam(open_s=3600, win=3600, dur=30, schedule=False, title='R5 01 ma trận')
qid = items[0]
q2id = mkq(5, base=Q2); e2, _ = mkexam(open_s=3600, win=3600, dur=30, schedule=False, base_e=E2, base_q=Q2, title='R5 C2 bài')
princ = {'TEACHER': 'r5t', 'TA': 'r5ta', 'ADMIN': 'admin', 'STU': 'r5s01', 'PENDING': 'r5s15', 'REMOVED': 'r5s16', 'OUTSIDE': 'r5x'}
routes = [('staff', 'GET', Q), ('staff', 'GET', f'{Q}/{qid}'), ('staff', 'POST', Q), ('member', 'GET', E), ('staff', 'POST', E), ('member', 'GET', f'{E}/{eid}'), ('teacher', 'PUT', f'{E}/{eid}/items'), ('teacher', 'POST', f'{E}/{eid}/schedule'), ('staff', 'GET', f'{E}/{eid}/results'), ('teacher', 'POST', f'{E}/{eid}/regrade'), ('staff', 'GET', f'{E}/{eid}/events'), ('student', 'POST', f'{E}/{eid}/attempts'), ('student', 'GET', f'{E}/{eid}/attempts/mine')]
bodies = {'POST ' + Q: dict(type='TRUE_FALSE', title='m', topic='m', stem='s', value=True), 'POST ' + E: dict(title='x', opens_at=iso(days=3), closes_at=iso(days=3, hours=1), duration_minutes=30), f'PUT {E}/{eid}/items': dict(items=[], version=1)}
bad = []; cells = 0; mat = []
for kind, m, p in routes:
    for pn, u in princ.items():
        tk = login(u); s, j, _ = call(m, p, tk, bodies.get(m + ' ' + p, {}), idem() if m == 'POST' else None); cells += 1
        if pn == 'ADMIN': exp = (403, None)
        elif pn in ('PENDING', 'REMOVED', 'OUTSIDE'): exp = (403, 'course')
        elif kind == 'member': exp = ('ok', None)
        elif kind == 'staff': exp = ('ok', None) if pn in ('TEACHER', 'TA') else (403, 'role')
        elif kind == 'teacher': exp = ('ok', None) if pn == 'TEACHER' else (403, 'role')
        else: exp = ('ok', None) if pn == 'STU' else (403, 'role')
        okc = (s < 300 or s in (404, 409, 422)) if exp[0] == 'ok' else (s == 403 and (exp[1] is None or reason(j) == exp[1]))
        if exp[0] == 'ok' and s in (404,): okc = False
        mat.append(f'{m} {p.split("/courses/")[1][37:] or "/"}|{pn}|{s}|{reason(j)}')
        if not okc: bad.append(f'{m} {p[-30:]} {pn} -> {s}/{reason(j)} (exp {exp})')
open(R + '/docs/regression/evidence-5/logs/pe01-matrix.txt', 'w').write('\n'.join(mat) + '\n')
chk('TC-PE01-12', not bad, f'{cells} ô (13 route × 7 danh tính); lệch: {bad[:5]}')
r13a = call('GET', '/courses/abc/questions', T)[0]; r13b = call('GET', f'/courses/{uuid.uuid4()}/questions', T)[0]
chk('TC-PE01-13', (r13a, r13b) == (404, 403), f'id không phải uuid {r13a}, uuid không tồn tại {r13b}')
chk('TC-PE01-14', (call('GET', Q, AD)[0], call('GET', Q, login('r5s02'))[0], call('GET', E, AD)[0], call('GET', '/me/exam-lock', AD)[0]) == (403, 403, 403, 200), 'ADMIN/SV(ngoài C2→ dùng C1 SV: 403 role) /me/exam-lock ADMIN 200')
# TC16 cách ly lớp: GV C1 dùng id C2 trong path C1
iso16 = []
for m, p, b in [('GET', f'{Q}/{q2id}', None), ('PUT', f'{Q}/{q2id}', dict(type='TRUE_FALSE', title='x', topic='x', stem='s', value=True, version=1)), ('GET', f'{E}/{e2}', None), ('PUT', f'{E}/{e2}', dict(title='x', opens_at=iso(days=2), closes_at=iso(days=2, hours=1), duration_minutes=20, version=1)), ('POST', f'{E}/{e2}/schedule', {}), ('GET', f'{E}/{e2}/results', None), ('DELETE', f'{E}/{e2}', None)]:
    s, j, t = call(m, p, T, b, idem() if m == 'POST' else None); iso16.append((m, s))
chk('TC-PE01-16', all(s == 404 for _, s in iso16), str(iso16))
eid17, _ = mkexam(open_s=3600, win=3600, dur=30, schedule=False, title='R5 17')
g = call('GET', f'{E}/{eid17}', T)[1]
s, j, _ = call('PUT', f'{E}/{eid17}/items', T, dict(items=[dict(question_id=q2id, points='1')], version=g['version']))
chk('TC-PE01-17', s == 422 and 'QUESTION_NOT_IN_COURSE' in codes(j), f'{s} {codes(j)}')
s18 = [call(m, p, login('r5s01'), b)[0] for m, p, b in [('GET', Q2, None), ('GET', E2, None), ('GET', f'{E2}/{e2}', None), ('POST', f'{E2}/{e2}/attempts', {})]]
chk('TC-PE01-18', s18 == [403] * 4, str(s18))
# canary C2
cq = mkq(7, base=Q2, title='QC-C2-CANARY')
l1 = call('GET', Q, T)[2] + call('GET', E, T)[2] + call('GET', f'{E}/{eid}/results.csv', T)[2]
chk('TC-PE01-19', 'QC-C2-CANARY' not in l1, 'canary lớp 2 không xuất hiện ở danh sách câu/bài/CSV lớp 1')
# TC28 409
q = mkq(11, ok=False); g = call('GET', f'{Q}/{q}', T)[1]
s1, j1, _ = call('PUT', f'{Q}/{q}', T, dict(type='TRUE_FALSE', title='đổi', topic='R5', stem='s', value=True, version=g['version'] + 5)); s2, j2, _ = call('PUT', f'{Q}/{q}', T, dict(type='TRUE_FALSE', title='đổi2', topic='R5', stem='s', value=True)); 
chk('TC-PE01-28', s1 == 409 and 'current_version' in json.dumps(j1.get('details')) and s2 == 422, f'sai version {s1} details={list((j1.get("details") or {}).keys()) if isinstance(j1.get("details"), dict) else j1.get("details")}; thiếu version {s2} {codes(j2)}')
import concurrent.futures as cf
with cf.ThreadPoolExecutor(2) as ex: rr = list(ex.map(lambda t: call('PUT', f'{Q}/{q}', T, dict(type='TRUE_FALSE', title='song song ' + t, topic='R5', stem='s', value=True, version=g['version']))[0], ['a', 'b']))
chk('TC-PE01-28b', sorted(rr) == [200, 409], f'hai PUT song song: {rr}')
sv = call('PUT', f'{Q}/{q}', login('r5s01'), dict(type='TRUE_FALSE', title='x', topic='R5', stem='s', value=True, version=1))[0]
rec('TC-PE01-29', sv == 403 and 'answer_key' in json.dumps(j1.get('details')), f'SV PUT {sv}; 409 cho GV có current (answer_key kèm: {"answer_key" in json.dumps(j1.get("details"))})')
# TC30 cursor
for i in range(32): mkq(100 + i, 'TRUE_FALSE', ok=False)
l = call('GET', Q, T)[1]; l2 = call('GET', Q + '?limit=101', T); l3 = call('GET', Q + '?limit=100', T)[1]; l0 = call('GET', Q + '?limit=0', T); lm = call('GET', Q + '?limit=-1', T); lc = call('GET', Q + '?cursor=rac', T)
seen = []; cur = None
for _ in range(10):
    jj = call('GET', Q + '?limit=10' + (f'&cursor={cur}' if cur else ''), T)[1]; seen += [x['id'] for x in jj['items']]; cur = jj.get('next_cursor')
    if not cur: break
chk('TC-PE01-30', len(l['items']) == 30 and len(l3['items']) <= 100 and l2[0] in (200, 422) and lc[0] == 422 and 'INVALID_CURSOR' in codes(lc[1]) and len(seen) == len(set(seen)) and len(seen) >= 33, f'default {len(l["items"])}; limit=101 {l2[0]}/{len(l2[1].get("items", []))}; limit=0 {l0[0]}; -1 {lm[0]}; cursor rác {lc[0]} {codes(lc[1])}; lật hết {len(seen)} dòng không trùng')
# TC33 audit canary
ac = db("select count(*) from audit_log where (coalesce(before::text,'')||coalesce(after::text,'')) like '%QCR5CANARY%'")
na = db(f"select count(*) from audit_log where course_id='{C1}' and (action ilike 'exam.%' or action ilike 'question.%' or action ilike 'testcases.%')")
chk('TC-PE01-33', ac == '0' and int(na) > 0, f'canary trong audit_log: {ac}; dòng audit exam/question: {na}')
# TC35 lớp lưu trữ
db("insert into courses(subject_code,class_code,name,semester,join_code,created_by) select 'INT1006','R5C3','R5 lớp lưu trữ','2026-2027-HK1','RFVC4K7',id from users where email='r5t@edupilot.local' on conflict do nothing")
db("insert into enrollments(course_id,user_id,role_in_course,status,joined_via,student_code_snapshot) select c.id,u.id,case when u.email='r5t@edupilot.local' then 'TEACHER'::enrollment_role when u.email='r5ta@edupilot.local' then 'TA' else 'STUDENT' end,'ACTIVE','ADMIN',u.student_code from courses c,users u where c.class_code='R5C3' and u.email in ('r5t@edupilot.local','r5ta@edupilot.local','r5s11@edupilot.local') on conflict do nothing")
C3 = db("select id from courses where class_code='R5C3'"); Q3 = f'/courses/{C3}/questions'; E3 = f'/courses/{C3}/exams'
e3, _ = mkexam(open_s=10, win=1200, dur=15, base_e=E3, base_q=Q3, title='R5 lưu trữ'); time.sleep(12)
aid3, j3, H3, s3, _ = start(e3, 'r5s11', E3)
db(f"update courses set status='ARCHIVED', archived_at=now(), join_enabled=false where id='{C3}'")
w1 = call('POST', Q3, T, dict(type='TRUE_FALSE', title='x', topic='x', stem='s', value=True))[0]; w2 = call('POST', E3, T, dict(title='x', opens_at=iso(days=2), closes_at=iso(days=2, hours=1), duration_minutes=20), idem())[0]
r1 = call('GET', E3, T)[0]; r2 = call('GET', Q3, T)[0]; sub = submit_exam(e3, aid3, 'r5s11', H3, E3)[0]
chk('TC-PE01-35', w1 == 409 and w2 == 409 and r1 == 200 and r2 == 200 and sub == 200, f'ghi {w1}/{w2}; đọc {r1}/{r2}; nộp lượt đang chạy {sub}')
# TC37 hợp đồng
ops = sh("grep -cE '^  /api/v1/courses/\\{id\\}/(questions|exams)' backend-go/api/openapi.yaml")
errs = [call('GET', f'{E}/{uuid.uuid4()}', T)[2], call('POST', E, T, {'title': ''}, idem())[2], call('PUT', f'{Q}/{q}', T, dict(version=1))[2]]
sql = [e for e in errs if re.search(r'select |from |relation|pg_|sqlstate', json.loads(e).get('message', ''), re.I)]
chk('TC-PE01-37', not sql, f'{ops} path questions/exams trong openapi; message không lộ SQL/tên bảng ({len(errs)} mã lỗi thử)')
# TC40
n40 = sh("grep -rnE '(SELECT|INSERT|UPDATE|DELETE) ' backend-go/internal/exam backend-go/internal/quiz --include=*.go | grep -v _test.go | grep -vE '^[^:]+:[0-9]+:\\s*//' | wc -l")
chk('TC-PE01-40', n40 == '0', f'SQL ngoài store/: {n40}')
# TC27
n27 = sh("cd backend-go && go list -deps ./internal/quiz ./internal/judge | grep -c internal/llm"); imps = sh("cd backend-go && grep -ln 'internal/llm' internal/exam/*.go | grep -v _test")
chk('TC-PE01-27', n27 == '0' and imps.strip().endswith('suggest.go'), f'deps llm={n27}; tệp exam import llm: {imps}')
n26 = sh("grep -rn 'float64\\|float32' backend-go/internal/quiz backend-go/internal/exam --include=*.go | grep -v _test.go | grep -v 'similarity/' | wc -l")
rec('TC-PE01-26', 'K', f'phần grep float: {n26} dòng (gốc 0); phần TestScoringDecimal chạy ở bước Go; tính tay 10 dòng chưa làm (xem TC-PE01-23..25 qua F19)')
