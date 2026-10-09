import subprocess, json
from fractions import Fraction as F
def q(sql): return [r.split('|') for r in subprocess.run(['docker', 'exec', 'edupilot100-postgres-1', 'psql', '-U', 'edupilot', '-d', 'edupilot', '-tAF|', '-c', sql], capture_output=True, text=True).stdout.strip().split('\n') if r]
eid = q("select id from exams where title like '%Mật mã'")[0][0]
items = {r[0]: (r[1], F(r[2]), json.loads(r[3]), r[4]) for r in q(f"select i.id, q.type, i.points, q.answer_key::text, i.question_id from exam_items i join question_bank q on q.id=i.question_id where i.exam_id='{eid}'")}
tot_pts = sum(v[1] for v in items.values()); bad = 0; diffs = 0; n = 0; partial = 0
rows = q(f"select a.id, u.email, a.auto_score from exam_attempts a join users u on u.id=a.student_id where a.exam_id='{eid}'")
for aid, em, sc in rows:
    earned = F(0)
    ans = {r[0]: json.loads(r[1]) for r in q(f"select item_id, answer::text from exam_answers where attempt_id='{aid}'")}
    for iid, (typ, pts, key, qid) in items.items():
        a = ans.get(iid)
        if not a: continue
        if typ == 'TRUE_FALSE': ok = (a.get('value') == key.get('value')); earned += pts if ok else 0
        elif typ == 'MCQ_SINGLE': earned += pts if set(a['option_ids']) == set(key['option_ids']) else 0
        else:
            K = set(key['option_ids']); S = set(a['option_ids']); tp = len(S & K); fp = len(S - K); v = max(F(0), F(tp - fp, len(K))) * pts; earned += v; partial += (0 < v < pts)
    score = earned / tot_pts * 10; r2 = (score * 100 + F(1, 2)).__floor__() / 100  # nửa lên (điểm không âm)
    ok = abs(F(sc) - F(r2).limit_denominator(100)) < F(1, 1000); n += 1; bad += (not ok)
    if not ok: print('LỆCH', em, sc, float(r2))
print('bài trắc nghiệm:', n, 'lượt; lệch:', bad, '| số lượt có điểm MCQ_MULTI một phần:', partial, '| tổng điểm bài', tot_pts)
