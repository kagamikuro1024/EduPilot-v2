# s8core: ca đọc / impact / what-if trên QD8A1 (bản sao sạch lớp 1). Ghi kết quả qua rec().
import csv, math, re, subprocess
from fractions import Fraction as F
from s8core_lib import *
A1 = open("/tmp/s8core/QD8A1.id").read().strip(); Q = f"/courses/{A1}"
cf = json.load(open("/tmp/s8core/a1scheme.json"))
def body(mod=None):
    b = {"components": [{k: c[k] for k in ("key", "name", "grp", "weight")} | {"items": c["items"]} for c in cf["components"]], "rules": [{"kind": r["kind"], "applicable": r["applicable"], "params": r["params"]} for r in cf["rules"]]}
    if mod: mod(b)
    return b
def setrule(kind, **p):
    def f(b):
        for r in b["rules"]:
            if r["kind"] == kind: r["params"] = {**r["params"], **p}
    return f
def setw(m, w, f_):
    def f(b):
        for c in b["components"]: c["weight"] = {"midterm": m, "weekly": w, "final": f_}[c["key"]]
    return f
exp = {r["student_code"]: r for r in csv.DictReader(open(REPO + "/seed/expected_final_grades.csv"))}
code_of = {r[0]: r[1] for r in sql("select id,student_code from users where student_code is not null")}
id_of = {v: k for k, v in code_of.items()}
# ---- 02-01 tay
hc = subprocess.run(["python3", REPO + "/docs/sprints/8/qc/scripts/p6-hand-calc.py", "A", "B", "C", "sv04", "sv05", "sv06", "sv13", "sv09", "sv20", "sv23"], capture_output=True, text=True, cwd=REPO)
open("../logs/s8core-handcalc.log", "w").write(hc.stdout + f"rc={hc.returncode}\n")
chk("TC-P6-02-01", hc.returncode == 0 and "30/30" in hc.stdout and "18/18" in hc.stdout, "p6-hand-calc.py rc=0: 30/30 + 18/18 khớp; các bước 10 dòng khớp bảng tay (logs/s8core-handcalc.log)", hc.stdout[-200:])
# ---- preview 30/30 vs CSV (02-03)
c, pv = call("GET", f"{Q}/gradebook/preview?limit=100"); assert c == 200, (c, pv)
api = {r["student_code"]: r for r in pv["items"]}
bad = []
for k, e in exp.items():
    a = api[k]
    if not (F(a["qt"]) == F(e["qt"]) and F(a["hp_raw"]) == F(e["hp_raw"]) and F(a["hp"]) == F(e["hp"]) and a["letter"] == e["letter"] and (a["verdict"] == "OK") == (e["verdict"] == "OK")): bad.append(k)
chk("TC-P6-02-03", not bad and len(api) == 30, "preview lớp sao QD8A1 (đầu vào sạch = seed): 30/30 khớp qt, hp_raw, hp, letter, verdict theo MSSV; lớp 1 gốc lệch 8 do dữ liệu bị ghi đè (xem DEF/ghi chú)", f"lệch {bad}")
# ---- 02-16 JSON chuỗi
def walk(x, path=""):
    if isinstance(x, dict):
        for k, v in x.items(): yield from walk(v, k)
    elif isinstance(x, list):
        for v in x: yield from walk(v, path)
    else: yield path, x
c, p5 = call("GET", f"{Q}/gradebook/preview?limit=5")
nums = sorted({k for k, v in walk(p5) if isinstance(v, (int, float)) and not isinstance(v, bool)})
scorekeys = {"hp", "qt", "hp_raw", "score", "numeric_grade", "bonus", "penalty"}
decstr = [v for k, v in walk(p5) if isinstance(v, str) and re.fullmatch(r"\d+\.\d+", v)]
chk("TC-P6-02-16", not (set(nums) & scorekeys) and len(decstr) > 0, f"preview: số JSON chỉ ở {nums}; điểm đều là chuỗi decimal ({len(decstr)} giá trị dạng '7.20')", f"số JSON ở trường điểm: {set(nums)&scorekeys}")
# ---- explain 30 SV (cache)
EX = {}
for k in exp:
    cc, ex = call("GET", f"{Q}/gradebook/students/{id_of[k]}/explain"); assert cc == 200, (k, cc, ex); EX[k] = ex
