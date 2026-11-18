#!/usr/bin/env python3
"""Ghi kết quả một ca vào results_m.jsonl (dòng sau thắng): rec.py TC-xx-yy P|F|N 'ghi chú'"""
import json, sys, os
f = os.path.join(os.path.dirname(os.path.abspath(__file__)), "results_m.jsonl")
tc, kq, note = sys.argv[1], sys.argv[2], sys.argv[3] if len(sys.argv) > 3 else ""
open(f, "a", encoding="utf8").write(json.dumps({"tc": tc, "kq": kq, "note": note}, ensure_ascii=False) + "\n"); print(kq, tc, note[:200])
