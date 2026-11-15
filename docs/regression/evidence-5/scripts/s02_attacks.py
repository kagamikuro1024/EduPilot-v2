import os; os.environ['SUITE']='PE-02'
from r5lib import *
import glob, json
meta = json.load(open('/tmp/qc2-c5/probe.json')); eid = meta['eid']
def ctx(u):
    t = login(u); s, j, _ = call('GET', E + '/' + eid + '/attempts/mine', t)
    if s != 200 or not j.get('attempt'): s, j, _, = call('POST', E + '/' + eid + '/attempts', t, {}, idem())
    aid = j['attempt']['id']; it = [x for x in j['items'] if x['type'] == 'CODE'][0]['item_id']; H = {'X-Exam-Tab': str(uuid.uuid4())}
    call('POST', E + '/' + eid + f'/attempts/{aid}/takeover', t, {}, H); return t, f'{E}/{eid}/attempts/{aid}', it, H
def run(c, lang, src):
    if c is None:
        if cnt[0] >= 8: POOL.pop(0); cur[0] = ctx(POOL[0]); cnt[0] = 0
        cnt[0] += 1; c = cur[0]
    t, A, it, H = c
    for _ in range(6):
        s, rr, tt = call('POST', A + f'/code/{it}/run', t, dict(language=lang, source=src), {**H, **idem()})
        if s == 202: break
        if s in (429, 503): time.sleep(8); continue
        return None, s, tt
    t0 = time.time()
    for _ in range(80):
        s, r, tt = call('GET', A + '/runs/' + rr['run_id'], t)
        if r.get('status') in ('DONE', 'ERROR'): return r, time.time(), tt
        time.sleep(1)
    return r, time.time(), tt
def verdict(r):
    if r is None: return 'NA'
    if r.get('compile_ok') is False: return 'CE'
    vs = [x['verdict'] for x in r.get('samples', [])]
    for v in vs:
        if v != 'AC': return v
    return 'AC' if vs else ('IE' if r.get('status') == 'ERROR' else '?')
POOL=['r5s03','r5s04','r5s05','r5s06','r5s07']; cnt=[0]; cur=[ctx(POOL[0])]
att = None
inn = ctx('r5s02')
rc0 = docker_rc = subprocess.run(['docker','inspect','ep-r-judge-1','--format','{{.RestartCount}}'],capture_output=True,text=True).stdout.strip()
def hostproc(): return subprocess.run(['docker','top','ep-r-judge-1'],capture_output=True,text=True).stdout
base_procs = len(hostproc().splitlines())
def mem(): return subprocess.run(['docker','stats','--no-stream','ep-r-judge-1','--format','{{.MemUsage}}|{{.PIDs}}'],capture_output=True,text=True).stdout.strip()
print('baseline', mem(), 'procs', base_procs)
D = 'attacks/'
def A_(name, lang, allowed, tc, note=''):
    src = open(name).read(); t0 = time.time(); r, _, tt = run(None, lang, src); dtm = time.time() - t0; v = verdict(r)
    print(f'  {os.path.basename(name):28s} {v:4s} {dtm:5.1f}s {json.dumps(r.get("samples") if r else tt)[:130]} {mem()}')
    return v, dtm
res = {}
for n, lang, allow in [('a01-infinite-loop.c','c11',{'TLE'}), ('a02-print-forever.c','c11',{'TLE','OLE'}), ('a03-print-5mb.c','c11',{'OLE'}), ('a04-print-1gb.c','c11',{'OLE','TLE'}),
    ('a05-alloc-gradual-512m.c','c11',{'MLE'}), ('a06-alloc-2g.c','c11',{'MLE','RE'}), ('a07-null-deref.c','c11',{'RE'}), ('a18-stack-overflow.c','c11',{'RE'}),
    ('a08-fork-bomb.c','c11',{'TLE','RE','MLE'}), ('a13-write-1gb-file.c','c11',{'MLE','RE','WA','OLE'}), ('a19-sleep-forever.c','c11',{'TLE'}), ('a20-stdin-hang.c','c11',{'TLE'}),
    ('a11-include-dev-random.c','c11',{'CE'}), ('a12-compile-bomb.cpp','cpp17',{'CE'})]:
    v, d = A_(D + n, lang, allow, '')
    res[n] = (v, d, v in allow); time.sleep(1)
