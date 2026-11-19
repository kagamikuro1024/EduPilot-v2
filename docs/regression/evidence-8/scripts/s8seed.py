# S8 SEED trên stack qc2s (DB qc2i_seed): ghi vào part-s8r.tsv (bản ghi sau thắng)
import sys, os, json, subprocess, ssl, urllib.request, urllib.error, re, uuid, time
REPO = os.path.expanduser('~/Documents/TA_Agent_qc2'); PART = f'{REPO}/docs/regression/evidence-8/part-s8r.tsv'
GW = 'http://127.0.0.1:18231'; B = GW + '/api/v1'; PW = 'Edupilot#Seed-2026'
def psql(q): return subprocess.run(['docker', 'exec', '-i', 'ep-r-postgres-1', 'psql', '-U', 'edupilot', '-d', 'qc2i_seed', '-tAc', q], capture_output=True, text=True).stdout.strip()
def sh(c, **k): return subprocess.run(c, shell=True, capture_output=True, text=True, cwd=REPO, **k)
def T(tc, st, note):
    note = ' '.join(str(note).split())[:900]; open(PART, 'a').write(f'{tc}\t{st}\t{note}\n'); print(st, tc, note[:200], flush=True)
def C(tc, ok, note_ok, note_bad=None): T(tc, 'P' if ok else 'F', note_ok if ok else (note_bad or note_ok)); return ok
_t = {}
def req(m, p, tok=None, body=None, key=True, raw=None, hdr=None):
    h = {'content-type': 'application/json'}
    if tok: h['authorization'] = 'Bearer ' + tok
    if m in ('POST', 'PUT', 'PATCH') and key: h['Idempotency-Key'] = str(uuid.uuid4())
    if hdr: h.update(hdr)
    d = raw if raw is not None else (json.dumps(body).encode() if body is not None else None)
    r = urllib.request.Request(B + p, data=d, method=m, headers=h)
    try: x = urllib.request.urlopen(r, timeout=60); s = x.status; t = x.read()
    except urllib.error.HTTPError as e: s = e.code; t = e.read()
    try: j = json.loads(t)
    except Exception: j = {'_raw': t[:200].decode('utf8', 'replace')}
    return s, j
def tok(u):
    if u in _t: return _t[u]
    s, j = req('POST', '/auth/login', None, {'email': u + '@edupilot.local', 'password': PW}); assert s == 200, (u, s, j); _t[u] = j['access_token']; return _t[u]
C1 = psql("select id from courses where class_code='761987'"); C2 = psql("select id from courses where class_code='761988'")
