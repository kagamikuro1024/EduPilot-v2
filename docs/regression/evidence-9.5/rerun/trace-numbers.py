#!/usr/bin/env python3
# TC-N09-04/05: trích số có đơn vị trong báo cáo research, tìm trong benchmarks/reports/** (json/md) và docs/research/2026-11-14-explain.md
import re, glob, json, sys, os
R = os.path.expanduser("~/Documents/TA_Agent_qc2")
rep = open(f"{R}/docs/research/2026-11-14-nfr-real-llm.md").read()
# bỏ khối bảng danh sách câu cần vá (không phải số đo)
rep = rep.split("## Danh sách câu cần vá")[0]
src_files = [f for f in glob.glob(f"{R}/benchmarks/reports/**/*", recursive=True) if f.endswith((".json", ".md"))] + [f"{R}/docs/research/2026-11-14-explain.md", f"{R}/docs/sprints/9.5/report.md"]
nums = set(); text = ""
NUM = re.compile(r"-?\d+(?:\.\d+)?")
def add(v, d=None):
    nums.add(round(v, 6))
for f in src_files:
    t = open(f, errors="ignore").read(); text += t + "\n"
    for m in NUM.finditer(t): add(float(m.group()))
    # số kiểu vi: 4.217 (nghìn)
    for m in re.finditer(r"\d{1,3}(?:\.\d{3})+", t): add(float(m.group().replace(".", "")))
    # ms->s, s->h, ratio %, ms->s
    for m in NUM.finditer(t):
        v = float(m.group()); add(v/1000); add(v*100); add(v/3600); add(v/60); add(v*1000)
def parse(s):
    s = s.strip()
    if re.fullmatch(r"\d{1,3}(?:\.\d{3})+", s): return float(s.replace(".", "")), 0
    s2 = s.replace(",", ".")
    d = len(s2.split(".")[1]) if "." in s2 else 0
    return float(s2), d
def found(x, d):
    for n in nums:
        if abs(n - x) < 10**(-d) * 0.5 + 1e-9: return True
    return False
UNITS = r"(ms|s|%|MB|GB|MiB|KB|h|giờ|QPS|điểm|phút|giây)"
pat = re.compile(r"(?<![\w./-])(\d{1,3}(?:\.\d{3})+|\d+(?:,\d+)?)\s*" + UNITS + r"(?![\w])")
seen = []; miss = []
for ln_no, ln in enumerate(rep.split("\n"), 1):
    for m in pat.finditer(ln):
        x, d = parse(m.group(1)); u = m.group(2)
        tag = (ln_no, m.group(0))
        seen.append(tag)
        ok = found(x, d)
        if not ok and u == "%": ok = found(x/100, d+2) or found(x, d)
        if not ok: miss.append(tag)
print("số có đơn vị:", len(seen), "không truy được:", len(miss))
for t in miss: print(t)
