from q6lib import *
from decimal import Decimal, ROUND_HALF_UP
def mq(i, typ, n, correct, expl=CANARY):
    if typ == 'TRUE_FALSE': b = dict(type=typ, title='qc8-%d' % i, topic='QC8', stem='Câu %d đúng?' % i, value=True, explanation=expl)
    else: b = dict(type=typ, title='qc8-%d' % i, topic='QC8', stem='Câu %d?' % i, options=[{'body': 'Đáp án %d của câu %d' % (k, i)} for k in range(n)], correct=list(correct), explanation=expl)
    q = call('POST', Q, T, b)[1]; call('PUT', Q + '/' + q['id'] + '/review', T, dict(decision='REQUEST', version=q['version'])); g = call('GET', Q + '/' + q['id'], T)[1]
    call('PUT', Q + '/' + q['id'] + '/review', TA, dict(decision='APPROVE', version=g['version'])); return q['id']
def mkexam8(title, open_s=62, win=300, reveal=True, appeal_days=7, hold=False, **kw):
    qs = [mq(0, 'MCQ_SINGLE', 4, (1,)), mq(1, 'MCQ_MULTI', 5, (0, 1, 2)), mq(2, 'TRUE_FALSE', 0, ())]
    qc = mkcode(8); time.sleep(10); g = call('GET', Q + '/' + qc, T)[1]
    if g.get('review_status') != 'APPROVED':
        call('PUT', Q + '/' + qc + '/review', T, dict(decision='REQUEST', version=g['version'])); g = call('GET', Q + '/' + qc, T)[1]; call('PUT', Q + '/' + qc + '/review', TA, dict(decision='APPROVE', version=g['version']))
    o = dt.datetime.now(dt.timezone.utc) + dt.timedelta(seconds=open_s); c = o + dt.timedelta(seconds=win)
    s, j = mke(title=title, opens_at=o.strftime('%Y-%m-%dT%H:%M:%SZ'), closes_at=c.strftime('%Y-%m-%dT%H:%M:%SZ'), duration_minutes=5, reveal_answers=reveal, appeal_days=appeal_days, **kw); eid = j['id']
    if s != 201: print('mke', s, j)
    s, jj, _ = call('PUT', E + '/' + eid + '/items', T, dict(items=[dict(question_id=qs[0], points='1'), dict(question_id=qs[1], points='3'), dict(question_id=qs[2], points='1'), dict(question_id=qc, points='5')], version=ge(eid)['version'])); print('items', s, codes(jj))
    print('sched', call('POST', E + '/' + eid + '/schedule', T, {})[0:1]); return eid, qs, qc, o, c
SMALL = '#include <cstdio>\nint main(){long long a,b;scanf("%lld %lld",&a,&b);printf("%lld\\n",a<50?a+b:0);}\n'
CODES = {'ok': (SUMSRC, 4), 'small': (SMALL, 3), 'wa': (SUMSRC.replace('a+b', 'a+b+1'), 0), 'ce': ('int main(){ x(); }', 0), 'none': (None, 0)}
SETS = [(0, 1, 2), (0, 1), (0,), (0, 3), (3,), (0, 1, 3), (), (0, 1, 2, 3), (0, 1, 2, 3, 4), (3, 4), (0, 3, 4), (0, 1, 3, 4)]
def tp_fp(s): tp = len([x for x in s if x < 3]); return tp, len(s) - tp
def D(x): return Decimal(str(x))
def expected(i, mode='PARTIAL'):
    q0 = 1 if i % 2 == 0 else 0; tp, fp = tp_fp(SETS[i]); q1 = D(3) * max(D(0), D(tp - fp) / D(3)); q2 = 1 if i % 3 != 0 else 0
    ck = ['ok', 'small', 'wa', 'ce', 'none'][i % 5]; q3 = D(5) * D(CODES[ck][1]) / D(4)
    return q0, q1, q2, q3, ck, (D(q0) + q1 + D(q2) + q3)
if __name__ == '__main__':
    eid, qs, qc, o, c = mkexam8('QC8 F19'); A = E + '/' + eid; json.dump(dict(eid=eid, qs=qs, qc=qc, c=c.isoformat()), open('/tmp/q3/q8.json', 'w'))
    time.sleep(max(0, (o - dt.datetime.now(dt.timezone.utc)).total_seconds()) + 3)
    rows = db(f"select i.id, q.title from exam_items i join question_bank q on q.id=i.question_id where i.exam_id='{eid}'").split('\n'); idx = {r.split('|')[0]: int(r.split('|')[1].split('-')[1]) for r in rows}
    plan = {}
    for i in range(12):
        u = 'sv%02d' % (i + 4); tok = login(u); r = call('POST', A + '/attempts', tok, {}, idem()); aid = r[1]['attempt']['id']; H = {'X-Exam-Tab': str(uuid.uuid4())}; call('POST', A + f'/attempts/{aid}/takeover', tok, {}, H)
        q0, q1, q2, q3, ck, tot = expected(i); ans = []; cit = None
        for it in r[1]['items']:
            n = idx[it['item_id']]; ob = lambda k: [o2['id'] for o2 in it['options'] if o2['body'].startswith('Đáp án %d ' % k)][0] if it.get('options') else None
            if n == 0: ans.append(dict(item_id=it['item_id'], answer=dict(option_ids=[ob(1 if q0 else 0)])))
            elif n == 1 and SETS[i]: ans.append(dict(item_id=it['item_id'], answer=dict(option_ids=[ob(k) for k in SETS[i]])))
            elif n == 2: ans.append(dict(item_id=it['item_id'], answer=dict(value=bool(q2))))
            elif it['type'] == 'CODE': cit = it['item_id']
        print(u, 'save', call('PUT', A + f'/attempts/{aid}/answers', tok, dict(items=ans), H)[0], end=' ')
        if CODES[ck][0]: print('code', call('POST', A + f'/attempts/{aid}/code/{cit}/submit', tok, dict(language='cpp17', source=CODES[ck][0]), {**H, **idem()})[0], end=' ')
        plan[u] = dict(aid=aid, exp=str(tot.quantize(D('0.01'), ROUND_HALF_UP)), parts=[str(q0), str(q1), str(q2), str(q3)], ck=ck); print(ck, plan[u]['exp'])
        plan[u]['tab'] = H['X-Exam-Tab']
        if i < 10: pass
    time.sleep(6)
    for u in list(plan)[:10]:
        tok = login(u); print(u, 'nộp', call('POST', A + f'/attempts/{plan[u]["aid"]}/submit', tok, {}, {'X-Exam-Tab': plan[u]['tab'], **idem()})[0], end='; ')
    # sv14, sv15: để hết giờ đóng (forced CLOSED); sv16: bắt đầu nhưng không làm
    print(); json.dump(plan, open('/tmp/q3/q8plan.json', 'w')); print('xong; đóng lúc', c.isoformat())
