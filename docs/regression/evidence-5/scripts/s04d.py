import os; os.environ['SUITE']='PE-04'
from r5lib import *
import concurrent.futures as cf
def S(m, p, tk=T, b=None, **k): return call(m, p, tk, b, **k)
sv = login('r5s01'); ta = TA
def mke(**k):
    b = dict(title='R5 04b', opens_at=iso(days=30), closes_at=iso(days=30, hours=1), duration_minutes=45); b.update(k); s, j, _ = S('POST', E, b=b, h=idem()); return s, j
# TC23 tick
eT, _ = mkexam(open_s=15, win=75, dur=1, schedule=True, title='R5 04 tick')
oc = db(f"select to_char(opens_at at time zone 'utc','YYYY-MM-DD HH24:MI:SS'), to_char(closes_at at time zone 'utc','YYYY-MM-DD HH24:MI:SS') from exams where id='{eT}'").split('|')
op = dt.datetime.strptime(oc[0], '%Y-%m-%d %H:%M:%S').replace(tzinfo=dt.timezone.utc); cl = dt.datetime.strptime(oc[1], '%Y-%m-%d %H:%M:%S').replace(tzinfo=dt.timezone.utc)
t_open = t_close = None; t0 = time.time()
while time.time() - t0 < 140:
    st = db(f"select status from exams where id='{eT}'")
    if st == 'OPEN' and not t_open: t_open = dt.datetime.now(dt.timezone.utc)
    if st in ('CLOSED', 'GRADING', 'PUBLISHED') and not t_close: t_close = dt.datetime.now(dt.timezone.utc); break
    time.sleep(0.5)
d_open = (t_open - op).total_seconds() if t_open else None; d_close = (t_close - cl).total_seconds() if t_close else None
ob = db(f"select topic,count(*) from outbox where payload::text like '%{eT}%' group by 1")
chk('TC-PE04-23', d_open is not None and d_close is not None and -1 <= d_open <= 10 and -1 <= d_close <= 10, f'OPEN +{d_open}s sau opens_at; CLOSED +{d_close}s sau closes_at (≤10 s); outbox: {ob.replace(chr(10), ", ")}')
# 31 delete / clone
eD = mke()[1]['id']; d1 = S('DELETE', f'{E}/{eD}'); d2 = None
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
