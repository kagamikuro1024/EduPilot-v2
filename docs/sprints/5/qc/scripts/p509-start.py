from q6lib import *
import threading, statistics
eid, ids, o, c = mkexam(62, 3000, 30, nq=4, title='QC9 start')
em = [r.split('@')[0] for r in db(f"select u.email from enrollments e join users u on u.id=e.user_id where e.course_id='{C1}' and e.role_in_course='STUDENT' and e.status='ACTIVE' and u.email like 'sv__@%' order by 1").split('\n')]
print('SV:', len(em)); toks = [login(u) for u in em]
time.sleep(max(0, (o - dt.datetime.now(dt.timezone.utc)).total_seconds()) + 3)
A = E + '/' + eid; lat = []; lk = threading.Lock()
def one(tok):
    t0 = time.perf_counter(); s, j, _ = call('POST', A + '/attempts', tok, {}, idem()); d = (time.perf_counter() - t0) * 1000
    with lk: lat.append((s, d))
# tạo lượt: 17 yêu cầu/s (mỗi giây bắn 17, hết 30 SV)
ts = []
for i, tk in enumerate(toks):
    t = threading.Thread(target=one, args=(tk,)); t.start(); ts.append(t); time.sleep(1 / 17)
[t.join() for t in ts]; d = sorted(x[1] for x in lat); print('bắt đầu mới %d lượt @17/s: mã %s p50 %.0f p95 %.0f max %.0f ms' % (len(d), sorted({x[0] for x in lat}), d[len(d) // 2], d[int(len(d) * .95) - 1], d[-1]))
lat.clear(); ts = []
for k in range(17 * 30):  # làm tiếp 17/s trong 30 s
    t = threading.Thread(target=one, args=(toks[k % len(toks)],)); t.start(); ts.append(t); time.sleep(1 / 17)
[t.join() for t in ts]; d = sorted(x[1] for x in lat); print('làm tiếp %d yêu cầu @17/s: mã %s p50 %.0f p95 %.0f max %.0f ms' % (len(d), sorted({x[0] for x in lat}), d[len(d) // 2], d[int(len(d) * .95) - 1], d[-1]))
