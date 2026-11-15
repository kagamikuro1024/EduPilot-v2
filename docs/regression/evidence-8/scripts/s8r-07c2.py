# s8r 07c: QD8R7 — MSSV #20 (07-14..16, 55..58), xuất inject (07-38)
from s8r_g import *
XD = "/tmp/qc2-s8r-xlsx"; MID = ITEM["midterm"]; H = ["MSSV", "Họ tên", "Điểm"]
def run(rows, mode="preview", skip=False):
    c, pre, _ = up_xlsx("teacher", C0, None, name="t.xlsx", data=mkxlsx([H] + rows)); assert c == 200, (c, pre)
    return imp("teacher", C0, pre["upload_id"], MID, mode, key=str(uuid.uuid4()) if mode == "commit" else None, skip_errors=skip)
def sc(u): r = sql(f"select score from grade_entries where course_id='{C0}' and item_id='{MID}' and student_id='{u}'"); return r[0][0] if r else None
# 07-38 inject: tên bắt đầu = + - @

st, b, hd = getx(f"{Q}/gradebook/export.xlsx", "teacher"); rx, nf, xml = xlsx_rows(b)
inj = [r for r in rx if len(r) > 1 and r[1] and r[1][0] in "=+-@"]; types = re.findall(r'<c [^>]*t="(\w+)"[^>]*>(?:(?!</c>).)*?(?:=1\+1|SUM\(A1\))', xml)
C("TC-P6-07-38", st == 200 and len(inj) == 4 and nf == 0, f"4 sinh viên tên bắt đầu = + - @ → export có {len(inj)} ô tên đúng chữ gốc, <f> trong sheet = {nf} (không công thức); ô kiểu {set(types) or 'inlineStr/s'}", f"{st} {len(inj)} {nf}")
