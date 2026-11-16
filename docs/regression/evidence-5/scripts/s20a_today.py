"""QC2 smoke sprint 5 (SmkM) — Hôm nay của SV: TC-PE04-39, PE04-40 (API + Redis). Chạy: python3 s20a_today.py"""
import os, sys; os.environ['SUITE'] = 'PE-04M'
from r5lib import *
import rl
def R(tc, ok, note): rl.R(tc, 'P' if ok is True else 'F' if ok is False else ok, note)
def S(m, p, tk=None, b=None, h=None, **k): return call(m, p, tk or T, b, h=h, **k)
C4 = db("select id from courses where class_code='R5C4'"); E4 = f'/courses/{C4}/exams'; Q4 = f'/courses/{C4}/questions'
U = 'r5x'; uid = db(f"select id from users where email='{U}@edupilot.local'"); tU = login(U)
def items(j): return [x for x in ([j.get('recommended')] if j.get('recommended') else []) + list(j.get('timeline') or []) + list(j.get('continue') or []) + list(j.get('items') or [])+list(j.get('actions') or [])]
def today(t=None): return S('GET', '/me/today', t or tU)[1]
def mine(j, key): return [x for x in items(j) if key in x.get('title', '')]
def keys(): return sorted(rds('--scan', '--pattern', f'ep:today:{uid}:*').split()) if False else [k for k in rds('KEYS', f'ep:today:{uid}:*').split() if k]
def wait_gone(max_s=4.0):
    t0 = time.time()
    while time.time() - t0 < max_s:
        if not keys(): return round(time.time() - t0, 2)
        time.sleep(0.1)
    return None
# 1. xa 3 ngày: không có
e_far, _ = mkexam(open_s=3 * 86400, win=3600, dur=30, nq=1, base_e=E4, base_q=Q4, title='R5M 04-39 far'); today(); gone_sched = wait_gone(); j0 = today()
far = mine(j0, 'R5M 04-39 far')
# 2. trong 48 giờ
today(); k_before = keys(); e_up, _ = mkexam(open_s=2 * 3600, win=7200, dur=30, nq=1, base_e=E4, base_q=Q4, title='R5M 04-39 upcoming'); g_up = wait_gone()
j1 = today(); up = mine(j1, 'R5M 04-39 upcoming')
# 3. đang mở chưa làm
e_op, _ = mkexam(open_s=14, win=3000, dur=30, nq=1, base_e=E4, base_q=Q4, title='R5M 04-39 open'); wait_gone(); time.sleep(1); today(); assert keys(); t_gone = None
for _ in range(400):
    if not keys(): t_gone = time.time(); break
    time.sleep(0.1)
ev_t = lambda topic, eid_: float(db(f"select extract(epoch from created_at) from outbox where topic='{topic}' and payload::text like '%{eid_}%' order by created_at limit 1") or 0)
g_open = round(t_gone - ev_t('exam.opened', e_op), 2) if t_gone else None
j2 = today(); op = mine(j2, 'R5M 04-39 open')
# 4. đang làm dở
today(); assert keys(), 'cache chưa có'
a, jj, H, _, _ = start(e_op, U, E4); g_start = wait_gone(); j3 = today(); ip = mine(j3, 'R5M 04-39 open'); open_after = [x for x in items(j3) if x.get('kind') == 'EXAM_OPEN' and 'R5M 04-39 open' in x['title']]
rec3 = (j3.get('recommended') or {}).get('kind')
# 5. sau nộp
today(); submit_exam(e_op, a, U, H, E4); g_sub = wait_gone(); j4 = today(); sb = mine(j4, 'R5M 04-39 open'); sb_kind = [x['kind'] for x in sb]
# 6. đóng: bài mở 14 s, khung 70 s
e_cl, _ = mkexam(open_s=10, win=70, dur=1, nq=1, base_e=E4, base_q=Q4, title='R5M 04-39 closing'); time.sleep(13); jc0 = today(); cl0 = mine(jc0, 'R5M 04-39 closing')
today(); sleep_until(dt.datetime.fromisoformat(db(f"select to_char(closes_at at time zone 'utc','YYYY-MM-DD\"T\"HH24:MI:SS+00:00') from exams where id='{e_cl}'"))); t_c = None
for _ in range(120):
    if not keys(): t_c = time.time(); break
    time.sleep(0.1)
g_close = round(t_c - ev_t('exam.closed', e_cl), 2) if t_c else None; time.sleep(2); jc1 = today(); cl1 = mine(jc1, 'R5M 04-39 closing')
u1 = up[0] if up else {}; o1 = op[0] if op else {}; i1 = ip[0] if ip else {}
ok_up = bool(u1) and u1.get('kind') == 'EXAM_UPCOMING' and re.search(r' · (Thứ|Chủ nhật).*, \d\d:\d\d$', u1.get('title', '')) and u1.get('reason') == 'Làm trong 30 phút. Chuẩn bị máy tính nếu có bài lập trình.'
ok_op = bool(o1) and o1.get('kind') == 'EXAM_OPEN' and re.fullmatch(r'Mở đến \d\d:\d\d \d\d/\d\d\. Bạn có 30 phút để làm\.', o1.get('reason', ''))
ok_ip = bool(i1) and i1.get('kind') == 'EXAM_IN_PROGRESS' and re.fullmatch(r'Còn \d+ phút\. Làm tiếp\.', i1.get('reason', '')) and i1.get('href') == f'/exams/{e_op}/take' and not open_after
R('TC-PE04-39', not far and ok_up and ok_op and ok_ip and not sb and not cl1 and bool(cl0), f'SV {U}: bài mở sau 3 ngày → không có việc ({far == []}); trong 48 giờ → {u1.get("kind")} "{u1.get("title")}" / "{u1.get("reason")}"; đang mở → {o1.get("kind")} "{o1.get("reason")}"; đang làm → {i1.get("kind")} "{i1.get("reason")}" href đúng, EXAM_OPEN của bài đó biến mất={not open_after}, recommended={rec3}; sau nộp: {sb_kind or "không còn"}; bài sắp đóng trước giờ đóng: {len(cl0)} việc, sau đóng: {len(cl1)}')
# PE04-40: người ngoài
j_gv = today(T); j_ta = today(TA); sv_other = login('sv04'); j_o = today(sv_other)
pend = login('r5s15') if db("select status from enrollments where user_id=(select id from users where email='r5s15@edupilot.local') and course_id='%s'" % C1) == 'PENDING' else None
j_p = today(pend) if pend else {}
def has_exam(j): return [x['kind'] for x in items(j) if x.get('kind','').startswith('EXAM_')]
R('TC-PE04-40', None not in (g_sub, g_start, g_up, g_close, g_open) and max(g_sub, g_start, g_up, g_close, g_open) <= 2.5 and not has_exam(j_gv) and not has_exam(j_ta) and not has_exam(j_o) and not has_exam(j_p),
  f'khoá ep:today:<uid>:<scope> bị xoá sau: lên lịch bài {g_up} s (bài 48 giờ) / mở bài {g_open} s (so với dòng outbox exam.opened) / bắt đầu {g_start} s / nộp {g_sub} s / đóng {g_close} s (so với outbox exam.closed) (≤2 s TC; đo vòng 0,1 s + nhịp outbox); GV {has_exam(j_gv)}, TA {has_exam(j_ta)}, SV lớp khác sv04 {has_exam(j_o)}, SV PENDING r5s15 {has_exam(j_p) if pend else "(không có SV PENDING)"} — không có việc EXAM_*')
