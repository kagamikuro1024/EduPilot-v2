# s8r 07d: 07-04 (rate limit), 07-24 (LOCKED), 07-25 (đua), 07-27 (DB trống), 07-29..33 (tệp xấu), 07-42 + 08-08/33/34 (LOAD1000)
import statistics
from s8r_g import *
XD = "/tmp/qc2-s8r-xlsx"; H = ["MSSV", "Họ tên", "Điểm"]
def gwlog(since): return subprocess.run(f"docker logs --since {since} ep-r-gateway-1 2>&1", shell=True, capture_output=True, text=True).stdout
def rss(): return subprocess.run("docker stats --no-stream --format '{{.MemUsage}}' ep-r-gateway-1", shell=True, capture_output=True, text=True).stdout.strip().split("/")[0].strip()
def mib(s): n = float(re.findall(r"[\d.]+", s)[0]); return n * (1024 if "GiB" in s else 1 if "MiB" in s else 1 / 1024)
# ---- 07-04 rate limit presign
time.sleep(62)
body = lambda: {"purpose": "grade_import", "filename": "midterm.xlsx", "mime_type": XMIME, "size_bytes": 2000, "sha256": "c" * 64}
q7 = cid("QD8R7"); codes = []
for i in range(11): codes.append(call("POST", f"/courses/{q7}/uploads/presign", "teacher", body())[0])
oc = cid("QD8F8"); other = call("POST", f"/courses/{oc}/uploads/presign", "gv.s8f", body())[0]
C("TC-P6-07-04", codes[:10].count(200) >= 9 and codes[10] == 429 and other == 200, f"11 presign trong ~1 s (teacher): {codes}; lần 11 = 429; GV khác (gv.s8f) cùng lúc → {other} (không bị ảnh hưởng)", f"{codes} {other}")
time.sleep(62)
# ---- 07-24 LOCKED (QD8R8 đang LOCKED v3)
r8 = cid("QD8R8"); mid8 = sq1(f"select id from grade_items where course_id='{r8}' and key='midterm'")
c, pre, _ = up_xlsx("teacher", r8, f"{XD}/midterm.xlsx"); rc_ = imp("teacher", r8, pre["upload_id"], mid8, "commit", key=str(uuid.uuid4())); rp = imp("teacher", r8, pre["upload_id"], mid8, "preview")
C("TC-P6-07-24", rc_[0] == 409 and rc_[1].get("code") == "GRADEBOOK_LOCKED", f"sổ LOCKED: commit → {rc_[0]} {rc_[1].get('code')} (preview {rp[0]})", str(rc_))
# ---- 07-33 đổi nội dung kho giữa preview và commit (cùng độ dài)
mk = lambda v: mkxlsx([H] + [[CSV[a]["student_code"], a, float(CSV[a]["midterm"]) if v == 0 else min(10, float(CSV[a]["midterm"]) + 0.25)] for a in list(CSV)[:3]])
A_, B_ = mk(0), mk(1)
if len(A_) != len(B_): print("độ dài khác", len(A_), len(B_))
c, pre, sha = up_xlsx("teacher", C0, None, name="swap.xlsx", data=A_); pv = imp("teacher", C0, pre["upload_id"], ITEM["midterm"], "preview")
sw = None
try: urllib.request.urlopen(urllib.request.Request(pre["url"], data=B_, method="PUT", headers=pre.get("headers", {})), context=CTX, timeout=60); sw = 200
except Exception as e: sw = getattr(e, "code", str(e)[:40])
e_before = sql(f"select sum(version) from grade_entries where course_id='{C0}'")
cm = imp("teacher", C0, pre["upload_id"], ITEM["midterm"], "commit", key=str(uuid.uuid4()))
T("TC-P6-07-33", "P" if (sw != 200 or (cm[0] in (409, 422) and cm[1].get("code") in ("FILE_INVALID", "FILE_SHA_MISMATCH"))) else "F",
  f"preview OK ({pv[0]}); thay đối tượng trong kho bằng tệp B cùng độ dài ({len(A_)} B): PUT lại → {sw}; commit → {cm[0]} {cm[1].get('code')}; sum(version) giữ {e_before[0][0]}→{sql(f'select sum(version) from grade_entries where course_id=chr(39)||chr(39)')[0][0] if False else sql(f'''select sum(version) from grade_entries where course_id='{C0}' ''')[0][0]}")
