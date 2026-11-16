#!/usr/bin/env python3
"""Gộp kết quả TSV (TC<TAB>KQ<TAB>ghi chú) vào docs/regression/smoke-<n>.md (n = 3 hoặc 4).
KQ: P / F / LT / K(a) / K(b) / K(c) / N. Dòng TSV cuối cùng của một mã thắng.
Dùng: python3 smk34-merge.py 3 [--dry]   (đọc evidence-<n>/results-smk34/*.tsv)"""
import re, sys, glob, os
n = sys.argv[1]; dry = '--dry' in sys.argv
R = os.path.expanduser('~/Documents/TA_Agent_qc2/docs')
smoke = f'{R}/regression/smoke-{n}.md'
lines = open(smoke, encoding='utf-8').read().split('\n')
defined = set()
for f in glob.glob(f'{R}/sprints/{n}/qc/tc-*.md'):
    for l in open(f, encoding='utf-8'):
        m = re.match(r'\| (TC-[A-Z0-9]+-\d+) \|', l)
        if m: defined.add(m.group(1))
ov = {}
for f in sorted(glob.glob(f'{R}/regression/evidence-{n}/results-smk34/*.tsv'), key=os.path.getmtime):
    for l in open(f, encoding='utf-8'):
        p = l.rstrip('\n').split('\t')
        if len(p) >= 3 and p[0].startswith('TC-'):
            assert p[1] in ('P','F','LT','K(a)','K(b)','K(c)','N'), (f, l)
            ov[p[0]] = (p[1], '\t'.join(p[2:]).replace('|', '/'))
def expand(code):
    m = re.match(r'(TC-[A-Z0-9]+-)(\d+)(?:…TC-[A-Z0-9]+-(\d+))?$', code)
    if not m or not m.group(3): return [code]
    a, b = int(m.group(2)), int(m.group(3))
    return sorted(c for c in defined if c.startswith(m.group(1)) and c[len(m.group(1)):].isdigit() and a <= int(c[len(m.group(1)):]) <= b) or [code]
def norm(kq, note):
    if kq == 'K':
        if note.startswith('(a)'): return 'K(a)', note
        if note.startswith('(b)'): return 'K(b)', note
        if note.startswith('(c)'): return 'K(c)', note
        if note.startswith('LỖI THỜI'): return 'LT', note
        return 'N', note
    return kq, note
i0 = lines.index('## Kết quả từng ca'); i1 = lines.index('## Trượt: bước tái hiện')
rows = []; seen = set()
for l in lines[i0 + 3:i1]:
    m = re.match(r'\| (TC-[^|]+?) \| (\S+) \| (.*) \|$', l)
    if not m: continue
    codes = expand(m.group(1))
    base = norm(m.group(2), m.group(3))
    if len(codes) > 1 and not any(c in ov for c in codes): rows.append((m.group(1), *base)); continue
    for c in codes:
        rows.append((c, *(ov.get(c) or base)))
        seen.add(c)
unknown = [c for c in ov if c not in {r[0] for r in rows}]
# mã chưa có hàng (vd. 'TC-…b'): thêm cuối
for c in unknown: rows.append((c, *ov[c]))
tot = {}
for c, k, _ in rows:
    tot[k] = tot.get(k, 0) + 1
T = len(rows)
hdr3 = f"Tổng TC cũ: {T} · Đạt: {tot.get('P',0)} · Trượt: {tot.get('F',0)} · Lỗi thời (LT): {tot.get('LT',0)} · Không kiểm được K (a/b/c): {tot.get('K(a)',0)}/{tot.get('K(b)',0)}/{tot.get('K(c)',0)} · Chưa chạy (N): {tot.get('N',0)}   (P+F+LT+K+N = T)"
us = []
for l in lines[i0 - 14 if False else 0:]:
    pass
ush = lines.index('## Tổng hợp theo US')
uslines = []
for l in lines[ush + 3:i0 - 1]:
    m = re.match(r'\| (\S+) \| \d+ \|', l)
    if not m: continue
    pre = 'TC-' + m.group(1).replace('US-', '').replace('-', '') + '-'
    rr = [r for r in rows if r[0].startswith(pre) and r[0][len(pre):].isdigit()]
    c = lambda k: sum(1 for r in rr if r[1] == k)
    uslines.append(f"| {m.group(1)} | {len(rr)} | {c('P')} | {c('F')} | {c('LT')} | {c('K(a)')+c('K(b)')+c('K(c)')} | {c('N')} | |")
out = lines[:2] + [hdr3, f"Tổng hợp N=0 đợt 2026-11-16 (qc2/SmkN): kết quả từng ca ở các tệp `evidence-{n}/results-smk34/*.tsv`; gộp bằng `evidence-3/scripts/smk34-merge.py {n}`."] + lines[4:ush] + \
    ['## Tổng hợp theo US', '| US | TC tổng | P | F | LT | K | N | Ghi chú |', '| --- | --- | --- | --- | --- | --- | --- | --- |'] + uslines + [''] + \
    ['## Kết quả từng ca', '| TC | KQ | Ghi chú |', '| --- | --- | --- |'] + [f'| {c} | {k} | {t} |' for c, k, t in rows] + [''] + lines[i1:]
print(hdr3); print({k: v for k, v in sorted(tot.items())})
if not dry: open(smoke, 'w', encoding='utf-8').write('\n'.join(out))
