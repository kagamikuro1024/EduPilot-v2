exec(open('q4_exams.py', encoding='utf-8').read().split("print('== AC1')")[0])
import subprocess
def db(sql): return subprocess.run(['docker','exec','edupilot100-postgres-1','psql','-U','edupilot','-d','edupilot','-tAF|','-c',sql],capture_output=True,text=True).stdout.strip()
CANARY = 'QCCANARY' + uuid.uuid4().hex[:8]
def mkq5(i, typ='MCQ_SINGLE', n=4, correct=(1,), expl=True):
    opts = [{'body': 'Đáp án %d của câu %d' % (k, i)} for k in range(n)]
    b = dict(type=typ, title='qc5-%d' % i, topic='QC5', stem='Câu %d?' % i, options=opts, correct=list(correct), explanation=CANARY + ' giải thích') if typ != 'TRUE_FALSE' else dict(type=typ, title='qc5-%d' % i, topic='QC5', stem='Câu %d đúng?' % i, value=True, explanation=CANARY)
    q = call('POST', Q, T, b)[1]; call('PUT', Q + '/' + q['id'] + '/review', T, dict(decision='REQUEST', version=q['version'])); g = call('GET', Q + '/' + q['id'], T)[1]
    call('PUT', Q + '/' + q['id'] + '/review', TA, dict(decision='APPROVE', version=g['version'])); return q['id']
def mkexam(opens_s, closes_s, dur, nq=6, **kw):
    o = dt.datetime.now(dt.timezone.utc) + dt.timedelta(seconds=opens_s); c = o + dt.timedelta(seconds=closes_s)
    s, j = mke(title=kw.pop('title', 'QC5 bài'), opens_at=o.strftime('%Y-%m-%dT%H:%M:%SZ'), closes_at=c.strftime('%Y-%m-%dT%H:%M:%SZ'), duration_minutes=dur, **kw); eid = j['id']
    ids = [mkq5(i, 'MCQ_SINGLE' if i % 3 else 'MCQ_MULTI', 4, (1,) if i % 3 else (0, 2)) for i in range(nq)]
    ids.append(mkq5(99, 'TRUE_FALSE'))
    s, jj, _ = call('PUT', E + '/' + eid + '/items', T, dict(items=[dict(question_id=q, points='1') for q in ids], version=ge(eid)['version']))
    s, jj, _ = call('POST', E + '/' + eid + '/schedule', T, {}); return eid, ids, o, c
