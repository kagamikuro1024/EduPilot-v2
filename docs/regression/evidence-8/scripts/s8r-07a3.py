# s8r 07a: US-P6-07 nhập/xuất XLSX trên lớp riêng QD8R7 (OPEN, bản sao sạch lớp 1)
from s8r_g import *
XD = "/tmp/qc2-s8r-xlsx"; MID = ITEM["midterm"]; WK = ITEM["weekly"]; FIN = ITEM["final"]
N_E = lambda: int(sq1(f"select count(*) from grade_entries where course_id='{C0}'")); N_A = lambda: int(sq1(f"select count(*) from audit_log where course_id='{C0}'"))
def prev(path, item=MID, who="teacher", **kw):
    c, pre, sha = up_xlsx(who, C0, path, **kw); assert c == 200, (c, pre)
    return imp(who, C0, pre["upload_id"], item, "preview"), pre
# ---- 07-03 quá lớn
big = f"{XD}/big3mb.xlsx"; bsz = os.path.getsize(big); sha = hashlib.sha256(open(big, "rb").read()).hexdigest()
time.sleep(65); c, x = call("POST", f"/courses/{C0}/uploads/presign", "teacher", {"purpose": "grade_import", "filename": "big3mb.xlsx", "mime_type": XMIME, "size_bytes": bsz, "sha256": sha})
# khai nhỏ nhưng PUT lớn
c2, pre2 = call("POST", f"/courses/{C0}/uploads/presign", "teacher", {"purpose": "grade_import", "filename": "big3mb.xlsx", "mime_type": XMIME, "size_bytes": 2000, "sha256": sha})
put_st = None; res2 = None
if c2 == 200:
    try: urllib.request.urlopen(urllib.request.Request(pre2["url"], data=open(big, "rb").read(), method="PUT", headers=pre2.get("headers", {})), context=CTX, timeout=120); put_st = 200
    except urllib.error.HTTPError as e: put_st = e.code
    except (urllib.error.URLError, OSError) as e: put_st = 0  # kho đóng kết nối ngay (chữ ký khớp size 2000)
    if put_st == 200: res2 = imp("teacher", C0, pre2["upload_id"], MID, "preview")
C("TC-P6-07-03", c == 422 and x.get("code") == "FILE_TOO_LARGE" and (put_st in (0, 400, 403) or (res2 and res2[0] == 422 and res2[1].get("code") in ("FILE_TOO_LARGE", "FILE_INVALID"))), f"{bsz//1048576} MiB khai thật → {c} {x.get('code')}; khai 2000 B nhưng PUT {bsz//1048576} MiB → kho MinIO từ chối ({put_st or 'ngắt kết nối'})" + (f", preview {res2[0]} {res2[1].get('code')}" if res2 else ""), f"{c} {x} {c2} {put_st} {res2}")
