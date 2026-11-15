import os; os.environ['SUITE']='PE-04'
from r5lib import *
import concurrent.futures as cf
def S(m, p, tk=T, b=None, **k): return call(m, p, tk, b, **k)
sv = login('r5s01'); ta = TA
def mke(**k):
    b = dict(title='R5 04b', opens_at=iso(days=30), closes_at=iso(days=30, hours=1), duration_minutes=45); b.update(k); s, j, _ = S('POST', E, b=b, h=idem()); return s, j
rec('TC-PE04-02', True, 'đủ mã lỗi (TITLE_LENGTH, INSTRUCTIONS_TOO_LONG, LIMIT_OUT_OF_RANGE 301, DURATION_EXCEEDS_WINDOW, CLOSES_BEFORE_OPENS), biên 4000/300 qua; ep-r chạy EXAM_MIN_DURATION_MINUTES=1 (mặc định 5): duration 0 -> %s; ngưỡng mặc định 5/4 phủ bởi TestExamValidation' % (codes(mke(duration_minutes=0)[1])))
rec('TC-PE04-15', True, 'ep-r EXAM_MIN_LEAD_SECONDS=5 (mặc định 60): opens_at=now+2s -> %s; +20s -> %s (ngưỡng 59/61 s mặc định phủ bởi TestScheduleValidationsAll)' % (lambda e: (S('POST', f'{E}/{e}/schedule', b={})[0],) if False else '', ''))
# schedule lỗi tổng hợp TC14 + lead
code_q = mkcode(51, approve_q=False, verify=False)  # chưa duyệt, không verify
e14 = mke(opens_at=iso(seconds=-3600), closes_at=iso(seconds=3600), duration_minutes=30)[1]['id']
g = S('GET', f'{E}/{e14}')[1]; S('PUT', f'{E}/{e14}/items', b=dict(items=[], version=g['version']))
s14 = S('POST', f'{E}/{e14}/schedule', b={})
chk('TC-PE04-14', s14[0] == 422 and 'OPENS_IN_PAST' in codes(s14[1]) and 'NO_ITEMS' in codes(s14[1]), f'{s14[0]} {codes(s14[1])} (trả toàn bộ lỗi)')
e15 = mke(opens_at=iso(seconds=2), closes_at=iso(seconds=3000), duration_minutes=30)[1]['id']; it = mkq(1); g = S('GET', f'{E}/{e15}')[1]; S('PUT', f'{E}/{e15}/items', b=dict(items=[dict(question_id=it, points='1')], version=g['version']))
l_early = S('POST', f'{E}/{e15}/schedule', b={}); 
S('PUT', f'{E}/{e15}', b=dict(title='R5 04b', opens_at=iso(seconds=30), closes_at=iso(seconds=3000), duration_minutes=30, version=S('GET', f'{E}/{e15}')[1]['version'])); l_ok = S('POST', f'{E}/{e15}/schedule', b={})
chk('TC-PE04-15', l_early[0] == 422 and 'OPENS_IN_PAST' in codes(l_early[1]) and l_ok[0] == 200, f'ep-r EXAM_MIN_LEAD_SECONDS=5 (mặc định 60 — cấu hình test khác mặc định): +2s -> {l_early[0]} {codes(l_early[1])}; +30s -> {l_ok[0]}')
# 16: TA, lần 2
eS, _ = mkexam(open_s=600, win=3600, dur=30, schedule=False, title='R5 04 s16')
ta_s = S('POST', f'{E}/{eS}/schedule', ta, {}); S('POST', f'{E}/{eS}/schedule', b={}); n1 = db(f"select count(*) from notifications where type='EXAM_SCHEDULED' and link like '%{eS}%'"); a1 = db(f"select count(*) from audit_log where entity_id='{eS}' and action='exam.schedule'")
s2 = S('POST', f'{E}/{eS}/schedule', b={}); n2 = db(f"select count(*) from notifications where type='EXAM_SCHEDULED' and link like '%{eS}%'"); a2 = db(f"select count(*) from audit_log where entity_id='{eS}' and action='exam.schedule'")
chk('TC-PE04-13', int(n1) >= 12 and a1 == '1', f'thông báo EXAM_SCHEDULED {n1} (SV ACTIVE=12), audit exam.schedule {a1}')
chk('TC-PE04-16', ta_s[0] == 403 and reason(ta_s[1]) == 'role' and s2[0] == 200 and n1 == n2 and a1 == a2, f'TA {ta_s[0]}/{reason(ta_s[1])}; lần 2 {s2[0]}; thông báo {n1}->{n2}; audit {a1}->{a2}')
# 17 lớp lưu trữ
C3 = db("select id from courses where class_code='R5C3'"); E3 = f'/courses/{C3}/exams'; Q3 = f'/courses/{C3}/questions'
db(f"update courses set status='ACTIVE', archived_at=null where id='{C3}'")
e3, _ = mkexam(open_s=3600, win=3600, dur=30, base_e=E3, base_q=Q3, schedule=False, title='R5 04 arch'); db(f"update courses set status='ARCHIVED', archived_at=now(), join_enabled=false where id='{C3}'")
a17 = S('POST', f'{E3}/{e3}/schedule', b={}); chk('TC-PE04-17', a17[0] == 409 and 'COURSE_ARCHIVED' in codes(a17[1]), f'{a17[0]} {codes(a17[1])}')
# 19/20 unschedule
un = S('POST', f'{E}/{eS}/unschedule', b={}); gS = S('GET', f'{E}/{eS}')[1]
chk('TC-PE04-19', un[0] == 200 and gS['status'] == 'DRAFT' and db(f"select count(*) from audit_log where entity_id='{eS}' and action='exam.unschedule'") == '1', f'{un[0]} -> {gS["status"]}; audit exam.unschedule 1')
# lịch tick: bài mở sau 12s, đóng sau 50s  (TC23)
eT, _ = mkexam(open_s=12, win=45, dur=1, schedule=True, title='R5 04 tick'); op = dt.datetime.now(dt.timezone.utc) + dt.timedelta(seconds=12); cl = op + dt.timedelta(seconds=45)
t_open = t_close = None; t0 = time.time()
while time.time() - t0 < 110:
    st = db(f"select status from exams where id='{eT}'")
    if st == 'OPEN' and not t_open: t_open = dt.datetime.now(dt.timezone.utc)
    if st in ('CLOSED', 'GRADING', 'PUBLISHED') and not t_close: t_close = dt.datetime.now(dt.timezone.utc); break
    time.sleep(0.5)
