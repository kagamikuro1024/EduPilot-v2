#!/usr/bin/env python3
"""SmkS4 2026-11-21: áp evidence-9.5/results-21nov.tsv (TC<TAB>P|F<TAB>ghi chú; dòng sau thắng; dòng không phải TC-N.. bỏ qua) vào các dòng TC của docs/regression/smoke-9.5.md, rồi tính lại dòng tổng (dòng 3)."""
import re, collections
H = "/Users/kuro/Documents/TA_Agent_qc2/docs/regression/"
res = {}
for l in open(H + "evidence-9.5/results-21nov.tsv", encoding="utf-8"):
    p = l.rstrip("\n").split("\t")
    if len(p) >= 3 and re.fullmatch(r"TC-N\d+-\d+", p[0]) and p[1] in ("P", "F"): res[p[0]] = (p[1], p[2].replace("|", "/"))
L = open(H + "smoke-9.5.md", encoding="utf-8").read().split("\n")
cnt = collections.Counter(); done = set()
for i, l in enumerate(L):
    m = re.match(r"\| (TC-N\d+-\d+) \| (\w+) \| (.*) \|$", l)
    if not m: continue
    t = m.group(1)
    if t in res: L[i] = f"| {t} | {res[t][0]} | {res[t][1]} |"; done.add(t)
    cnt[re.match(r"\| \S+ \| (\w+) \|", L[i]).group(1)] += 1
T = sum(cnt.values())
print("áp", len(done), "/", len(res), "thiếu:", sorted(set(res) - done), dict(cnt), T)
for i, l in enumerate(L):
    if l.startswith("Tổng TC cũ:"):
        L[i] = re.sub(r"Tổng tính = \d+ · Đạt: \d+ · Trượt \(F xác nhận\): \d+ · K: \d+ · Chưa chạy \(N\): \d+   \(P\+F\+K\+N = \d+\)",
                      f"Tổng tính = {T} · Đạt: {cnt['P']} · Trượt (F xác nhận): {cnt['F']} · K: {cnt['K']} · Chưa chạy (N): {cnt['N']}   (P+F+K+N = {T})", l)
        L[i] = re.sub(r"\[cập nhật SmkS3[^\]]*\]", "[cập nhật SmkS4 2026-11-21, HEAD b98b9926; smk91 + ep-r]", L[i])
open(H + "smoke-9.5.md", "w", encoding="utf-8").write("\n".join(L))