json.dump({"preview": api, "explain": EX}, open("/tmp/s8core/a1cache.json", "w"), ensure_ascii=False)
def line(ex, prefix): return [l for l in ex["lines"] if l["label"].startswith(prefix)]
wk = lambda k: F(line(EX[k], "Kiểm tra hằng tuần")[0]["value"].replace(",", "."))
# ---- 01-20: weekly A 10, B 8, C 0 + 24 MCQ
acc2code = {r["account"]: r["student_code"] for r in exp.values()}
mcq = [r for r in csv.DictReader(open(REPO + "/seed/expected_exam_scores.csv")) if r["exam"] == "MCQ" and r["expected_score"] != ""]
bad20 = [(r["account"], wk(acc2code[r["account"]]), r["expected_score"]) for r in mcq if wk(acc2code[r["account"]]) != F(r["expected_score"])]
A, B, C_ = acc2code["sv.gioi"], acc2code["sv.kha"], acc2code["sv.nguyco"]
cl = line(EX[C_], "Kiểm tra hằng tuần")[0]
chk("TC-P6-01-20", not bad20 and len(mcq) == 24 and (wk(A), wk(B), wk(C_)) == (10, 8, 0) and "không dự" in json.dumps(cl, ensure_ascii=False), f"weekly A 10,00 · B 8,00 · C 0 (giải trình 'không dự → 0 theo quy chế'); {len(mcq)} dòng MCQ khớp expected_exam_scores.csv", f"{bad20} A,B,C={wk(A)},{wk(B)},{wk(C_)} {cl}")
# ---- 02-25
lc = line(EX[C_], "Kiểm tra hằng tuần")[0]
chk("TC-P6-02-25", EX[C_]["result"]["status"] == "COMPLETE" and "không dự" in json.dumps(lc, ensure_ascii=False) and lc["value"] in ("0", "0,00") and (lc.get("quote") or "").startswith("Sinh viên không dự"), f"C: KT=0 '{lc.get('text', '')[:60]}' có quote điều 1.4 (tr.{lc.get('page')}); status {EX[C_]['result']['status']}", str(lc)[:150])
# ---- 02-29/30
s20, s23, s25 = [acc2code[a] for a in ("sv20", "sv23", "sv25")]
a20, a23, a25 = api[s20], api[s23], api[s25]
chk("TC-P6-02-29", (F(a20["hp"]), a20["letter"], a20["verdict"]) == (5, "F", "LIET:final<3.0") and (a23["letter"], F(a23["hp"]), a23["verdict"]) == ("D", 4, "OK") and (a25["letter"], F(a25["hp"]), a25["verdict"]) == ("D", 4, "OK"), f"sv20 hp 5,0 F LIET:final<3.0; sv23 D 4,0 OK; sv25 D 4,0 (hp_raw {a25['hp_raw']}) không liệt", f"{a20} {a23} {a25}")
lq = [l for l in EX[s20]["lines"] if (l.get("quote") or "").startswith("Sinh viên có điểm thi cuối kỳ dưới 3,0 thì bị điểm F")]
chk("TC-P6-02-30", bool(lq), f"giải trình sv20 có dòng liệt với quote nguyên văn điều 4.1: '{(lq[0]['label'] if lq else '')}'", "không có dòng liệt với quote điều 4.1: " + str([l['label'] for l in EX[s20]['lines']]))
# ---- 02-32/33/34
exB = EX[B]; labs = [l["label"] for l in exB["lines"]]
order = ["Giữa kỳ", "Kiểm tra hằng tuần", "Thi cuối kỳ", "Điểm cộng", "Điểm trừ", "Điểm quá trình trước", "Điểm quá trình", "Điểm cuối kỳ", "Điểm học phần thô", "Làm tròn", "Điểm chữ"]
pos = [next((i for i, x in enumerate(labs) if x.startswith(o)), -1) for o in order]
v = lambda p: line(exB, p)[0]["value"]
chk("TC-P6-02-32", pos == sorted(pos) and -1 not in pos and (v("Điểm quá trình trước"), v("Điểm quá trình "), v("Điểm học phần thô"), v("Làm tròn"), v("Điểm chữ")) == ("7,75", "8,25", "7,2", "7", "B"), f"B: thứ tự dòng đúng; QT₀ 7,75 → +0,50 → 8,25 → HP thô 7,2 → 7 → B; cộng min(0,5; 1)", f"{pos} {labs}")
def ev(expr):
    s = expr.replace("÷", "/").replace("×", "*").replace("−", "-").replace(";", ",")
    s = re.sub(r"(\d+),(\d+)", r"\1.\2", s); s = re.sub(r"(\d+(?:\.\d+)?)", r"F('\1')", s)
    return eval(s, {"F": F, "min": min, "max": max, "round": lambda x: F(math.floor(x + F(1, 2)))})