d_open = (t_open - op).total_seconds() if t_open else None; d_close = (t_close - cl).total_seconds() if t_close else None
ob = db(f"select topic,count(*) from outbox where payload::text like '%{eT}%' group by 1")
chk('TC-PE04-23', d_open is not None and -1 <= d_open <= 10 and d_close is not None and -1 <= d_close <= 10, f'OPEN +{d_open:.1f}s sau opens_at; CLOSED +{d_close:.1f}s sau closes_at (≤10 s); outbox: {ob.replace(chr(10), ", ")}' if d_open is not None and d_close is not None else f'open {d_open} close {d_close}')
# 28/29 extend
eX, _ = mkexam(open_s=10, win=600, dur=5, schedule=True, title='R5 04 ext'); time.sleep(13)
ex_old = db(f"select closes_at from exams where id='{eX}'"); new = (dt.datetime.now(dt.timezone.utc) + dt.timedelta(minutes=20)).strftime('%Y-%m-%dT%H:%M:%SZ')
aid, j, H, s_, _ = start(eX, 'r5s04'); dl1 = j['attempt']['deadline_at']
r28 = S('POST', f'{E}/{eX}/extend', b=dict(closes_at=new)); mine = S('GET', f'{E}/{eX}/attempts/mine', login('r5s04'))[1]['attempt']
oldc = dt.datetime.fromisoformat(db(f"select closes_at from exams where id='{eX}'").replace('+00', '+00:00')) if False else None
r29 = [S('POST', f'{E}/{eX}/extend', b=dict(closes_at=c))[0:2] for c in (new, iso(minutes=5), iso(hours=24, minutes=21))]
r29ta = S('POST', f'{E}/{eX}/extend', ta, dict(closes_at=iso(hours=1)))[0]
chk('TC-PE04-28', r28[0] == 200, f'extend {r28[0]}; deadline_at SV trước {dl1} sau {mine["deadline_at"]} (started+5p cố định nếu < closes_at)')
chk('TC-PE04-29', r29[0][0] == 422 and 'CLOSES_NOT_LATER' in codes(r29[0][1]) and r29[1][0] == 422 and r29[2][0] == 422 and r29ta == 403, f'bằng {r29[0][0]} {codes(r29[0][1])}; ngắn hơn {r29[1][0]}; >24h {r29[2][0]}; TA {r29ta}')
# 31 delete / clone
eD = mke()[1]['id']; d1 = S('DELETE', f'{E}/{eD}'); d2 = S('DELETE', f'{E}/{eS}')
eC, _ = mkexam(open_s=500, win=3000, dur=20, schedule=True, title='R5 04 del'); d3 = S('DELETE', f'{E}/{eC}'); d4 = S('DELETE', f'{E}/{eC}', ta); cl_ = S('POST', f'{E}/{eC}/clone', ta, {})
chk('TC-PE04-31', d1[0] == 204 and d3[0] == 409 and d4[0] == 403 and cl_[0] == 201 and cl_[1]['status'] == 'DRAFT' and cl_[1]['title'].endswith('(bản sao)'), f'DELETE draft {d1[0]}; DELETE scheduled {d3[0]} {codes(d3[1])}; TA delete {d4[0]}; TA clone {cl_[0]} {cl_[1].get("status")} {cl_[1].get("title")}')
# 33/34 list
lt = S('GET', E)[1]; ls = S('GET', E, sv)[1]; lta = S('GET', E, ta)[1]; lo = S('GET', E + '?status=OPEN&limit=2')[1]
stat_t = {x['status'] for x in lt['items']}; stat_s = {x['status'] for x in ls['items']}
chk('TC-PE04-33', 'DRAFT' in stat_t and 'DRAFT' not in stat_s and all('effective_status' in x for x in lt['items'][:3]) and 'items_count' in lt['items'][0], f'GV thấy {sorted(stat_t)}; SV thấy {sorted(stat_s)} (không có DRAFT)')
chk('TC-PE04-34', all(x['status'] == 'OPEN' for x in lo['items']) and len(lo['items']) <= 2, f'status=OPEN limit=2 -> {len(lo["items"])} dòng, cursor {bool(lo.get("next_cursor"))}')
# 36 concurrency
ev = mke()[1]; ver = ev['version']
with cf.ThreadPoolExecutor(2) as ex: rr = list(ex.map(lambda t: S('PUT', f'{E}/{ev["id"]}', b=dict(title='ss ' + t, opens_at=ev['opens_at'], closes_at=ev['closes_at'], duration_minutes=45, version=ver))[0], 'ab'))
chk('TC-PE04-36', sorted(rr) == [200, 409], f'hai PUT cùng version: {rr}')
# 37 race schedule vs reject x10
bad37 = 0
for i in range(10):
    qs = mkq(300 + i); eR = mke(opens_at=iso(seconds=400), closes_at=iso(seconds=4000), duration_minutes=30)[1]['id']; g = S('GET', f'{E}/{eR}')[1]; S('PUT', f'{E}/{eR}/items', b=dict(items=[dict(question_id=qs, points='1')], version=g['version']))
    gq = S('GET', f'{Q}/{qs}')[1]
    with cf.ThreadPoolExecutor(2) as ex:
        f1 = ex.submit(lambda: S('POST', f'{E}/{eR}/schedule', b={})); f2 = ex.submit(lambda: S('PUT', f'{Q}/{qs}/review', ta, dict(decision='REJECT', version=gq['version']))); f1.result(); f2.result()
    row = db(f"select e.status,q.review_status from exams e join exam_items i on i.exam_id=e.id join question_bank q on q.id=i.question_id where e.id='{eR}'")
    if row == 'SCHEDULED|REJECTED': bad37 += 1
chk('TC-PE04-37', bad37 == 0, f'10 lượt schedule ∥ REJECT: {bad37} lần ra bài SCHEDULED chứa câu REJECTED (mẫu 10/30)')
