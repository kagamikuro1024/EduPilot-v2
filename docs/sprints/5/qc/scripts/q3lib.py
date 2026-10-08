"""QC US-PE-03: thư viện nhỏ gọi API qua Caddy của stack EP_PORT_OFFSET=100 (https://localhost:543), không verify TLS."""
import json, ssl, urllib.request, urllib.error, uuid, io, zipfile
CTX = ssl._create_unverified_context()
BASE = 'https://localhost:543/api/v1'
PW = 'Edupilot#Seed-2026'
def call(m, p, tok=None, body=None, h=None, raw=None, ctype=None):
    hd = {'Origin': 'https://localhost:543', 'X-Forwarded-For': '198.51.100.%d' % (hash(str(uuid.uuid4())) % 250 + 1)}
    if tok: hd['Authorization'] = 'Bearer ' + tok
    data = raw
    if body is not None: data = json.dumps(body).encode(); hd['Content-Type'] = 'application/json'
    if ctype: hd['Content-Type'] = ctype
    hd.update(h or {})
    rq = urllib.request.Request(BASE + p, data=data, method=m, headers=hd)
    try:
        r = urllib.request.urlopen(rq, context=CTX, timeout=60); s, t = r.status, r.read().decode()
    except urllib.error.HTTPError as e: s, t = e.code, e.read().decode()
    try: j = json.loads(t)
    except Exception: j = {}
    return s, j, t
def login(u): 
    s, j, _ = call('POST', '/auth/login', body={'email': u + '@edupilot.local', 'password': PW}); assert s == 200, (u, s); return j['access_token']
def idem(): return {'Idempotency-Key': str(uuid.uuid4())}
def codes(j):
    det = j.get('details'); out = {d.get('code') for d in det if isinstance(d, dict)} if isinstance(det, list) else set()
    if isinstance(j.get('code'), str): out.add(j['code'])
    return sorted(c for c in out if c)
def mkzip(files):
    b = io.BytesIO()
    with zipfile.ZipFile(b, 'w', zipfile.ZIP_DEFLATED) as z:
        for n, c in files.items(): z.writestr(n, c)
    return b.getvalue()
def multipart(fields, fname, content):
    bd = '----qc' + uuid.uuid4().hex
    out = b''
    for k, v in fields.items(): out += ('--%s\r\nContent-Disposition: form-data; name="%s"\r\n\r\n%s\r\n' % (bd, k, v)).encode()
    out += ('--%s\r\nContent-Disposition: form-data; name="file"; filename="%s"\r\nContent-Type: application/zip\r\n\r\n' % (bd, fname)).encode() + content + ('\r\n--%s--\r\n' % bd).encode()
    return out, 'multipart/form-data; boundary=' + bd