for n, lang in [('x09-read-files.c','c11'),('x10-sockets.c','c11'),('x14-exec.c','c11'),('x15w.c','c11'),('x15r.c','c11'),('x16-priv.c','c11'),('x17-thr.c','c11')]:
    v, d = A_('attacks-x/' + n, lang, {'AC'}, ''); res[n] = (v, d, v == 'AC'); time.sleep(1)
# TC44: bài đúng của SV khác ngay sau bom
t0 = time.time(); rb, _, _ = run(None, 'c11', open(D+'a08-fork-bomb.c').read()); ri, _, _ = run(inn, 'cpp17', SUM); ac_t = time.time() - t0; vi = verdict(ri)
print('innocent after bomb', vi, '%.1fs' % ac_t)
rc1 = subprocess.run(['docker','inspect','ep-r-judge-1','--format','{{.RestartCount}}'],capture_output=True,text=True).stdout.strip()
time.sleep(30); m30 = mem(); procs_after = len(hostproc().splitlines())
print('after 30s', m30, 'procs', procs_after, 'restart', rc0, rc1)
disk = subprocess.run(['docker','exec','ep-r-judge-1','sh','-c','du -sk /tmp /w /dev/shm /opt 2>/dev/null | tail -5'],capture_output=True,text=True).stdout
print('disk', disk)
json.dump(dict(res={k:list(v) for k,v in res.items()}, vi=vi, ac_t=ac_t, rc=[rc0,rc1], mem30=m30, procs=[base_procs, procs_after], disk=disk), open('/tmp/qc2-c5/s02.json','w'))
G = lambda n: res[n][2]
def one(tc, names, note=''):
    ok = all(G(n) for n in names); rec(tc, ok, '; '.join(f'{n.split("-")[0]}={res[n][0]}({res[n][1]:.0f}s)' for n in names) + ' ' + note)
one('TC-PE02-33', ['a01-infinite-loop.c','a02-print-forever.c']); one('TC-PE02-34', ['a03-print-5mb.c','a04-print-1gb.c'])
one('TC-PE02-35', ['a05-alloc-gradual-512m.c','a06-alloc-2g.c']); one('TC-PE02-36', ['a07-null-deref.c','a18-stack-overflow.c'])
one('TC-PE02-37', ['a08-fork-bomb.c','x17-thr.c'], 'procPeak/FORKED không quan sát được qua API (verdict an toàn + 0 tiến trình mồ côi: host procs %d→%d)' % (base_procs, procs_after))
one('TC-PE02-38', ['x09-read-files.c'], '(a09 qua biến thể in LEAK nếu đọc được)')
one('TC-PE02-39', ['x10-sockets.c'], '(36 cặp ip:port, không connect nào thành công)')
ce = res['a11-include-dev-random.c'][2] and res['a12-compile-bomb.cpp'][2] and max(res['a11-include-dev-random.c'][1], res['a12-compile-bomb.cpp'][1]) <= 25
rec('TC-PE02-40', ce, 'a11=%s %.0fs a12=%s %.0fs' % (res['a11-include-dev-random.c'][0], res['a11-include-dev-random.c'][1], res['a12-compile-bomb.cpp'][0], res['a12-compile-bomb.cpp'][1]))
one('TC-PE02-41', ['a13-write-1gb-file.c','x15r.c'], 'x15w ghi rồi x15r đọc: AC = không rò chéo')
one('TC-PE02-42', ['x14-exec.c','x16-priv.c'])
one('TC-PE02-43', ['a19-sleep-forever.c','a20-stdin-hang.c'])
rec('TC-PE02-44', vi == 'AC' and ac_t <= 20 and rc0 == rc1 and procs_after <= base_procs + 1, f'bài đúng sau fork bomb {vi} {ac_t:.1f}s; RestartCount {rc0}->{rc1}; mem30 {m30}; procs {base_procs}->{procs_after}')
