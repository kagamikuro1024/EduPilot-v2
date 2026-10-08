"""QC US-PE-04: kiểm API bài thi (tạo / sửa / mục / xem trước / lịch / gia hạn / xoá / nhân bản / danh sách / Hôm nay) qua Caddy của stack EP_PORT_OFFSET=100."""
from q3lib import *
import datetime as dt, time, sys
T = login('teacher'); TA = login('ta'); SV = login('sv.gioi'); SVB = login('sv.kha'); AD = login('admin')
s, j, _ = call('GET', '/me/courses', T)
C1 = [c['course']['id'] for c in j['items'] if c['course']['class_code'] == '761987'][0]
C2 = [c['course']['id'] for c in j['items'] if c['course']['class_code'] == '761988'][0]
Q = f'/courses/{C1}/questions'; E = f'/courses/{C1}/exams'
now = lambda **k: (dt.datetime.now(dt.timezone.utc) + dt.timedelta(**k)).strftime('%Y-%m-%dT%H:%M:%SZ')
def mkq(title, approve=True, typ='MCQ_SINGLE'):
    b = dict(type=typ, title=title, topic='QC-exam', stem='s') | (dict(value=True) if typ == 'TRUE_FALSE' else dict(options=[{'body': 'a'}, {'body': 'b'}], correct=[0]))
    q = call('POST', Q, T, b)[1]
    if approve:
        call('PUT', Q + '/' + q['id'] + '/review', T, dict(decision='REQUEST', version=q['version']))
        g = call('GET', Q + '/' + q['id'], T)[1]; call('PUT', Q + '/' + q['id'] + '/review', TA, dict(decision='APPROVE', version=g['version']))
    return q['id']
def mke(**k):
    b = dict(title='QC bài thi', opens_at=now(days=30), closes_at=now(days=30, hours=1), duration_minutes=45); b.update(k)
    s, j, t = call('POST', E, T, b, idem()); return s, j
def ge(eid, tok=T): return call('GET', E + '/' + eid, tok)[1]
print('== AC1')
s, j = mke(); eid = j.get('id'); print('01', s, j.get('status'), j.get('shuffle_questions'), j.get('shuffle_options'), j.get('max_score'), j.get('rounding_step'), j.get('multi_scoring'), j.get('appeal_days'), j.get('version'))
for n, b in [('title rỗng', dict(title='')), ('title 121', dict(title='x' * 121)), ('instr 4001', dict(instructions='x' * 4001)), ('dur 4', dict(duration_minutes=4)), ('dur 301', dict(duration_minutes=301)), ('dur > khung', dict(duration_minutes=90))]:
    s, j = mke(**b); print('02', n, s, codes(j))
for n, b in [('instr 4000', dict(instructions='x' * 4000)), ('dur 5', dict(duration_minutes=5)), ('dur 300', dict(duration_minutes=300, closes_at=now(days=30, hours=6)))]:
    s, j = mke(**b); print('02 biên', n, s)
s, j = mke(opens_at='2026-12-01T08:00:00+07:00', closes_at='2026-12-01T09:00:00+07:00'); a = j.get('opens_at'); s, j2 = mke(opens_at='2026-12-01T01:00:00Z', closes_at='2026-12-01T02:00:00Z'); print('03 múi giờ +07 vs Z cùng thời điểm:', a == j2.get('opens_at'), a)
print('03 TA tạo', mke()[0], 'SV', call('POST', E, SV, dict(title='x'), idem())[0], 'ADMIN', call('POST', E, AD, dict(title='x'), idem())[0], 'ngoài lớp (SVB ở lớp 761987) tạo ở C2', call('POST', f'/courses/{C2}/exams', T, dict(title='x', opens_at=now(days=30), closes_at=now(days=30, hours=1), duration_minutes=45), idem())[0])
k = idem(); b = dict(title='idem', opens_at=now(days=30), closes_at=now(days=30, hours=1), duration_minutes=45)
r1 = call('POST', E, T, b, k); r2 = call('POST', E, T, b, k); r3 = call('POST', E, T, b | dict(title='khác'), k); print('04 cùng key cùng id:', r1[1].get('id') == r2[1].get('id'), 'khác thân:', r3[0], codes(r3[1]))
print('== AC2')
ok = [mkq('ok%d' % i) for i in range(5)]; code_q = None
s, j, _ = call('POST', Q, T, dict(type='CODE', title='code', topic='QC', stem='s')); cq = j['id']
call('PUT', Q + '/' + cq + '/code', T, dict(version=j['version'], languages=['c11'], reference=dict(language='c11', source='#include <stdio.h>\nint main(){int a,b;scanf("%d %d",&a,&b);printf("%d\\n",a+b);}')))
call('POST', Q + '/' + cq + '/testcases', T, dict(name='t', input='1 2\n', expected='3\n', is_sample=True))
v = call('POST', Q + '/' + cq + '/reference/verify', T, {}, idem())[1]
for _ in range(60):
    if call('GET', '/jobs/' + v['job_id'], T)[1].get('status') in ('SUCCEEDED', 'FAILED'): break
    time.sleep(1)
