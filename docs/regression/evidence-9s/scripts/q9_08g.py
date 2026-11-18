#!/usr/bin/env python3
import sys, os, json; sys.path.insert(0, os.path.dirname(__file__))
from q9 import *
U = json.load(open("/tmp/q9s/ui.json")); F = json.load(open("/tmp/q9s/ui08f.json")); Q = f"/api/v1/courses/{U['course']}"
s, x, _ = submit(Q, F["aid"], tok("sv33"), "v2.md", ("Bản nộp lại của sv33. SUNBURST. " * 12).encode()); print("nộp lại", s)
T = tok("teacher"); print("review bản cũ:", call("POST", f"{Q}/grading-results/{F['res']['sv33']}/review", T, {"version": 1}, None)[:2])
print("review RUNNING:", call("POST", f"{Q}/grading-results/{F['res']['sv34']}/review", T, {"version": 1}, None)[:2])
