# GATEPU-05 (@real, lớp dữ liệu trên gateway thật ep-r): ETag/304, 409, 422, cursor, SSE thật, job. Không Playwright.
import ssl, json, time, urllib.request, importlib.util, sys
spec = importlib.util.spec_from_file_location('lib', 's8flow-lib.py'); L = importlib.util.module_from_spec(spec); spec.loader.exec_module(L)
ctx = ssl.create_default_context(); ctx.check_hostname = False; ctx.verify_mode = ssl.CERT_NONE; B = 'https://localhost:1143/api/v1'
def req(m, p, tk, body=None, h=None, stream=False, to=15):
    hd = {'Authorization': 'Bearer ' + tk, **(h or {})}; d = None
    if body is not None: d = json.dumps(body).encode(); hd['Content-Type'] = 'application/json'
    r = urllib.request.Request(B + p, data=d, method=m, headers=hd)
    try:
        x = urllib.request.urlopen(r, context=ctx, timeout=to); return x.status, dict(x.headers), x
    except urllib.error.HTTPError as e: return e.code, dict(e.headers), e
adm, gv = L.tok('admin'), L.tok('gv.s8f'); out = {}
s, h, x = req('GET', '/admin/llm/providers', adm); et = h.get('ETag') or h.get('Etag'); body = json.loads(x.read()); s2 = req('GET', '/admin/llm/providers', adm, h={'If-None-Match': et})[0]; out['etag'] = (s, et, s2)
p = [i for i in body['items'] if i['name'] == 'QC fake'][0]
v = p['version']; upd = lambda ver, extra={}: req('PUT', f"/admin/llm/providers/{p['id']}", adm, {'type': 'fake', 'name': 'QC fake', 'enabled': True, 'models': [{'model': 'fake-chat', 'kind': 'chat', 'price_in': '0', 'price_out': '0'}], 'version': ver, **extra})
a = upd(v); b = upd(v)  # hai tab cùng version
out['409'] = (a[0], b[0], json.loads(b[2].read()).get('code')); 
c = req('PUT', f"/admin/llm/providers/{p['id']}", adm, {'type': 'fake', 'name': '', 'models': [], 'version': 99999}); out['422/409 sai'] = (c[0], json.loads(c[2].read()).get('code'))
cur = req('GET', '/admin/llm/providers', adm); ver = [i for i in json.loads(cur[2].read())['items'] if i['name'] == 'QC fake'][0]['version']; d = req('PUT', f"/admin/llm/providers/{p['id']}", adm, {'type': 'fake', 'name': '', 'models': [{'model': 'fake-chat', 'kind': 'chat'}], 'version': ver}); out['422'] = (d[0], json.loads(d[2].read()).get('code'))
C = L.psql("select id from courses where class_code='QD8H3'"); r1 = req('GET', f'/courses/{C}/gradebook?limit=1', gv); j1 = json.loads(r1[2].read()); nc = j1.get('next_cursor'); r2 = req('GET', f'/courses/{C}/gradebook?limit=1&cursor={nc}', gv); j2 = json.loads(r2[2].read()); r3 = req('GET', f'/courses/{C}/gradebook?limit=1&cursor=%%%bad', gv)
out['cursor'] = {'p1': r1[0], 'rows1': len(j1['rows']), 'next': bool(nc), 'p2': r2[0], 'rows2': len(j2['rows']), 'khac_nhau': j1['rows'][0]['student']['id'] != j2['rows'][0]['student']['id'], 'cursor_hong': r3[0], 'limit500': len(json.loads(req('GET', f'/courses/{C}/gradebook?limit=500', gv)[2].read())['rows']) if req('GET', f'/courses/{C}/gradebook?limit=500', gv)[0] == 200 else 'status ' + str(req('GET', f'/courses/{C}/gradebook?limit=500', gv)[0])}
# SSE thật
sse = req('GET', '/events', gv, h={'Accept': 'text/event-stream'}, to=6); first = []
try:
    for _ in range(6):
        l = sse[2].readline().decode().strip()
        if l: first.append(l)
except Exception as e: first.append('read:' + type(e).__name__)
out['sse'] = (sse[0], sse[1].get('Content-Type'), first[:4])
out['sse_last_event_id'] = req('GET', '/events', gv, h={'Accept': 'text/event-stream', 'Last-Event-ID': '0-0'}, to=5)[0]
jid = L.psql("select id from jobs order by created_at desc limit 1"); jj = req('GET', f'/jobs/{jid}', adm); out['job'] = (jj[0], list(json.loads(jj[2].read()).keys())[:8])
print(json.dumps(out, ensure_ascii=False)); open('../../evidence-3/logs/qc2-gatepu05-api.json', 'w').write(json.dumps(out, ensure_ascii=False, indent=1))
