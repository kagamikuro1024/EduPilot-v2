#!/usr/bin/env python3
"""QC: sinh tệp roster thử cho tc-US-P2-10 (chạy: python3 gen.py <thư mục ra>). zip bomb sinh lúc chạy, không commit."""
import sys, os, zipfile, io, random
out = sys.argv[1] if len(sys.argv) > 1 else '.'
os.makedirs(out, exist_ok=True)
def w(name, text, mode='w', enc='utf-8'):
    with open(os.path.join(out, name), mode, **({} if 'b' in mode else {'encoding': enc, 'newline': ''})) as f: f.write(text)
def rows(n, prefix='qc-r', start=1, bad=()):
    r = ['Email,Họ và tên,MSSV']
    for i in range(start, start + n):
        line = i - start + 2
        e = f'{prefix}{i}@example.test'; name = f'Sinh Viên {i}'; m = f'2022{i:05d}'
        if line == 7 and 7 in bad: e = 'khong-phai-email'
        if line == 19 and 19 in bad: name = ''
        r.append(f'{e},{name},{m}')
    return '\r\n'.join(r) + '\r\n'
w('ok-30.csv', rows(30, 'qc-ok'))
w('loi-2.csv', rows(30, 'qc-le', start=200, bad=(7, 19)))
w('bom-semicolon.csv', '\ufeff' + rows(5, 'qc-bs', start=100).replace(',', ';'))
w('501-dong.csv', rows(501, 'qc-501', start=1000))
w('fake-ext.csv', b'\x7fELF\x02\x01\x01\x00' + os.urandom(2000), 'wb')
w('png-as.csv', b'\x89PNG\r\n\x1a\n' + os.urandom(500), 'wb')
w('formula.csv', 'Email,Họ và tên,MSSV\r\nqc-f1@example.test,"=HYPERLINK(""http://evil"",""x"")",20220101\r\nqc-f2@example.test,+cmd|\' /C calc\'!A0,20220102\r\nqc-f3@example.test,@SUM(1),20220103\r\n')
w('dong-dai.csv', 'Email,Họ và tên,MSSV\r\nqc-long@example.test,' + 'A' * 12000 + ',20220201\r\n')
w('ten-dai.csv', 'Email,Họ và tên,MSSV\r\nqc-ten@example.test,' + ('Nguyễn ' * 30) + ',20220301\r\n')
w('control.csv', 'Email,Họ và tên,MSSV\r\nqc-ctl@example.test,Ten\x00Loi,20220401\r\n')
w('thieu-cot.csv', 'Họ và tên,MSSV\r\nA,20220501\r\n')
w('trung-mssv.csv', 'Email,Họ và tên,MSSV\r\nqc-t1@example.test,Một,20220601\r\nqc-t2@example.test,Hai,20220601\r\nqc-t1@example.test,Một lặp,20220602\r\n')
w('injection.csv', 'Email,Họ và tên,MSSV\r\nqc-i1@example.test,<script>alert(1)</script>,20220701\r\nqc-i2@example.test,\'; drop table users;--,20220702\r\n')
def xlsx(path, data, sheet_extra=b''):
    ss = []; rowsx = []
    for ri, row in enumerate(data, 1):
        cells = []
        for ci, v in enumerate(row):
            col = 'ABC'[ci]
            esc = str(v).replace('&', '&amp;').replace('<', '&lt;')
            cells.append(f'<c r="{col}{ri}" t="inlineStr"><is><t>{esc}</t></is></c>')
        rowsx.append(f'<row r="{ri}">{"".join(cells)}</row>')
    sheet = '<?xml version="1.0" encoding="UTF-8"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>' + ''.join(rowsx) + '</sheetData></worksheet>'
    with zipfile.ZipFile(path, 'w', zipfile.ZIP_DEFLATED) as z:
        z.writestr('[Content_Types].xml', '<?xml version="1.0"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/></Types>')
        z.writestr('_rels/.rels', '<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>')
        z.writestr('xl/workbook.xml', '<?xml version="1.0"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Sheet1" sheetId="1" r:id="rId1"/></sheets></workbook>')
        z.writestr('xl/_rels/workbook.xml.rels', '<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>')
        z.writestr('xl/worksheets/sheet1.xml', sheet)
xlsx(os.path.join(out, 'ok.xlsx'), [['Email', 'Họ và tên', 'MSSV']] + [[f'qc-x{i}@example.test', f'Xlsx {i}', f'2022{i + 800:05d}'] for i in range(1, 6)])
# zip bomb: sheet 25 MiB nén nhỏ
big = '<?xml version="1.0"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>' + ('<row r="1"/>' * 2000000) + '</sheetData></worksheet>'
xlsx(os.path.join(out, 'zipbomb.xlsx'), [['Email', 'Họ và tên', 'MSSV']])
with zipfile.ZipFile(os.path.join(out, 'zipbomb.xlsx'), 'a', zipfile.ZIP_DEFLATED) as z:
    z.writestr('xl/worksheets/sheet2.xml', big)
# xlsx hỏng
data = open(os.path.join(out, 'ok.xlsx'), 'rb').read()
open(os.path.join(out, 'cut.xlsx'), 'wb').write(data[:len(data) // 2])
open(os.path.join(out, 'text.xlsx'), 'w').write('Email,Họ và tên,MSSV\nqc-tx@example.test,A,20220901\n')
print('ok', os.listdir(out))
