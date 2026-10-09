from q6lib import *
st = json.load(open('/tmp/q3/q6.json')); eid = st['eid']; A = E + '/' + eid
j = call('GET', A + '/attempts/mine', SV)[1]; aid = j['attempt']['id']; ci = [x for x in j['items'] if x['type'] == 'CODE'][0]; iid = ci['item_id']
TAB = str(uuid.uuid4()); H = {'X-Exam-Tab': TAB}; print('takeover', call('POST', A + f'/attempts/{aid}/takeover', SV, {}, H)[0])
CB = A + f'/attempts/{aid}/code/{iid}'
def draft(lang, src, rev, h=H, tok=SV): return call('PUT', CB + '/draft', tok, dict(language=lang, source=src, base_rev=rev), h)

# run
def run(lang, src, tok=SV, key=None): return call('POST', CB + '/run', tok, dict(language=lang, source=src), {**H, **(key or idem())})
def poll(path, tok=SV):
    for _ in range(40):
        s, j, t = call('GET', path, tok)
        if s == 200 and j.get('status') in ('DONE', 'ERROR'): return j, t
        time.sleep(1)
    return j, t
CASES = {'ok': SUMSRC, 'wa': SUMSRC.replace('a+b', 'a+b+1'), 'ce': 'int main(){ undefined_fn(); }', 'tle': '#include <cstdio>\nint main(){ for(;;); }', 're': '#include <cstdlib>\nint main(){ abort(); }', 'ml': '#include <vector>\nint main(){ std::vector<char> v(1<<30,1); long s=0; for(char c:v) s+=c; return s==7; }', 'stderr': '#include <cstdio>\nint main(){ fprintf(stderr,"%s /tmp/secret/a.cpp","QCSTDERR"); long long a,b; scanf("%lld %lld",&a,&b); printf("%lld\\n",a+b); }'}
for n, src in CASES.items():
    t0 = time.time(); s, rr, tt = run('cpp17', src)
    if s != 202: print('16', n, s, tt[:200]); continue
    res, txt = poll(A + f'/attempts/{aid}/runs/' + rr['run_id']); print('16', n, 'rid', s, '%.1fs' % (time.time() - t0), json.dumps(res, ensure_ascii=False)[:420], '| HID', HID in txt, 'stderr canary', 'QCSTDERR' in txt, '/tmp/secret' in txt)
    time.sleep(0.3)
