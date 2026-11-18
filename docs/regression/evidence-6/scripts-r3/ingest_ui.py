#!/usr/bin/env python3
"""Đọc /tmp/qc3e/ui/results.jsonl → thêm vào results-r3.tsv các TC thật (id không có hậu tố); dòng cuối thắng."""
import json,re
RES='/Users/kuro/Documents/TA_Agent_qc3/docs/regression/evidence-6/results-r3.tsv'
last={}
for l in open('/tmp/qc3e/ui/results.jsonl'):
    d=json.loads(l)
    if re.fullmatch(r'TC-P\d-\d\d-\d\d',d['tc']): last[d['tc']]=d
with open(RES,'a') as f:
    for tc,d in last.items(): f.write(f"{tc}\t{'P' if d['ok'] else 'F'}\t{' '.join(d['note'].split())[:600]}\n")
print(len(last),'ingested')
