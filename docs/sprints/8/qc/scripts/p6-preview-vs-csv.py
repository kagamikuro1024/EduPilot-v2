#!/usr/bin/env python3
"""QC P6 — so /tmp/pv.json (GET …/gradebook/preview?limit=100) với seed/expected_final_grades.csv độc lập với check-grade-seed.mjs.
   Khớp theo MSSV (student_code); so qt, hp_raw, hp, letter, verdict (OK / LIET). Thoát 1 nếu lệch."""
import csv, json, sys
from decimal import Decimal as D
pv = {r["student_code"]: r for r in json.load(open(sys.argv[1] if len(sys.argv) > 1 else "/tmp/pv.json"))["items"]}
bad = 0; n = 0
for r in csv.DictReader(open("seed/expected_final_grades.csv")):
    p = pv.get(r["student_code"]); n += 1
    if not p: print("THIẾU", r["student_code"], r["account"]); bad += 1; continue
    ok = D(p["qt"]) == D(r["qt"]) and D(p["hp_raw"]) == D(r["hp_raw"]) and D(p["hp"]) == D(r["hp"]) and p["letter"] == r["letter"] and (p["verdict"] == "OK") == (r["verdict"] == "OK")
    if not ok: print("LỆCH", r["account"], {k: p[k] for k in ("qt", "hp_raw", "hp", "letter", "verdict")}, "BA:", {k: r[k] for k in ("qt", "hp_raw", "hp", "letter", "verdict")}); bad += 1
print(f"{n - bad}/{n} khớp (so qt, hp_raw, hp, letter, verdict)"); sys.exit(1 if bad else 0)
