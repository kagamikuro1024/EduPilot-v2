#!/usr/bin/env python3
"""Hợp nhất kết quả smoke sprint 6 (qc3, lượt R3): đọc smoke-6.md hiện tại + results-r3.tsv (TC<TAB>KQ<TAB>ghi chú; dòng cuối thắng),
mở rộng dòng nhóm N thành từng ca, xoá ca lỗi thời có nguồn, ghi lại smoke-6.md với đếm khớp. Chạy lại bao nhiêu lần cũng được."""
import re,glob,os,sys,collections
R='/Users/kuro/Documents/TA_Agent_qc3/docs/regression'
TCD='/Users/kuro/Documents/TA_Agent_qc3p/docs/sprints/6/qc'
HANDOFF=open(f'{R}/evidence-6/HANDOFF-qc3-s6.txt').read() if os.path.exists(f'{R}/evidence-6/HANDOFF-qc3-s6.txt') else ''
SM=f'{R}/smoke-6.md'; RES=f'{R}/evidence-6/results-r3.tsv'; ORIG=f'{R}/evidence-6/smoke-6.orig.md'
# Ca lỗi thời có nguồn quyết định: xoá hẳn (ghi vào defects-6.md)
LT={'TC-P3-05-19':'FEAT-escalation-notify US-P4-01 AC7 (sprint 7, APPROVED): nút «Nhờ giảng viên hỗ trợ» nay hiện dưới mọi câu trả lời DONE; quyết định ẩn 2026-10-10 chỉ tới sprint 7','TC-P3-05-44':'như TC-P3-05-19 (US-P4-01 AC7 thay quyết định ẩn nút 2026-10-10)','TC-P3-01-05':'proposals #13 (số 00009 bỏ trống, calendar là 00011)','TC-P3-03-31':'GET /admin/llm/usage mở cho TEACHER: openapi «ADMIN, TEACHER» (sprint 9/10)','TC-P3-04-45':'US-P4-07 2026-10-14: khủng hoảng hai tầng, câu mẫu đổi'}
if not os.path.exists(ORIG): open(ORIG,'w').write(open(SM).read())
src=open(ORIG).read()
us_ids=collections.OrderedDict()
for f in sorted(glob.glob(f'{TCD}/tc-US-*.md')):
    u=re.search(r'tc-(US-P\d-\d\d)',f).group(1); us_ids[u]=sorted(set(re.findall(r'(?m)^\| (TC-P\d-\d\d-\d\d) \|',open(f).read())))
allids=[i for v in us_ids.values() for i in v]
old={}
for ln in src.splitlines():
    m=re.match(r'\| (TC-P\d-\d\d-\d\d) \| ([^|]+?) \| (.*) \|$',ln)
    if m: old[m.group(1)]=(m.group(2).strip(),m.group(3)); continue
    m=re.match(r'\|\.*(\d\d) … (TC-P\d-\d\d)-(\d\d) \((\d+) ca\) \| N',ln) or re.match(r'\| (TC-P\d-\d\d)-(\d\d) … (TC-P\d-\d\d)-(\d\d) \((\d+) ca\) \| N',ln)
    if m:
        g=m.groups()
        if len(g)==5: pre,a,_,b=g[0],int(g[1]),g[2],int(g[3])
        else: pre,a,b=g[1],int(g[0]),int(g[2])
        for k in range(a,b+1): old[f'{pre}-{k:02d}']=('N','')
res={}
if os.path.exists(RES):
    for ln in open(RES):
        p=ln.rstrip('\n').split('\t')
        if len(p)>=2 and p[0].startswith('TC-'): res[p[0]]=(p[1],p[2] if len(p)>2 else '')
rows=[];cnt=collections.Counter();per=collections.defaultdict(collections.Counter)
for u,ids in us_ids.items():
    for i in ids:
        if i in LT: continue
        kq,note=old.get(i,('N',''))
        if kq in('K','K(c)','K(b)','K(a)','T','N'): kq,note='N','chờ chạy lượt R3'
        if i in res: kq,note=res[i]
        rows.append((i,kq,note)); cnt[kq]+=1; per[u][kq]+=1
tot=len(rows); P=cnt['P']; F=cnt['F']; N=tot-P-F
hdr=f'''# Smoke hồi quy sprint 6 — kết quả
Mã chạy: 634d59c5 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc3 · Cập nhật: lần cuối theo `evidence-6/results-r3.tsv`
Tổng TC (đã xoá ca lỗi thời có nguồn): {tot} · Đạt: {P} · Trượt: {F} · Chờ test N: {N}   (P+F+N = {tot}). Ca lỗi thời đã xoá: {len(LT)} (xem cuối `defects-6.md`).
'''
# giữ phần mô tả cách chạy từ bản gốc (từ «Quy ước» tới trước «## Tổng hợp theo US»)
mid=src[src.index('Quy ước:'):src.index('## Tổng hợp theo US')]
tab='## Tổng hợp theo US\n| US | TC tổng | P | F | N | Ghi chú |\n|---|---|---|---|---|---|\n'
for u,ids in us_ids.items():
    c=per[u]; t=sum(c.values()); tab+=f'| {u} | {t} | {c["P"]} | {c["F"]} | {t-c["P"]-c["F"]} |  |\n'
body='\n## Kết quả từng ca\n| TC | KQ | Ghi chú |\n|---|---|---|\n'+''.join(f'| {i} | {k} | {n} |\n' for i,k,n in rows)
tail=src[src.index('## Trượt: bước tái hiện'):src.index('## Ca K / T / N và lý do')]
open(SM,'w').write(hdr+'\n'+mid+tab+body+'\n'+tail+'## Ca còn N\nXem bảng; mọi ca N nằm trong hàng đợi chạy (không còn nhãn K/T).\n\n'+HANDOFF)
d=f'{R}/defects-6.md'; s=open(d).read()
for k,v in LT.items():
    line=f'- LT-đã-xoá {k}: {v}\n'
    if line not in s: s+= ('' if s.endswith('\n') else '\n')+line; open(d,'w').write(s)
print(tot,P,F,N)
open('/tmp/qc3-progress.tsv.s6','w').write(f'sprint 6\t{tot}\t{P}\t{F}\t0\t')
