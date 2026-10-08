from q6lib import *
import os, importlib.util
sp = importlib.util.spec_from_file_location('w', 'p507-winnow.py'); W = importlib.util.module_from_spec(sp); sp.loader.exec_module(W)
D = 'p507-src/'; rd = lambda n: open(D + n, encoding='utf-8').read()
SKEL = '''#include <bits/stdc++.h>
using namespace std;
long long scan_input(vector<long long>& v, int n) {
    for (int i = 0; i < n; i++) { long long x; cin >> x; v.push_back(x); }
    long long s = 0; for (auto y : v) { s += y; if (s > 1000000007LL) s -= 1000000007LL; }
    return s;
}
'''
def filler(i): return SKEL + 'int main(){ vector<long long> v; int n=%d; long long r=scan_input(v,n); ' % (i + 3) + ''.join('r = r * %d + v.size() %% %d; ' % (7 + i * k, 3 + k) for k in range(1, 4 + i)) + 'for (int j=0;j<%d;j++) cout << r + j * %d << "\\n"; }\n' % (i + 2, i + 5)
SUBS = {'sv16': rd('A-original.cpp'), 'sv17': rd('B-renamed-reformatted.cpp'), 'sv18': rd('Aprime-reordered.cpp'), 'sv19': rd('C-different-algorithm.cpp'), 'sv20': rd('short.cpp'),
        'sv21': filler(1), 'sv22': filler(2), 'sv23': filler(3), 'sv24': filler(4)}
qid = mkcode(1, {'cpp17': SKEL}); print('qid', qid)
time.sleep(10); g = call('GET', Q + '/' + qid, T)[1]
if g.get('review_status') != 'APPROVED':
    call('PUT', Q + '/' + qid + '/review', T, dict(decision='REQUEST', version=g['version'])); g = call('GET', Q + '/' + qid, T)[1]; print('approve', call('PUT', Q + '/' + qid + '/review', TA, dict(decision='APPROVE', version=g['version']))[0:1])
o = dt.datetime.now(dt.timezone.utc) + dt.timedelta(seconds=70); c = o + dt.timedelta(seconds=300)
s, j = mke(title='QC7 sim', opens_at=o.strftime('%Y-%m-%dT%H:%M:%SZ'), closes_at=c.strftime('%Y-%m-%dT%H:%M:%SZ'), duration_minutes=5); print('mke', s, j if s != 201 else ''); eid = j['id']; A = E + '/' + eid
print('items', call('PUT', A + '/items', T, dict(items=[dict(question_id=qid, points='10')], version=ge(eid)['version']))[0], 'sched', call('POST', A + '/schedule', T, {})[0:1])
json.dump(dict(eid=eid, qid=qid, c=c.isoformat()), open('/tmp/q3/q7.json', 'w'))
time.sleep(max(0, (o - dt.datetime.now(dt.timezone.utc)).total_seconds()) + 3)
AT = {}
for u, src in SUBS.items():
    tok = login(u); r = call('POST', A + '/attempts', tok, {}, idem())
    if r[0] not in (200, 201): print(u, 'start', r[0], codes(r[1])); continue
    aid = r[1]['attempt']['id']; iid = r[1]['items'][0]['item_id']; H = {'X-Exam-Tab': str(uuid.uuid4())}; call('POST', A + f'/attempts/{aid}/takeover', tok, {}, H)
    s = call('POST', A + f'/attempts/{aid}/code/{iid}/submit', tok, dict(language='cpp17', source=src), {**H, **idem()}); AT[u] = (tok, aid, iid, H); print(u, 'submit', s[0], codes(s[1]))
time.sleep(8)
json.dump({u: [v[1], v[2]] for u, v in AT.items()}, open('/tmp/q3/q7at.json', 'w'))
for u, (tok, aid, iid, H) in AT.items(): print(u, 'nộp bài', call('POST', A + f'/attempts/{aid}/submit', tok, {}, {**H, **idem()})[0])
print('xong, đợi đóng bài lúc', c.strftime('%H:%M:%S'))
