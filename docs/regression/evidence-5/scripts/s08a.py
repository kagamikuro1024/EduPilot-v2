import os; os.environ['SUITE']='PE-08'
from r5lib import *
from decimal import Decimal, ROUND_HALF_UP
import glob
def S(m, p, tk=T, b=None, h=None, **k): return call(m, p, tk, b, h=h, **k)
D = Decimal
SMALL = '#include <cstdio>\nint main(){long long a,b;scanf("%lld %lld",&a,&b);printf("%lld\\n",a<50?a+b:0);}\n'
CODES = {'ok': (SUM, D(6)), 'small': (SMALL, D(4)), 'wa': (SUM.replace('a+b', 'a+b+1'), D(0)), 'ce': ('int main(){ x(); }', D(0)), 'none': (None, D(0))}
SETS = [(0, 1, 2), (0, 1), (0,), (0, 3), (3,), (0, 1, 3), (), (0, 1, 2, 3), (0, 1, 2, 3, 4), (3, 4), (0, 3, 4), (0, 1, 3, 4)]
def expected(i):
    q0 = D(1) if i % 2 == 0 else D(0); tp = len([x for x in SETS[i] if x < 3]); fp = len(SETS[i]) - tp
    q1 = D(3) * max(D(0), D(tp - fp) / D(3)); q2 = D(1) if i % 3 != 0 else D(0)
    ck = ['ok', 'small', 'wa', 'ce', 'none'][i % 5] if i < 8 else 'p507'; q3 = D(5) * CODES[ck][1] / D(6) if ck != 'p507' else D(0)
    return q0, q1, q2, q3, ck, q0 + q1 + q2 + q3
mq = lambda i, typ, n, cor: mkq(i, typ, n, cor)
q0 = mkq(0, 'MCQ_SINGLE', 4, (1,)); q1 = mkq(1, 'MCQ_MULTI', 5, (0, 1, 2)); q2 = mkq(2, 'TRUE_FALSE'); qc = mkcode(88)
eid, _ = mkexam(open_s=1000, win=3000, dur=5, items=[q0, q1, q2, qc], schedule=False, title='R5 F19', reveal_answers=True, appeal_days=7)
g = S('GET', f'{E}/{eid}')[1]
pts = ['1', '3', '1', '5']; its = [q0, q1, q2, qc]
s, jj, _ = S('PUT', f'{E}/{eid}/items', b=dict(items=[dict(question_id=q, points=p) for q, p in zip(its, pts)], version=g['version'])); assert s == 200, jj
OPEN = 25; WIN = 200
g = S('GET', f'{E}/{eid}')[1]; s, jj, _ = S('PUT', f'{E}/{eid}', b=dict(title='R5 F19', opens_at=iso(seconds=OPEN), closes_at=iso(seconds=OPEN + WIN), duration_minutes=3, reveal_answers=True, appeal_days=7, version=g['version'])); assert s == 200, jj
s, jj, _ = S('POST', f'{E}/{eid}/schedule', b={}); assert s == 200, jj
c = dt.datetime.fromisoformat(db(f"select to_char(closes_at at time zone 'utc','YYYY-MM-DD\"T\"HH24:MI:SS+00:00') from exams where id='{eid}'")); o = dt.datetime.fromisoformat(db(f"select to_char(opens_at at time zone 'utc','YYYY-MM-DD\"T\"HH24:MI:SS+00:00') from exams where id='{eid}'"))
print('eid', eid, 'đóng lúc', c); A = f'{E}/{eid}'
sleep_until(o + dt.timedelta(seconds=2))
rows = db(f"select i.id, q.title from exam_items i join question_bank q on q.id=i.question_id where i.exam_id='{eid}'").split('\n'); idx = {r.split('|')[0]: r.split('|')[1] for r in rows}
src_p = {n: open(f'p507-src/{n}.cpp').read() for n in ('A-original', 'B-renamed-reformatted', 'C-different-algorithm', 'Aprime-reordered')}
p507 = [src_p['A-original'], src_p['B-renamed-reformatted'], src_p['C-different-algorithm'], src_p['Aprime-reordered']]
plan = {}
for i in range(12):
    u = sv(i + 1); t = login(u); r = S('POST', A + '/attempts', t, {}, idem()); aid = r[1]['attempt']['id']; H = {'X-Exam-Tab': str(uuid.uuid4())}; S('POST', A + f'/attempts/{aid}/takeover', t, {}, H)
    a0, a1, a2, a3, ck, tot = expected(i); ans = []; cit = None
    for it in r[1]['items']:
        ttl = idx[it['item_id']]
        if ttl.endswith('-0'): ans.append(dict(item_id=it['item_id'], answer=dict(option_ids=[[o_['id'] for o_ in it['options'] if o_['body'].startswith('Đáp án %d ' % k)][0] for k in [1 if a0 else 0]])))
        elif ttl.endswith('-1') and SETS[i]: ans.append(dict(item_id=it['item_id'], answer=dict(option_ids=[[o_['id'] for o_ in it['options'] if o_['body'].startswith('Đáp án %d ' % k)][0] for k in SETS[i]])))
        elif ttl.endswith('-99'): ans.append(dict(item_id=it['item_id'], answer=dict(value=bool(a2))))
        elif it['type'] == 'CODE': cit = it['item_id']
    sv_ = S('PUT', A + f'/attempts/{aid}/answers', t, dict(items=ans), h=H)[0]
    src = p507[i - 8] if i >= 8 else CODES[ck][0]
    if src: sub = S('POST', A + f'/attempts/{aid}/code/{cit}/submit', t, dict(language='cpp17', source=src), {**H, **idem()})[0]
    else: sub = '-'
    plan[u] = dict(aid=aid, exp=str(tot.quantize(D('0.01'), ROUND_HALF_UP)), parts=[str(x) for x in (a0, a1, a2, a3)], ck=ck, tab=H['X-Exam-Tab']); print(u, 'save', sv_, 'code', sub, ck, plan[u]['exp'], flush=True)
json.dump(dict(eid=eid, close=c.isoformat(), plan=plan), open('/tmp/qc2-c5/s08.json', 'w'))
time.sleep(8)
for i in range(10):
    u = sv(i + 1); r = S('POST', A + f'/attempts/{plan[u]["aid"]}/submit', login(u), {}, {'X-Exam-Tab': plan[u]['tab'], **idem()}); print(u, 'nộp', r[0], end='; ')
print('\nr5s11, r5s12: để hết giờ (đóng ép nộp)')
