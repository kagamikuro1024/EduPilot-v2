# Dựng smoke-9.1.md từ log Playwright (lượt 1 + lượt chạy lại ca lỗi) + ca GREP chạy tay
import re, json, glob, os, collections
H = os.path.dirname(os.path.abspath(__file__)); L = H + "/logs"
rows = json.load(open('/tmp/qc2-f-s91.json'))
def expand(title):
    out = set()
    for m in re.finditer(r'TC-UX(\d\d)-([0-9A-Za-z/\-,]+)', title):
        st = m.group(1)
        for part in re.split(r'[/,]', m.group(2)):
            if re.fullmatch(r'\d+[a-z]?|PQ|PG|P\d*|PQ\d*|X\d', part): out.add(f"TC-UX{st}-{part}")
            elif re.fullmatch(r'\d+-\d+', part):
                a, b = part.split('-')
                for n in range(int(a), int(b) + 1): out.add(f"TC-UX{st}-{n}")
    for m in re.finditer(r'TC-UX91-(X\d)', title): out.add(f"TC-UX91-{m.group(1)}")
    return out
LINE = re.compile(r'^\s+(✓|✘|-)\s+\d+ \[(\w+)\] › e2e/(?:qc2-r91/)?(\S+?)\.spec\.ts:\d+:\d+ › (.*?)(?: \([\d.]+m?s\))?$')
res = {}   # (file,title,project) -> st
for f in sorted(glob.glob(L + "/*.log")):
    if "grep-batch" in f: continue
    rerun = f.endswith(".rerun.log")
    for ln in open(f, errors="ignore"):
        m = LINE.match(ln.rstrip("\n"))
        if not m: continue
        st, proj, file, title = m.groups()
        if st == '-': continue
        key = (file.replace("qc-uiux91-", ""), title.strip(), proj)
        if rerun or key not in res: res[key] = "P" if st == "✓" else "F"
tc = collections.defaultdict(lambda: [0, 0, set()])
for (file, title, proj), st in res.items():
    for t in expand(title):
        tc[t][0 if st == "P" else 1] += 1; tc[t][2].add(file)
ov = json.load(open(H + "/overrides91.json")) if os.path.exists(H + "/overrides91.json") else {}
out_rows = []; cnt = collections.Counter(); byw = collections.defaultdict(collections.Counter)
for t, (f, m, cells) in rows.items():
    wave = os.path.basename(f).replace("tc-", "").replace(".md", "")
    us = t.split('-')[1]
    if t in ov: kq, note = ov[t]
    elif t in tc and tc[t][1] == 0 and tc[t][0] > 0: kq, note = "P", "spec " + ",".join(sorted(tc[t][2]))
    elif t in tc and tc[t][1] > 0: kq, note = "F", f"spec {','.join(sorted(tc[t][2]))}: {tc[t][1]} lượt đỏ / {tc[t][0]} xanh (xem logs/)"
    else: kq, note = "M", f"chưa chạy (mẫu; dạng {m}, không có test tự động hoặc test bị skip)"
    cnt[kq] += 1; byw[wave][kq] += 1; out_rows.append((t, kq, note, wave))
json.dump({"rows": out_rows, "cnt": cnt, "byw": {k: dict(v) for k, v in byw.items()}}, open(H + "/s91-rows.json", "w"), ensure_ascii=False)
print(cnt, len(out_rows))
