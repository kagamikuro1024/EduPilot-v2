"""QC2 smoke sprint 5 — thư viện chung (stack ep-r). Ghi kết quả từng TC vào evidence-5/results.jsonl."""
import json, ssl, urllib.request, urllib.error, uuid, io, zipfile, time, subprocess, os, datetime as dt, re
CTX = ssl._create_unverified_context()
BASE = os.environ.get('QC_API', 'https://localhost:1143/api/v1'); PW = 'Edupilot#Seed-2026'
OUT = '/Users/kuro/Documents/TA_Agent_qc2/docs/regression/evidence-5/results.jsonl'
_tok = {}
def call(m, p, tok=None, body=None, h=None, raw=None, ctype=None, timeout=60):
    hd = {'Origin': 'https://localhost:1143', 'X-Forwarded-For': '198.51.100.%d' % (uuid.uuid4().int % 250 + 1)}
    if tok: hd['Authorization'] = 'Bearer ' + tok
    data = raw
    if body is not None: data = json.dumps(body).encode(); hd['Content-Type'] = 'application/json'
    if ctype: hd['Content-Type'] = ctype
    hd.update(h or {})
    rq = urllib.request.Request(BASE + p, data=data, method=m, headers=hd)
    try: r = urllib.request.urlopen(rq, context=CTX, timeout=timeout); s, t, hh = r.status, r.read().decode(), dict(r.headers)
    except urllib.error.HTTPError as e: s, t, hh = e.code, e.read().decode(), dict(e.headers)
    try: j = json.loads(t)
    except Exception: j = {}
    call.last_headers = hh
    return s, j, t
TOKF = '/tmp/qc2-c5/toks.json'
def _tokfile():
    try: return json.load(open(TOKF))
    except Exception: return {}
def login(u, fresh=False):
    if u in _tok and not fresh: return _tok[u]
    tf = _tokfile()
    if u in tf and not fresh and tf[u][1] > time.time() + 120: _tok[u] = tf[u][0]; return _tok[u]
    for i in range(5):
        s, j, t = call('POST', '/auth/login', body={'email': u + '@edupilot.local', 'password': PW}, h=idem())
        if s == 200:
            _tok[u] = j['access_token']; tf = _tokfile(); tf[u] = [_tok[u], time.time() + 780]; json.dump(tf, open(TOKF, 'w')); return _tok[u]
        time.sleep(2 + i * 3)
    raise Exception('login %s %s %s' % (u, s, t[:200]))
def idem(): return {'Idempotency-Key': str(uuid.uuid4())}
def codes(j):
    det = j.get('details'); out = {d.get('code') for d in det if isinstance(d, dict)} if isinstance(det, list) else set()
    if isinstance(j.get('code'), str): out.add(j['code'])
    return sorted(c for c in out if c)
def reason(j):
    d = j.get('details'); return d.get('reason') if isinstance(d, dict) else None
def db(sql): return subprocess.run(['docker','exec','ep-r-postgres-1','psql','-U','edupilot','-d','edupilot','-tAF|','-c',sql],capture_output=True,text=True).stdout.strip()
def rds(*a): return subprocess.run(['docker','exec','ep-r-redis-1','redis-cli',*a],capture_output=True,text=True).stdout.strip()
def mkzip(files):
    b = io.BytesIO()
    with zipfile.ZipFile(b, 'w', zipfile.ZIP_DEFLATED) as z:
        for n, c in files.items(): z.writestr(n, c)
    return b.getvalue()
def multipart(fields, fname, content, ctype='application/zip'):
    bd = '----qc' + uuid.uuid4().hex; out = b''
    for k, v in fields.items(): out += ('--%s\r\nContent-Disposition: form-data; name="%s"\r\n\r\n%s\r\n' % (bd, k, v)).encode()
    out += ('--%s\r\nContent-Disposition: form-data; name="file"; filename="%s"\r\nContent-Type: %s\r\n\r\n' % (bd, fname, ctype)).encode() + content + ('\r\n--%s--\r\n' % bd).encode()
    return out, 'multipart/form-data; boundary=' + bd
def iso(**k): return (dt.datetime.now(dt.timezone.utc) + dt.timedelta(**k)).strftime('%Y-%m-%dT%H:%M:%SZ')
def sleep_until(t): time.sleep(max(0, (t - dt.datetime.now(dt.timezone.utc)).total_seconds()))
# --- ghi kết quả
SUITE = os.environ.get('SUITE', 'x')
def rec(tc, ok, note=''):
    v = 'P' if ok is True else 'F' if ok is False else ok
    open(OUT, 'a').write(json.dumps(dict(tc=tc, r=v, note=str(note)[:300], suite=SUITE), ensure_ascii=False) + '\n')
    print(('PASS ' if v == 'P' else 'FAIL ' if v == 'F' else v + ' ') + tc, str(note)[:200], flush=True)
def chk(tc, cond, note=''): rec(tc, bool(cond), note)
# --- fixture
T, TA, AD = login('r5t'), login('r5ta'), login('admin')
C1 = db("select id from courses where class_code='R5C1'"); C2 = db("select id from courses where class_code='R5C2'")
Q = f'/courses/{C1}/questions'; E = f'/courses/{C1}/exams'; Q2 = f'/courses/{C2}/questions'; E2 = f'/courses/{C2}/exams'
def sv(n): return 'r5s%02d' % n
def approve(qid, base=Q):
    g = call('GET', base + '/' + qid, T)[1]
    if g.get('review_status') == 'APPROVED': return True
    s, _, _ = call('PUT', base + '/' + qid + '/review', T, dict(decision='REQUEST', version=g['version'])); g = call('GET', base + '/' + qid, T)[1]
    s, j, _ = call('PUT', base + '/' + qid + '/review', TA, dict(decision='APPROVE', version=g['version'])); return s == 200
