"""TC-PE08-15 (regrade attempt idempotent), TC-PE03-33 (sửa test sau đóng), TC-PE08-52 (chấm lại sau sửa test, giảm điểm có lý do) trên E30 (bài đã PUBLISHED)."""
import os; os.environ['SUITE'] = 'PE-08E3'
from r5lib import *
from decimal import Decimal as D
import rl
def R(tc, ok, note): rl.R(tc, 'P' if ok is True else 'F' if ok is False else ok, note)
def S(m, p, tk=None, b=None, h=None, **k): return call(m, p, tk or T, b, h=h, **k)
st = json.load(open('/tmp/qc2L/e30.json')); eid = st['eid']; plan = st['plan']; EX = f'{E}/{eid}'; users = list(plan)
u0 = users[0]; a0 = plan[u0]['aid']; sc0 = db(f"select auto_score from exam_attempts where id='{a0}'"); n0 = db(f"select count(*) from code_submissions where attempt_id='{a0}'")
r1 = S('POST', f'{EX}/regrade', b=dict(scope='attempt', attempt_id=a0, reason='kiểm idempotent'), h=idem()); time.sleep(10); r2 = S('POST', f'{EX}/regrade', b=dict(scope='attempt', attempt_id=a0, reason='kiểm idempotent 2'), h=idem()); time.sleep(10)
sc1 = db(f"select auto_score from exam_attempts where id='{a0}'"); n1 = db(f"select count(*) from code_submissions where attempt_id='{a0}'")
R('TC-PE08-15', r1[0] == 202 and r2[0] == 202 and sc0 == sc1 and n0 == n1, f'regrade scope=attempt ×2: {r1[0]}/{r2[0]}; auto_score {sc0} → {sc1}; số bản nộp {n0} → {n1} (không dòng đôi)')
qid = db(f"select i.question_id from exam_items i join question_bank q on q.id=i.question_id where i.exam_id='{eid}' and q.type='CODE'"); P = f'{Q}/{qid}'
tl = S('GET', P + '/testcases?limit=100')[1]['items']; hid = [t for t in tl if not t['is_sample']][0]; ver = S('GET', P)[1]['version']
before = {u: db(f"select auto_score from exam_attempts where id='{plan[u]['aid']}'") for u in users}
ta = S('PUT', f'{P}/testcases/{hid["id"]}', TA, dict(name=hid['name'], input=hid.get('input') or '5 5\n', expected='999\n', is_sample=False, weight=hid['weight']))
tv0 = db(f"select tests_version from code_problems where question_id='{qid}'")
gv = S('PUT', f'{P}/testcases/{hid["id"]}', b=dict(name=hid['name'], input=hid.get('input') or '5 5\n', expected='999\n', is_sample=False, weight=hid['weight'])); tv1 = db(f"select tests_version from code_problems where question_id='{qid}'")
same = db(f"select count(*) from exam_attempts where exam_id='{eid}' and auto_score is not null"); unchanged = all(db(f"select auto_score from exam_attempts where id='{plan[u]['aid']}'") == before[u] for u in users)
op = mkexam(open_s=14, win=300, dur=5, items=[qid], title='R5L 03-33 dùng câu', nq=1)[0]; time.sleep(16)
lock = S('PUT', f'{P}/testcases/{hid["id"]}', b=dict(name=hid['name'], input=hid.get('input') or '5 5\n', expected='999\n', is_sample=False, weight=hid['weight']))
R('TC-PE03-33', gv[0] == 200 and ta[0] == 403 and tv1 == str(int(tv0) + 1) and unchanged and lock[0] == 409, f'sửa test ẩn của câu đã dùng ở bài PUBLISHED: GV {gv[0]} (tests_version {tv0} → {tv1}); TA {ta[0]}; điểm 30 lượt chưa đổi tới khi regrade: {unchanged}; khi câu đang ở bài OPEN: sửa test → {lock[0]} {codes(lock[1])}')
rg = S('POST', f'{EX}/regrade', b=dict(scope='all', reason='Sửa test sai (QC): chấm lại theo bộ test mới'), h=idem()); time.sleep(5)
for _ in range(60):
    if db(f"select regrading from exams where id='{eid}'") == 'f': break
    time.sleep(3)
time.sleep(15); after = {u: db(f"select auto_score from exam_attempts where id='{plan[u]['aid']}'") for u in users}
chg = [(u, before[u], after[u]) for u in users if after[u] != before[u]]; down = [c for c in chg if D(c[2]) < D(c[1])]; nn = db(f"select count(*) from notifications where link like '%{eid}%' and type ilike '%REGRAD%'"); msg = db(f"select body from notifications where link like '%{eid}%' and type ilike '%REGRAD%' order by created_at desc limit 1")
R('TC-PE08-52', rg[0] == 202 and len(down) > 0 and db(f"select status from exams where id='{eid}'") == 'PUBLISHED', f'GV sửa expected test ẩn → regrade scope=all có lý do: {rg[0]}; chấm lại bản nộp cuối theo tests_version {tv1}: {len(chg)} lượt đổi điểm, {len(down)} lượt GIẢM (vd {down[:2]}); bài vẫn PUBLISHED (không huỷ công bố); thông báo EXAM_REGRADED {nn} dòng: "{(msg or "")[:120]}"')
