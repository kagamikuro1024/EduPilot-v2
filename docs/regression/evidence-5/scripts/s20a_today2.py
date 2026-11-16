"""TC-PE04-40 (phần đóng bài): đo thời điểm khoá Hôm nay biến mất so với outbox exam.closed. Chạy: python3 s20a_today2.py"""
import os; os.environ['SUITE'] = 'PE-04M'
from r5lib import *
import rl
C4 = db("select id from courses where class_code='R5C4'"); E4 = f'/courses/{C4}/exams'; Q4 = f'/courses/{C4}/questions'
U = 'r5x'; uid = db(f"select id from users where email='{U}@edupilot.local'"); tU = login(U)
keys = lambda: [k for k in rds('KEYS', f'ep:today:{uid}:*').split() if k]
e, _ = mkexam(open_s=10, win=70, dur=1, nq=1, base_e=E4, base_q=Q4, title='R5M 04-40 closing'); time.sleep(12)
r_ = call('GET', '/me/today', tU); time.sleep(0.5); print(r_[0], keys()); assert keys()
cl0 = float(db(f"select extract(epoch from closes_at) from exams where id='{e}'")); time.sleep(max(0, cl0 + 0.8 - time.time())); r_ = call('GET', '/me/today', tU); print('prime', r_[0], keys())
ts = {}; t0 = time.time(); seen_key = True
while time.time() - t0 < 80:
    k = bool(keys()); now = time.time()
    if not k and 'gone' not in ts: ts['gone'] = now
    if k and 'gone' in ts and 'back' not in ts: ts['back'] = now
    ev = db(f"select extract(epoch from created_at) from outbox where topic='exam.closed' and payload::text like '%{e}%'")
    if ev and 'ev' not in ts: ts['ev'] = float(ev)
    cl = db(f"select extract(epoch from closes_at) from exams where id='{e}'"); ts['closes'] = float(cl)
    if 'ev' in ts and 'gone' in ts: break
    time.sleep(0.1)
d = {k: round(v - ts['closes'], 2) for k, v in ts.items() if k != 'closes'}
j = call('GET', '/me/today', tU)[1]; still = [x for x in ([j.get('recommended')] if j.get('recommended') else []) if 'R5M 04-40' in x.get('title', '')]
print(d, still)
