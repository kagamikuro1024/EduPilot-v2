#!/bin/bash
# usage: upd6.sh "việc đang làm"
cd ~/Documents/TA_Agent_qc3/docs/regression
python3 evidence-6/scripts-r3/merge6.py >/dev/null
IFS=$'\t' read -r a t p f z _ < $HOME/.config/edupilot-team/progress/qc3.s6
python3 - "$1" <<'P'
import sys
t,p,f=open('$HOME/.config/edupilot-team/progress/qc3.s6').read().split('\t')[1:4]
lines=open('$HOME/.config/edupilot-team/progress/qc3.tsv').read().splitlines()
out=[]
for l in lines:
    out.append(f'sprint 6\t{t}\t{p}\t{f}\t0\t{sys.argv[1]}' if l.startswith('sprint 6\t') else l)
open('$HOME/.config/edupilot-team/progress/qc3.tsv','w').write('\n'.join(out)+'\n')
P
grep '^sprint 6' $HOME/.config/edupilot-team/progress/qc3.tsv