g = call('GET', Q + '/' + cq, T)[1]; call('PUT', Q + '/' + cq + '/review', TA, dict(decision='REQUEST', version=g['version'])); g = call('GET', Q + '/' + cq, T)[1]; print('   duyệt câu code', call('PUT', Q + '/' + cq + '/review', TA, dict(decision='APPROVE', version=g['version']))[0])
e = ge(eid); it = lambda ids: dict(items=[dict(question_id=q, points='1') for q in ids], version=ge(eid)['version'])
s, j, _ = call('PUT', E + '/' + eid + '/items', T, it(ok)); print('06 5 MCQ', s, ge(eid).get('kind'), [x['position'] for x in ge(eid).get('items', [])])
s, j, _ = call('PUT', E + '/' + eid + '/items', T, it(ok[::-1] + [cq])); g = ge(eid); print('06 đảo + CODE', s, g.get('kind'), [x['question_id'] for x in g['items']][:2] == ok[::-1][:2], 'số mục', len(g['items']))
dr = mkq('nháp', approve=False); pend = mkq('pend', approve=False); call('PUT', Q + '/' + pend + '/review', T, dict(decision='REQUEST', version=1)); rej = mkq('rej', approve=False)
call('PUT', Q + '/' + rej + '/review', T, dict(decision='REQUEST', version=1)); call('PUT', Q + '/' + rej + '/review', T, dict(decision='REJECT', version=call('GET', Q + '/' + rej, T)[1]['version']))
arch = mkq('arch'); call('POST', Q + '/' + arch + '/archive', T, {})
other = call('POST', f'/courses/{C2}/questions', T, dict(type='TRUE_FALSE', title='c2', topic='x', stem='s', value=True))[1]['id']
for n, ids in [('DRAFT', [dr]), ('PENDING', [pend]), ('REJECTED', [rej]), ('lưu trữ', [arch]), ('lớp khác', [other]), ('trùng', [ok[0], ok[0]]), ('rỗng', [])]:
    s, j, _ = call('PUT', E + '/' + eid + '/items', T, it(ids)); print('07', n, s, codes(j))
for pts in ('0.01', '100', '0', '100.01'):
    s, j, _ = call('PUT', E + '/' + eid + '/items', T, dict(items=[dict(question_id=ok[0], points=pts)], version=ge(eid)['version'])); print('07 points', pts, s, codes(j))
call('PUT', E + '/' + eid + '/items', T, it(ok + [cq]))
print('== AC3')
p1 = call('GET', E + '/' + eid + '/preview', T); p2 = call('GET', E + '/' + eid + '/preview', T)
print('10 preview', p1[0], 'preview:', p1[1].get('preview'), 'thứ tự khác giữa hai lần:', [q.get('id') for q in p1[1].get('items', p1[1].get('questions', []))] != [q.get('id') for q in p2[1].get('items', p2[1].get('questions', []))], 'khoá:', list(p1[1].keys())[:10])
print('11 SV preview', call('GET', E + '/' + eid + '/preview', SV)[0], 'ADMIN', call('GET', E + '/' + eid + '/preview', AD)[0], 'GV bắt đầu lượt', call('POST', E + '/' + eid + '/attempts', T, {}, idem())[0])
print('== AC4')
print('16 TA lên lịch', call('POST', E + '/' + eid + '/schedule', TA, {})[0])
s, j = mke(opens_at=now(seconds=-5), closes_at=now(hours=1)); print('14 opens_at quá khứ (tạo)', s, codes(j))
s, j = mke(opens_at=now(seconds=59), closes_at=now(hours=1), duration_minutes=5); e59 = j.get('id'); print('15 tạo với opens=+59s', s, codes(j))
