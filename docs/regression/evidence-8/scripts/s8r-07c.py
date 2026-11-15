# s8r 07c: QD8R7 — MSSV #20 (07-14..16, 55..58), xuất inject (07-38)
from s8r_g import *
XD = "/tmp/qc2-s8r-xlsx"; MID = ITEM["midterm"]; H = ["MSSV", "Họ tên", "Điểm"]
def run(rows, mode="preview", skip=False):
    c, pre, _ = up_xlsx("teacher", C0, None, name="t.xlsx", data=mkxlsx([H] + rows)); assert c == 200, (c, pre)
    return imp("teacher", C0, pre["upload_id"], MID, mode, key=str(uuid.uuid4()) if mode == "commit" else None, skip_errors=skip)
def sc(u): r = sql(f"select score from grade_entries where course_id='{C0}' and item_id='{MID}' and student_id='{u}'"); return r[0][0] if r else None
# 07-14: X khai MSSV của Y ở hồ sơ tài khoản
Y = mkstudent(C0, "20229555", "QC Bạn Y", "ROSTER"); X = mkstudent(C0, "20229555", "QC Bạn X", "ROSTER", snapshot="XSELF0001")
c, r = run([["20229555", "Y", 9]]); c2, r2 = run([["20229555", "Y", 9]], "commit")
C("TC-P6-07-14", c == 200 and r["errors"] == [] and c2 == 200 and sc(Y) in ("9", "9.00") and sc(X) is None, f"X.users.student_code = MSSV của Y nhưng snapshot X = XSELF0001; dòng '20229555, 9' → điểm vào Y (snapshot khớp, ROSTER) = {sc(Y)}, X = {sc(X)} (không nhận)", f"{c} {r} {c2} {r2} {sc(Y)} {sc(X)}")
# 07-15 trùng snapshot
mkstudent(C0, "20229666", "QC Trùng 1", "ROSTER"); mkstudent(C0, "20229666", "QC Trùng 2", "ROSTER")
c, r = run([["20229666", "T", 8]]); codes = [e["code"] for e in r.get("errors", [])]
C("TC-P6-07-15", codes == ["AMBIGUOUS_STUDENT_CODE"], f"2 SV ROSTER cùng snapshot 20229666 → preview {codes}", f"{c} {r}")
# 07-16 số 0 đầu
Z0 = mkstudent(C0, "00123456", "QC Số không", "ROSTER")
c, r = run([["00123456", "a", 7], ["123456", "b", 6]]); er = {e["row"]: e["code"] for e in r.get("errors", [])}
c2, r2 = run([["00123456", "a", 7]], "commit")
C("TC-P6-07-16", er == {3: "STUDENT_NOT_IN_COURSE"} and r["rows_ok"] == 1 and sc(Z0) in ("7", "7.00"), f"'00123456' (chuỗi) khớp đúng người (điểm {sc(Z0)}); '123456' → {er} (không mất số 0)", f"{c} {r} {c2} {sc(Z0)}")
# 07-55 CODE tự khai
Z = mkstudent(C0, "20229777", "QC Tự khai", "CODE")
c, r = run([["20229777", "Z", 9], [CSV["sv.kha"]["student_code"], "B", 6]]); er = {e["row"]: e["code"] for e in r.get("errors", [])}
cn, rn = run([["20229777", "Z", 9], [CSV["sv.kha"]["student_code"], "B", 6]], "commit", False); z0 = sq1(f"select count(*) from grade_entries where student_id='{Z}'")
cs, rs = run([["20229777", "Z", 9], [CSV["sv.kha"]["student_code"], "B", 6]], "commit", True); z1 = sq1(f"select count(*) from grade_entries where student_id='{Z}'")
C("TC-P6-07-55", er == {2: "STUDENT_CODE_UNVERIFIED"} and cn == 409 and rn.get("code") == "IMPORT_HAS_ERRORS" and cs == 200 and z0 == z1 == "0" and sc(SV("sv.kha")) in ("6", "6.00"), f"Z (CODE, tự khai 20229777): preview {er}; commit không skip → {cn} {rn.get('code')}; skip_errors → {cs}, ghi B=6, entries của Z {z0}→{z1}", f"{c} {r} {cn} {rn} {cs} {rs}")
setc("sv.kha", "midterm", CSV["sv.kha"]["midterm"])
# 07-56 CODE trùng với ROSTER
R = mkstudent(C0, "20229888", "QC Roster", "ROSTER"); Z2 = mkstudent(C0, "20229888", "QC Code trùng", "CODE")
c, r = run([["20229888", "R", 8]]); c2, r2 = run([["20229888", "R", 8]], "commit")
C("TC-P6-07-56", r["errors"] == [] and c2 == 200 and sc(R) in ("8", "8.00") and sc(Z2) is None, f"20229888 có ROSTER (R) + CODE (Z2): preview không lỗi/AMBIGUOUS; điểm vào R={sc(R)}, Z2={sc(Z2)}", f"{c} {r} {c2} {sc(R)} {sc(Z2)}")
# 07-57 ADMIN
A_ = mkstudent(C0, "20229999", "QC Admin thêm", "ADMIN"); c, r = run([["20229999", "A", 7]]); c2, r2 = run([["20229999", "A", 7]], "commit")
C("TC-P6-07-57", r["errors"] == [] and c2 == 200 and sc(A_) in ("7", "7.00"), f"20229999 joined_via=ADMIN → khớp, điểm {sc(A_)}", f"{c} {r} {c2} {sc(A_)}")
# 07-58 nhập tay cho CODE
c, r = call("PUT", f"{Q}/grade-entries", "teacher", {"item_id": MID, "student_id": Z, "score": "6.5"}); cg, g = call("GET", f"{Q}/gradebook?limit=100", "teacher")
C("TC-P6-07-58", c == 200 and any(x["student"]["id"] == Z for x in g["rows"]), f"PUT tay cho Z (CODE) → {c}, điểm {sc(Z)}; sổ vẫn liệt kê Z ({len(g['rows'])} dòng)", f"{c} {r}")
# 07-38 inject: tên bắt đầu = + - @
for i, nm in enumerate(["=1+1", "+SUM(A1)", "-2+3", "@cmd|' /C calc'!A0"]): mkstudent(C0, f"2022INJ{i}", nm, "ROSTER")
st, b, hd = getx(f"{Q}/gradebook/export.xlsx", "teacher"); rx, nf, xml = xlsx_rows(b)
inj = [r for r in rx if r and r[1] and r[1][0] in "=+-@"]; types = re.findall(r'<c [^>]*t="(\w+)"[^>]*>(?:(?!</c>).)*?(?:=1\+1|SUM\(A1\))', xml)
C("TC-P6-07-38", st == 200 and len(inj) == 4 and nf == 0, f"4 sinh viên tên bắt đầu = + - @ → export có {len(inj)} ô tên đúng chữ gốc, <f> trong sheet = {nf} (không công thức); ô kiểu {set(types) or 'inlineStr/s'}", f"{st} {len(inj)} {nf}")
