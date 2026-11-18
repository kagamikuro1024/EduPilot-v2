#!/usr/bin/env python3
"""Dựng lại docs/regression/smoke-7.md từ: bảng cũ (smoke-7.md hiện có) + kết quả mới /tmp/qc3e/master7.json
(bản ghi cuối của mỗi TC thắng; không xoá kết quả cũ trong evidence). Ca K/T cũ -> N (luật mới của PM); ca LT có nguồn bị xoá."""
import json,re,sys,collections
SM='/Users/kuro/Documents/TA_Agent_qc3/docs/regression/smoke-7.md'
CAT=json.load(open('/tmp/qc3e/cat.json')); order=CAT['order']+[x for x in CAT['st'] if x not in CAT['order']]
DEL={'TC-P5-06-19':'proposals sprint 7 #34 (PM ACCEPTED 2026-10-15)','TC-S7-SEED-10':'proposals sprint 8 #5 (ACCEPTED có sửa; WATCH không còn ghim)'}
try: NEW=json.load(open('/tmp/qc3e/master7.json'))
except Exception: NEW={}
old={}; oldnote={}
for l in open('/tmp/qc3e/smoke-7.orig.md'):
    if not l.startswith('| TC-') : continue
    c=[x.strip() for x in l.rstrip('\n').strip('|').split('|',2)]
    names=re.findall(r'(TC-[\w-]+)',c[0]); st=c[1]; note=c[2] if len(c)>2 else ''
    if len(names)==1: ids=names
    else:
        a,b=names; n=1 if b=='TC-id' else int(re.search(r'\((\d+) ca\)',c[0]).group(1))
        O=CAT['order']
        if a=='TC-E1': ids=['TC-E1','TC-E2','TC-E3','TC-E4','TC-27']
        elif b=='TC-id': ids=[a]
        elif a in O and b in O: ids=O[O.index(a):O.index(b)+1]
        else:
            pre=a.rsplit('-',1)[0]
            ids=[f'{pre}-E{i}' for i in range(1,n)]+[pre+'-P1'] if b.endswith('P1') else [f'{pre}-E{i}' for i in range(1,n+1)]
    for i in ids: old[i]=st; oldnote[i]=note
keys=list(CAT['st'])
res={}
for k in keys:
    if k in DEL: continue
    if k in NEW: res[k]=(NEW[k][0],NEW[k][1])
    else:
        st=old[k]; note=oldnote[k]
        if st in('K','T'): res[k]=('N',f'(trước: {st}) {note}')
        else: res[k]=(st,note)
def us(k):
    m=re.match(r'TC-(P\d-\d\d|S7-SEED)',k)
    return 'US-'+m.group(1) if m else 'US-CI-02'
cnt=collections.OrderedDict()
for k,(s,_) in res.items(): cnt.setdefault(us(k),collections.Counter())[s]+=1
tot=collections.Counter(s for s,_ in res.values()); T=len(res)
hdr=open('/tmp/qc3e/smoke-7.orig.md').read().split('## Tổng hợp theo US')[0].split('\n')
hdr[1]='Mã chạy: 634d59c5 (sprint/11-p10-eval, bản dựng trên ep-r) · Stack ep-r · Người chạy: qc3 (Qc3S7, chạy lại từ bản ghi qc2 ngày 2026-11-15) · Ngày: 2026-11-18'
hdr[2]=f'Tổng TC: {T} (582 cũ − {len(DEL)} lỗi thời có nguồn đã xoá) · Đạt: {tot["P"]} · Trượt: {tot["F"]} · Chưa chạy N: {tot["N"]}   (P+F+N = {tot["P"]+tot["F"]+tot["N"]} = {T})'
hdr[4]='Quy ước: P = đạt; F = trượt; N = chưa chạy (chạy tới P/F thật, không còn K/T). Ca lỗi thời có nguồn quyết định đã xoá khỏi bảng, xem cuối defects-7.md. Bản ghi mới của qc3 ghi «[qc3 18/11]».'
out='\n'.join(hdr)+'\n## Tổng hợp theo US\n| US | TC tổng | P | F | N | Ghi chú |\n|---|---|---|---|---|---|\n'
for u,c in cnt.items(): out+=f'| {u} | {sum(c.values())} | {c["P"]} | {c["F"]} | {c["N"]} |  |\n'
out+='\n## Kết quả từng ca\n(Mỗi ca P/F một dòng; ca N liền nhau cùng ghi chú được gom một dòng.)\n| TC | KQ | Ghi chú |\n|---|---|---|\n'
i=0
while i<len(keys):
    k=keys[i]
    if k in DEL: i+=1; continue
    s,n=res[k]
    if s=='N' and n.startswith('chưa chạy (mẫu)'):
        j=i
        while j+1<len(keys) and keys[j+1] not in DEL and res[keys[j+1]]==(s,n): j+=1
        lab=k if j==i else f'{k} … {keys[j]} ({j-i+1} ca)'
        out+=f'| {lab} | N | {n} |\n'; i=j+1; continue
    tag='' if k not in NEW else '[qc3 18/11] '
    out+=f'| {k} | {s} | {tag}{n.replace(chr(10)," ").replace("|","/")} |\n'; i+=1
open(SM,'w').write(out)
print(T,dict(tot))
json.dump({u:dict(c) for u,c in cnt.items()},open('/tmp/qc3e/us-counts.json','w'))
