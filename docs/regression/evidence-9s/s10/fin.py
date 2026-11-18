#!/usr/bin/env python3
"""fin.py TC-id 'tiền tố tên chk' [KQ]: lấy dòng CUỐI có tên bắt đầu bằng tiền tố → ghi dòng chốt tc = TC-id (dòng chốt thắng khi dựng smoke-N.md)."""
import json, sys, os
d = os.path.dirname(os.path.abspath(__file__)); f = os.path.join(d, "results_m.jsonl")
rows = [json.loads(l) for l in open(f, encoding="utf8")]
for arg in sys.argv[1:]:
    tc, pre = arg.split("=", 1)
    hit = [r for r in rows if r["tc"].startswith(pre) and r["tc"] != tc]
    if not hit: print("KHÔNG TÌM THẤY", tc, pre); continue
    r = hit[-1]; open(f, "a", encoding="utf8").write(json.dumps({"tc": tc, "kq": r["kq"], "note": (r["tc"].split(" ",1)[1] + " — " if " " in r["tc"] else "") + r["note"]}, ensure_ascii=False) + "\n"); print(r["kq"], tc, r["note"][:100])
