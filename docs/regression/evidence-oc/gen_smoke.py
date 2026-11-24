import re
s=open('../smoke-11-opencode-go.md').read()
head=s[:s.index('## Kết quả từng ca')]; tail=s[s.index('## Quan sát cho PM'):]
R=[l.split('\t',2) for l in open('results.tsv').read().splitlines() if l]
tc=open('../tc-11-opencode-go.md').read(); grp={}
for l in tc.splitlines():
    m=re.match(r'\| (TC-OC-\d+) \|[^|]*\|[^|]*\|[^|]*\|[^|]*\| (LOCAL|STACK) \|',l)
    if m: grp[m.group(1)]=m.group(2)
P=sum(r[1]=='P' for r in R);F=sum(r[1]=='F' for r in R);N=sum(r[1]=='N' for r in R)
head=re.sub(r'Tổng TC: .*',f'Tổng TC: {len(R)} · Đạt: {P} · Trượt: {F} · Chặn: {N} (TC nguồn: `tc-11-opencode-go.md`; lỗi: `defects-opencode-go.md`)',head)
head=re.sub(r'Mã đã chạy: .*',"Mã đã chạy: `8f548396` (sprint/11-p10-eval; dev `0e4bd078`, `3600257a`, `eec49533`, `76157fa7`, `843a6f4a`, `9e738647`, `18c134dd`) · LOCAL: worktree chỉ đọc qc3p, gọi thật OpenCode Go (trừ decision: không gọi thật) · STACK: `ep-qc` gateway/worker @0e4bd078, @3600257a, @843a6f4a · Người chạy: qc3 · Ngày: 2026-11-24",head,count=1)
out=head+'## Kết quả từng ca\n| TC | Nhóm | KQ | Ghi chú |\n| --- | --- | --- | --- |\n'
for t,r,n in sorted(R,key=lambda x:int(x[0].split('-')[-1])): out+=f"| {t} | {grp.get(t,'')} | {r} | {' '.join(n.split()).replace('|','/')} |\n"
open('../smoke-11-opencode-go.md','w').write(out+'\n'+tail)
print(len(R),P,F,N)
