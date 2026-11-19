import csv, json, sys
from decimal import Decimal as D
pv = {r["student_code"]: r for r in json.load(open(sys.argv[1]))["items"]}
bad = n = 0
for r in csv.DictReader(open("seed/expected_final_grades.csv")):
    p = pv.get(r["student_code"]); n += 1
    if not p: print("THIẾU", r["student_code"]); bad += 1; continue
    ok = D(p["qt"]) == D(r["qt"]) and D(p["hp_raw"]) == D(r["hp_raw"]) and D(p["hp"]) == D(r["hp"]) and p["letter"] == r["letter"]
    if not ok: print("LỆCH", r["student_code"], p["qt"], r["qt"], p["hp"], r["hp"], p["letter"], r["letter"]); bad += 1
print(f"{n - bad}/{n}"); sys.exit(1 if bad else 0)
