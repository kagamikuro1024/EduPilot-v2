#!/usr/bin/env python3
"""QC P6 — so /tmp/pv.json (GET …/gradebook/preview?limit=100) với seed/expected_final_grades.csv, độc lập với check-grade-seed.mjs.
   Mặc định khớp theo MSSV (SEED_RNG mặc định). `--by-account` khớp theo email `<account>@edupilot.local` (lấy từ Postgres của stack QC): dùng khi SEED_RNG khác.
   So qt, hp_raw, hp, letter, verdict (OK / LIET). Thoát 1 nếu lệch."""
import csv, json, subprocess, sys
from decimal import Decimal as D
args = [a for a in sys.argv[1:] if not a.startswith("--")]
by_acc = "--by-account" in sys.argv
items = json.load(open(args[0] if args else "/tmp/pv.json"))["items"]
if by_acc:
    out = subprocess.run(["docker", "exec", "edupilot100-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-tAF,", "-c", "select id, email from users"], capture_output=True, text=True).stdout.split()
    mail = dict(l.split(",") for l in out)
    pv = {mail[r["student_id"]].split("@")[0]: r for r in items}
else:
    pv = {r["student_code"]: r for r in items}
bad = 0; n = 0
for r in csv.DictReader(open("seed/expected_final_grades.csv")):
    key = r["account"] if by_acc else r["student_code"]; p = pv.get(key); n += 1
    if not p: print("THIẾU", key); bad += 1; continue
    ok = D(p["qt"]) == D(r["qt"]) and D(p["hp_raw"]) == D(r["hp_raw"]) and D(p["hp"]) == D(r["hp"]) and p["letter"] == r["letter"] and (p["verdict"] == "OK") == (r["verdict"] == "OK")
    if not ok: print("LỆCH", r["account"], {k: p[k] for k in ("qt", "hp_raw", "hp", "letter", "verdict")}, "BA:", {k: r[k] for k in ("qt", "hp_raw", "hp", "letter", "verdict")}); bad += 1
print(f"{n - bad}/{n} khớp theo {'account' if by_acc else 'MSSV'} (so qt, hp_raw, hp, letter, verdict)"); sys.exit(1 if bad else 0)
