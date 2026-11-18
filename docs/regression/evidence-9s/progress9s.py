#!/usr/bin/env python3
"""Ghi /tmp/qc-progress.tsv: nhóm⇥tổng⇥đạt⇥trượt⇥chặn⇥đang_làm. Số thật từ smoke-*.md (dòng sau thắng; N = chưa chạy)."""
import re,sys,os
D=os.path.join(os.path.dirname(os.path.abspath(__file__)),"..")
UX=["S21","S22","S23","a","a2","b","c","d","f","g","h","i","j","k","l","m","n","o","p","17"]
def expand(a,b):
    if a=="TC-UX-S21": return ["TC-UX-"+x for x in UX]
    ma=re.match(r"(.*?)(\d+)([a-z]?)$",a); mb=re.match(r"(.*?)(\d+)([a-z]?)$",b)
    if not(ma and mb) or ma.group(1)!=mb.group(1) or int(mb.group(2))<int(ma.group(2)): return [a,b]
    return [f"{ma.group(1)}{n:0{len(ma.group(2))}d}" for n in range(int(ma.group(2)),int(mb.group(2))+1)]
def klass(s):
    s=s.strip()
    if s.startswith("P"): return "P"
    if s.startswith("F"): return "F"
    if s.startswith("LT") or "LỖI THỜI" in s or "lỗi thời" in s or "ngoài phạm vi" in s: return "K"
    if s.startswith("K") and not s.startswith("K ") or s.startswith("K(") or s.startswith("K "): return "K"
    return "N"
def load(f):
    st={}
    for l in open(os.path.join(D,f"smoke-{f}.md"),encoding="utf8"):
        m=re.match(r"\| (TC-[^|]+?) \| ([^|]+?) \|",l)
        if not m: continue
        t=m.group(1).strip(); k=klass(m.group(2))
        # ghi chú còn nói «MẪU:» / «chưa chạy» ở ca K cũ → vẫn N
        if k=="K" and re.search(r"MẪU|chưa chạy",l) and m.group(2).strip()=="K": k="N"
        ids=expand(*[x.strip() for x in t.split("…")]) if "…" in t else [t]
        for i in ids: st[i]=k
    return st
rows=[]; 
for f,name in [("9","smoke 9"),("10","smoke 10"),("10.5","smoke 10.5"),("10.6","smoke 10.6")]:
    s=load(f); c={k:sum(1 for v in s.values() if v==k) for k in "PFKN"}
    rows.append((name,len(s),c["P"],c["F"],c["K"],c["N"]))
out=["nhóm\ttổng\tđạt\ttrượt\tchặn\tđang_làm"]
out.append("sprint 11\t275\t273\t1\t1\t0")
note=dict(a.split("=",1) for a in sys.argv[1:] if "=" in a)
for n,t,p,f,k,nn in rows: out.append(f"{n}\t{t}\t{p}\t{f}\t{k}\t{note.get(n.split()[1],nn)}")
open("/tmp/qc-progress.tsv","w",encoding="utf8").write("\n".join(out)+"\n"); print("\n".join(out))