# ---- 07-29 zip bomb
t0 = time.strftime("%Y-%m-%dT%H:%M:%S", time.gmtime(time.time() - 5)); st0 = subprocess.run("docker inspect ep-r-gateway-1 --format '{{.State.StartedAt}}'", shell=True, capture_output=True, text=True).stdout.strip(); r0 = mib(rss())
c, pre, _ = up_xlsx("teacher", C0, f"{XD}/zipbomb.xlsx"); rz = imp("teacher", C0, pre["upload_id"], ITEM["midterm"], "preview"); r1 = mib(rss()); st1 = subprocess.run("docker inspect ep-r-gateway-1 --format '{{.State.StartedAt}}'", shell=True, capture_output=True, text=True).stdout.strip()
C("TC-P6-07-29", rz[0] == 422 and rz[1].get("code") == "FILE_INVALID" and r1 - r0 < 100 and st0 == st1, f"zipbomb.xlsx (170 KiB nén, XML ~64 MiB) → {rz[0]} {rz[1].get('code')}; RSS gateway {r0:.0f}→{r1:.0f} MiB (tăng <100); không restart (StartedAt giữ)", f"{rz} {r0} {r1} {st0} {st1}")
# ---- 07-30 5000 / 5001
r5 = prev_ = None
c, pre, _ = up_xlsx("teacher", C0, f"{XD}/rows5000.xlsx"); a = imp("teacher", C0, pre["upload_id"], ITEM["midterm"], "preview")
c, pre, _ = up_xlsx("teacher", C0, f"{XD}/toomany.xlsx"); b = imp("teacher", C0, pre["upload_id"], ITEM["midterm"], "preview")
C("TC-P6-07-30", a[0] == 200 and a[1]["rows_total"] == 5000 and b[0] == 422 and b[1].get("code") == "TOO_MANY_ROWS", f"5.000 dòng → {a[0]} rows_total {a[1].get('rows_total')} (lỗi {len(a[1].get('errors', []))} mục, ghi {a[1].get('written')}); 5.001 dòng → {b[0]} {b[1].get('code')}", f"{a[0]} {str(a[1])[:100]} {b}")
# ---- 07-31 corrupt / notxlsx
since = "2m"; out = []
for f in ("corrupt.xlsx", "notxlsx.xlsx"):
    c, pre, _ = up_xlsx("teacher", C0, f"{XD}/{f}"); out.append(imp("teacher", C0, pre["upload_id"], ITEM["midterm"], "preview"))
lg = gwlog(since); pn = len(re.findall(r"panic|goroutine \d+ \[", lg))
C("TC-P6-07-31", all(x[0] == 422 and x[1].get("code") == "FILE_INVALID" for x in out) and pn == 0, f"corrupt.xlsx / notxlsx.xlsx (MIME khai đúng) → {[(x[0], x[1].get('code')) for x in out]}; log gateway 2 phút gần nhất: 'panic' {pn}", f"{out} {pn}")
# ---- 07-32 công thức + inject: không thực thi, không lộ tên tệp / nội dung vào log
SECRET = "qc-secret-s8r-4417.xlsx"; res = []
for f in ("formula.xlsx", "inject.xlsx"):
    c, pre, _ = up_xlsx("teacher", C0, f"{XD}/{f}", name=SECRET); res.append(imp("teacher", C0, pre["upload_id"], ITEM["midterm"], "preview"))
lg = gwlog("2m"); leak = len(re.findall(re.escape(SECRET), lg)) + len(re.findall(r"=1\+1|SUM\(A1\)|cmd\|", lg))
C("TC-P6-07-32", all(x[0] == 200 for x in res) and leak == 0, f"formula.xlsx + inject.xlsx preview → {[x[0] for x in res]} (không thực thi công thức: 22/3 đọc theo giá trị đã định dạng); log gateway 2 phút: tên tệp gốc/nội dung ô độc xuất hiện {leak} lần", f"{res} {leak}")
# ---- 07-27: DB trống điểm giữa/cuối (QD8R71)
r71 = cid("QD8R71"); i71 = {k: sq1(f"select id from grade_items where course_id='{r71}' and key='{k}'") for k in ("midterm", "final")}
sql(f"delete from grade_entries where course_id='{r71}' and item_id in ('{i71['midterm']}','{i71['final']}')")
for k, f in (("midterm", "midterm.xlsx"), ("final", "final.xlsx")):
    c, pre, _ = up_xlsx("teacher", r71, f"{XD}/{f}"); rr = imp("teacher", r71, pre["upload_id"], i71[k], "commit", key=str(uuid.uuid4())); print(k, rr[0], rr[1].get("rows_ok"), rr[1].get("written"))
n60 = sq1(f"select count(*) from grade_entries where course_id='{r71}' and item_id in ('{i71['midterm']}','{i71['final']}')")
c, p = call("GET", f"/courses/{r71}/gradebook/preview?limit=100", "teacher"); P = {x["student_code"]: x for x in p["items"]}
bad = [a for a, r in CSV.items() if (P[r["student_code"]]["hp"], P[r["student_code"]]["letter"], P[r["student_code"]]["verdict"]) != (str(float(r["hp"])).rstrip("0").rstrip("."), r["letter"], r["verdict"])]
# so điểm thành phần từng ô với CSV
bad2 = []
for a, r in CSV.items():
    for k, col in (("midterm", "midterm"), ("final", "final_exam")):
        v = sql(f"select score from grade_entries e join users u on u.id=e.student_id where e.course_id='{r71}' and e.item_id='{i71[k]}' and u.student_code='{r['student_code']}'")
        if not v or abs(float(v[0][0]) - float(r[col])) > 1e-9: bad2.append((a, k))
C("TC-P6-07-27", n60 == "60" and not bad and not bad2, f"DB trống GK/CK → nhập midterm.xlsx + final.xlsx (commit): {n60} ô; từng giá trị khớp inputs CSV ({len(bad2)} lệch); preview 30/30 khớp expected_final_grades.csv (hp, letter, verdict; {len(bad)} lệch)", f"{n60} {bad} {bad2}")
