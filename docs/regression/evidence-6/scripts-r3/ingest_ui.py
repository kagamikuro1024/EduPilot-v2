#!/usr/bin/env python3
"""Đọc dòng MỚI của /tmp/qc3e/ui/results.jsonl (theo offset ở /tmp/qc3e/ui/ingested.n) → results-r3.tsv; id thật; dòng cuối thắng."""
import json,re,os
RES='/Users/kuro/Documents/TA_Agent_qc3/docs/regression/evidence-6/results-r3.tsv'; OFF='/tmp/qc3e/ui/ingested.n'
k=int(open(OFF).read()) if os.path.exists(OFF) else 0
lines=open('/tmp/qc3e/ui/results.jsonl').read().splitlines(); last={}
for l in lines[k:]:
    d=json.loads(l)
    if re.fullmatch(r'TC-P\d-\d\d-\d\d',d['tc']): last[d['tc']]=d
with open(RES,'a') as f:
    for tc,d in last.items(): f.write(f"{tc}\t{'P' if d['ok'] else 'F'}\t{' '.join(d['note'].split())[:600]}\n")
open(OFF,'w').write(str(len(lines))); print(len(last),'ingested')
