#!/usr/bin/env python3
"""QC P6 US-P6-07: sinh tệp XLSX thử nhập (không cần openpyxl: ghi OOXML tối thiểu bằng zipfile).
Chạy từ gốc repo:  python3 docs/sprints/8/qc/scripts/p6-make-xlsx.py [thư-mục-ra=/tmp/qc-p6-xlsx]
Đầu vào: seed/expected_final_grades.csv (MSSV + điểm — nguồn BA). Tệp ra (mọi thứ do QC sinh, không động vào seed):
  midterm.xlsx / final.xlsx   30 dòng khoá MSSV (chuỗi), cột "MSSV","Điểm" → TC nhập đúng, nhập 2 lần
  errors.xlsx                 dòng lỗi cố ý (xem ERRORS bên dưới) → TC báo dòng lỗi đúng mã, đúng dòng
  formula.xlsx                ô công thức: =22/3 định dạng 0.00 (đọc "7.33" → nhận) và =22/3 định dạng chung (→ SCORE_PRECISION)
  real-mssv.xlsx              MSSV dạng số thực 20229002.0 (ô số) → vẫn khớp B
  toomany.xlsx                5.001 dòng dữ liệu → 422 TOO_MANY_ROWS;  rows5000.xlsx: đúng 5.000 dòng → qua giới hạn dòng
  zipbomb.xlsx                sheet1.xml giải nén ~64 MiB, nén ~175 KiB → bị chặn (422 FILE_INVALID), gateway không phình RAM
  corrupt.xlsx / notxlsx.xlsx byte rác / PDF đổi đuôi → 422 FILE_INVALID, không panic
  big3mb.xlsx                 > 2 MiB (GRADE_IMPORT_MAX_BYTES) → 422 FILE_TOO_LARGE
  inject.xlsx                 họ tên bắt đầu '=' , '+', '-', '@' (dùng cho test xuất — nhập thử xem không thành công thức)
"""
import csv, os, random, sys, zipfile
from xml.sax.saxutils import escape

OUT = sys.argv[1] if len(sys.argv) > 1 else "/tmp/qc-p6-xlsx"
os.makedirs(OUT, exist_ok=True)
rows = list(csv.DictReader(open("seed/expected_final_grades.csv")))

CT = '<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/><Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/></Types>'
RELS = '<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>'
WB = '<?xml version="1.0" encoding="UTF-8"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Sheet1" sheetId="1" r:id="rId1"/></sheets></workbook>'
WBR = '<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/></Relationships>'
# style 0 = General, style 1 = 0.00 (numFmtId 2)
STY = '<?xml version="1.0" encoding="UTF-8"?><styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><fonts count="1"><font><sz val="11"/><name val="Calibri"/></font></fonts><fills count="2"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill></fills><borders count="1"><border><left/><right/><top/><bottom/><diagonal/></border></borders><cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs><cellXfs count="2"><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/><xf numFmtId="2" fontId="0" fillId="0" borderId="0" xfId="0" applyNumberFormat="1"/></cellXfs></styleSheet>'


def cell(ref, v, kind="s", style=0, formula=None):
    if v is None:
        return ""
    if kind == "n":
        return f'<c r="{ref}" s="{style}"><v>{v}</v></c>'
    if kind == "f":
        return f'<c r="{ref}" s="{style}"><f>{escape(formula)}</f><v>{v}</v></c>'
    return f'<c r="{ref}" t="inlineStr"><is><t>{escape(str(v))}</t></is></c>'


def sheet(table):  # table: list of list of (v, kind, style, formula) | None | plain str
    out = ['<?xml version="1.0" encoding="UTF-8"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>']
    for i, r in enumerate(table, 1):
        out.append(f'<row r="{i}">')
        for j, c in enumerate(r):
            ref = "ABCDEFGH"[j] + str(i)
            if c is None:
                continue
            if isinstance(c, tuple):
                if c[1] == "f":
                    out.append(cell(ref, c[0], "f", c[2], c[3]))
                else:
                    out.append(cell(ref, c[0], c[1]))
            else:
                out.append(cell(ref, c))
        out.append("</row>")
    out.append("</sheetData></worksheet>")
    return "".join(out)


