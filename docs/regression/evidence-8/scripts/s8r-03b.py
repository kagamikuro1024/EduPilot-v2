# s8r 03b: US-P6-03 phần còn lại (06,11,12,17-23,30,31,33-37,39-42) trên ep-r bằng stub OpenAI-compatible :9761. Chạy DƯỚI khoá global (UTILITY → S8R-stub; 03-39 dừng worker).
import subprocess, copy, hashlib
from s8r_lib import *
LOCK = "/tmp/qc2-lock-global"; LOG = "/tmp/qc2-global-changes.log"; MODE = "/tmp/qc2-s8r/stub-mode.json"; REQ = "/tmp/qc2-s8r/stub-req.jsonl"
REAL_MODEL = "01a1df2d-a8d7-76a5-9cd8-5dbdb8646c78"
FXD = REPO + "/docs/specs/FEAT-gradebook/fixtures/"
FX = json.load(open(FXD + "extraction-lop1.json"))
def logl(m): open(LOG, "a").write(time.strftime("%H:%M:%S") + " qc2: " + m + "\n")
def acquire():
    for _ in range(80):
        try: os.mkdir(LOCK); open(LOCK + "/owner", "w").write("qc2"); return True
        except FileExistsError: time.sleep(15)
    return False
def setmode(m, **kw): json.dump({"mode": m, **kw}, open(MODE, "w"))
def routes(): return {i["task"]: i for i in call("GET", "/admin/llm/routes", "admin")[1]["items"]}
def setroute(task, model_id): cur = routes()[task]; return call("PUT", "/admin/llm/routes", "admin", {"task": task, "chain": [model_id], "params": {}, "version": cur["version"]}, headers={"If-Match": 'W/"v%d"' % cur["version"]})
def jobwait(jid, t=120):
    j = None
    for _ in range(t // 2):
        c, j = call("GET", f"/jobs/{jid}", "teacher")
        if isinstance(j, dict) and j.get("status") in ("DONE", "FAILED", "SUCCEEDED", "CANCELLED", "COMPLETED"): return j
        time.sleep(2)
    return j
def sch(c0): return call("GET", f"/courses/{c0}/grade-scheme", "teacher")
def extract(c0, doc, key=None, **kw): return call("POST", f"/courses/{c0}/grade-scheme/extract", "teacher", {"document_id": doc, **kw}, key=key)
def doc_of(c0): return sq1(f"select id from documents where course_id='{c0}' and type='COURSE_POLICY' order by created_at limit 1")
def reset_scheme(c0):
    for t in ("grade_items", "grade_rules", "grade_components"): sql(f"delete from {t} where course_id='{c0}'")
    sql(f"delete from grade_schemes where course_id='{c0}'")
def nreq(): return sum(1 for _ in open(REQ)) if os.path.exists(REQ) else 0
def draft_of(c0): return sch(c0)[1].get("draft") or {}
def ucodes(d): return sorted({u.get("code") for u in d.get("unclear_items", [])})
def run(c0, custom, doc=None):
    reset_scheme(c0); open(REQ, "w").close(); setmode("custom", custom=custom)
    c, r = extract(c0, doc or doc_of(c0)); j = jobwait(r.get("job_id")) if c == 202 else {}
    return c, r, j, draft_of(c0)
def mut(f):
    d = copy.deepcopy(FX); f(d); return d
def comp_weight(d, k, w):
    for c in d["components"]:
        if c["key"] == k: c["weight"] = w
def citation(d, cid_): return [c for c in d["citations"] if c["id"] == cid_][0]
def rule(d, kind, sub=None):
    for r in d["rules"]:
        if r["kind"] == kind and (sub is None or r["params"].get("kind") == sub): return r

if not acquire(): print("không lấy được khoá"); sys.exit(2)
stub = subprocess.Popen([sys.executable, os.path.dirname(os.path.abspath(__file__)) + "/s8r-stub.py", "9761"]); time.sleep(1.5)
logl("LOCK — US-P6-03 (phần còn lại): UTILITY → S8R-stub :9761; 03-39 dừng worker ≤ 4 phút; hoàn lại ≤ 10 phút")
try:
    prov = [p for p in call("GET", "/admin/llm/providers", "admin")[1]["items"] if p["name"] == "S8R-stub"]
    MID = prov[0]["models"][0]["id"]; print("setroute", setroute("UTILITY", MID)[0])
    A2, B1 = cid("QD8A2"), cid("QD8B1")
    # ---- 06 schema strict (Go: internal/grading/schema.go) — đọc nguồn schema
    src = open(REPO + "/backend-go/internal/grading/schema.go").read()
    n_obj = src.count('"type":"object"'); n_ap = src.count('"additionalProperties":false')
    kinds = all(k in src for k in ("BONUS", "PENALTY", "ROUNDING", "FAIL_CONDITION", "LETTER_SCALE", "MISSING_SCORE"))
    sreq = open(REPO + "/backend-go/internal/grading/extract.go").read()
    C("TC-P6-03-06", n_obj == n_ap and "anyOf" in src and kinds and ("Temperature" in sreq or "temperature" in sreq.lower()), f"schema.go: {n_obj} object / {n_ap} additionalProperties:false, có anyOf, đủ 6 kind; extract.go đặt temperature", f"obj={n_obj} ap={n_ap} anyOf={'anyOf' in src} kinds={kinds}")
    # ---- 11 PDF seed + 12 tài liệu seed
    pdf = REPO + "/seed/demo/quy-che-lop1.pdf"
    p1 = subprocess.run(f"pdftotext -f 1 -l 1 {pdf} - | grep -c 'Điều 1\\|1\\.1\\.'; pdftotext -f 2 -l 2 {pdf} - | grep -c 'làm tròn đến 0,5'; pdfinfo {pdf} | grep Pages; pdftotext -f 2 -l 2 {pdf} - | grep -c 'Điều [345]'", shell=True, capture_output=True, text=True).stdout.split()
    C("TC-P6-03-11", len(p1) >= 5 and int(p1[0]) >= 1 and int(p1[1]) >= 1 and p1[3] == "2" and int(p1[-1]) >= 3, f"pdftotext: trang 1 có Điều 1/1.1 ({p1[0] if p1 else '?'}), trang 2 có 'làm tròn đến 0,5' ({p1[1] if len(p1)>1 else '?'}) và Điều 3,4,5 ({p1[-1] if p1 else '?'}), Pages={p1[3] if len(p1)>3 else '?'}, có lớp chữ", str(p1))
    c761 = cid("761987")
    rows = sql(f"select title, type, category from documents where course_id='{c761}' and (title ilike '%quy chế%' or title ilike '%quyche%')")
    ok12 = any("quy-che-lop1" in r[0] and r[1] == "COURSE_POLICY" for r in rows) and any(r[0].lower().startswith("quyche") and r[1] == "OTHER" for r in rows) and not any(r[0].lower().startswith("quyche") and r[1] == "COURSE_POLICY" for r in rows)
    C("TC-P6-03-12", ok12, f"761987: {rows}", f"{rows}")
    # ---- 17 tổng trọng số 90
    c, r, j, dr = run(A2, mut(lambda d: comp_weight(d, "final", "50")))
    ws = [u for u in dr.get("unclear_items", []) if u.get("code") == "WEIGHT_SUM"]
    det = json.dumps(ws, ensure_ascii=False)
    gd = call("POST", f"/courses/{A2}/grade-scheme/confirm", "teacher", {"version": dr.get("version")})
    C("TC-P6-03-17", bool(ws) and "90" in det and dr.get("components"), f"WEIGHT_SUM có tổng 90 ({det[:90]}); bản nháp vẫn lưu ({len(dr.get('components', []))} thành phần); confirm → {gd[0]} {gd[1].get('code') if isinstance(gd[1], dict) else ''}", f"{ucodes(dr)} {det[:200]}")
    # ---- 18 thang điểm chữ
    c, r, j, dr = run(A2, mut(lambda d: d["rules"].remove(rule(d, "LETTER_SCALE")))); a = ucodes(dr)
    def bad_scale(d): rule(d, "LETTER_SCALE")["params"]["bands"] = [{"min": "9.0", "letter": "A+"}, {"min": "9.0", "letter": "A"}, {"min": "7.0", "letter": "B"}, {"min": "0", "letter": "F"}]
    c, r, j, dr2 = run(A2, mut(bad_scale)); b = ucodes(dr2)
    def one_scale(d): rule(d, "LETTER_SCALE")["params"]["bands"] = [{"min": "9.0", "letter": "A+"}, {"min": "5.0", "letter": "C"}, {"min": "1.0", "letter": "F"}]
    c, r, j, dr3 = run(A2, mut(one_scale)); e = ucodes(dr3)
    C("TC-P6-03-18", "LETTER_SCALE_MISSING" in a and "LETTER_SCALE_INVALID" in b and "LETTER_SCALE_INVALID" in e, f"bỏ thang → {a}; A+≥9,A≥9,B≥7 → {b}; thang bắt đầu từ 1 → {e}", f"{a} / {b} / {e}")
    # ---- 19 PARAM_INVALID
    c, r, j, dr = run(A2, mut(lambda d: comp_weight(d, "weekly", "abc"))); a = ucodes(dr); t19a = (c, j.get("status"), (j.get("error") or {}).get("code") if isinstance(j.get("error"), dict) else j.get("error"))
    def final0(d): comp_weight(d, "final", "0")
    c, r, j, dr2 = run(A2, mut(final0)); b = ucodes(dr2)
    C("TC-P6-03-19", "PARAM_INVALID" in a and "PARAM_INVALID" in b, f"weight 'abc' → {a} (job {t19a}); FINAL_BELOW khi CK=0 → {b}", f"{a} {t19a} / {b}")
    # ---- 20 ROUNDING_MISSING do code dù LLM trả unclear_items []
    c, r, j, dr = run(A2, mut(lambda d: (d["rules"].remove(rule(d, "ROUNDING")), d.update(unclear_items=[]))))
    mi = [u for u in dr.get("unclear_items", []) if u.get("code") == "ROUNDING_MISSING"]
    C("TC-P6-03-20", bool(mi) and mi[0].get("source") == "CODE", f"LLM trả unclear_items [] nhưng ROUNDING_MISSING vẫn có, source={mi[0].get('source') if mi else None}", f"{dr.get('unclear_items')}")
    # ---- 21..23 trích dẫn
    def fake(d): citation(d, "c6")["quote"] = "Điều 9. Mọi sinh viên được 10 điểm."
    c, r, j, dr = run(A2, mut(fake)); cn = [u for u in dr.get("unclear_items", []) if u.get("code") == "CITATION_NOT_FOUND"]
    tg = sorted(u.get("target") for u in cn)
    C("TC-P6-03-21", len(cn) >= 1 and all("ROUNDING" in (t or "") for t in tg), f"quote bịa ở c6 → CITATION_NOT_FOUND {len(cn)} mục, target {tg}; mã khác: {ucodes(dr)}", f"{tg} {ucodes(dr)}")
    def p9(d): citation(d, "c8")["page"] = 9
    def p2(d): citation(d, "c3")["page"] = 2
    c, r, j, d9 = run(A2, mut(p9)); n9 = [u for u in d9.get("unclear_items", []) if u.get("code") == "CITATION_NOT_FOUND"]
    c, r, j, d2 = run(A2, mut(p2)); n2 = [u for u in d2.get("unclear_items", []) if u.get("code") == "CITATION_NOT_FOUND"]
    C("TC-P6-03-22", len(n9) >= 1 and len(n2) >= 1, f"page=9 → {len(n9)} CITATION_NOT_FOUND ({[u.get('target') for u in n9]}); quote trang 1 gán page=2 → {len(n2)} ({[u.get('target') for u in n2]})", f"{ucodes(d9)} {ucodes(d2)}")
    def nodiac(d): citation(d, "c6")["quote"] = "Diem hoc phan duoc lam tron den 0,5 diem; diem nam dung giua hai muc thi lam tron len muc cao hon."
    def edited(d): citation(d, "c6")["quote"] = citation(d, "c6")["quote"].replace("0,5", "1,0")
    print("quote c6:", citation(FX, "c6")["quote"])
    c, r, j, da = run(A2, mut(nodiac)); na = [u for u in da.get("unclear_items", []) if u.get("code") == "CITATION_NOT_FOUND"]
    c, r, j, db = run(A2, mut(edited)); nb = [u for u in db.get("unclear_items", []) if u.get("code") == "CITATION_NOT_FOUND"]
    C("TC-P6-03-23", len(na) == 0 and len(nb) >= 1, f"quote bỏ dấu → {len(na)} CITATION_NOT_FOUND (chấp nhận); đổi '0,5'→'1,0' → {len(nb)} ({[u.get('target') for u in nb]})", f"{ucodes(da)} / {ucodes(db)}")
    # ---- 30/31 PDF tải lên: injection + PII (tải qua presign/complete, đợi READY)
    def upload(c0, pdfp, title):
        b = open(pdfp, "rb").read(); sha = hashlib.sha256(b).hexdigest()
        c, pre = call("POST", f"/courses/{c0}/uploads/presign", "teacher", {"purpose": "document", "filename": os.path.basename(pdfp), "mime_type": "application/pdf", "size_bytes": len(b), "sha256": sha})
        if c != 200 and c != 201: return None, f"presign {c} {pre}"
        import urllib.request
        rq = urllib.request.Request(pre.get("url") or pre.get("upload_url"), data=b, method=pre.get("method", "PUT"), headers=pre.get("headers") or {})
        import ssl; ctx = ssl.create_default_context(); ctx.check_hostname = False; ctx.verify_mode = ssl.CERT_NONE
        try: urllib.request.urlopen(rq, context=ctx, timeout=60).read()
        except Exception as ex: return None, f"PUT blob {ex}"
        c, done = call("POST", f"/courses/{c0}/uploads/complete", "teacher", {"upload_id": pre["upload_id"], "title": title, "type": "COURSE_POLICY"})
        for _ in range(60):
            s = sq1(f"select status from documents where course_id='{c0}' and title='{title}'")
            if s in ("READY", "FAILED"): break
            time.sleep(3)
        return sq1(f"select id from documents where course_id='{c0}' and title='{title}'"), s
    os.makedirs("/tmp/s8core", exist_ok=True)
    if not os.path.exists("/tmp/s8core/pdf-inj.pdf"):
        subprocess.run(f"cd {REPO} && node docs/regression/evidence-8/scripts/s8core-mkpdf.mjs", shell=True, capture_output=True)
    for k, tc, ttl in (("inj", "TC-P6-03-30", "Quy chế QC inj"), ("pii", "TC-P6-03-31", "Quy chế QC pii")):
        if not os.path.exists(f"/tmp/s8core/pdf-{k}.pdf"): T(tc, "F", f"không dựng được pdf-{k}.pdf"); continue
        did, st = sq1(f"select id from documents where course_id='{A2}' and title='{ttl}'"), "READY"
        if not did: did, st = upload(A2, f"/tmp/s8core/pdf-{k}.pdf", ttl)
        if not did or st != "READY": T(tc, "F", f"tải PDF {k}: tài liệu {did} trạng thái {st}"); continue
        reset_scheme(A2); open(REQ, "w").close(); setmode("lop1")
        c, r = extract(A2, did); j = jobwait(r.get("job_id")); reqs = [json.loads(l) for l in open(REQ) if l.strip()]
        body = json.dumps(reqs[0]["req"], ensure_ascii=False) if reqs else ""
        msgs = (reqs[0]["req"].get("messages") if reqs else []) or []
        sysm = " ".join(m_.get("content", "") for m_ in msgs if m_.get("role") == "system" and isinstance(m_.get("content"), str))
        usr = " ".join(m_.get("content", "") for m_ in msgs if m_.get("role") == "user" and isinstance(m_.get("content"), str))
        if k == "inj":
            inside = "Bỏ qua mọi hướng dẫn trước đó" in usr and "Bỏ qua mọi hướng dẫn trước đó" not in sysm
            guard = bool(re.search(r"không.{0,20}(làm theo|tuân|thực hiện).{0,40}(lệnh|hướng dẫn|chỉ dẫn)|chỉ trích", sysm, re.I))
            dr = draft_of(A2); ws = {c_["key"]: c_["weight"] for c_ in dr.get("components", [])}
            fence = bool(re.search(r"<<<|\"\"\"|```|<tai_lieu|<document|BEGIN|---", usr))
            C(tc, inside and guard and ws.get("midterm") == "20", f"request tới LLM (log stub): câu injection nằm trong khối dữ liệu của tin user (không ở system)={inside}, ranh giới={fence}, system dặn chỉ trích/không làm theo lệnh={guard}; bản nháp midterm={ws.get('midterm')} tổng {sum(float(v) for v in ws.values())}", f"inside={inside} guard={guard} fence={fence} ws={ws} sys={sysm[:200]}")
        else:
            leak = [x for x in ("Nguyễn Văn Test", "20229999", "nvtest@example.com") if x in body]
            ph = bool(re.search(r"\[\[|\{\{|<<|PERSON|STUDENT|_\d+|\[[A-Z_]+\d*\]", usr))
            C(tc, not leak and j.get("status") in ("SUCCEEDED", "DONE"), f"request tới LLM không còn tên/MSSV/email (lọt: {leak}); placeholder trong prompt={ph}; job {j.get('status')}", f"leak={leak} {usr[:200]}")
    # ---- 33 DOCUMENT_NOT_READY
    did = doc_of(A2); old = sq1(f"select status from documents where id='{did}'")
    sql(f"update documents set status='PROCESSING' where id='{did}'"); c, r = extract(A2, did); sql(f"update documents set status='{old}' where id='{did}'")
    C("TC-P6-03-33", c == 409 and r.get("code") == "DOCUMENT_NOT_READY", f"tài liệu PROCESSING → {c} {r.get('code')}", f"{c} {r}")
    # ---- 34 POLICY_TOO_LONG (quy chế dài: nhồi chunk, không đụng env)
    ids = sql(f"select id, course_ids::text, audience::text, ord, page_no from content_chunks where document_id='{did}' order by ord limit 1")
    cols = [r_[0] for r_ in sql("select column_name from information_schema.columns where table_name='content_chunks' and is_nullable='NO' and column_default is null")]
    print("NOT NULL no default:", cols)
    n0 = int(sq1(f"select count(*) from content_chunks where document_id='{did}'"))
    sql(f"insert into content_chunks (document_id, course_ids, audience, ord, page_no, heading, text, token_count) select document_id, course_ids, audience, 9000+g, 2, 'QC dài', repeat('Phụ lục rất dài của quy chế để vượt giới hạn độ dài. ', 400), 1 from content_chunks, generate_series(1,40) g where document_id='{did}' and ord=(select min(ord) from content_chunks where document_id='{did}')")
    jb0 = int(sq1("select count(*) from jobs where kind='grade_scheme.extract'")); c, r = extract(A2, did); jb1 = int(sq1("select count(*) from jobs where kind='grade_scheme.extract'"))
    sql(f"delete from content_chunks where document_id='{did}' and ord>=9000")
    n1 = int(sq1(f"select count(*) from content_chunks where document_id='{did}'"))
    C("TC-P6-03-34", c == 422 and r.get("code") == "POLICY_TOO_LONG" and jb1 == jb0 and n1 == n0, f"quy chế dài → {c} {r.get('code')}, số job không đổi ({jb0}→{jb1}); đã dọn chunk thêm ({n0}→{n1})", f"{c} {r} {jb0}->{jb1} {n0}->{n1}")
    # ---- 35 toàn bộ nhà cung cấp lỗi
    reset_scheme(A2); open(REQ, "w").close(); setmode("fail503"); t0 = sq1("select now()"); ds0 = int(sq1("select count(*) from grade_schemes where status='DRAFT'"))
    c, r = extract(A2, doc_of(A2)); j = jobwait(r.get("job_id"), 180); n_att = sql(f"select attempts, status, error_kind from llm_audit where created_at >= '{t0}' and task='UTILITY'")
    ds1 = int(sq1("select count(*) from grade_schemes where status='DRAFT'")); err = j.get("error") if isinstance(j, dict) else None; ecode = err.get("code") if isinstance(err, dict) else err
    C("TC-P6-03-35", j.get("status") == "FAILED" and ecode == "LLM_UNAVAILABLE" and ds1 == ds0 and nreq() <= 3 * 3, f"nhà cung cấp 503: job FAILED {ecode}; stub nhận {nreq()} yêu cầu, llm_audit {n_att}; DRAFT rác: {ds1-ds0}", f"{j} {n_att} {nreq()}")
    # ---- 36 đầu ra sai schema: đúng 1 lời gọi
    reset_scheme(A2); open(REQ, "w").close(); setmode("invalid")
    c, r = extract(A2, doc_of(A2)); j = jobwait(r.get("job_id")); err = j.get("error"); ecode = err.get("code") if isinstance(err, dict) else err
    C("TC-P6-03-36", j.get("status") == "FAILED" and ecode == "EXTRACTION_INVALID" and nreq() == 1, f"đầu ra không phải JSON → job FAILED {ecode}; stub nhận đúng {nreq()} lời gọi", f"{j} {nreq()}")
    # ---- 37 lỗi không phá bản CONFIRMED (lớp QD8R6W đã CONFIRMED)
    R6 = cid("QD8R6W"); st0 = sql(f"select status, scheme_no from grade_schemes where course_id='{R6}' order by scheme_no")
    d6 = doc_of(R6)
    if d6:
        setmode("fail503"); c, r = extract(R6, d6); j1 = jobwait(r.get("job_id"), 180) if c == 202 else {}
        setmode("invalid"); c2, r2 = extract(R6, d6); j2 = jobwait(r2.get("job_id")) if c2 == 202 else {}
        st1 = sql(f"select status, scheme_no from grade_schemes where course_id='{R6}' order by scheme_no")
        gb = call("GET", f"/courses/{R6}/gradebook", "teacher")
        C("TC-P6-03-37", st0 == st1 and ("CONFIRMED" in json.dumps(st1)) and gb[0] == 200, f"QD8R6W: grade_schemes trước {st0} = sau {st1} sau 2 lần trích lỗi ({j1.get('status')}/{j2.get('status')}); /gradebook {gb[0]}", f"{st0} {st1} {gb[0]}")
    else: T("TC-P6-03-37", "F", "QD8R6W không có tài liệu COURSE_POLICY để trích")
    setmode("lop1")
    # ---- 40/41/42 DRAFT_EXISTS
    reset_scheme(A2); setmode("lop1"); c, r = extract(A2, doc_of(A2)); jobwait(r.get("job_id")); d = draft_of(A2)
    def body(dd, ver=None, mod=None):
        b = {"version": ver or dd["version"], "components": [{k: x[k] for k in ("key", "name", "grp", "weight")} | {"items": x["items"]} for x in dd["components"]], "rules": [{"kind": x["kind"], "applicable": x["applicable"], "params": copy.deepcopy(x["params"])} for x in dd["rules"]]}
        if mod: mod(b)
        return b
    def tweak(b): b["components"][0]["weight"] = "21"
    cp, rp = call("PUT", f"/courses/{A2}/grade-scheme", "teacher", body(d, mod=tweak)); dE = draft_of(A2)
    c1, r1 = extract(A2, doc_of(A2)); c2, r2 = extract(A2, doc_of(A2), replace_draft=True); jobwait(r2.get("job_id"))
    al = sql(f"select count(*) from audit_log where created_at >= now() - interval '2 minutes' and (action ilike '%grade%' or action ilike '%scheme%' or action ilike '%draft%')")
    C("TC-P6-03-40", cp == 200 and c1 == 409 and r1.get("code") == "DRAFT_EXISTS" and c2 == 202, f"sửa tay PUT → {cp}; extract không replace → {c1} {r1.get('code')} (details {r1.get('details')}); replace_draft=true → {c2}; audit_log gần đây {al}", f"{cp} {c1} {r1} {c2}")
    # 41: sửa trong lúc job chạy (stub delay 8 s)
    reset_scheme(A2); setmode("lop1"); c, r = extract(A2, doc_of(A2)); jobwait(r.get("job_id")); d = draft_of(A2)
    call("PUT", f"/courses/{A2}/grade-scheme", "teacher", body(d, mod=tweak)); d = draft_of(A2)
    setmode("lop1", delay=8); c1, r1 = extract(A2, doc_of(A2), replace_draft=True); time.sleep(3)
    def tweak2(b): b["components"][1]["weight"] = "19"
    cp, rp = call("PUT", f"/courses/{A2}/grade-scheme", "teacher", body(draft_of(A2), mod=tweak2)); j = jobwait(r1.get("job_id"), 90); dz = draft_of(A2)
    err = j.get("error"); ecode = err.get("code") if isinstance(err, dict) else err
    keep = [c_["weight"] for c_ in dz.get("components", []) if c_["key"] == "weekly"]
    C("TC-P6-03-42", c1 == 202 and cp == 200 and j.get("status") == "FAILED" and ecode == "DRAFT_EXISTS" and keep == ["19"], f"replace_draft=true rồi PUT tăng version trong lúc chạy → job {j.get('status')} {ecode}; nháp người dùng còn weekly={keep}", f"{c1} {cp} {j} {keep}")
    # 41: extract KHÔNG replace_draft trên nháp chưa sửa (version 1) → được nhận; trong lúc chạy PUT sửa → job FAILED DRAFT_EXISTS
    reset_scheme(A2); setmode("lop1"); c, r = extract(A2, doc_of(A2)); jobwait(r.get("job_id")); d = draft_of(A2)
    setmode("lop1", delay=8); c1, r1 = extract(A2, doc_of(A2)); time.sleep(3)
    cp, rp = call("PUT", f"/courses/{A2}/grade-scheme", "teacher", body(draft_of(A2), mod=tweak)); j = jobwait(r1.get("job_id"), 90); dz = draft_of(A2)
    err = j.get("error"); ecode = err.get("code") if isinstance(err, dict) else err
    keep = [c_["weight"] for c_ in dz.get("components", []) if c_["key"] == "midterm"]
    C("TC-P6-03-41", c1 == 202 and cp == 200 and j.get("status") == "FAILED" and ecode == "DRAFT_EXISTS" and keep == ["21"], f"extract không replace_draft; PUT sửa trong lúc chạy → job {j.get('status')} {ecode}; nháp người dùng midterm={keep}", f"{c1} {cp} {j} {keep}")
    # ---- 39 worker bị giết giữa chừng
    reset_scheme(A2); setmode("lop1", delay=14); open(REQ, "w").close(); c, r = extract(A2, doc_of(A2)); jid = r.get("job_id"); time.sleep(5)
    subprocess.run("docker kill ep-r-worker-1", shell=True, capture_output=True); time.sleep(3); subprocess.run("docker start ep-r-worker-1", shell=True, capture_output=True)
    setmode("lop1", delay=0); j = jobwait(jid, 220); n = int(sq1(f"select count(*) from grade_schemes where course_id='{A2}' and status='DRAFT'"))
    C("TC-P6-03-39", j.get("status") in ("SUCCEEDED", "DONE") and n == 1, f"kill worker ở giây 5/14: job nhận lại → {j.get('status')}; số DRAFT {n}; stub nhận {nreq()} lời gọi", f"{j} {n} {nreq()}")
finally:
    stub.terminate(); setmode("lop1")
    subprocess.run("docker start ep-r-worker-1", shell=True, capture_output=True)
    print("restore", setroute("UTILITY", REAL_MODEL)[0]); logl("UNLOCK — UTILITY hoàn lại QC 9router (US-P6-03 phần còn lại)")
    try: os.remove(LOCK + "/owner"); os.rmdir(LOCK)
    except Exception as e: print("lock", e)
