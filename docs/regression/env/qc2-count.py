#!/usr/bin/env python3
"""Đếm P/F/N... cho mọi smoke-*.md theo luật: dòng CUỐI của mỗi mã TC thắng; mã dạng A…B được bung ra từng mã;
nhiều mã trong một ô (phẩy) cũng bung. Nhãn: P, F, LT (LT/D/T/LỖI THỜI), K ((a)(b)(c)/K), còn lại (S, M, X, «chưa chạy»…) = N.
Dùng: python3 env/qc2-count.py [--codes]   (--codes: in mã không phải P)"""
import re, glob, os, sys, collections
R = os.path.join(os.path.dirname(os.path.abspath(__file__)), '..')
def expand(a, b):
    ma = re.match(r'^(.*?)(\d+|M\d+)$', a); mb = re.match(r'^(.*?)(\d+|M\d+)$', b)
    if not ma or not mb or ma.group(1) != mb.group(1): return [a, b]
    pa, sa = ma.groups(); sb = mb.group(2)
    if sa.startswith('M') != sb.startswith('M'): return [a, b]
    if sa.startswith('M'): return [f'{pa}M{i}' for i in range(int(sa[1:]), int(sb[1:]) + 1)]
    return [f'{pa}{i:0{len(sa)}d}' for i in range(int(sa), int(sb) + 1)]
def label(s):
    if re.match(r'^(P|F)\b', s): return s[0]
    if re.match(r'^(LT|D|T)\b|^LỖI THỜI', s): return 'LT'
    if re.match(r'^(K\w*|\(a\)|\(b\)|\(c\))', s): return 'K'
    return 'N'
def count(path, oc=False):
    last = {}
    for l in open(path, encoding='utf-8'):
        if not l.startswith('|'): continue
        c = [x.strip() for x in l.strip().strip('|').split('|')]
        if len(c) < 2 or not re.match(r'^(TC-|P0-|AUDIT|LANDING|DOCK|DEV-|MISC|COLOR|ANTI)', c[0]): continue
        lab = label(c[2] if oc else c[1])
        ids = []
        parts = c[0].split(',') if re.fullmatch(r'(TC-|P0-)\S+(, (TC-|P0-)\S+)+', c[0]) else [c[0]]
        for part in parts:
            part = part.strip()
            ids += expand(*[x.strip() for x in part.split('…')]) if '…' in part else [part]
        for i in ids: last[i] = lab
    return last
if __name__ == '__main__':
    for p in sorted(glob.glob(R + '/smoke-*.md')):
        sp = os.path.basename(p)[6:-3]
        last = count(p, 'opencode' in sp)
        c = collections.Counter(last.values())
        print(f"{sp}\ttổng {len(last)}\tP {c['P']}\tF {c['F']}\tN {c['N']}\tLT {c['LT']}\tK {c['K']}")
        if '--codes' in sys.argv:
            print('   ', [k for k, v in last.items() if v != 'P'])