def write(name, table=None, raw_sheet=None, compress=zipfile.ZIP_DEFLATED):
    with zipfile.ZipFile(os.path.join(OUT, name), "w", compress) as z:
        z.writestr("[Content_Types].xml", CT); z.writestr("_rels/.rels", RELS); z.writestr("xl/workbook.xml", WB)
        z.writestr("xl/_rels/workbook.xml.rels", WBR); z.writestr("xl/styles.xml", STY)
        z.writestr("xl/worksheets/sheet1.xml", raw_sheet if raw_sheet is not None else sheet(table))


H = ["MSSV", "Họ tên", "Điểm"]
def good(col):
    return [H] + [[r["student_code"], r["account"], (r[col], "n")] for r in rows if r[col]]

write("midterm.xlsx", good("midterm"))
write("final.xlsx", good("final_exam"))

B, K = rows[1], rows[2]  # B = sv.kha 20229002, C = sv.nguyco 20229003
ERRORS = [H,
    ["99999999", "ngoài lớp", ("8", "n")],               # dòng 2  STUDENT_NOT_IN_COURSE
    [B["student_code"], "B lần 1", ("7", "n")],          # dòng 3  hợp lệ
    [B["student_code"], "B lần 2", ("7.5", "n")],        # dòng 4  DUPLICATE_IN_FILE
    [K["student_code"], "C", ("11", "n")],               # dòng 5  SCORE_OUT_OF_RANGE (max 10)
    [rows[3]["student_code"], "sv04", ("8.333", "s")],   # dòng 6  SCORE_PRECISION (chuỗi 8.333)
    [rows[4]["student_code"], "sv05", "abc"],            # dòng 7  SCORE_NOT_NUMBER
    [None, None, None],                                  # dòng 8  dòng trống → bỏ qua, KHÔNG lỗi
    [rows[5]["student_code"], "sv06", "8,5"],            # dòng 9  hợp lệ (dấu phẩy)
    [rows[6]["student_code"], "sv07", ("-1", "n")],      # dòng 10 SCORE_OUT_OF_RANGE (âm)
    [rows[7]["student_code"], "sv08", None],             # dòng 11 điểm trống: không xoá điểm cũ, không lỗi
]
write("errors.xlsx", ERRORS)
# Ghi chú kỳ vọng (preview): rows_total=9 (không tính dòng trống), errors = {2:NOT_IN_COURSE, 4:DUPLICATE_IN_FILE, 5:OUT_OF_RANGE, 6:PRECISION, 7:NOT_NUMBER, 10:OUT_OF_RANGE}

write("formula.xlsx", [H,
    [B["student_code"], "B =22/3 định dạng 0.00 → đọc '7.33' → nhận", ("7.3333333333", "f", 1, "22/3")],
    [K["student_code"], "C =22/3 định dạng chung → 7.333… → SCORE_PRECISION", ("7.3333333333", "f", 0, "22/3")],
])
write("real-mssv.xlsx", [H, [(B["student_code"] + ".0", "n"), "B mssv số thực", ("6", "n")]])

def many(n):
    random.seed(1)
    return [H] + [[f"{20300000 + i}", f"x{i}", ("5", "n")] for i in range(n)]
write("rows5000.xlsx", many(5000)); write("toomany.xlsx", many(5001))

# zip bomb: 64 MiB XML (lặp 1 ô), nén ≪ 100 KiB
head = '<?xml version="1.0" encoding="UTF-8"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row r="1">'
write("zipbomb.xlsx", raw_sheet=head + ('<c r="A1" t="inlineStr"><is><t>' + "A" * 1000 + "</t></is></c>") * (64 * 1024) + "</row></sheetData></worksheet>")

open(os.path.join(OUT, "corrupt.xlsx"), "wb").write(os.urandom(4096))
open(os.path.join(OUT, "notxlsx.xlsx"), "wb").write(b"%PDF-1.4\n1 0 obj<<>>endobj\n%%EOF\n")
rnd = random.Random(7)
write("big3mb.xlsx", [H] + [[f"{rnd.randrange(10**8)}", "".join(rnd.choice("abcdefghijklmnopqrstuvwxyz") for _ in range(40)), ("5", "n")] for _ in range(60000)], compress=zipfile.ZIP_STORED)
write("inject.xlsx", [H, ["=1+1", "=cmd|' /C calc'!A0", ("5", "n")], ["+84", "-2+3", ("5", "n")], ["@SUM(A1)", "tên", ("5", "n")]])

for f in sorted(os.listdir(OUT)):
    print(f"{os.path.getsize(os.path.join(OUT, f)):>10}  {f}")
