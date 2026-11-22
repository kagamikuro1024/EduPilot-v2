#!/usr/bin/env python3
"""Ghi ~/.config/edupilot-team/progress/qc2.tsv từ smoke-*.md (sprint⇥tổng(trừ LT)⇥đạt⇥trượt⇥0⇥ghi chú). Dùng: python3 env/qc2-progress.py ["ghi chú chung"]"""
import re, sys, os
R = os.path.expanduser('~/Documents/TA_Agent_qc2/docs/regression')
NOTE = {'8': 'SEED sprint 8 (SEED-22/36 P sau sửa)', '4': 'SmkT3 + P209–P212; P212-14 P (seed 281 s, qc2seedt2); N: GATEP2-99 dọn cuối', '9.1': 'đóng', '9.5': 'đóng, 5 F là DEF đã báo dev', '3': 'đóng', '10.5': 'phần qc2 xong; N còn lại của QC (D04-11, D08-02/03)'}
out = []
for sp in ['1', '1.5', '2', '3', '4', '8', '9.1', '9.5', '10.5', '10.6']:
    p = f'{R}/smoke-{sp}.md'
    if not os.path.exists(p): continue
    last = {}  # dòng sau của cùng TC thắng (file chung 10.5/10.6 có dòng N cũ + dòng chạy lại)
    for l in open(p):
        m = re.match(r'\| ((?:TC-|P0-)[^|]+?) \| (P|F|N|LT|D|X|K\w*)(?: \||\s)', l)
        if m: last[m.group(1)] = m.group(2)
    c = {}
    for v in last.values(): c[v] = c.get(v, 0) + 1
    P, F, N = c.get('P', 0), c.get('F', 0), sum(v for k, v in c.items() if k not in ('P', 'F', 'LT'))
    tot = P + F + N
    out.append(f'{sp}\t{tot}\t{P}\t{F}\t0\t{NOTE.get(sp, "")} (N={N})' + (' ' + sys.argv[1] if len(sys.argv) > 1 else ''))
open(os.path.expanduser('~/.config/edupilot-team/progress/qc2.tsv'), 'w').write('\n'.join(out) + '\n')
print('\n'.join(out))
