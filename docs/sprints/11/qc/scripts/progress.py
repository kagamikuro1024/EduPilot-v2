# Ghi /tmp/qc-progress.tsv từ results-p10.jsonl + danh sách TC trong các tệp tc-*.md (nhóm, tổng, đạt, trượt, chặn/không kiểm được, đang làm)
import re, json, sys, os
D = os.path.dirname(os.path.abspath(__file__)); Q = os.path.dirname(D)
def ids(fn, pat=None):
    out = []
    for l in open(os.path.join(Q, fn), encoding="utf8"):
        if not l.startswith("| TC-"): continue
        if pat and not re.search(pat, l): continue
        out.append(l.split("|")[1].strip())
    return sorted(set(out))
res = {}
for l in open(os.path.join(D, "results-p10.jsonl")):
    d = json.loads(l)
    for m in re.finditer(r"TC-(\d\d)-(\d+[a-z]?)", d["tc"]):
        k = f"TC-{m.group(1)}-{m.group(2)}"; res.setdefault(k, {})[d["tc"]] = d["st"]
extra = json.load(open(os.path.join(D, "progress-extra.json"))) if os.path.exists(os.path.join(D, "progress-extra.json")) else {}
def tally(all_ids):
    p = f = k = 0
    for i in all_ids:
        r = res.get(i)
        if not r: r2 = extra.get(i)
        else: r2 = "FAIL" if "FAIL" in r.values() else "PASS"
        if r2 == "PASS": p += 1
        elif r2 == "FAIL": f += 1
        elif r2 == "K": k += 1
    return p, f, k
groups = [("Đợt1 quan sát AI", ids("tc-US-P10-01-03.md")), ("Đợt2 insights", ids("tc-US-P10-04-05.md")), ("Đợt2 điểm thi", ids("tc-US-P10-06-07.md")),
          ("/assignments SV", ["A"] * 6), ("P-11-15", ["B"] * 4), ("US-P10-14 audit", ids("tc-US-P10-14.md")),
          ("@real F3", ["F3"] * 3), ("@real F5", ["F5"]), ("@real F9/F11", ["F9", "F11"]), ("@real F16", ["F16"]), ("@real F17", ["F17"]), ("U1 phiên", ["U1"] * 3), ("LCP 7 route", ["LCP"] * 7)]
cur = sys.argv[1] if len(sys.argv) > 1 else ""
with open("/tmp/qc-progress.tsv", "w") as f:
    for name, tc in groups:
        p, fl, k = tally(tc) if name.startswith(("Đợt", "US-P10-14")) else (extra.get(name, [0, 0, 0]) + [0, 0, 0])[:3]
        f.write(f"{name}\t{len(tc)}\t{p}\t{fl}\t{k}\t{cur if cur.startswith(name[:3]) or not cur else ''}\n")
print(open("/tmp/qc-progress.tsv").read())
