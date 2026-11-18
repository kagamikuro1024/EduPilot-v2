#!/usr/bin/env python3
"""apply105.py: nối kết quả cuối của từng TC (results105.jsonl, dòng sau thắng) vào smoke-10.5.md, khối «Lượt 2026-11-19»."""
import json, re, os
D = os.path.dirname(os.path.abspath(__file__)); md = os.path.join(D, "..", "..", "smoke-10.5.md")
last = {}
for l in open(os.path.join(D, "results105.jsonl"), encoding="utf8"):
    r = json.loads(l)
    if re.fullmatch(r"TC-D\d\d-\d\d", r["tc"]): last[r["tc"]] = r
txt = open(md, encoding="utf8").read()
H = "\n## Lượt 2026-11-19 — chạy nốt 105 ca «N» (mã gateway 4ac1638a, worker 6abb09af, stack ep-qc; lớp riêng QD105A/QD105B; không đụng 761987/761988)\n| TC | KQ | Ghi chú |\n| --- | --- | --- |\n"
head, _, tail = txt.partition(H)
if not _: head = txt.rstrip("\n") + "\n"
rows = "\n".join(f"| {k} | {r['kq']} | {r['note'].replace('|','/').replace(chr(10),' ')[:700]} |" for k, r in sorted(last.items()))
open(md, "w", encoding="utf8").write(head + H + rows + "\n")
print(len(last), "ca →", md)
