from q6lib import *
import re
st = json.load(open('/tmp/q3/q8e.json')); pl = json.load(open('/tmp/q3/q8epl.json')); eid = st['eid']; A = E + '/' + eid; c = dt.datetime.fromisoformat(st['c'])
while dt.datetime.now(dt.timezone.utc) < c + dt.timedelta(seconds=45): time.sleep(2)
print('22 sau đóng 45 s:', db(f"select status, publish_hold from exams where id='{eid}'"), '| lượt', db(f"select status,count(*) from exam_attempts where exam_id='{eid}' group by 1"))
s, j, t = call('GET', '/me/today', T); m = re.search(r'\{[^{}]*EXAM_PUBLISH_HOLD[^{}]*\}', t); print('22 Today GV:', m.group(0)[:300] if m else 'KHÔNG CÓ', '| TA', 'EXAM_PUBLISH_HOLD' in call('GET', '/me/today', TA)[2])
sv = login('sv17'); aid = pl['sv17'][0]; print('29 result trước công bố', call('GET', A + f'/attempts/{aid}/result', sv)[0:1], codes(call('GET', A + f'/attempts/{aid}/result', sv)[1]), call('GET', A + f'/attempts/{aid}/result', sv)[2][:120])
g = call('GET', A, T)[1]; print('23 TA hold', call('PUT', A + '/publish-hold', TA, dict(hold=False, version=g['version']))[0], '| sai version', call('PUT', A + '/publish-hold', T, dict(hold=False, version=99))[0:1])
t0 = time.time(); print('22 bỏ hoãn', call('PUT', A + '/publish-hold', T, dict(hold=False, version=g['version']))[0]); 
for _ in range(30):
    if db(f"select status from exams where id='{eid}'") == 'PUBLISHED': break
    time.sleep(1)
print('22 công bố sau bỏ hoãn: %.0fs' % (time.time() - t0), db(f"select status from exams where id='{eid}'"), '| outbox', db(f"select count(*) from outbox where topic='exam.published' and payload::text like '%{eid}%'"))
print('23 hold sau PUBLISHED', call('PUT', A + '/publish-hold', T, dict(hold=True, version=call('GET', A, T)[1]['version']))[0:1], codes(call('PUT', A + '/publish-hold', T, dict(hold=True, version=call('GET', A, T)[1]['version']))[1]))
s, j, t = call('GET', A + f'/attempts/{aid}/result', sv); mc = j['items'][0]
print('26 reveal=false: answer', mc.get('answer'), 'explanation', mc.get('explanation'), 'correct', mc.get('correct'), 'mine', str(mc.get('mine'))[:40], '| exam.reveal_answers', j['exam'].get('reveal_answers'), '| canary explanation trong thân:', 'QCCANARY' in t)
print('34 appeal_days=0:', call('POST', A + f'/attempts/{aid}/appeal', sv, dict(reason='x'), idem())[0:1], codes(call('POST', A + f'/attempts/{aid}/appeal', sv, dict(reason='x'), idem())[1]))
# 60 leak scan: mọi GET SV ở exam 1 (PUBLISHED, reveal bật) và exam 2 (reveal tắt) với canary đáp án/giải thích/test ẩn
st1 = json.load(open('/tmp/q3/q8.json')); pl1 = json.load(open('/tmp/q3/q8plan.json')); A1 = E + '/' + st1['eid']
hid = db("select name||'|'||input||'|'||expected from code_tests where name like 'QC-HID-%' limit 1") if 0 else ''
ref = db("select 'x'") if 0 else ''
exp_can = [r for r in db("select distinct explanation from question_bank where explanation like 'QCCANARY%'").split('\n')]
print('60 canary giải thích:', exp_can)
def scan(tag, tok, path):
    s, j, t = call('GET', path, tok); h = HID in t or REF in t; return s, h, t
paths = {'exam2': (sv, [A, A + '/attempts/mine', A + f'/attempts/{aid}/result']), 'exam1': (login('sv04'), [A1, A1 + '/attempts/mine', A1 + f'/attempts/{pl1["sv04"]["aid"]}/result'])}
for k, (tk, ps) in paths.items():
    for p in ps + ['/me/exam-lock', '/me/today', f'/courses/{C1}/exams']:
        s, h, t = scan(k, tk, p); print('60', k, p.split('/')[-1][:12] or 'exam', s, 'QC-HID/REF:', h, '| giải thích canary:', [cn[:12] for cn in exp_can if cn and cn in t], '| đáp án đúng khoá:', '"answer_key"' in t)
print('60 hidden test tên/đầu vào trong result exam1:', any(x in call('GET', A1 + f'/attempts/{pl1["sv04"]["aid"]}/result', login('sv04'))[2] for x in ['hid-big', HID]))
# 63 ma trận quyền
rt = {'results': ('GET', '/results'), 'stats': ('GET', '/stats'), 'csv': ('GET', '/results.csv'), 'appeals': ('GET', '/appeals')}
toks = {'GV': T, 'TA': TA, 'SV': login('sv04'), 'SVngoài': login('sv20'), 'ADMIN': AD}
M = {n: {w: call(m, A1 + p, tk)[0] for w, tk in toks.items()} for n, (m, p) in rt.items()}
for n, r in M.items(): print('63', n, r)
