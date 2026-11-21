#!/bin/bash
# usage: upd5.sh "việc đang làm"  — chạy make_report, cập nhật dòng sprint 5 của qc3.tsv
cd ~/Documents/TA_Agent_qc3/docs/regression/evidence-5/scripts-r3
out=$(python3 make_report.py | tail -1)
python3 - "$1" "$out" <<'P'
import sys,re,os,ast
t,d=sys.argv[2].split(' ',1); d=ast.literal_eval(d); tot=int(t)
f=os.path.expanduser('~/.config/edupilot-team/progress/qc3.tsv')
lines=open(f).read().splitlines(); new=f"sprint 5\t{tot}\t{d.get('P',0)}\t{d.get('F',0)}\t0\t{sys.argv[1]} (còn S {d.get('S',0)})"
for i,l in enumerate(lines):
    if l.startswith('sprint 5\t'): lines[i]=new; break
else: lines.append(new)
open(f,'w').write('\n'.join(lines)+'\n'); print(new)
P
