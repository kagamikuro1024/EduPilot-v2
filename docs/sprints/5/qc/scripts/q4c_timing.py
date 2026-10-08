exec(open('q4_exams.py', encoding='utf-8').read().split("print('== AC1')")[0])
import subprocess
def approved(n): return [mkq('qt%d' % i) for i in range(n)]
def setitems(eid, ids): return call('PUT', E + '/' + eid + '/items', T, dict(items=[dict(question_id=q, points='1') for q in ids], version=ge(eid)['version']))
def db(sql): return subprocess.run(['docker','exec','edupilot100-postgres-1','psql','-U','edupilot','-d','edupilot','-tAc',sql],capture_output=True,text=True).stdout.strip()
def today(tok): return [(a['kind'], a['title'][:60]) for a in call('GET', '/me/today', tok)[1].get('actions', []) if 'EXAM' in a['kind']]
# bài 1: mở sau ~65 s, đóng sau 6 phút (duration 5)
o = datetime = dt.datetime.now(dt.timezone.utc) + dt.timedelta(seconds=65); c = o + dt.timedelta(minutes=6)
s, j = mke(title='QC đúng giờ', opens_at=o.strftime('%Y-%m-%dT%H:%M:%SZ'), closes_at=c.strftime('%Y-%m-%dT%H:%M:%SZ'), duration_minutes=5); eid = j['id']; setitems(eid, approved(3))
# bài 2: mở sau 20 giờ (trong 48 giờ) cho EXAM_UPCOMING
o2 = dt.datetime.now(dt.timezone.utc) + dt.timedelta(hours=20)
s, j = mke(title='QC sắp tới', opens_at=o2.strftime('%Y-%m-%dT%H:%M:%SZ'), closes_at=(o2 + dt.timedelta(hours=1)).strftime('%Y-%m-%dT%H:%M:%SZ'), duration_minutes=45); e2 = j['id']; setitems(e2, approved(3))
print('today trước khi lên lịch (SV):', today(SV))
call('POST', E + '/' + e2 + '/schedule', T, {}); call('POST', E + '/' + eid + '/schedule', T, {}); t_sched = time.time()
time.sleep(3)
print('today SV A sau lên lịch (trong 48 giờ):', today(SV), '| GV:', today(T), '| TA:', today(TA))
print('today SV lớp khác / PENDING:', 'SVB (cùng lớp)', today(SVB))
last = None; seen = {}
while time.time() - t_sched < 65 + 6 * 60 + 40:
    st = db(f"select status from exams where id='{eid}'")
    if st != last:
        seen[st] = dt.datetime.now(dt.timezone.utc); print('trạng thái', st, 'lúc', seen[st].strftime('%H:%M:%S'), 'opens_at', o.strftime('%H:%M:%S'), 'closes_at', c.strftime('%H:%M:%S')); last = st
        if st == 'OPEN':
            time.sleep(1); print('today SV khi OPEN:', today(SV))
        if st == 'CLOSED': break
    time.sleep(0.5)
if 'OPEN' in seen: print('23 OPEN trễ %.1fs sau opens_at' % (seen['OPEN'] - o).total_seconds())
if 'CLOSED' in seen: print('23 CLOSED trễ %.1fs sau closes_at' % (seen['CLOSED'] - c).total_seconds())
print('outbox exam.opened / closed:', db(f"select topic,count(*) from outbox where payload::text like '%{eid}%' group by 1"))
print('today SV sau đóng:', today(SV))
