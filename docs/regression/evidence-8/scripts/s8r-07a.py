# s8r 07a: US-P6-07 nhập/xuất XLSX trên lớp riêng QD8R7 (OPEN, bản sao sạch lớp 1)
from s8r_g import *
XD = "/tmp/qc2-s8r-xlsx"; MID = ITEM["midterm"]; WK = ITEM["weekly"]; FIN = ITEM["final"]
N_E = lambda: int(sq1(f"select count(*) from grade_entries where course_id='{C0}'")); N_A = lambda: int(sq1(f"select count(*) from audit_log where course_id='{C0}'"))
def prev(path, item=MID, who="teacher", **kw):
    c, pre, sha = up_xlsx(who, C0, path, **kw); assert c == 200, (c, pre)
    return imp(who, C0, pre["upload_id"], item, "preview"), pre
# ---- 07-01 presign + PUT, không có tệp ở đĩa gateway
d0 = subprocess.run("docker diff ep-r-gateway-1 | grep -i xlsx | wc -l", shell=True, capture_output=True, text=True).stdout.strip()
c, pre, sha = up_xlsx("teacher", C0, f"{XD}/midterm.xlsx")
d1 = subprocess.run("docker diff ep-r-gateway-1 | grep -i xlsx | wc -l", shell=True, capture_output=True, text=True).stdout.strip()
C("TC-P6-07-01", c == 200 and pre.get("upload_id") and pre.get("url", "").startswith("http") and d1 == d0, f"presign → 200 upload_id + URL ký (kho MinIO :9700), PUT thành công; `docker diff` gateway: tệp .xlsx mới {d0}→{d1} (không ghi đĩa cục bộ)", f"{c} {pre} {d0} {d1}")
# ---- 07-02 loại tệp
r = []
for nm, mime in (("midterm.pdf", XMIME), ("midterm.csv", XMIME), ("midterm.xlsx", "application/pdf")):
    r.append(call("POST", f"/courses/{C0}/uploads/presign", "teacher", {"purpose": "grade_import", "filename": nm, "mime_type": mime, "size_bytes": 2000, "sha256": "a" * 64}))
C("TC-P6-07-02", all(c == 422 and x.get("code") == "FILE_TYPE_NOT_ALLOWED" for c, x in r), f"đuôi .pdf / .csv / .xlsx+mime pdf → {[(c, x.get('code')) for c, x in r]}", str(r))
# ---- 07-03 quá lớn
big = f"{XD}/big3mb.xlsx"; bsz = os.path.getsize(big); sha = hashlib.sha256(open(big, "rb").read()).hexdigest()
c, x = call("POST", f"/courses/{C0}/uploads/presign", "teacher", {"purpose": "grade_import", "filename": "big3mb.xlsx", "mime_type": XMIME, "size_bytes": bsz, "sha256": sha})
# khai nhỏ nhưng PUT lớn
c2, pre2 = call("POST", f"/courses/{C0}/uploads/presign", "teacher", {"purpose": "grade_import", "filename": "big3mb.xlsx", "mime_type": XMIME, "size_bytes": 2000, "sha256": sha})
put_st = None; res2 = None
if c2 == 200:
    try: urllib.request.urlopen(urllib.request.Request(pre2["url"], data=open(big, "rb").read(), method="PUT", headers=pre2.get("headers", {})), context=CTX, timeout=120); put_st = 200
    except urllib.error.HTTPError as e: put_st = e.code
    except (urllib.error.URLError, OSError) as e: put_st = 0  # kho đóng kết nối ngay (chữ ký khớp size 2000)
    if put_st == 200: res2 = imp("teacher", C0, pre2["upload_id"], MID, "preview")
C("TC-P6-07-03", c == 422 and x.get("code") == "FILE_TOO_LARGE" and (put_st in (0, 400, 403) or (res2 and res2[0] == 422 and res2[1].get("code") in ("FILE_TOO_LARGE", "FILE_INVALID"))), f"{bsz//1048576} MiB khai thật → {c} {x.get('code')}; khai 2000 B nhưng PUT {bsz//1048576} MiB → kho MinIO từ chối ({put_st or 'ngắt kết nối'})" + (f", preview {res2[0]} {res2[1].get('code')}" if res2 else ""), f"{c} {x} {c2} {put_st} {res2}")
# ---- 07-05 sha sai
b = open(f"{XD}/midterm.xlsx", "rb").read()
c, pre, _ = up_xlsx("teacher", C0, None, name="midterm.xlsx", sha="b" * 64, data=b)
r1 = imp("teacher", C0, pre["upload_id"], MID, "preview")
C("TC-P6-07-05", r1[0] == 422 and r1[1].get("code") == "FILE_INVALID" and N_E() == 60, f"presign khai sha256 sai + PUT tệp thật → preview {r1[0]} {r1[1].get('code')}; không ghi (grade_entries 60)", str(r1))
# ---- 07-06 phân quyền 3 vai × 3 lệnh
res = []
for u in ("ta", "sv.gioi", "admin"):
    res.append(call("POST", f"/courses/{C0}/uploads/presign", u, {"purpose": "grade_import", "filename": "midterm.xlsx", "mime_type": XMIME, "size_bytes": len(b), "sha256": hashlib.sha256(b).hexdigest()})[0])
    res.append(imp(u, C0, str(uuid.uuid4()), MID, "preview")[0]); res.append(imp(u, C0, str(uuid.uuid4()), MID, "commit")[0])
