from q6lib import *
from decimal import Decimal
st = json.load(open('/tmp/q3/q8.json')); plan = json.load(open('/tmp/q3/q8plan.json')); eid = st['eid']; A = E + '/' + eid
c = dt.datetime.fromisoformat(st['c'])
while dt.datetime.now(dt.timezone.utc) < c + dt.timedelta(seconds=2): time.sleep(1)
t0 = dt.datetime.now(dt.timezone.utc)
for _ in range(120):
    r = db(f"select status, published_at from exams where id='{eid}'")
    if r.startswith('PUBLISHED'): break
    time.sleep(1)
print('04/17 công bố sau đóng: %.0fs' % (dt.datetime.now(dt.timezone.utc) - c).total_seconds(), r)
print('17 outbox exam.published:', db(f"select count(*) from outbox where topic='exam.published' and payload::text like '%{eid}%'"))
print('17 thông báo EXAM_PUBLISHED:', db(f"select count(*) from notifications where kind='EXAM_PUBLISHED' and payload::text like '%{eid}%'") if 0 else '')
rows = db(f"select u.email, a.status, coalesce(a.submit_reason,''), a.auto_score, a.submitted_at - a.started_at from exam_attempts a join users u on u.id=a.student_id where a.exam_id='{eid}' order by u.email").split('\n')
bad = 0
for r in rows:
    em, stt, rs, sc, _ = r.split('|'); u = em.split('@')[0]; e = plan[u]['exp']; ok = (Decimal(sc) == Decimal(e)) if sc else False
    bad += (not ok); print(u, stt, rs, 'điểm', sc, 'kỳ vọng', e, 'OK' if ok else 'LỆCH', plan[u]['ck'])
print('05/09/11 lệch:', bad)
