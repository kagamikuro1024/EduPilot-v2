"""QC2 smoke sprint 5 (SmkL) — chấm điểm chi tiết trên ep-r (không đổi cấu hình toàn cục): TC-PE08-10/11/12/13/14/15/50 (+ PE01-25).
Mỗi bài dài 1 phút, SV làm trong cửa sổ; điểm tính tay bằng Fraction độc lập với hệ thống."""
import os, sys; os.environ['SUITE'] = 'PE-08S'
from r5lib import *
from fractions import Fraction as F
from decimal import Decimal as D, ROUND_HALF_UP
import random, rl
def R(tc, ok, note): rl.R(tc, 'P' if ok is True else 'F' if ok is False else ok, note)
def S(m, p, tk=None, b=None, h=None, **k): return call(m, p, tk or T, b, h=h, **k)
ACT = [x.split('@')[0] for x in db(f"select u.email from enrollments e join users u on u.id=e.user_id where e.course_id='{C1}' and u.email like 'r5s%' and e.status='ACTIVE' order by 1").splitlines() if int(x[3:5]) not in (13, 14)]
def r2(x, step): s = F(str(step)); return (x / s + F(1, 2)).__floor__() * s   # làm tròn nửa lên theo bước
def q2(x): return str((D(x.numerator) / D(x.denominator)).quantize(D('0.01'), ROUND_HALF_UP))
def mkx(title, items_pts, **kw):
    eid, _ = mkexam(open_s=14, win=95, dur=1, items=[i for i, _ in items_pts], schedule=False, title=title, **kw)
    g = S('GET', f'{E}/{eid}')[1]; s, jj, _ = S('PUT', f'{E}/{eid}/items', b=dict(items=[dict(question_id=q, points=p) for q, p in items_pts], version=g['version'])); assert s == 200, jj
    g = S('GET', f'{E}/{eid}')[1]; s, jj, _ = S('PUT', f'{E}/{eid}', b=dict(title=title, opens_at=iso(seconds=14), closes_at=iso(seconds=14 + 95), duration_minutes=1, version=g['version'], **kw)); assert s == 200, jj
    s, jj, _ = S('POST', f'{E}/{eid}/schedule', b={}); assert s == 200, jj
    return eid
def idx_items(eid): return {r.split('|')[0]: r.split('|')[1] for r in db(f"select i.id, q.title from exam_items i join question_bank q on q.id=i.question_id where i.exam_id='{eid}'").splitlines()}
def answer(eid, user, picks, code=None):
    """picks: {title_suffix: list option index (0-based của 'Đáp án k') | bool}"""
    aid, j, H, s, _ = start(eid, user); assert aid, (user, s, _); t = login(user); idx = idx_items(eid); ans = []; cit = None
    for it in j['items']:
        ttl = idx[it['item_id']]; k = ttl.rsplit('-', 1)[-1]
        if it['type'] == 'CODE': cit = it['item_id']; continue
        if k in picks:
            v = picks[k]
            if isinstance(v, bool): ans.append(dict(item_id=it['item_id'], answer=dict(value=v)))
            elif v: ans.append(dict(item_id=it['item_id'], answer=dict(option_ids=[[o['id'] for o in it['options'] if o['body'].startswith('Đáp án %d ' % n)][0] for n in v])))
    if ans: S('PUT', f'{E}/{eid}/attempts/{aid}/answers', t, dict(items=ans), h=H)
    if code and cit: S('POST', f'{E}/{eid}/attempts/{aid}/code/{cit}/submit', t, dict(language='cpp17', source=code), {**H, **idem()})
    return aid
