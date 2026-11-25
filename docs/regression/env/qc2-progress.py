#!/usr/bin/env python3
"""Ghi ~/.config/edupilot-team/progress/qc2.tsv từ smoke-*.md (sprint⇥tổng(trừ LT)⇥đạt⇥trượt⇥0⇥ghi chú). Dùng: python3 env/qc2-progress.py ["ghi chú chung"]"""
import re, sys, os, importlib.util
_s = importlib.util.spec_from_file_location('qc2count', os.path.join(os.path.dirname(os.path.abspath(__file__)), 'qc2-count.py')); qc2count = importlib.util.module_from_spec(_s); _s.loader.exec_module(qc2count)
R = os.path.expanduser('~/Documents/TA_Agent_qc2/docs/regression')
NOTE = {'8': 'SEED sprint 8 (SEED-22/36 P sau sửa)', '4': 'SmkT3 + P209–P212; P212-14 P (seed 281 s, qc2seedt2); GATEP2-99 dọn xong (P-11-100)', '9.1': 'đóng', '9.5': 'đóng (N08-11 P với OpenCode Go)', '3': 'đóng', '10.5': 'đóng'}
out = []
for sp in ['1', '1.5', '2', '3', '4', '8', '9', '9.1', '9.5', '10', '10.5', '10.6', '10.7', '10.8']:  # sprint của qc2; 5–7 là của qc3
    p = f'{R}/smoke-{sp}.md'
    if not os.path.exists(p): continue
    c = {}
    for v in qc2count.count(p).values(): c[v] = c.get(v, 0) + 1  # dòng cuối của mỗi mã thắng; A…B được bung (env/qc2-count.py)
    P, F, N = c.get('P', 0), c.get('F', 0), sum(v for k, v in c.items() if k not in ('P', 'F', 'LT'))
    tot = P + F + N
    out.append(f'{sp}\t{tot}\t{P}\t{F}\t0\t{NOTE.get(sp, "")} (N={N})' + (' ' + sys.argv[1] if len(sys.argv) > 1 else ''))
open(os.path.expanduser('~/.config/edupilot-team/progress/qc2.tsv'), 'w').write('\n'.join(out) + '\n')
print('\n'.join(out))
