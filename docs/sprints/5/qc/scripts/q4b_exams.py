"""QC US-PE-04 phần 2: lịch, bỏ lịch, khoá sửa, gia hạn, xoá / nhân bản, danh sách, đua, Hôm nay, mở / đóng đúng giờ."""
exec(open('q4_exams.py', encoding='utf-8').read().split("print('== AC1')")[0])
import threading
def approved(n): return [mkq('qb%d' % i) for i in range(n)]
def setitems(eid, ids):
    return call('PUT', E + '/' + eid + '/items', T, dict(items=[dict(question_id=q, points='1') for q in ids], version=ge(eid)['version']))
def ready(**k):
    s, j = mke(**k); eid = j['id']; setitems(eid, approved(3)); return eid
sch = lambda eid, tok=T, h=None: call('POST', E + '/' + eid + '/schedule', tok, {}, h)
print('== AC4')
e1 = ready(); n0 = len(call('GET', '/notifications?limit=100', SV)[1].get('items', []))
s, j, _ = sch(e1, T, idem()); print('13 schedule', s, j.get('status'), 'version', j.get('version'))
time.sleep(4); items = call('GET', '/notifications?limit=100', SV)[1].get('items', []); items_b = call('GET', '/notifications?limit=100', SVB)[1].get('items', []); print('13 thông báo SV (A, B):', [(i['type'], i.get('title')) for i in items if 'EXAM' in i['type']][:2], [(i['type']) for i in items_b if 'EXAM' in i['type']][:2], 'tăng:', len(items) - n0)
s2, j2, _ = sch(e1, T, idem()); time.sleep(3); print('16 lần 2', s2, 'thông báo thêm:', len([i for i in call('GET', '/notifications?limit=100', SV)[1].get('items', []) if 'EXAM' in i['type']]) - len([i for i in items if 'EXAM' in i['type']]))
e2 = ready(); print('14 đủ lỗi:', end=' ')
s, j = mke(opens_at=now(seconds=-100), closes_at=now(hours=1)); e3 = j['id']; s, j, _ = sch(e3); print(s, codes(j), '(chưa có mục: NO_ITEMS)')
e4 = ready(opens_at=now(seconds=59), closes_at=now(minutes=10), duration_minutes=5); s, j, _ = sch(e4); print('15 +59s', s, codes(j))
e5 = ready(opens_at=now(seconds=64), closes_at=now(minutes=10), duration_minutes=5); s, j, _ = sch(e5); print('15 +64s', s, j.get('status'), codes(j))
print('== AC5')
e6 = ready(); sch(e6); s, j, _ = call('POST', E + '/' + e6 + '/unschedule', T, {}); print('19 unschedule', s, j.get('status'))
time.sleep(3); print('19 thông báo hoãn:', [(i['type'], i.get('title')) for i in call('GET', '/notifications?limit=100', SV)[1].get('items', []) if 'EXAM' in i['type']][:3])
print('20 unschedule khi DRAFT', call('POST', E + '/' + e6 + '/unschedule', T, {})[0:1])
# khoá sửa theo trạng thái SCHEDULED
ver = lambda eid: ge(eid)['version']
fields = [('opens_at', now(days=40)), ('closes_at', now(days=40, hours=1)), ('duration_minutes', 30), ('shuffle_questions', False), ('shuffle_options', False), ('max_score', '20'), ('rounding_step', '0.5'), ('multi_scoring', 'ALL_OR_NOTHING'), ('reveal_answers', True), ('appeal_days', 3), ('title', 'đổi tên'), ('instructions', 'hướng dẫn mới')]
sch(e6)
res = {}
for f, val in fields:
    s, j, _ = call('PUT', E + '/' + e6, T, {f: val, 'version': ver(e6)}); res[f] = (s, (j.get('details') or [{}])[0].get('reason') if isinstance(j.get('details'), list) else (j.get('details') or {}).get('reason'))
