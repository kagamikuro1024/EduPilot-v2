#!/usr/bin/env python3
"""rec.py TC-Gxx-yy P|F|N 'ghi chú' → results_106.jsonl"""
import json, sys, os
f = os.path.join(os.path.dirname(os.path.abspath(__file__)), "results_106.jsonl")
open(f, "a", encoding="utf8").write(json.dumps({"tc": sys.argv[1], "kq": sys.argv[2], "note": sys.argv[3] if len(sys.argv) > 3 else ""}, ensure_ascii=False) + "\n"); print(sys.argv[2], sys.argv[1])
