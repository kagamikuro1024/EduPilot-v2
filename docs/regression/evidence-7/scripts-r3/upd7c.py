#!/usr/bin/env python3
"""Áp /tmp/qc3e/res7c.json ({tc:[P|F,note]}) vào smoke-7.md (mở rộng dòng gom N, cập nhật tổng) + /tmp/qc3-progress.tsv."""
import json,re,sys,os
SM='/Users/kuro/Documents/TA_Agent_qc3/docs/regression/smoke-7.md'
CAT=json.load(open('/tmp/qc3e/cat.json')); ORDER=CAT['order']+[x for x in CAT['st'] if x not in CAT['order']]
RES=json.load(open('/tmp/qc3e/res7c.json'))
L=open(SM).read().split('\n')
def us(i):
    m=re.match(r'TC-(P\d-\d\d)-',i)
    if m: return 'US-'+m.group(1)
    if i.startswith('TC-S7-SEED'): return 'US-S7-SEED'
    return 'US-CI-02'
out=[];stat={}
for l in L:
    m=re.match(r'^\| (TC-[\w-]+)(?: … (TC-[\w-]+) \((\d+) ca\))? \| ([PFN]) \| (.*) \|$',l)
    if not m: out.append(l); continue
    a,b,n,st,note=m.groups()
    ids=[a] if not b else ORDER[ORDER.index(a):ORDER.index(b)+1]
    assert not b or len(ids)==int(n),(a,b,len(ids),n)
    for i in ids:
        if i in RES: st2,note2=RES[i]; stat[i]=st2; out.append(f'| {i} | {st2} | {note2} |')
        else: stat[i]=st; out.append(f'| {i} | {st} | {note} |')
# đếm
rows=[(re.match(r'^\| (TC-[\w-]+) \| ([PFN]) \|',l)) for l in out]; rows=[r.groups() for r in rows if r]
tot=len(rows); P=sum(1 for _,s in rows if s=='P'); F=sum(1 for _,s in rows if s=='F'); N=sum(1 for _,s in rows if s=='N')
byus={}
for i,s in rows: d=byus.setdefault(us(i),[0,0,0,0]); d[0]+=1; d['PFN'.index(s)+1]+=1
res=[]
for l in out:
    m=re.match(r'^\| (US-[\w-]+) \| \d+ \| \d+ \| \d+ \| \d+ \|(.*)$',l)
    if m: d=byus[m.group(1)]; l=f'| {m.group(1)} | {d[0]} | {d[1]} | {d[2]} | {d[3]} |{m.group(2)}'
    if l.startswith('Tổng TC:'): l=re.sub(r'Đạt: \d+ · Trượt: \d+ · Chưa chạy N: \d+ +\(P\+F\+N = \d+ = \d+\)',f'Đạt: {P} · Trượt: {F} · Chưa chạy N: {N}   (P+F+N = {P+F+N} = {tot})',l)
    res.append(l)
open(SM,'w').write('\n'.join(res))
print('tot',tot,'P',P,'F',F,'N',N,'| N ids:',[i for i,s in rows if s=='N'])
tsv=open('/tmp/qc3-progress.tsv').read().split('\n')
act=sys.argv[1] if len(sys.argv)>1 else f'N={N}'
tsv=[f'sprint 7\t{tot}\t{P}\t{F}\t0\t{act}' if l.startswith('sprint 7\t') else l for l in tsv]
open('/tmp/qc3-progress.tsv','w').write('\n'.join(tsv))
