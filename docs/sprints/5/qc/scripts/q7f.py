from q6lib import *
import statistics, math
st = json.load(open('/tmp/q3/q7e.json')); eid = st['eid']; A = E + '/' + eid
while dt.datetime.now(dt.timezone.utc) < dt.datetime.fromisoformat(st['c']) + dt.timedelta(seconds=5): time.sleep(2)
for _ in range(60):
    if db(f"select count(*) from similarity_reports where exam_id='{eid}'") != '0': break
    time.sleep(2)
rows = [r.split('|') for r in db(f"select ua.email, ub.email, r.score, r.flagged from similarity_reports r join exam_attempts aa on aa.id=r.attempt_a join users ua on ua.id=aa.student_id join exam_attempts ab on ab.id=r.attempt_b join users ub on ub.id=ab.student_id where r.exam_id='{eid}' order by r.score desc limit 8").split('\n')]
print('35 cặp:', [(a.split('@')[0], b.split('@')[0], s, f) for a, b, s, f in rows]); print('35 số dòng / số cờ', db(f"select count(*), count(*) filter (where flagged) from similarity_reports where exam_id='{eid}'"))
s, j, _ = call('GET', A + '/similarity?flagged=true', T); print('flagged API', s, len(j['items']), [(i['score'], i['review_state']) for i in j['items']])
for nm, tok in [('GV', T), ('TA', TA), ('SV', SV)]:
    s, jj, t = call('GET', '/me/today', tok); print('40 today', nm, 'EXAM_SIMILARITY' in t)
s, jj, t = call('GET', '/me/today', T); import re; m = re.search(r'\{[^{}]*EXAM_SIMILARITY[^{}]*\}', t); print('   mục:', m.group(0)[:300] if m else t[:200])
for it in j['items']: print('review', call('PUT', A + f'/similarity/{it["id"]}/review', T, dict(state='CLEARED'))[0])
print('40 sau xử lý: today có EXAM_SIMILARITY', 'EXAM_SIMILARITY' in call('GET', '/me/today', T)[2])
