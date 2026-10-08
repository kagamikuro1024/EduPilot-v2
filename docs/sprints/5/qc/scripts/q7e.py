from q6lib import *
import random
random.seed(7)
OPS = ['+', '-', '*', '^', '|', '&', '<<', '>>', '%']; CTL = ['if (x %s %d) { y %s= x; }', 'while (x %s %d) { x = x %s 3; }', 'for (int k = 0; k < %d; k++) { y %s= k %s x; }']
def prog(i):
    random.seed(1000 + i); body = []
    for _ in range(random.randint(8, 14)):
        t = random.choice([0, 1, 2])
        if t == 0: body.append('if (x %s %d) { y %s= x %s %d; }' % (random.choice(['<', '>', '==', '!=', '<=']), random.randint(1, 99), random.choice(OPS), random.choice(OPS), random.randint(1, 9)))
        elif t == 1: body.append('for (int k = 0; k < %d; k++) { y %s= k %s x; if (y > %d) break; }' % (random.randint(2, 30), random.choice(OPS), random.choice(OPS), random.randint(5, 999)))
        else: body.append('x = x %s (y %s %d); while (x > %d) { x %s= 2; }' % (random.choice(OPS), random.choice(OPS), random.randint(1, 50), random.randint(10, 500), random.choice(OPS)))
    return '#include <iostream>\nint f%d(int x){ int y = 0; %s return y; }\nint main(){ int a; std::cin >> a; std::cout << f%d(a); }\n' % (i, ' '.join(body), i)
P = [prog(i) for i in range(5)]; P += [P[0].replace('f0', 'cp%d' % k).replace('int y', 'int w').replace('y ', 'w ').replace(' y', ' w') for k in range(3)]  # sao chép đổi tên
qid = mkcode(2); print('qid', qid); time.sleep(10); g = call('GET', Q + '/' + qid, T)[1]
if g.get('review_status') != 'APPROVED':
    call('PUT', Q + '/' + qid + '/review', T, dict(decision='REQUEST', version=g['version'])); g = call('GET', Q + '/' + qid, T)[1]; print('approve', call('PUT', Q + '/' + qid + '/review', TA, dict(decision='APPROVE', version=g['version']))[0:1])
o = dt.datetime.now(dt.timezone.utc) + dt.timedelta(seconds=70); c = o + dt.timedelta(seconds=300)
s, j = mke(title='QC7 flag', opens_at=o.strftime('%Y-%m-%dT%H:%M:%SZ'), closes_at=c.strftime('%Y-%m-%dT%H:%M:%SZ'), duration_minutes=5); eid = j['id']; A = E + '/' + eid
print('items', call('PUT', A + '/items', T, dict(items=[dict(question_id=qid, points='10')], version=ge(eid)['version']))[0], call('POST', A + '/schedule', T, {})[0:1])
json.dump(dict(eid=eid, c=c.isoformat()), open('/tmp/q3/q7e.json', 'w'))
time.sleep(max(0, (o - dt.datetime.now(dt.timezone.utc)).total_seconds()) + 3)
AT = []
for i, src in enumerate(P):
    u = 'sv%02d' % (i + 4); tok = login(u); r = call('POST', A + '/attempts', tok, {}, idem())
    if r[0] not in (200, 201): print(u, r[0]); continue
    aid = r[1]['attempt']['id']; iid = r[1]['items'][0]['item_id']; H = {'X-Exam-Tab': str(uuid.uuid4())}; call('POST', A + f'/attempts/{aid}/takeover', tok, {}, H)
    call('POST', A + f'/attempts/{aid}/code/{iid}/submit', tok, dict(language='cpp17', source=src), {**H, **idem()}); AT.append((tok, aid, H))
time.sleep(6)
for tok, aid, H in AT: call('POST', A + f'/attempts/{aid}/submit', tok, {}, {**H, **idem()})
print('xong', len(AT), 'lượt; đóng', c.isoformat())