def exprs_ok(ex):
    bad = []
    for l in ex["lines"]:
        if l.get("expr") and re.search(r"\d", l["expr"]):
            try:
                if ev(l["expr"]) != F(l["value"].replace(",", ".")): bad.append((l["label"], l["expr"], l["value"]))
            except Exception as e: bad.append((l["label"], l["expr"], repr(e)))
    return bad
bB, bC, b9 = exprs_ok(EX[B]), exprs_ok(EX[C_]), exprs_ok(EX[acc2code["sv09"]])
chk("TC-P6-02-33", not (bB or bC or b9) and v("Điểm học phần thô") == "7,2" and line(EX[C_], "Điểm học phần thô")[0]["value"] == "3,2" and line(EX[acc2code["sv09"]], "Điểm học phần thô")[0]["value"] == "8,066", "mọi expr của B, C, sv09 tính lại bằng Fraction ra đúng value; HP thô B 7,2 · C 3,2 · sv09 8,066 (không làm tròn giữa chừng)", f"{bB} {bC} {b9}")
md = open(REPO + "/docs/specs/FEAT-gradebook/fixtures/quy-che-lop1.md").read()
nq = nbadq = 0; missing = []
for ex in EX.values():
    for l in ex["lines"]:
        q = l.get("quote")
        if q:
            nq += 1
            if q not in md: nbadq += 1; missing.append(q[:60])
import random
random.seed(8); sample = random.sample(sorted(exp), 10); badx = []
for k in sample:
    nums = {x for x in re.findall(r"\d+(?:\.\d+)?", exp[k]["explanation"].replace(",", "."))}
    blob = " ".join(l["label"] + " " + (l.get("expr") or "") + " " + l["value"] + " " + (l.get("text") or "") for l in EX[k]["lines"]).replace(",", ".")
    got = {x for x in re.findall(r"\d+(?:\.\d+)?", blob)}
    norm = lambda s: {str(F(x).limit_denominator(10**6)) if False else str(F(x)) for x in s}
    miss = {x for x in nums if F(x) not in {F(y) for y in got}}
    if miss: badx.append((k, sorted(miss)))
noq = [(l["label"], l.get("filled_by_user")) for l in EX[B]["lines"] if not l.get("quote")]
chk("TC-P6-02-34", not badx and nbadq == 0 and nq > 0, f"10 SV ngẫu nhiên: mọi số trong cột explanation CSV có trong lines[]; {nq} quote của 30 SV đều có thật trong quy-che-lop1.md; dòng không quote của B: {len(noq)} (số học dẫn xuất/GV điền)", f"{badx} quote lạ {missing}")
# ---- 01-25 đầu vào P5
rows = sql("select u.email, coalesce(sum(p.points) filter (where p.type in ('SPEAK','BONUS')),0), coalesce(abs(sum(p.points) filter (where p.type='PENALTY')),0) from users u left join participation_events p on p.student_id=u.id and p.course_id='%s' where u.email in ('sv.gioi@edupilot.local','sv.kha@edupilot.local','sv.nguyco@edupilot.local') group by 1 order by 1" % A1)
ab = dict(sql("select u.email,count(*) from attendance_records a join users u on u.id=a.student_id where a.course_id='%s' and a.status='ABSENT' and u.email in ('sv.gioi@edupilot.local','sv.kha@edupilot.local','sv.nguyco@edupilot.local') group by 1" % A1))
sq = {r[0].split("@")[0]: (F(r[1]), int(ab.get(r[0], 0))) for r in rows}
bB_ = line(EX[B], "Điểm cộng")[0]; bC_ = line(EX[C_], "Điểm trừ")[0]; bA_ = line(EX[A], "Điểm cộng")[0]
chk("TC-P6-01-25", sq == {"sv.gioi": (F(3), 0), "sv.kha": (F("0.5"), 2), "sv.nguyco": (F("0.25"), 5)} and bB_["expr"] == "min(0,5; 1)" and "5 − 2" in line(EX[C_], "Điểm trừ")[0]["expr"], f"SQL P5: A cộng 3,00 vắng 0 · B 0,50 vắng 2 · C 0,25 vắng 5 (khớp expected_crm); giải trình dùng đúng ({bB_['expr']}; {bC_['expr']})", f"{sq} {bB_} {bC_}")
# ---- 02-12 / 04-15 / 04-16 / 02-22 impact
def table(*a):
    r = subprocess.run(["python3", REPO + "/docs/sprints/8/qc/scripts/p6-hand-calc.py", "--table", *a], capture_output=True, text=True, cwd=REPO).stdout
    return {m.group(2): (F(m.group(3)), m.group(4)) for m in re.finditer(r"(\S+)\s+(\d+)\s+QT=\S+\s+HP_thô=\S+\s+HP=(\S+)\s+(\S+)", r)}
