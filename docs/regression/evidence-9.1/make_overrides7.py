# SmkS lượt 7: ghi override P từ log *.rerun7*.log (✓ thật, mọi TC trong tiêu đề test; chỉ tính project desktop) + kết quả tay trong results-smkS.tsv (TC⇥P|F⇥ghi chú; bản ghi sau thắng, thắng cả log).
# Chạy: python3 make_overrides7.py && python3 build_smoke91.py && python3 build_defects91.py(nếu có)
import json, os, re, glob
H = os.path.dirname(os.path.abspath(__file__)); ov = json.load(open(H + "/overrides91.json"))
LINE = re.compile(r'^\s+(✓|✘)\s+\d+ \[desktop\] › e2e/(?:qc2-r91/)?(\S+?)\.spec\.ts:\d+:\d+ › (.*?)(?: \([\d.]+m?s\))?$')
def expand(title):
    out = set()
    for m in re.finditer(r'TC-UX(\d\d)-([0-9A-Za-z/,]+)', title):
        st = m.group(1)
        for part in re.split(r'[/,]', m.group(2)):
            x = re.fullmatch(r'UX(\d\d)-(\w+)', part)
            if re.fullmatch(r'\d+[a-z]?|PQ|PG|ID', part): out.add(f"TC-UX{st}-{re.sub('[a-z]$','',part)}")
    for m in re.finditer(r'/(UX(\d\d)-(\d+|PQ|PG))\b', title): out.add(f"TC-{m.group(1)}")
    return out
last = {}  # tc -> (✓/✘, file, log)
for f in sorted(glob.glob(H + "/logs/*.rerun7*.log")):
    for ln in open(f, errors="ignore"):
        m = LINE.match(ln.rstrip("\n"))
        if not m: continue
        st, file, title = m.groups()
        for t in expand(title):
            last.setdefault(t, {})[os.path.basename(f)] = (st, file)
n = 0
SKIP = {'TC-UX16-9', 'TC-UX16-10', 'TC-UX12-2'}  # nhóm tiêu đề không có assertion riêng: chấm tay trong results-smkS.tsv
for t, d in last.items():
    if t in SKIP: continue
    logs = sorted(d)
    st, file = d[logs[-1]]
    if st == "✓":
        ov[t] = ["P", f"RERUN7 SmkS (ep-r, lớp sạch QR91A khi cần): spec {file} ✓ ({logs[-1]})"]; n += 1
man = H + "/results-smkS.tsv"
if os.path.exists(man):
    for ln in open(man, encoding="utf-8"):
        p = ln.rstrip("\n").split("\t")
        if len(p) >= 3 and p[1] in ("P", "F"): ov[p[0]] = [p[1], p[2]]; n += 1
json.dump(ov, open(H + "/overrides91.json", "w"), ensure_ascii=False, indent=1)
print("ghi", n)
