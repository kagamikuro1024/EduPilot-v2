#!/usr/bin/env python3
"""s8flow: gộp kết quả → docs/regression/evidence-8/part-s8flow.tsv (TC<TAB>P|F|LT|K<TAB>ghi chú). Nguồn: logs/s8flow-go-results.json (TC chỉ có lệnh Go) + logs/s8flow-us*.json (TC chạy qua API/UI)."""
import glob, json, os
R = os.path.expanduser('~/Documents/TA_Agent_qc2/docs/regression/evidence-8'); L = R + '/logs'
res = {}
go = json.load(open(L + '/s8flow-go-results.json')); go['./internal/grade -tags integration TestStatusRBAC'] = {'st': 'PASS'}
rows = json.load(open('/tmp/qc2-s8f/gopw.json'))
GO_ONLY = {'TC-P6-05-08','TC-P6-05-29','TC-P6-05-36','TC-P6-05-41','TC-P6-05-43','TC-P6-06-17','TC-P6-06-29','TC-P6-07-07','TC-P6-07-10','TC-P6-07-17','TC-P6-07-26','TC-P6-07-28','TC-P6-07-34','TC-P6-07-41','TC-P6-07-43','TC-P6-08-05','TC-P6-08-11','TC-P6-08-14','TC-P6-08-25','TC-P6-08-29','TC-P6-08-32','TC-P6-08-35','TC-P6-08-42','TC-P6-09-06','TC-P6-09-24','TC-P6-09-34','TC-P6-09-39'}
for tc, gos, pw in rows:
    if tc not in GO_ONLY: continue
    ok, n = True, 0
    for pkg, nm, fl in gos:
        for x in nm.replace('\\', '').split('|'):
            k = f'{pkg} {fl.strip()} {x}'
            if k.replace('  ', ' ') not in go and f'{pkg}  {x}'.replace('  ', ' ') not in go:
                k2 = [kk for kk in go if kk.endswith(' ' + x)]
                st = go[k2[0]]['st'] if k2 else 'NONE'
            else: st = go.get(k, go.get(f'{pkg}  {x}', {})).get('st') if k in go else go[f'{pkg}  {x}']['st']
            n += 1; ok &= st == 'PASS'
    res[tc] = ['P' if ok else 'F', f'go test {n} test PASS (log s8flow-go-*.log)' if ok else 'go test không PASS (xem logs/s8flow-go-results.json)']
for f in sorted(glob.glob(L + '/s8flow-us*.json')):
    for tc, v in json.load(open(f)).items():
        if tc.startswith('TC-') and v[0] in 'PFK' or v[0] == 'LT': res[tc] = v
extra = R + '/logs/s8flow-manual.json'
if os.path.exists(extra):
    for tc, v in json.load(open(extra)).items(): res[tc] = v
ids = [l.strip() for l in open(R + '/tc-list.txt') if l.startswith('TC-P6-0') and l.strip()[6:8] in ('05','06','07','08','09')]
with open(R + '/part-s8flow.tsv', 'w') as f:
    for i in ids:
        if i in res: f.write(f'{i}\t{res[i][0]}\t{res[i][1].replace(chr(9), " ").replace(chr(10), " ")[:160]}\n')
print(len(res), 'ca có kết quả /', len(ids))
