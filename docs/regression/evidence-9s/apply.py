#!/usr/bin/env python3
"""apply.py <smoke-N.md> <results.jsonl> [heading]: nối vào smoke-N.md các dòng KQ cuối của từng TC (tc = mã TC chính xác; dòng sau thắng) chưa có trong khối «Lượt 2026-11-18 (kết quả)»."""
import json, re, sys, os
smoke, res = sys.argv[1], sys.argv[2]
D = os.path.dirname(os.path.abspath(__file__)); md = os.path.join(D, "..", smoke)
last = {}
for l in open(res, encoding="utf8"):
    r = json.loads(l)
    if re.fullmatch(r"TC-[A-Za-z0-9-]+", r["tc"]): last[r["tc"]] = r
txt = open(md, encoding="utf8").read()
H = "\n## " + (sys.argv[3] if len(sys.argv) > 3 else "Lượt 2026-11-18 — kết quả chạy thật (mã dc67bd93, stack ep-qc, lớp riêng QD10A/QD10B; không đụng 761987/761988)") + "\n| TC | KQ | Ghi chú |\n| --- | --- | --- |\n"
if H not in txt: txt += H
# gỡ dòng cũ của chính khối này để ghi lại bản mới nhất
head, _, tail = txt.partition(H)
tail = "\n".join(l for l in tail.split("\n") if not re.match(r"\| (%s) \| " % "|".join(re.escape(k) for k in last), l))
rows = "\n".join(f"| {k} | {r['kq']} | {r['note'].replace('|','/').replace(chr(10),' ')[:600]} |" for k, r in last.items())
open(md, "w", encoding="utf8").write(head + H + tail.rstrip("\n") + ("\n" if tail.strip() else "") + rows + "\n")
print(len(last), "ca →", smoke)