C("TC-P6-07-06", res == [403] * 9, f"TA/STUDENT/ADMIN × (presign, preview, commit) → {res}", str(res))
# ---- 07-08 preview midterm: không ghi
e0, a0 = N_E(), N_A(); (c, r), pre = prev(f"{XD}/midterm.xlsx")
C("TC-P6-07-08", c == 200 and r["rows_total"] == 30 and r["rows_ok"] == 30 and r["errors"] == [] and len(r["sample"]) <= 20 and (N_E(), N_A()) == (e0, a0), f"preview midterm.xlsx → rows_total {r['rows_total']}, rows_ok {r['rows_ok']}, overwrites {r['overwrites']}, unchanged {r['unchanged']}, errors [], sample {len(r['sample'])} (≤20); grade_entries {e0}→{N_E()}, audit_log {a0}→{N_A()} không đổi", f"{c} {r}")
# ---- 07-11 errors.xlsx
(c, r), pre_err = prev(f"{XD}/errors.xlsx"); errs = {e["row"]: e["code"] for e in r["errors"]}
exp = {2: "STUDENT_NOT_IN_COURSE", 4: "DUPLICATE_IN_FILE", 5: "SCORE_OUT_OF_RANGE", 6: "SCORE_PRECISION", 7: "SCORE_NOT_NUMBER", 10: "SCORE_OUT_OF_RANGE"}
C("TC-P6-07-11", errs == exp and r["rows_ok"] in (2, 3) and 8 not in errs and 11 not in errs, f"errors.xlsx → {len(r['errors'])} lỗi {errs}; rows_total {r['rows_total']}, rows_ok {r['rows_ok']}; dòng 8 trống & dòng 11 (điểm trống) không lỗi", f"{errs} vs {exp} {r}")
# ---- 07-12 formula, 07-13 real-mssv
(c, r), _ = prev(f"{XD}/formula.xlsx"); e = {x["row"]: x["code"] for x in r["errors"]}
C("TC-P6-07-12", c == 200 and e == {3: "SCORE_PRECISION"} and r["rows_ok"] == 1, f"formula.xlsx: dòng B (=22/3 định dạng 0.00 → '7.33') hợp lệ; dòng C (=22/3 chung) → {e}; rows_ok {r['rows_ok']}", f"{c} {r}")
(c, r), _ = prev(f"{XD}/real-mssv.xlsx")
C("TC-P6-07-13", c == 200 and r["rows_ok"] == 1 and r["errors"] == [], f"MSSV ô số thực 20229002.0 → rows_ok {r['rows_ok']}, 0 lỗi (khớp B: điểm 6 vs hiện có 7,5 → overwrites {r['overwrites']})", f"{c} {r}")
# ---- 07-50/51/52/53
oc = cid("QD8F8"); co, preo, _ = up_xlsx("gv.s8f", oc, f"{XD}/midterm.xlsx")
r50 = imp("teacher", C0, preo["upload_id"], MID, "preview"); r50b = imp("teacher", C0, str(uuid.uuid4()), MID, "preview")
oi = sq1(f"select id from grade_items where course_id='{oc}' and key='midterm'"); (c, pre, _) = up_xlsx("teacher", C0, f"{XD}/midterm.xlsx"); r51 = imp("teacher", C0, pre["upload_id"], oi, "preview")
C("TC-P6-07-50", r50[0] in (404, 422) and r50b[0] in (404, 422) and N_E() == 60, f"upload_id lớp khác → {r50[0]} {r50[1].get('code')}; upload_id không tồn tại → {r50b[0]}; không ghi", f"{r50} {r50b}")
C("TC-P6-07-51", r51[0] == 404, f"item_id của lớp khác → {r51[0]} {r51[1].get('code')}", str(r51))
rk = call("POST", f"/courses/{C0}/grade-entries/import", "teacher", {"upload_id": pre["upload_id"], "item_id": MID, "mode": "commit", "mapping": {"student_code_col": "A", "score_col": "C", "header_row": 1}}, key=False)
C("TC-P6-07-52", rk[0] in (400, 422) and rk[1].get("code") == "IDEMPOTENCY_KEY_REQUIRED" and N_E() == 60, f"commit thiếu Idempotency-Key → {rk[0]} {rk[1].get('code')}; không ghi", str(rk))
rm = imp("teacher", C0, pre["upload_id"], MID, "preview", mapping={"student_code_col": "A", "score_col": "A", "header_row": 1})
codes = {e["code"] for e in rm[1].get("errors", [])}
C("TC-P6-07-53", rm[0] == 200 and rm[1]["rows_ok"] == 0 and codes and all(x.startswith("SCORE_") or x.startswith("STUDENT_") for x in codes), f"mapping cột điểm = cột MSSV → {rm[0]}; rows_ok {rm[1].get('rows_ok')}; mã lỗi {sorted(codes)}; không 500", str(rm)[:300])
# ---- 07-23 EXAM đích
(c, pre, _) = up_xlsx("teacher", C0, f"{XD}/midterm.xlsx"); rp = imp("teacher", C0, pre["upload_id"], WK, "preview"); rc_ = imp("teacher", C0, pre["upload_id"], WK, "commit")
C("TC-P6-07-23", rp[0] == 409 and rp[1].get("code") == "ITEM_NOT_EDITABLE" and rc_[0] == 409, f"item EXAM làm đích: preview {rp[0]} {rp[1].get('code')}, commit {rc_[0]} {rc_[1].get('code')}", f"{rp} {rc_}")
