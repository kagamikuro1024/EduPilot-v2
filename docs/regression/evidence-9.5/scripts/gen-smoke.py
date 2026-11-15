#!/usr/bin/env python3
"""QC2 r95 — dựng docs/regression/smoke-9.5.md từ bộ TC + /tmp/qc2-r95-results.tsv (TC<TAB>KQ<TAB>ghi chú; dòng sau thắng).
KQ: P | F | K (a/b/c) | LT (lỗi thời) | CM (chưa chạy, mẫu). Thiếu dòng = CM."""
import re, glob, collections, sys
R = "/Users/kuro/Documents/TA_Agent_qc2"
tcs = []  # (id, us)
for f in sorted(glob.glob(R + "/docs/sprints/9.5/qc/tc-US-NFR-*.md")):
    us = re.search(r"NFR-(\d+)", f).group(1)
    for l in open(f, encoding="utf8"):
        m = re.match(r"\| (TC-N\d+-\d+) \|", l)
        if m: tcs.append((m.group(1), "US-NFR-" + us))
res = {}
for l in open("/tmp/qc2-r95-results.tsv", encoding="utf8"):
    p = l.rstrip("\n").split("\t")
    if len(p) >= 3: res[p[0]] = (p[1], p[2])
CM_NOTE = "chưa chạy (mẫu): chưa có script, ngoài mẫu «chính + nhánh lỗi/phân quyền»"
rows = [(t, us) + res.get(t, ("CM", CM_NOTE)) for t, us in tcs]
cnt = collections.Counter(r[2] for r in rows)
T = len(rows)
assert sum(cnt.values()) == T
by = collections.defaultdict(collections.Counter)
for t, us, k, n in rows: by[us][k] += 1
head = open(R + "/docs/regression/evidence-9.5/smoke-head.md", encoding="utf8").read().format(T=T, P=cnt["P"], F=cnt["F"], K=cnt["K"], LT=cnt["LT"], CM=cnt["CM"])
o = [head, "## Tổng hợp theo US", "| US | TC tổng | P | F | K (a/b/c) | LT | CM | Ghi chú |", "| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |"]
notes = {l.split("|")[0]: l.split("|")[1] for l in open(R + "/docs/regression/evidence-9.5/us-notes.txt", encoding="utf8").read().splitlines() if "|" in l}
for us in sorted(by):
    c = by[us]; o.append(f"| {us} | {sum(c.values())} | {c['P']} | {c['F']} | {c['K']} | {c['LT']} | {c['CM']} | {notes.get(us, '')} |")
o.append(f"| **Cộng** | **{T}** | **{cnt['P']}** | **{cnt['F']}** | **{cnt['K']}** | **{cnt['LT']}** | **{cnt['CM']}** | |")
o += ["", "## Kết quả từng ca", "Ký hiệu: P đạt · F trượt · K không chạy (a thiết bị · b người thật · c ngoài phạm vi) · LT lỗi thời (spec/quyết định sau đổi) · CM chưa chạy (mẫu).", "| TC | KQ | Ghi chú |", "| --- | --- | --- |"]
i = 0
while i < len(rows):
    t, us, k, n = rows[i]
    if k in ("K", "LT", "CM") and n:
        j = i
        while j + 1 < len(rows) and rows[j + 1][2] == k and rows[j + 1][3] == n and rows[j + 1][1] == us: j += 1
        o.append(f"| {t if j == i else t + '…' + rows[j][0]} | {k} | {n} |")
        i = j + 1
    else:
        o.append(f"| {t} | {k} | {n} |"); i += 1
open(R + "/docs/regression/smoke-9.5.md.body", "w", encoding="utf8").write("\n".join(o) + "\n")
print(T, dict(cnt))
