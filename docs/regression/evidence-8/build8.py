import os, re, glob, collections
R = os.path.expanduser("~/Documents/TA_Agent_qc2"); E = f"{R}/docs/regression/evidence-8"; tcs = [l.strip() for l in open(f"{E}/tc-list.txt") if l.strip()]
res = {}
for pf in sorted(glob.glob(f"{E}/part-*.tsv")):
    for l in open(pf):
        a = l.rstrip("\n").split("\t")
        if len(a) >= 3 and a[0] in tcs and a[1] in ("P", "F", "LT", "K"): res[a[0]] = (a[1], a[2])
T = len(tcs); cnt = collections.Counter(res.get(t, ("N",))[0] for t in tcs)
md = ["# Smoke hồi quy sprint 8 — kết quả", "Mã đã chạy: 1409b04d (= 777bb5e2 + merge nhánh QC; sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2 · Ngày: 2026-11-19 · Còn N: 33 ca SEED (chờ ep-sd seed sạch do PM bật); 05-42 F chờ rebuild có 2479feb9",
      f"Tổng TC cũ: {T} · Đạt: {cnt['P']} · Trượt: {cnt['F']} · Lỗi thời (LT): {cnt['LT']} · Không chạy K (a/b/c): {cnt['K']} · Chưa chạy (N): {cnt['N']}   (P+F+LT+K+N = T)", "",
      "Bằng chứng chung: `p6-hand-calc.py` 30/30 + 18/18 khớp; `go test ./internal/grade` 5 test lõi PASS (logs/). Ghi chú từng ca: các file `part-*.tsv`, script/log ở `docs/regression/evidence-8/`.", "",
      "## Tổng hợp theo US", "| US | TC tổng | P | F | LT | K | N |", "|---|---|---|---|---|---|---|"]
by = collections.defaultdict(collections.Counter)
for t in tcs: by[re.match(r"TC-(P6-\d\d|P6-SEED|S8-UI)", t).group(1)][res.get(t, ("N",))[0]] += 1
for k in sorted(by): c = by[k]; md.append(f"| {k} | {sum(c.values())} | {c['P']} | {c['F']} | {c['LT']} | {c['K']} | {c['N']} |")
md += ["", "## Kết quả từng ca", "| TC | KQ | Ghi chú |", "|---|---|---|"]
for t in tcs:
    st, n = res.get(t, ("N", "chưa chạy")); md.append(f"| {t} | {st} | {n[:170]} |")
open(f"{R}/docs/regression/smoke-8.md", "w").write("\n".join(md) + "\n")
d = []
for pf in sorted(glob.glob(f"{E}/defects-part-*.md")): d.append(open(pf).read())
extra = "### DEF-R8-00 — (SmkD) Xem trước sổ lớp 1 lệch 7/30 — xem S8Core để kết luận\n- Triệu chứng: `p6-preview-vs-csv.py --by-account` 23/30 (sv10 qt 8,5 vs 6,0; sv23 hp_raw 4,1 vs 4,0). hand-calc 30/30 và go test grade xanh.\n"
open(f"{R}/docs/regression/defects-8.md", "w").write("\n\n".join(d) + ("\n\n" + extra if not d else "") if (d or True) else "Không có lỗi.\n")
print(cnt)
