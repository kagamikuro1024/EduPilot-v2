#!/bin/bash
# usage: upd6.sh "việc đang làm"  — merge6.py (sinh smoke-6.md) rồi ghi dòng `sprint 6` vào qc3.tsv
cd ~/Documents/TA_Agent_qc3/docs/regression
python3 evidence-6/scripts-r3/merge6.py >/dev/null
python3 - "$1" <<'P'
import sys,os
H=os.path.expanduser('~/.config/edupilot-team/progress/')
t,p,f=open(H+'qc3.s6').read().split('\t')[1:4]
L=open(H+'qc3.tsv').read().splitlines(); out=[]; done=False
for l in L:
    if l.startswith('sprint 6\t'): out.append(f'sprint 6\t{t}\t{p}\t{f}\t0\t{sys.argv[1]}'); done=True
    else: out.append(l)
if not done: out.append(f'sprint 6\t{t}\t{p}\t{f}\t0\t{sys.argv[1]}')
open(H+'qc3.tsv','w').write('\n'.join(out)+'\n')
P
grep '^sprint 6' ~/.config/edupilot-team/progress/qc3.tsv
