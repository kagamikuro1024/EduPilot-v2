#!/usr/bin/env python3
"""QC US-P6-03 AC5: mọi citation {page, quote} phải có thật, đúng TRANG, trong văn bản quy chế.
Dùng:  python3 p6-citation-check.py <quy-che.md> <extraction.json | draft.json>   (rc=0 khi mọi trích dẫn khớp)
  - quy-che.md: nguồn có dấu '<!-- trang N -->' (docs/specs/FEAT-gradebook/fixtures/quy-che-lop{1,2}.md)
  - JSON: có khoá "citations":[{"page":N,"quote":"…"}] (fixture BA hoặc bản nháp lấy từ `GET …/grade-scheme` → .draft.extraction/.citations)
Chuẩn hoá giống spec: NFC, gộp khoảng trắng, hạ chữ thường (bỏ dấu cho phần dung sai OCR: chỉ để báo "khớp sau bỏ dấu" riêng).
"""
import json, re, sys, unicodedata

def norm(s): return re.sub(r"\s+", " ", unicodedata.normalize("NFC", s)).strip().lower()
def fold(s): return "".join(c for c in unicodedata.normalize("NFD", s) if unicodedata.category(c) != "Mn").replace("đ", "d")

md = open(sys.argv[1], encoding="utf-8").read()
parts = re.split(r"<!--\s*trang\s+(\d+)\s*-->", md)
pages = {int(parts[i]): norm(re.sub(r"<!--.*?-->", "", parts[i + 1], flags=re.S)) for i in range(1, len(parts), 2)}
data = json.load(open(sys.argv[2], encoding="utf-8"))
cits = data["citations"] if isinstance(data, dict) and "citations" in data else data
bad = 0
for c in cits:
    q, pg = norm(c["quote"]), int(c["page"])
    if pg not in pages: print(f"FAIL {c.get('id','?')}: trang {pg} không tồn tại"); bad += 1
    elif q in pages[pg]: print(f"OK   {c.get('id','?')} tr.{pg}")
    elif fold(q) in fold(pages[pg]): print(f"OK~  {c.get('id','?')} tr.{pg} (chỉ khớp sau bỏ dấu)")
    else:
        other = [p for p, t in pages.items() if q in t]
        print(f"FAIL {c.get('id','?')}: không có trong tr.{pg}" + (f" (có ở tr.{other})" if other else " (không có ở trang nào)")); bad += 1
print(f"{len(cits) - bad}/{len(cits)} trích dẫn đúng"); sys.exit(1 if bad else 0)