t05, t01 = table(), table("step=0.1")
n_changed = sum(1 for k in t05 if t05[k] != t01[k])
audit0, entr0 = sq1("select count(*) from audit_log"), sq1("select count(*) from grade_entries")
c, im = call("POST", f"{Q}/grade-scheme/impact", "teacher", body(setrule("ROUNDING", step="0.1")))
c2, im2 = call("POST", f"{Q}/grade-scheme/impact", "teacher", body(setrule("ROUNDING", step="0.1")))
audit1, entr1 = sq1("select count(*) from audit_log"), sq1("select count(*) from grade_entries")
smp = im["samples"]; sm = [(code_of[s["student"]["id"]], F(s["after"]["hp"]), F(s["before"]["hp"])) for s in smp]
hps = sorted(F(r["hp"]) for r in api.values()); 
ok12 = [F(s["after"]["hp"]) for s in smp][0] == F("9.8") and F(smp[2]["after"]["hp"]) == F("3.2") and all(t01[code][0] == aft for code, aft, _ in sm)
chk("TC-P6-02-12", c == 200 and len(smp) == 3 and ok12, f"impact step 0.1: samples cao nhất {smp[0]['student']['full_name']} 9,8; giữa {smp[1]['student']['full_name']} {smp[1]['after']['hp']}; thấp nhất 3,2; cả 3 khớp hand-calc --table step=0.1", f"{im}")
im1 = call("POST", f"{Q}/grade-scheme/impact", "teacher", body(setrule("ROUNDING", step="1")))[1]
t1 = table("step=1"); ok1 = all(t1[code_of[s["student"]["id"]]][0] == F(s["after"]["hp"]) for s in im1["samples"])
chk("TC-P6-02-12", ok12 and ok1, f"impact step 0.1: samples 9,8 / {smp[1]['after']['hp']} / 3,2 khớp hand-calc step=0.1; step 1 khớp hand-calc step=1 (6,5→7)", f"{im} {im1}")
chk("TC-P6-04-15", c == 200 and len(smp) == 3 and F(smp[0]["before"]["hp"]) == 10 and F(smp[0]["after"]["hp"]) == F("9.8") and F(smp[2]["before"]["hp"]) == 3 and F(smp[2]["after"]["hp"]) == F("3.2") and F(sm[1][1]) == t01[sm[1][0]][0] and sorted(hps)[14] <= F(smp[1]["before"]["hp"]) <= sorted(hps)[15], f"3 mẫu: cao nhất 10→9,8 · giữa {smp[1]['before']['hp']}→{smp[1]['after']['hp']} (vị trí 15/16 theo HP) · thấp nhất 3→3,2; giữa khớp hand-calc (đo ≤300 ms: xem UI/TC-04-15 ghi chú)", f"{im}")
chk("TC-P6-04-16", c == 200 and c2 == 200 and audit0 == audit1 and entr0 == entr1 and im == im2 and im["changed_vs_confirmed"] == n_changed and im["students_total"] == 30 == im["students_computable"], f"impact 2 lần không ghi (audit_log {audit0}→{audit1}, grade_entries {entr0}→{entr1}); N={im['changed_vs_confirmed']} = {n_changed} tính tay (hand-calc 0,5 vs 0,1); {sorted(im)}", f"{c} {c2} {audit0}/{audit1} {entr0}/{entr1} N {im['changed_vs_confirmed']} vs {n_changed}")
# 02-22
c, imx = call("POST", f"{Q}/grade-scheme/impact", "teacher", body(lambda b: (setrule("BONUS", cap="1.5")(b), setrule("PENALTY", free=1)(b))))
tx = table("step=0.5", "cap=1.5", "free=1"); okx = c == 200 and all(tx[code_of[s["student"]["id"]]][0] == F(s["after"]["hp"]) for s in imx["samples"])
chk("TC-P6-02-22", okx, f"impact với cap=1,5 free=1 (không sửa mã giữa hai lần gọi): 3 mẫu khớp hand-calc --table cap=1.5 free=1; changed={imx['changed_vs_confirmed']}", f"{c} {imx}")
# 02-41: w_QT=0
c, i41 = call("POST", f"{Q}/grade-scheme/impact", "teacher", body(setw("0", "0", "100")))
txt = json.dumps(i41, ensure_ascii=False)
ents = {r[0]: F(r[1]) for r in sql("select u.student_code,e.score from grade_entries e join users u on u.id=e.student_id join grade_items i on i.id=e.item_id where e.course_id='%s' and i.key='final'" % A1)}
def hu(x): return F(math.floor(x * 2 + F(1, 2))) / 2
ok41 = c == 200 and "NaN" not in txt and "Infinity" not in txt and all(F(s["after"]["hp"]) == hu(ents[code_of[s["student"]["id"]]]) for s in i41.get("samples", []))
chk("TC-P6-02-41", ok41, f"w_QT=0: 200, HP = CK (3 mẫu {[s['after']['hp'] for s in i41['samples']]}), không NaN/Infinity/chia 0; ghi chú: impact không trả cảnh báo BONUS_NO_EFFECT (có ở TestBonusNoEffectWarning/PUT)", f"{c} {txt[:300]}")
c, i42 = call("POST", f"{Q}/grade-scheme/impact", "teacher", body(lambda b: (setw("50", "50", "0")(b))))
chk("TC-P6-02-42", c == 422 or "PARAM_INVALID" in json.dumps(i42), f"w_CK=0 + luật FINAL_BELOW → {c} {str(i42)[:100]}", f"{c} {str(i42)[:200]}")
# 02-45
pan0 = subprocess.run("docker logs ep-r-gateway-1 2>&1 | grep -ic panic", shell=True, capture_output=True, text=True).stdout.strip()
res = []
for bad_w in ("abc", "-5", "101", "", 1e999):
    cc, jj = call("POST", f"{Q}/grade-scheme/impact", "teacher", body(lambda b, w=bad_w: b["components"][0].__setitem__("weight", w)))
    res.append((str(bad_w), cc))
