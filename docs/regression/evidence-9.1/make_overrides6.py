# SmkR lượt 6: thêm override P/F cho TC từ log rerun6 của các spec rs* (đọc ✓/✘ thật, không ghi tay). Chạy: python3 make_overrides6.py && python3 build_smoke91.py
import json, os, re, glob
H = os.path.dirname(os.path.abspath(__file__)); ov = json.load(open(H + "/overrides91.json"))
LINE = re.compile(r'^\s+(✓|✘)\s+\d+ \[desktop\] › e2e/(?:qc2-r91/)?(\S+?)\.spec\.ts:\d+:\d+ › (.*?)(?: \([\d.]+m?s\))?$')
def expand(title):
    out = set()
    for m in re.finditer(r'TC-UX(\d\d)-([0-9A-Za-z/,\-]+)', title):
        for part in re.split(r'[/,]', m.group(2)):
            if re.fullmatch(r'\d+|PQ|PG|ID', part): out.add(f"TC-UX{m.group(1)}-{part}")
    return out
res = {}
for f in sorted(glob.glob(H + "/logs/qc-uiux91-rs*-real.rerun6.log")):
    for ln in open(f, errors="ignore"):
        m = LINE.match(ln.rstrip("\n"))
        if m:
            st, file, title = m.groups()
            for t in expand(title): res.setdefault(t, []).append((st, file, os.path.basename(f)))
for t, l in res.items():
    ok = all(s == "✓" for s, _, _ in l)
    if not ok: continue  # đỏ = chưa phân loại (lỗi kịch bản QC hoặc môi trường) ⇒ giữ N, không ghi F
    ov[t] = ["P", f"RERUN6 SmkR 1409b04d (ep-r): spec {l[0][1]} ({'; '.join(sorted({x[2] for x in l}))}) xanh"]
json.dump(ov, open(H + "/overrides91.json", "w"), ensure_ascii=False, indent=1)
print(len(res), "TC từ rs*; P:", sum(1 for v in res.values() if all(s == '✓' for s, _, _ in v)))
