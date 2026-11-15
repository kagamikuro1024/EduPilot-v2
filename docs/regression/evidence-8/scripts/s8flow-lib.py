"""s8flow lib: gọi API ep-r bằng urllib, cache token theo exp, psql, tải tệp qua presign."""
import base64, hashlib, json, os, ssl, subprocess, time, urllib.request, urllib.error, uuid
B = 'https://localhost:1143/api/v1'; PW = 'Edupilot#Seed-2026'; REPO = os.path.expanduser('~/Documents/TA_Agent_qc2')
ctx = ssl._create_unverified_context(); _tok = {}
def _exp(t): return json.loads(base64.urlsafe_b64decode(t.split('.')[1] + '==')).get('exp', 0)
def raw(m, p, tok=None, body=None, key=None, data=None, hdr=None, full=False):
    h = {'Content-Type': 'application/json', **(hdr or {})}
    if tok: h['Authorization'] = f'Bearer {tok}'
    if m in ('POST', 'PUT', 'PATCH', 'DELETE') and 'Idempotency-Key' not in h and key != False: h['Idempotency-Key'] = key or str(uuid.uuid4())
    d = data if data is not None else (json.dumps(body).encode() if body is not None else None)
    for _try in range(10):
        try:
            r = urllib.request.urlopen(urllib.request.Request(p if p.startswith('http') else B + p, data=d, method=m, headers=h), context=ctx); st, bt, hd = r.status, r.read(), r.headers
        except urllib.error.HTTPError as e: st, bt, hd = e.code, e.read(), e.headers
        except (urllib.error.URLError, ConnectionError, TimeoutError): st, bt, hd = 599, b'', {}
        if st in (502, 503, 504, 599) and not os.environ.get('NORETRY'): time.sleep(15); continue
        break
    if full: return st, bt, hd
    try: return st, json.loads(bt)
    except Exception: return st, bt
def tok(u):
    if '@' not in u: u += '@edupilot.local'
    t = _tok.get(u)
    if t and _exp(t) - time.time() > 60: return t
    for i in range(6):
        s, d = raw('POST', '/auth/login', body={'email': u, 'password': PW})
        if s == 200: _tok[u] = d['access_token']; return d['access_token']
        if s in (429, 423): time.sleep(20); os.system('''docker exec ep-r-redis-1 sh -c "for p in 'ep:auth:blockip:*' 'ep:auth:ipfail:*' 'ep:auth:fail:*' 'ep:rl:auth*'; do redis-cli --scan --pattern \\"\\$p\\" | xargs -r redis-cli del; done" >/dev/null''')
        elif s in (502,503,504,599): time.sleep(15)
        else: time.sleep(5)
    raise RuntimeError(f'login {u} {s} {d}')
def api(u, m, p, body=None, key=None, **kw): return raw(m, p, tok(u) if u else None, body, key, **kw)
def psql(q):
    r = subprocess.run(['docker', 'exec', '-i', 'ep-r-postgres-1', 'psql', '-U', 'edupilot', '-d', 'edupilot', '-At', '-F', '|', '-c', q], capture_output=True, text=True)
    if r.returncode: raise RuntimeError(r.stderr)
    return r.stdout.strip()
def mkclone(cc, mode=''):
    env = {**os.environ}; r = subprocess.run(['bash', f'{REPO}/docs/regression/evidence-8/scripts/s8flow-clone.sh', cc, mode], capture_output=True, text=True, env=env)
    cid = psql(f"select id from courses where class_code='{cc}'"); return cid
def put_obj(url, hdrs, b): urllib.request.urlopen(urllib.request.Request(url, data=b, method='PUT', headers=hdrs), context=ctx)
def upload(u, cid, purpose, filename, mime, b, sha=None, size=None, extra=None):
    sha = sha or hashlib.sha256(b).hexdigest()
    s, pre = api(u, 'POST', f'/courses/{cid}/uploads/presign', {'purpose': purpose, 'filename': filename, 'mime_type': mime, 'size_bytes': size if size is not None else len(b), 'sha256': sha, **(extra or {})})
    if s != 200: return s, pre, None
    put_obj(pre['url'], pre.get('headers', {}), b); return s, pre, sha
class Rec:
    def __init__(self, path): self.path = path; self.r = {}
    def __call__(self, tc, st, note=''): self.r[tc] = (st, note[:160]); print(tc, st, note[:160], flush=True)
    def save(self):
        with open(self.path, 'w') as f:
            for k, (s, n) in sorted(self.r.items()): f.write(f'{k}\t{s}\t{n}\n')