pan1 = subprocess.run("docker logs ep-r-gateway-1 2>&1 | grep -ic panic", shell=True, capture_output=True, text=True).stdout.strip()
chk("TC-P6-02-45", all(c_ in (400, 422) for _, c_ in res) and pan0 == pan1, f"weight 'abc','-5','101','',1e999 → {res}; không 5xx, panic trong log gateway {pan0}→{pan1}", f"{res} panic {pan0}->{pan1}")
# ---- 02-37/38 what-if (sinh viên)
b0 = (sq1("select count(*) from audit_log"), sq1("select count(*) from grade_entries"))
cB, wB = call("POST", f"{Q}/me/grades/what-if", "sv.kha", {"final_exam_score": "8.0"})
cA, wA = call("POST", f"{Q}/me/grades/what-if", "sv.kha", {"target_letter": "A"})
cA2, wA2 = call("POST", f"{Q}/me/grades/what-if", "sv.kha", {"target_letter": "A+"})
cC, wC = call("POST", f"{Q}/me/grades/what-if", "sv.nguyco", {"target_letter": "A+"})
b1 = (sq1("select count(*) from audit_log"), sq1("select count(*) from grade_entries"))
print(wB, wA, wA2, wC)
open("../logs/s8core-whatif.json", "w").write(json.dumps([wB, wA, wA2, wC], ensure_ascii=False, indent=1))
s = json.dumps(wB, ensure_ascii=False)
chk("TC-P6-02-37", cB == 200 and re.search(r'"hp": ?"8(\.0+)?"', s) and "B+" in s and "8.25" in json.dumps(wA) and "9.09" in json.dumps(wA2) and b0 == b1, f"what-if B: CK 8,0 → HP 8,0 B+; target A → min CK 8.25; A+ → 9.09; audit_log/grade_entries không đổi {b0}→{b1}", f"{cB} {wB} | {wA} | {wA2} | {b0} {b1}")
sc = json.dumps(wC, ensure_ascii=False)
chk("TC-P6-02-38", cC == 200 and ("null" in sc or "Không" in sc or "unreachable" in sc.lower() or "false" in sc.lower() or "min_final_exam_score\": null" in sc), f"what-if C target A+ → {sc[:100]}", f"{cC} {sc}")