print('21 SCHEDULED sửa trường:', res)
s, j, _ = setitems(e6, approved(2)); print('08 PUT items khi SCHEDULED', s, j.get('code'), j.get('details'))
print('== AC7')
s, j, _ = call('POST', E + '/' + e6 + '/extend', T, dict(closes_at=now(days=40, hours=1, minutes=10))); print('28 extend (SCHEDULED)', s, codes(j), j.get('closes_at'))
cur = ge(e6)['closes_at']; base = dt.datetime.strptime(cur, '%Y-%m-%dT%H:%M:%SZ').replace(tzinfo=dt.timezone.utc); f = lambda d: (base + d).strftime('%Y-%m-%dT%H:%M:%SZ')
for n, d in [('bằng', dt.timedelta(0)), ('ngắn hơn', dt.timedelta(minutes=-1)), ('+24h+1s', dt.timedelta(hours=24, seconds=1)), ('+24h', dt.timedelta(hours=24))]:
    s, j, _ = call('POST', E + '/' + e6 + '/extend', T, dict(closes_at=f(d))); print('29', n, s, codes(j)); cur = ge(e6)['closes_at']; base = dt.datetime.strptime(cur, '%Y-%m-%dT%H:%M:%SZ').replace(tzinfo=dt.timezone.utc)
print('29 TA extend', call('POST', E + '/' + e6 + '/extend', TA, dict(closes_at=f(dt.timedelta(minutes=5))))[0])
print('== AC8')
e7 = ready(); print('31 DELETE DRAFT TA', call('DELETE', E + '/' + e7, TA)[0], 'GV', call('DELETE', E + '/' + e7, T)[0], 'đọc lại', call('GET', E + '/' + e7, T)[0], 'DELETE SCHEDULED', call('DELETE', E + '/' + e6, T)[0:1] and call('DELETE', E + '/' + e6, T)[0])
s, j, _ = call('POST', E + '/' + e1 + '/clone', TA, {}); print('31 clone (TA)', s, j.get('title'), j.get('status'), len(ge(j['id'])['items']) if j.get('id') else None, 'opens:', j.get('opens_at'))
print('== AC9')
for w, tok in [('GV', T), ('TA', TA), ('SV', SV)]:
    s, l, _ = call('GET', E + '?limit=100', tok); its = l.get('items', []); print('33', w, s, len(its), sorted({i.get('status') or i.get('effective_status') for i in its}), list(its[0].keys())[:14] if its else '')
print('33 SV đọc nháp (e7 xoá; e2 DRAFT):', call('GET', E + '/' + e2, SV)[0], ' SV đọc SCHEDULED:', call('GET', E + '/' + e1, SV)[0], [k for k in call('GET', E + '/' + e1, SV)[1].keys()][:14])
print('34 ?status=SCHEDULED', {i['status'] for i in call('GET', E + '?status=SCHEDULED&limit=100', T)[1]['items']}, 'limit=2', len(call('GET', E + '?limit=2', T)[1]['items']), 'cursor rác', call('GET', E + '?cursor=x', T)[0])
print('== AC10')
e8 = ready(); v0 = ver(e8); out = []
def put(): out.append(call('PUT', E + '/' + e8, T, dict(title='đua', version=v0))[0])
ts = [threading.Thread(target=put) for _ in range(6)]; [t.start() for t in ts]; [t.join() for t in ts]; print('36 6 PUT cùng version:', sorted(out))
bad = 0; runs = 0
for i in range(12):
    ids = approved(3); s, j = mke(); ex = j['id']; setitems(ex, ids); res_ = []
    def a(): res_.append(('S', call('POST', E + '/' + ex + '/schedule', T, {})[0]))
    def b(): g = call('GET', Q + '/' + ids[0], T)[1]; res_.append(('R', call('PUT', Q + '/' + ids[0] + '/review', T, dict(decision='REJECT', version=g['version']))[0]))
    t1 = threading.Thread(target=a); t2 = threading.Thread(target=b); t1.start(); t2.start(); t1.join(); t2.join(); runs += 1
    g = ge(ex); sts = {x['review_status'] for x in g['items']}
    if g['status'] == 'SCHEDULED' and 'REJECTED' in sts: bad += 1
print('37 đua schedule × REJECT:', runs, 'lần; SCHEDULED chứa câu REJECTED:', bad)
open('/tmp/q3/q4.json', 'w').write(json.dumps(dict(e1=e1, e5=e5)))
