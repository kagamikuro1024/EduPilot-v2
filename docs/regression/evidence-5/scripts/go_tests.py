"""Chạy các test Go 'G' của TC sprint 5 (tất cả -tags integration, testcontainers). Ra results-go.jsonl + log."""
import re, subprocess, glob, json, os, sys
os.environ.update(TESTCONTAINERS_RYUK_DISABLED='true', DOCKER_HOST='unix:///Users/kuro/.colima/default/docker.sock')
ROOT = '/Users/kuro/Documents/TA_Agent_qc2'; BE = ROOT + '/backend-go'
rows = []
for f in sorted(glob.glob(ROOT + '/docs/sprints/5/qc/tc-US-P*.md')):
    for l in open(f, encoding='utf-8'):
        if not l.startswith('| TC-'): continue
        tc = l.split('|')[1].strip()
        if '**G**' not in l and 'go test' not in l: continue
        m = re.search(r"-run ('([^']+)'|(\S+))", l)
        if not m or '-race -count=1 ./...' in l: continue
        pat = m.group(2) or m.group(3)
        if pat.startswith('Test') or '|' in pat: rows.append((tc, pat))
allgo = {}
for p in glob.glob(BE + '/**/*_test.go', recursive=True):
    for n in re.findall(r'^func (Test\w+)\(', open(p, encoding='utf-8').read(), re.M): allgo.setdefault(n, set()).add(os.path.dirname(p))
out = open(ROOT + '/docs/regression/evidence-5/results-go.jsonl', 'w'); log = open(ROOT + '/docs/regression/evidence-5/logs/go-tests.log', 'w')
cache = {}
for tc, pat in rows:
    names = pat.split('|'); pk = {}
    for n in names:
        for d in allgo.get(n, []): pk.setdefault(d, []).append(n)
    miss = [n for n in names if n not in allgo]
    if not pk:
        out.write(json.dumps(dict(tc=tc, r='F', note='không tìm thấy test: ' + ','.join(miss), pat=pat)) + '\n'); continue
    ok = True; notes = []
    for d, ns in pk.items():
        key = (d, tuple(ns))
        if key not in cache:
            r = subprocess.run(['go', 'test', '-count=1', '-tags', 'integration', '-timeout', '8m', d, '-run', '^(' + '|'.join(ns) + ')$', '-v'], cwd=BE, capture_output=True, text=True)
            cache[key] = (r.returncode, r.stdout + r.stderr); log.write('### %s %s\n%s\n' % (d, ns, (r.stdout + r.stderr)[-3000:])); log.flush()
        rc, txt = cache[key]
        passed = len(re.findall(r'^--- PASS', txt, re.M)); failed = re.findall(r'^--- FAIL: (\S+)', txt, re.M); skipped = len(re.findall(r'^--- SKIP', txt, re.M))
        if rc != 0: ok = False
        notes.append('%s pass=%d skip=%d fail=%s' % (os.path.basename(d), passed, skipped, ','.join(failed) or ('rc=%d' % rc if rc else '0')))
    if miss: notes.append('thiếu test: ' + ','.join(miss)); ok = ok and False
    out.write(json.dumps(dict(tc=tc, r='P' if ok else 'F', note='; '.join(notes)[:300], pat=pat), ensure_ascii=False) + '\n'); out.flush()
