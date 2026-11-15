# s8r-pw2tsv: đọc JSON của Playwright (tên test bắt đầu [TC-A,TC-B]) → ghi part-s8r.tsv (đạt = P; trượt = F kèm thông điệp). Dùng: s8r-pw2tsv.py <json> [ghi-chú-thêm]
import json, re, sys, os
sys.path.insert(0, os.path.dirname(__file__))
from s8r_lib import T
d = json.load(open(sys.argv[1])); extra = sys.argv[2] if len(sys.argv) > 2 else ""
def walk(s):
    for sp in s.get("specs", []):
        m = re.match(r"\[([^\]]+)\]\s*(.*)", sp["title"])
        if not m: continue
        r = sp["tests"][0]["results"][-1]; ok = r["status"] == "passed"
        msg = re.sub(r"\x1b\[[0-9;]*m", "", (r.get("error") or {}).get("message", "")); msg = " ".join(msg.split())[:120]
        for tc in m.group(1).split(","):
            tc = tc.strip()
            if tc.startswith("TC-"): T(tc, "P" if ok else "F", (f"UI thật ep-r: {m.group(2)} — xanh {extra}" if ok else f"UI thật: {m.group(2)} — {msg}"))
    for c in s.get("suites", []): walk(c)
for s in d["suites"]: walk(s)
