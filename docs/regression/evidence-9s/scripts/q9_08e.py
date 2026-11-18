#!/usr/bin/env python3
"""TC-P7-08-05: bài có câu đầu chứa ký tự ngoài BMP + dấu tổ hợp (stub quote=1 → trích dẫn = câu đầu) trong lớp QD9V; ghi /tmp/q9s/ui08e.json."""
import sys, os, json; sys.path.insert(0, os.path.dirname(__file__))
from q9 import *
U = json.load(open("/tmp/q9s/ui.json")); Q = f"/api/v1/courses/{U['course']}"; T = tok("teacher")
txt = "😀 e\u0301 Em bắt đầu bài làm bằng một câu có biểu tượng 😀 và chữ ghép e\u0301 để thử vị trí UTF-16, SUNBURST là điểm vào của chiến dịch. " + "Câu tiếp theo trình bày về nhóm APT29 và chuỗi cung ứng phần mềm. " * 30 + "[[S9G quote=1]]"
with stub_ctx():
    aid = mkassign(Q, T, "Q8E" + str(int(time.time()) % 100000), due_days=3)
    s, x, _ = submit(Q, aid, tok("sv32"), "emoji.md", txt.encode()); assert s in (200, 201), (s, x)
    s, g, _ = call("POST", f"{Q}/assignments/{aid}/grade-all", T, {}); assert s == 202, (s, g); wait_done(aid, 120); time.sleep(2)
sub = sub_id(aid, "sv32"); res = result(aid, "sv32")
ev = [e for c in res["criteria"] for e in c.get("evidence", [])]
print("evidence:", json.dumps(ev, ensure_ascii=False)[:400])
u16 = txt.encode("utf-16-le"); e0 = ev[0] if ev else None
ok = bool(e0) and u16[e0["start"] * 2:e0["end"] * 2].decode("utf-16-le") == e0["quote"]
print("offset UTF-16 cắt đúng trích dẫn:", ok, "| theo code point:", bool(e0) and txt[e0["start"]:e0["end"]] == e0["quote"])
json.dump({"aid": aid, "sub": sub, "res": res["id"], "quote": e0["quote"] if e0 else None, "utf16_ok": ok}, open("/tmp/q9s/ui08e.json", "w"))
