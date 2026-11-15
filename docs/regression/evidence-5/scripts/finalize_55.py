"""Gộp kết quả Playwright mock (pw55b.json) vào smoke-5.5.md: bảng theo spec (chạy: python3 finalize_55.py)."""
import json, collections
d = json.load(open('/tmp/qc2-c5/pw55b.json')); rows = collections.defaultdict(collections.Counter); fails = []
def walk(s, f):
    for sp in s.get('specs', []):
        for t in sp['tests']:
            r = t['results'][-1]['status'] if t['results'] else 'none'; rows[f][t['projectName'] + ':' + r] += 1
            if r not in ('passed', 'skipped'): fails.append((f, t['projectName'], sp['title'][:80], (t['results'][-1].get('error', {}).get('message', '') or '')[:110].replace('\n', ' ')))
    for s2 in s.get('suites', []): walk(s2, f)
for s in d['suites']: walk(s, s['file'])
out = ['', '## Kết quả spec mock (Playwright, bản dựng cổng, --workers=1)', '| spec | kết quả |', '| --- | --- |'] + [f'| {f} | {dict(v)} |' for f, v in rows.items()] + ['', '### Ca không đạt (đầu 40)'] + [f'- {f} [{p}] {t} — {m}' for f, p, t, m in fails[:40]]
open('/Users/kuro/Documents/TA_Agent_qc2/docs/regression/smoke-5.5.md', 'a').write('\n'.join(out) + '\n')
print(len(fails))