def wait_graded(eid, n, to=240):
    for _ in range(to // 3):
        if int(db(f"select count(*) from exam_attempts where exam_id='{eid}' and status='GRADED'")) >= n: return True
        time.sleep(3)
    return False
def score(eid, aid): return db(f"select auto_score from exam_attempts where id='{aid}'")
part = sys.argv[1] if len(sys.argv) > 1 else 'all'
# ---------- PE08-10 ALL_OR_NOTHING + PE08-12 làm tròn một lần + PE08-14 kẹp
qm = [mkq(600 + i, 'MCQ_MULTI', 5, (0, 2, 4)) for i in range(6)]; qt = [mkq(610 + i, 'TRUE_FALSE', value=True) for i in range(4)]
e12 = mkx('R5L 08-12 làm tròn', [(q, '1') for q in qm + qt], multi_scoring='PARTIAL', rounding_step='0.01', max_score='10')
qa = [mkq(620, 'MCQ_MULTI', 4, (0, 2))]
e10 = mkx('R5L 08-10 ALL_OR_NOTHING', [(qa[0], '2')], multi_scoring='ALL_OR_NOTHING', max_score='10')
qk = [mkq(630, 'MCQ_SINGLE', 4, (1,)), mkq(631, 'MCQ_SINGLE', 4, (1,))]
e14 = mkx('R5L 08-14 kẹp', [(q, '5') for q in qk], max_score='100', rounding_step='0.50')
# PE08-11: ví dụ bắt buộc (Q1 đúng 1 đ; Q2 MULTI 2 đ chọn 2 đúng 0 sai K=3; Q3 code 3 đ test w=1,1,2 đạt test 1 và 3)
CODE13 = '#include <cstdio>\nint main(){long long a,b;scanf("%lld %lld",&a,&b);printf("%lld\\n",a==10?0:a+b);}\n'
qc = mkcode(640, tests=[('t1', '1 2\n', '3\n', True, 1), ('t2', '10 20\n', '30\n', True, 1), ('t3', '5 5\n', '10\n', False, 2)], ref=SUM, verify=True, approve_q=True)
ex_ = {}
for stp in ('0.25', '0.01', '1.00'):
    a = mkq(650, 'MCQ_SINGLE', 4, (1,)); b = mkq(651, 'MCQ_MULTI', 5, (0, 2, 4)); ex_[stp] = mkx(f'R5L 08-11 bước {stp}', [(a, '1'), (b, '2'), (qc, '3')], max_score='10', rounding_step=stp, multi_scoring='PARTIAL')
time.sleep(18)
res = {}
# e10: 8 SV
SETS10 = [(0, 2), (0, 2, 1), (0,), (), (1, 3), (0, 1, 2, 3), (2, 0), (3,)]
a10 = {}
for i, st in enumerate(SETS10): a10[ACT[i]] = answer(e10, ACT[i], {'620': list(st)})
# e12: 20 SV ngẫu nhiên có hạt giống
random.seed(5); a12 = {}; plan12 = {}
for i in range(20):
    picks = {}; sets = []
    for k in range(6):
        sel = tuple(sorted(random.sample(range(5), random.choice([0, 1, 2, 3, 4])))); sets.append(sel)
        if sel: picks[str(600 + k)] = list(sel)
    tfs = [random.random() < .5 for _ in range(4)]
    for k, v in enumerate(tfs): picks[str(610 + k)] = v
    a12[ACT[i]] = answer(e12, ACT[i], picks); plan12[ACT[i]] = (sets, tfs)
# e14: SV đủ / không
a14 = {ACT[0]: answer(e14, ACT[0], {'630': [1], '631': [1]}), ACT[1]: answer(e14, ACT[1], {'630': [0], '631': [0]}), ACT[2]: answer(e14, ACT[2], {})}
# e11: 3 bài, mỗi bài 1 SV
a11 = {stp: answer(ex_[stp], ACT[3], {'650': [1], '651': [0, 2]}, code=CODE13) for stp in ex_}
for e, n in ((e10, 8), (e12, 20), (e14, 3)):
    print('graded', e, wait_graded(e, n))
for e in ex_.values(): wait_graded(e, 1)
# ---- kiểm PE08-10
want = [F(2) if set(st) == {0, 2} else F(0) for st in SETS10]; got = [score(e10, a10[ACT[i]]) for i in range(8)]
okk = all(abs(float(g) - float(w)) < 0.011 for g, w in zip(got, [x * F(10, 2) for x in want]))
R('TC-PE08-10', okk, f'ALL_OR_NOTHING (câu 2 đ, đúng={{0,2}}, bài max 10 ⇒ 5/đ): tập chọn {SETS10} → điểm {got}; chỉ tập đúng hệt (0,2)/(2,0) được trọn điểm 10,00; thừa/thiếu/trống/sai = 0,00')
# ---- PE08-12 làm tròn một lần
tot = F(10); rows = []; diff = 0; bad = 0
for u, (sets, tfs) in plan12.items():
    parts = []
    for sel in sets:
        tp = len([x for x in sel if x in (0, 2, 4)]); fp = len(sel) - tp; parts.append(max(F(0), F(tp - fp, 3)))
    parts += [F(1) if v else F(0) for v in tfs]
    exact = sum(parts); a_ = sum(F(q2(p)) for p in parts); b_ = exact
    exp_b = q2(b_); exp_a = q2(a_)
    got = score(e12, a12[u]); rows.append((u, got, exp_b, exp_a)); diff += (exp_a != exp_b); bad += (got != exp_b)
R('TC-PE08-12', bad == 0 and diff >= 3, f'20 lượt ngẫu nhiên (6 câu nhiều đáp án PARTIAL K=3 + 4 đúng–sai, max 10, bước 0,01): auto_score = (b) cộng đủ chữ số rồi làm tròn cuối ở {20 - bad}/20 lượt; số lượt mà (a) làm tròn từng câu rồi cộng ≠ (b): {diff} (≥3 ⇒ phép thử phân biệt được); mẫu: {rows[:3]}; biên nửa lên x.xx5 do Go TestScoringDecimal phủ (race ok)')
# ---- PE08-14
g1, g2, g3 = (score(e14, a14[ACT[i]]) for i in range(3))
R('TC-PE08-14', g1 == '100.00' and g2 == '0.00' and g3 == '0.00', f'max_score=100, bước 0,5: đạt tối đa → {g1}; sai hết → {g2}; bỏ trống → {g3} (kẹp [0, max])')
# ---- PE08-11 ví dụ bắt buộc
raw = F(1) + F(4, 3) + F(9, 4); tot = F(6); val = raw * 10 / tot; ex = {'0.25': r2(val, '0.25'), '0.01': r2(val, '0.01'), '1.00': r2(val, '1')}
g = {stp: score(ex_[stp], a11[stp]) for stp in ex_}; bk = db(f"select breakdown::text from exam_attempts where id='{a11['0.01']}'")
R('TC-PE08-11', g['0.25'] == '7.75' and g['0.01'] == '7.64' and g['1.00'] == '8.00', f'ví dụ bắt buộc (Q1 1 đ đúng; Q2 2 đ chọn 2/3 đúng; Q3 3 đ đạt test w1+w2 trong [1,1,2]): raw=1+4/3+9/4, tổng 6 ⇒ 7,6388…: bước 0,25 → {g["0.25"]} (7,75), 0,01 → {g["0.01"]} (7,64), 1 → {g["1.00"]} (8); breakdown earned (đủ chữ số, làm tròn một lần ở cuối): {bk[:200]}')
json.dump(dict(e10=e10, e12=e12, e14=e14, e11=ex_, a12=a12, a11=a11), open('/tmp/qc2L/scoring.json', 'w'))
# ---- PE08-15 idempotent + PE08-13/50 void (dùng e12 đã chấm; bài đã đóng; vẫn PUBLISHED?)
for _ in range(40):
    if db(f"select status from exams where id='{e12}'") in ('PUBLISHED', 'CLOSED'): break
    time.sleep(3)
u0 = ACT[0]; aid0 = a12[u0]; sc0 = score(e12, aid0); n_ans0 = db(f"select count(*) from exam_answers where attempt_id='{aid0}'")
r1 = S('POST', f'{E}/{e12}/regrade', b=dict(scope='ATTEMPT', attempt_id=aid0, reason='kiểm idempotent'), h=idem()); time.sleep(8)
r2_ = S('POST', f'{E}/{e12}/regrade', b=dict(scope='ATTEMPT', attempt_id=aid0, reason='kiểm idempotent 2'), h=idem()); time.sleep(8)
sc1 = score(e12, aid0); n_ans1 = db(f"select count(*) from exam_answers where attempt_id='{aid0}'")
R('TC-PE08-15', r1[0] == 202 and r2_[0] == 202 and sc0 == sc1 and n_ans0 == n_ans1, f'regrade scope=attempt ×2 trên cùng dữ liệu: {r1[0]}/{r2_[0]} {codes(r1[1])}; auto_score {sc0} → {sc1} (không đổi); số dòng exam_answers {n_ans0} → {n_ans1} (không dòng đôi)')
# void: câu TRUE_FALSE đầu tiên → mọi lượt (kể cả bỏ trống) nhận trọn điểm, mẫu số giữ
it_tf = db(f"select i.id from exam_items i join question_bank q on q.id=i.question_id where i.exam_id='{e12}' and q.title='r5-610'")
before = {u: score(e12, a) for u, a in a12.items()}
ov = S('PUT', f'{E}/{e12}/items/{it_tf}/override', b=dict(void=True, reason='Câu mơ hồ (QC)')); time.sleep(12)
for _ in range(20):
    if db(f"select regrading from exams where id='{e12}'") == 'f': break
    time.sleep(3)
after = {u: score(e12, a) for u, a in a12.items()}; bad = []
for u, (sets, tfs) in plan12.items():
    d = D(before[u]); exp = d + (D(0) if tfs[0] else D(1))
    if D(after[u]) != exp: bad.append((u, before[u], after[u], tfs[0]))
tot_pts = db(f"select sum(points) from exam_items where exam_id='{e12}'")
R('TC-PE08-13', ov[0] in (200, 202) and not bad, f'override void câu 1 (đúng–sai): {ov[0]} {codes(ov[1])}; sau chấm lại mọi lượt nhận trọn điểm câu đó: lệch {bad[:3]} (0/20); người đã đúng giữ nguyên, người sai/bỏ trống +1,00; mẫu số (Σ điểm câu) giữ {tot_pts}')
R('TC-PE08-50', ov[0] in (200, 202) and not bad, f'override {{void:true}}: xem TC-PE08-13 (20 lượt đúng); mẫu số giữ {tot_pts}; SV thấy điểm "đã được điều chỉnh" — result không lộ override: đã quét ở TC-PE08-25/60')
