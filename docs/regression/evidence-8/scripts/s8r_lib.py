# s8r: thư viện QC lượt chạy lại sprint 8 (P6-03..09, SEED, UI) trên ep-r. Ghi kết quả vào part-s8r.tsv (bản ghi sau thắng).
import sys, os
sys.path.insert(0, os.path.dirname(__file__))
from s8core_lib import *
import s8core_lib as _c, urllib.error, time as _t
_oraw, _osql = _c.raw, _c.sql
def _raw(*a, **k):  # gateway/DB bị khoá global của nhóm khác (chaos) → chờ rồi thử lại, không ghi F vì môi trường
    for _ in range(45):
        try: return _oraw(*a, **k)
        except (urllib.error.URLError, ConnectionError, TimeoutError, OSError): _t.sleep(10)
    return _oraw(*a, **k)
def _sql(q, db="edupilot"):
    for _ in range(45):
        try: return _osql(q, db)
        except RuntimeError as e:
            if "ERROR" in str(e) and "connection" not in str(e).lower() and "starting up" not in str(e).lower() and "shutting" not in str(e).lower(): raise
            _t.sleep(10)
    return _osql(q, db)
_c.raw = _raw; _c.sql = _sql; raw = _raw; sql = _sql
def sq1(q, db="edupilot"):
    r = sql(q, db); return r[0][0] if r else None
_c.sq1 = sq1
PART = f"{REPO}/docs/regression/evidence-8/part-s8r.tsv"
def T(tc, st, note):
    note = " ".join(str(note).split())[:165]
    with open(PART, "a") as f: f.write(f"{tc}\t{st}\t{note}\n")
    print(st, tc, note, flush=True)
def C(tc, cond, ok, bad=None, st="F"):
    T(tc, "P" if cond else st, ok if cond else (bad or ok)); return cond
import zipfile, io, re, html
def xlsx_rows(b):
    z = zipfile.ZipFile(io.BytesIO(b)); names = z.namelist()
    ss = []
    if "xl/sharedStrings.xml" in names:
        x = z.read("xl/sharedStrings.xml").decode()
        ss = [html.unescape(re.sub(r"<[^>]+>", "", m)) for m in re.findall(r"<si>(.*?)</si>", x, re.S)]
    sh = [n for n in names if n.startswith("xl/worksheets/sheet")][0]; x = z.read(sh).decode(); out = []; formulas = 0
    for r in re.findall(r"<row[^>]*>(.*?)</row>", x, re.S):
        row = []
        for m in re.finditer(r"<c ([^>]*?)(?:/>|>(.*?)</c>)", r, re.S):
            at, body = m.group(1), m.group(2) or ""
            if "<f" in body: formulas += 1
            t = re.search(r't="(\w+)"', at); t = t.group(1) if t else "n"
            if t == "s": v = ss[int(re.search(r"<v>(.*?)</v>", body).group(1))]
            elif t == "inlineStr": v = html.unescape(re.sub(r"<[^>]+>", "", body))
            else:
                mm = re.search(r"<v>(.*?)</v>", body); v = mm.group(1) if mm else ""
            row.append(v)
        out.append(row)
    return out, formulas, x
def getx(path, user, hdr=None):
    return raw("GET", path, {"authorization": "Bearer " + tok(user), **(hdr or {})})
import hashlib as _h
XMIME = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
def up_xlsx(user, course, path, name=None, mime=XMIME, sha=None, size=None, data=None):
    b = data if data is not None else open(path, "rb").read(); sha = sha or _h.sha256(b).hexdigest()
    for _ in range(9):  # presign ≤ 10/phút/người (TC-07-04): chờ khi 429, không coi là lỗi
        c, pre = call("POST", f"/courses/{course}/uploads/presign", user, {"purpose": "grade_import", "filename": name or os.path.basename(path), "mime_type": mime, "size_bytes": size if size is not None else len(b), "sha256": sha})
        if c != 429: break
        _t.sleep(10)
    if c != 200: return c, pre, None
    req = urllib.request.Request(pre["url"], data=b, method="PUT", headers=pre.get("headers", {}))
    try: urllib.request.urlopen(req, context=CTX, timeout=120)
    except urllib.error.HTTPError as e: return -e.code, pre, e.read()
    return 200, pre, sha
def imp(user, course, upload_id, item, mode, mapping=None, key=None, **extra):
    body = {"upload_id": upload_id, "item_id": item, "mode": mode, "mapping": mapping or {"student_code_col": "A", "score_col": "C", "header_row": 1}, **extra}
    return call("POST", f"/courses/{course}/grade-entries/import", user, body, key=key)
from xml.sax.saxutils import escape as _esc
def mkxlsx(rows):
    """xlsx tối thiểu (zipfile): ô str → inlineStr, số → số, None → bỏ qua."""
    out = ['<?xml version="1.0" encoding="UTF-8"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>']
    for i, r in enumerate(rows, 1):
        out.append(f'<row r="{i}">')
        for j, v in enumerate(r):
            if v is None: continue
            ref = "ABCDEFGH"[j] + str(i)
            out.append(f'<c r="{ref}"><v>{v}</v></c>' if isinstance(v, (int, float)) else f'<c r="{ref}" t="inlineStr"><is><t>{_esc(str(v))}</t></is></c>')
        out.append("</row>")
    out.append("</sheetData></worksheet>")
    bio = io.BytesIO()
    with zipfile.ZipFile(bio, "w", zipfile.ZIP_DEFLATED) as z:
        z.writestr("[Content_Types].xml", '<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/></Types>')
        z.writestr("_rels/.rels", '<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>')
        z.writestr("xl/workbook.xml", '<?xml version="1.0" encoding="UTF-8"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Sheet1" sheetId="1" r:id="rId1"/></sheets></workbook>')
        z.writestr("xl/_rels/workbook.xml.rels", '<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>')
        z.writestr("xl/worksheets/sheet1.xml", "".join(out))
    return bio.getvalue()
def mkstudent(course, code, name, via="ROSTER", snapshot="same", email=None):
    """Tạo sinh viên MỚI (không đụng user seed) + ghi danh vào lớp riêng; snapshot='same'→=code, None→NULL."""
    u = sq1("select uuidv7()"); em = email or "s8r." + u[-12:] + "@s8r.local"
    sql(f"insert into users (id,email,full_name,role,student_code,email_verified_at) values ('{u}','{em}','{name}','STUDENT','{code}',now())")
    snap_ = code if snapshot == "same" else snapshot; sn = "null" if snap_ is None else f"'{snap_}'"
    sql(f"insert into enrollments (course_id,user_id,role_in_course,status,joined_via,student_code_snapshot) values ('{course}','{u}','STUDENT','ACTIVE','{via}',{sn})")
    return u
