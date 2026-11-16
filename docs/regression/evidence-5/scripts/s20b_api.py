"""QC2 smoke sprint 5 (SmkM) — API: TC-PE07-21 (quét route bằng SV), PE08-34 (vắng), PE08-44 (xuất CSV: quyền, 5.000 dòng, log). Chạy: python3 s20b_api.py"""
import os, sys, hashlib; os.environ['SUITE'] = 'PE-08M'
from r5lib import *
import rl
def R(tc, ok, note): rl.R(tc, 'P' if ok is True else 'F' if ok is False else ok, note)
def S(m, p, tk=None, b=None, h=None, **k): return call(m, p, tk or T, b, h=h, **k)
st = json.load(open('/tmp/qc2L/e30.json')); eid = st['eid']; plan = st['plan']; EX = f'{E}/{eid}'
# ---- PE07-21: mọi thao tác của exam.Routes() gọi bằng SV có lượt (r5s01 trong E30, có sự kiện liêm chính)
src = open('/Users/kuro/Documents/TA_Agent_qc2/backend-go/internal/exam/routes.go', encoding='utf-8').read()
rt = [(int(n), m, p, md) for n, m, p, md in re.findall(r'\{(\d+), "(\w+)", ([^,]+), (\w+), Idem', src)] + [(43, 'GET', '/me/exam-lock', 'none')]
sv = 'r5s01'; tsv = login(sv); aid = plan[sv]['aid']; qid = db(f"select question_id from exam_items where exam_id='{eid}' limit 1"); iid = db(f"select id from exam_items where exam_id='{eid}' limit 1"); tid = '01a1e136-0000-7000-8000-000000000000'
ev_in_db = db(f"select count(*) from exam_events where attempt_id='{aid}'")
def pathof(p):
    p = p.replace('q + ', '').replace('qi + ', '').strip()
    return p
consts = dict(q='/questions', qi='/questions/{qid}', e='/exams', ei='/exams/{eid}', at='/exams/{eid}/attempts/{aid}')
def resolve(p):
    parts = [x.strip() for x in p.split('+')]; s = ''.join(consts.get(x, x.strip('"')) for x in parts)
    return s.format(qid=qid, eid=eid, aid=aid, tid=tid, itemId=iid, runId=tid, sid=tid, id=tid)
sc_rows = []; leak = []; STAFF_ONLY_MUT = 0
for no, m, p, md in rt:
    path = resolve(p) if no != 43 else '/me/exam-lock'; url = path if no == 43 else f'/courses/{C1}' + path
    if m == 'GET':
        s, j, t = call('GET', url, tsv)
        bad = [k for k in ('exam_events', 'TAB_HIDDEN', 'TAB_VISIBLE', 'PASTE', 'TAB_TAKEOVER', 'integrity') if k in t]
        if bad: leak.append((no, s, bad))
        sc_rows.append((no, m, s))
    elif md in ('staff', 'teacher'):
        s, j, t = call(m, url, tsv, {} if m != 'DELETE' else None, h=idem()); sc_rows.append((no, m, s))
        if s not in (403, 404): leak.append((no, s, 'mutating staff route reachable'))
    else: sc_rows.append((no, m, 'bỏ (route của chính SV, không trả sự kiện: xem test)'))
stu_statuses = {s for no, m, s in sc_rows if isinstance(s, int)}
n_get = sum(1 for no, m, s in sc_rows if m == 'GET'); n_mut = sum(1 for no, m, s in sc_rows if m != 'GET' and isinstance(s, int))
R('TC-PE07-21', len(rt) == 57 and not leak and ev_in_db != '0' and next(s for n, m, s in sc_rows if n == 51) in (403, 404), f'quét {len(rt)} thao tác của exam.Routes() bằng SV {sv} (lượt có {ev_in_db} sự kiện liêm chính trong DB): {n_get} GET + {n_mut} thao tác sửa của Staff/Teacher; thân phản hồi chứa exam_events/TAB_*/PASTE/integrity: {leak or "0 route"}; #51 GET events → {next(s for n, m, s in sc_rows if n == 51)}; #50/#44/#45 → {[s for n, m, s in sc_rows if n in (44, 45, 50)]}; route route-của-SV POST (29,31,32,33,35,38,39,40,42) không chạy (đổi trạng thái), không có đường trả sự kiện cho SV trong mã (chỉ events.go #51 Teacher)')
# ---- PE08-34: SV vắng
users = list(plan); absent = [x for x in db(f"select u.email from enrollments e join users u on u.id=e.user_id where e.course_id='{C1}' and e.status='ACTIVE' and u.email like 'r5s%' and not exists (select 1 from exam_attempts a where a.exam_id='{eid}' and a.student_id=u.id)").splitlines()]
res = S('GET', EX + '/results?limit=100')[1]; rows_abs = [x for x in res['items'] if x.get('status') == 'ABSENT' or (x.get('progress') == 'absent')]
sample = res['items'][0]; keys_ = sorted(sample.keys())
s, _, csvt = call('GET', f'/courses/{C1}/exams/{eid}/results.csv', T); lines = csvt.lstrip('\ufeff').strip().split('\n'); hdr = lines[0].split(';'); ia = hdr.index('trang_thai'); idg = hdr.index('diem_tu_dong')
abs_rows = [l.split(';') for l in lines[1:] if l.split(';')[ia] == 'ABSENT']; blank = all(r[idg] == '' for r in abs_rows)
ap = S('POST', f'{EX}/attempts/{plan[users[0]]["aid"]}/appeal', login(absent[0].split('@')[0]), dict(reason='tôi vắng'), idem()); ap2 = S('POST', f'{EX}/attempts/00000000-0000-7000-8000-000000000000/appeal', login(absent[0].split('@')[0]), dict(reason='x'), idem())
res_abs = S('GET', EX + '/result' , login(absent[0].split('@')[0]))
R('TC-PE08-34', len(absent) >= 1 and len(rows_abs) == len(absent) and all(x.get('score') in (None, '') for x in rows_abs) and len(abs_rows) == len(absent) and blank and ap[0] in (403, 404, 409) and ap2[0] in (403, 404, 409), f'E30: {len(absent)} SV ACTIVE không có lượt → bảng điểm Staff {len(rows_abs)} hàng progress=absent điểm trống; CSV {len(abs_rows)} dòng "ABSENT" ô diem_tu_dong trống={blank}; SV vắng POST appeal lượt của người khác {ap[0]} {codes(ap[1])}, lượt giả {ap2[0]} {codes(ap2[1])} (không phúc khảo được); không 0 điểm')
# ---- PE08-44 (phần API): quyền + log
csv_by = {k: call('GET', f'/courses/{C1}/exams/{eid}/results.csv', t)[0] for k, t in (('GV', T), ('TA', TA), ('SV', tsv), ('ADMIN', AD))}
names = [l.split(';')[1] for l in lines[1:6]]
lg = subprocess.run("docker logs --since 10m ep-r-gateway-1 2>&1", shell=True, capture_output=True, text=True).stdout
named = [n for n in names if n and n in lg]; csv_hits = [l for l in lg.splitlines() if 'results.csv' in l][:2]
aud = db(f"select count(*) from audit_log where created_at > now() - interval '10 minutes' and (action ilike '%csv%' or action ilike '%export%')")
rec = dict(csv_by=csv_by, named=named, aud=aud, hit=len(csv_hits), cols=len(hdr), rows=len(lines) - 1)
json.dump(rec, open('/tmp/qc2-c5b/pe08_44.json', 'w'))
print(json.dumps(rec, ensure_ascii=False))
