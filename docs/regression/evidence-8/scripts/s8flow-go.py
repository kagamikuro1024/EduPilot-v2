#!/usr/bin/env python3
"""s8flow: chạy các test Go nêu trong cột «Bước» của tc-US-P6-05..09 (tuần tự, chia chunk), ghi kết quả từng test vào logs/s8flow-go-results.json."""
import json, re, subprocess, os, sys, time
R = os.path.expanduser('~/Documents/TA_Agent_qc2'); L = f'{R}/docs/regression/evidence-8/logs'
rows = json.load(open('/tmp/qc2-s8f/gopw.json'))
want = {}  # (pkg, flags) -> ordered names
for tc, gos, _ in rows:
    for pkg, nm, fl in gos:
        key = (pkg, fl.strip())
        for x in nm.replace('\\', '').split('|'):
            if x not in want.setdefault(key, []): want[key].append(x)
res_path = f'{L}/s8flow-go-results.json'
res = json.load(open(res_path)) if os.path.exists(res_path) else {}
env = {**os.environ, 'DOCKER_HOST': f'unix://{os.path.expanduser("~")}/.colima/default/docker.sock', 'TESTCONTAINERS_RYUK_DISABLED': 'true'}
for (pkg, fl), names in want.items():
    for i in range(0, len(names), 4):
        ch = [n for n in names[i:i+4] if f'{pkg} {fl} {n}' not in res]
        if not ch: continue
        args = ['go', 'test', '-count=1', '-v', '-timeout', '900s'] + fl.split() + ['-run', '^(' + '|'.join(ch) + ')$', pkg]
        t = time.time()
        p = subprocess.run(args, cwd=f'{R}/backend-go', env=env, capture_output=True, text=True)
        out = p.stdout + p.stderr
        open(f'{L}/s8flow-go-{pkg.split("/")[-1]}-{abs(hash(tuple(ch)))%10**6}.log', 'w').write(' '.join(args) + '\n' + out)
        race = 'DATA RACE' in out
        for n in ch:
            st = 'PASS' if re.search(rf'--- PASS: {n}\b', out) else 'SKIP' if re.search(rf'--- SKIP: {n}\b', out) else 'FAIL' if re.search(rf'--- FAIL: {n}\b', out) else 'NONE'
            if race and st == 'PASS': st = 'FAIL'
            res[f'{pkg} {fl} {n}'] = {'st': st, 'rc': p.returncode, 'sec': round(time.time() - t)}
        json.dump(res, open(res_path, 'w'), indent=1, ensure_ascii=False)
        print(pkg, fl, ch, [res[f'{pkg} {fl} {n}']['st'] for n in ch], flush=True)
open(f'{L}/s8flow-go.done', 'w').write('DONE')
