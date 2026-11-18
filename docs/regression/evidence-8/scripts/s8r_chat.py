import importlib.util, os, json, time, ssl, urllib.request, uuid
spec = importlib.util.spec_from_file_location('lib', os.path.dirname(os.path.abspath(__file__)) + '/s8flow-lib.py'); L = importlib.util.module_from_spec(spec); spec.loader.exec_module(L)
PART = L.REPO + '/docs/regression/evidence-8/part-s8r.tsv'
def T(tc, st, n):
    n = ' '.join(str(n).split())[:420]; open(PART, 'a').write(f"{tc}\t{st}\t{n}\n"); print(st, tc, n[:220], flush=True)
def cid(code): return L.psql(f"select id from courses where class_code='{code}'")
def now_db(): return L.psql("select now()")
def audit_since(t): return L.psql(f"select count(*) from llm_audit where created_at >= '{t}' and task in ('CHAT','UTILITY','CLASSIFY','DECIDE')")
def ask(user, course, text, sid=None, channel=None):
    """POST /chat/sessions (nếu chưa có) + POST messages (SSE). Trả (sid, văn bản, danh sách sự kiện)."""
    if sid is None:
        s, d = L.api(user, 'POST', '/chat/sessions', {'course_id': course}); sid = d['id']
    tk = L.tok(user)
    req = urllib.request.Request(L.B + f'/chat/sessions/{sid}/messages', data=json.dumps({'content': text}).encode(), method='POST', headers={'Authorization': 'Bearer ' + tk, 'Content-Type': 'application/json', 'Idempotency-Key': str(uuid.uuid4()), 'Accept': 'text/event-stream'})
    ev = []; txt = ''
    try:
        r = urllib.request.urlopen(req, context=L.ctx, timeout=120)
        cur = None
        for raw in r:
            l = raw.decode('utf-8', 'replace').rstrip('\n')
            if l.startswith('event:'): cur = l[6:].strip()
            elif l.startswith('data:'):
                try: dd = json.loads(l[5:].strip())
                except Exception: dd = l[5:].strip()
                ev.append((cur, dd))
                if isinstance(dd, dict) and cur == 'token': txt += str(dd.get('t') or dd.get('text') or '')
                if cur in ('done', 'error'): break
    except Exception as e: ev.append(('exc', str(e)))
    return sid, txt, ev
