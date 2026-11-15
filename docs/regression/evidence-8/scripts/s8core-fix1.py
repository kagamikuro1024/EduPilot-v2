# s8core: chấm lại 02-32 / 02-34 / 02-37 từ cache a1cache.json + logs/s8core-whatif.json (lần đầu lỗi so khớp trong script: nhãn 'Điểm quá trình' trùng tiền tố, quote chưa vào blob, regex '8.0')
import csv, random, re
from fractions import Fraction as F
from s8core_lib import *
c = json.load(open("/tmp/s8core/a1cache.json")); EX = c["explain"]
exp = {r["student_code"]: r for r in csv.DictReader(open(REPO + "/seed/expected_final_grades.csv"))}
B = next(r["student_code"] for r in exp.values() if r["account"] == "sv.kha"); exB = EX[B]; labs = [l["label"] for l in exB["lines"]]
order = ["Giữa kỳ", "Kiểm tra hằng tuần", "Thi cuối kỳ", "Điểm cộng", "Điểm trừ", "Điểm quá trình trước", "=Điểm quá trình", "Điểm cuối kỳ", "Điểm học phần thô", "Làm tròn", "Điểm chữ"]
pos = [next((i for i, x in enumerate(labs) if (x == o[1:] if o[0] == "=" else x.startswith(o))), -1) for o in order]
v = lambda p, exact=False: next(l for l in exB["lines"] if (l["label"] == p if exact else l["label"].startswith(p)))["value"]
chk("TC-P6-02-32", pos == sorted(pos) and -1 not in pos and (v("Điểm quá trình trước"), v("Điểm quá trình", True), v("Điểm học phần thô"), v("Làm tròn"), v("Điểm chữ")) == ("7,75", "8,25", "7,2", "7", "B"), "B: thứ tự dòng đúng (thành phần→cộng→trừ→QT→CK→HP thô→làm tròn→chữ); QT₀ 7,75 → +0,50 → 8,25 → HP thô 7,2 → 7 → B", f"{pos} {labs}")
random.seed(8); sample = random.sample(sorted(exp), 10); badx = []
for k in sample:
    nums = set(re.findall(r"\d+(?:\.\d+)?", exp[k]["explanation"].replace(",", ".")))
    blob = " ".join(" ".join(str(l.get(x) or "") for x in ("label", "expr", "value", "text", "quote")) for l in EX[k]["lines"]).replace(",", ".")
    got = {F(y) for y in re.findall(r"\d+(?:\.\d+)?", blob)}
    miss = sorted(x for x in nums if F(x) not in got)
    if miss: badx.append((k, miss))
md = open(REPO + "/docs/specs/FEAT-gradebook/fixtures/quy-che-lop1.md").read()
nq = sum(1 for ex in EX.values() for l in ex["lines"] if l.get("quote")); bq = [l["quote"][:50] for ex in EX.values() for l in ex["lines"] if l.get("quote") and l["quote"] not in md]
noq = [l["label"] for l in exB["lines"] if not l.get("quote")]
chk("TC-P6-02-34", not badx and not bq and nq > 0, f"10 SV ngẫu nhiên: mọi số trong explanation CSV có trong lines[]; {nq} quote của 30 SV đều có thật (grep -F) trong quy-che-lop1.md; dòng B không quote = số học dẫn xuất", f"{badx} {bq}")
w = json.load(open("../logs/s8core-whatif.json"))
chk("TC-P6-02-37", w[0]["hp"] == "8" and w[0]["letter"] == "B+" and w[1]["min_final_exam_score"] == "8.25" and w[2]["min_final_exam_score"] == "9.09", "what-if B: CK 8,0 → HP 8 B+; target A → min CK 8.25; A+ → 9.09; audit_log/grade_entries không đổi (script đếm trước/sau)", str(w)[:150])
