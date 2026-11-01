#!/usr/bin/env python3
"""QC P6 — TC-P6-SEED-14/15/34: luồng lớp 2 (demo bước 6) qua API thật. Cần stack đã seed (xem p6-env.sh). In từng bước + kết quả.
   python3 p6-lop2.py [upload]     tải seed/demo/quy-che-lop2.pdf → extract → in bản nháp + mục chưa rõ
   python3 p6-lop2.py confirm      điền làm tròn 0,1 (nửa lên) → xác nhận → in trạng thái (đã có bản nháp)
   python3 p6-lop2.py unknown      tải một PDF lạ (không có bản ghi demo) → extract → phải báo lỗi rõ, không bản nháp rác"""
import hashlib, json, os, ssl, subprocess, sys, time, urllib.request
GW = os.environ.get("GW", "https://localhost:543"); PW = os.environ.get("PW", "Edupilot#Seed-2026")
ctx = ssl._create_unverified_context()
def call(m, p, tok=None, body=None, key=None, raw=None, hdr=None):
    h = {"Content-Type": "application/json", **(hdr or {})}
    if tok: h["Authorization"] = f"Bearer {tok}"
    if key: h["Idempotency-Key"] = key
    data = raw if raw is not None else (json.dumps(body).encode() if body is not None else None)
    try:
        r = urllib.request.urlopen(urllib.request.Request(p if p.startswith("http") else GW + p, data=data, method=m, headers=h), context=ctx)
        t = r.read(); return r.status, (json.loads(t) if t and t[:1] in b"{[" else t)
    except urllib.error.HTTPError as e:
        t = e.read(); return e.code, (json.loads(t) if t[:1] == b"{" else t)
def login(e): return call("POST", "/api/v1/auth/login", body={"email": e, "password": PW})[1]["access_token"]
T = login("teacher@edupilot.local")
psql = lambda q: subprocess.run(["docker", "exec", "edupilot100-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-tAc", q], capture_output=True, text=True).stdout.strip()
C2 = psql("select id from courses where class_code='761988'"); Q = f"/api/v1/courses/{C2}"
mode = sys.argv[1] if len(sys.argv) > 1 else "upload"
def upload(path, title):
    b = open(path, "rb").read(); sha = hashlib.sha256(b).hexdigest()
    s, pre = call("POST", f"{Q}/uploads/presign", T, {"purpose": "document", "filename": os.path.basename(path), "mime_type": "application/pdf", "size_bytes": len(b), "sha256": sha})
    req = urllib.request.Request(pre["url"], data=b, method=pre["method"], headers=pre["headers"]); urllib.request.urlopen(req, context=ctx)
    s, d = call("POST", f"{Q}/uploads/complete", T, {"upload_id": pre["upload_id"], "title": title, "type": "COURSE_POLICY"}, key=f"qc-lop2-{sha[:12]}-{int(time.time())}")
    print("complete", s, d if s >= 400 else ""); return d
if mode in ("upload", "unknown"):
    path, title = ("seed/demo/quy-che-lop2.pdf", "Quy chế lớp 2") if mode == "upload" else ("/tmp/unknown-policy.pdf", "PDF lạ không có bản ghi")
    upload(path, title)
    for _ in range(60):
        st = psql(f"select status from documents where course_id='{C2}' and title='{title}'")
        if st in ("READY", "FAILED"): break
        time.sleep(3)
    print("document", st); doc = psql(f"select id from documents where course_id='{C2}' and title='{title}'")
    s, ex = call("POST", f"{Q}/grade-scheme/extract", T, {"document_id": doc}, key=f"qc-extract-{doc[:8]}-{int(time.time())}"); print("extract", s, json.dumps(ex)[:160])
    for _ in range(40):
        s, st = call("GET", f"{Q}/grade-scheme/status", T)
        if st.get("state") not in ("EXTRACTING", "PENDING", None): break
        time.sleep(3)
    print("status", json.dumps(st, ensure_ascii=False)[:300])
    s, g = call("GET", f"{Q}/grade-scheme", T); d = g.get("draft"); print("draft", "KHÔNG CÓ" if not d else json.dumps({k: d.get(k) for k in ("version", "unclear_items", "rounding")}, ensure_ascii=False)[:400])
    print("draft components", [(c["key"], c["weight"]) for c in (d or {}).get("components", [])])
    print("lop2 draft rows:", psql(f"select status, scheme_no from grade_schemes where course_id='{C2}'"))
