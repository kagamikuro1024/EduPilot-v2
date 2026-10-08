from q6lib import *
import importlib.util
sp = importlib.util.spec_from_file_location('w', 'p507-winnow.py'); W = importlib.util.module_from_spec(sp); sp.loader.exec_module(W)
st = json.load(open('/tmp/q3/q7.json')); eid = st['eid']; A = E + '/' + eid
while dt.datetime.now(dt.timezone.utc) < dt.datetime.fromisoformat(st['c']) + dt.timedelta(seconds=5): time.sleep(2)
for _ in range(60):
    n = db(f"select count(*) from similarity_reports where exam_id='{eid}'")
    if n != '0': break
    time.sleep(2)
print('34 sau đóng bài: số dòng similarity_reports', n, '| run_id khác nhau', db(f"select count(distinct run_id) from similarity_reports where exam_id='{eid}'"), '| dấu vết queued', db(f"select count(*) from audit_log where action='exam.similarity.queued' and target_id='{eid}'") if 0 else db("select count(*) from audit_log where action='exam.similarity.queued'"))
s, j, t = call('GET', A + '/similarity', T); print('GET similarity', s, json.dumps(j, ensure_ascii=False)[:900])
att = json.load(open('/tmp/q3/q7at.json')); who = {v[0]: u for u, v in att.items()}
for it in (j.get('items') or []):
    print('  ', it.get('score'), it.get('flagged'), {k: it[k] for k in it if k in ('student_a', 'student_b', 'a', 'b', 'review_state')} if 0 else json.dumps(it, ensure_ascii=False)[:230])
print('db pairs:'); print(db(f"select ua.email, ub.email, r.score, r.flagged, r.shared_fingerprints, r.review_state from similarity_reports r join exam_attempts aa on aa.id=r.attempt_a join users ua on ua.id=aa.student_id join exam_attempts ab on ab.id=r.attempt_b join users ub on ub.id=ab.student_id where r.exam_id='{eid}' order by r.score desc limit 12"))