CANARY = 'QCR5CANARY' + uuid.uuid4().hex[:8]
def mkq(i=0, typ='MCQ_SINGLE', n=4, correct=(1,), base=Q, title=None, expl=None, ok=True, value=True):
    if typ == 'TRUE_FALSE': b = dict(type=typ, title=title or 'r5-%d' % i, topic='R5', stem='Câu %d đúng?' % i, value=value, explanation=expl or CANARY)
    else: b = dict(type=typ, title=title or 'r5-%d' % i, topic='R5', stem='Câu %d?' % i, options=[{'body': 'Đáp án %d của câu %d' % (k, i)} for k in range(n)], correct=list(correct), explanation=expl or CANARY)
    s, q, _ = call('POST', base, T, b)
    assert s in (200, 201), (s, q)
    if ok: approve(q['id'], base)
    return q['id']
SUM = '#include <cstdio>\nint main(){long long a,b;scanf("%lld %lld",&a,&b);printf("%lld\\n",a+b);}\n'
HID = 'QCHID' + uuid.uuid4().hex[:6]
def wait_job(jid, tok=None, n=90):
    for _ in range(n):
        j = call('GET', '/jobs/' + jid, tok or T)[1]
        if j.get('status') in ('SUCCEEDED', 'FAILED', 'DEAD'): return j
        time.sleep(1)
    return j
def mkcode(i=1, tests=None, base=Q, verify=True, approve_q=True, starter=None, checker=None, float_eps=None, ref=None, tl=1000):
    s, j, _ = call('POST', base, T, dict(type='CODE', title='r5code-%d' % i, topic='R5', difficulty='EASY', stem='Đọc a, b in a+b')); qid = j['id']; P = base + '/' + qid
    s, j, _ = call('PUT', P + '/code', T, dict(languages=['c11', 'cpp17'], time_limit_ms=tl, memory_limit_mb=256, starter_code=starter or {'cpp17': '// khởi đầu\n'}, reference=dict(language='cpp17', source=ref or SUM), version=j['version'], **({'checker': checker} if checker else {}), **({'float_eps': float_eps} if float_eps else {}))); assert s == 200, (s, j)
    tids = []
    for nm, inp, exp, smp, w in (tests or [('mau1', '1 2\n', '3\n', True, 1), ('mau2', '10 20\n', '30\n', True, 1), (HID + 'a', '5 5\n', '10\n', False, 2), (HID + 'b', '100 200\n', '300\n', False, 2)]):
        s, j, _ = call('POST', P + '/testcases', T, dict(name=nm, input=inp, expected=exp, is_sample=smp, weight=w)); assert s in (200, 201), (s, j); tids.append(j['id'])
    call('POST', P + '/testcases/approve', T, dict(ids=tids))
    if verify:
        s, j, _ = call('POST', P + '/reference/verify', T, {}, idem())
        if s in (200, 202) and j.get('job_id'): wait_job(j['job_id'])
    if approve_q: approve(qid, base)
    return qid
def mkexam(open_s=15, win=600, dur=10, items=None, base_e=E, base_q=Q, nq=3, with_code=False, title='R5 bài', schedule=True, **kw):
    b = dict(title=title, opens_at=iso(seconds=open_s), closes_at=iso(seconds=open_s + win), duration_minutes=dur); b.update(kw)
    s, j, _ = call('POST', base_e, T, b, idem()); assert s == 201, (s, j); eid = j['id']
    if items is None:
        ids = [mkq(i, 'MCQ_SINGLE' if i % 3 else 'MCQ_MULTI', 4, (1,) if i % 3 else (0, 2), base=base_q) for i in range(nq)]
        ids.append(mkq(99, 'TRUE_FALSE', base=base_q))
        if with_code: ids.append(mkcode(base=base_q))
        items = ids
    g = call('GET', base_e + '/' + eid, T)[1]
    s, jj, _ = call('PUT', base_e + '/' + eid + '/items', T, dict(items=[dict(question_id=q, points='1') for q in items], version=g['version'])); assert s == 200, (s, jj)
    if schedule:
        if open_s < 120:
            g = call('GET', base_e + '/' + eid, T)[1]
            s, jj, _ = call('PUT', base_e + '/' + eid, T, dict(title=title, opens_at=iso(seconds=open_s), closes_at=iso(seconds=open_s + win), duration_minutes=dur, version=g['version'], **{k: v for k, v in kw.items() if k not in ('title',)})); assert s == 200, (s, jj)
        s, jj, _ = call('POST', base_e + '/' + eid + '/schedule', T, {}); assert s == 200, (s, jj)
    return eid, items
def start(eid, u, base_e=E):
    t = login(u); s, j, _ = call('POST', base_e + '/' + eid + '/attempts', t, {}, idem())
    if s not in (200, 201): return None, None, None, s, j
    aid = j['attempt']['id']; H = {'X-Exam-Tab': str(uuid.uuid4())}; call('POST', base_e + f'/{eid}/attempts/{aid}/takeover', t, {}, H)
    return aid, j, H, s, j
def submit_exam(eid, aid, u, H, base_e=E): return call('POST', base_e + f'/{eid}/attempts/{aid}/submit', login(u), {}, {**H, **idem()})
def results():
    d = {}
    for l in open(OUT):
        r = json.loads(l); d[r['tc']] = r
    return d
