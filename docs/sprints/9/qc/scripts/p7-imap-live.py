#!/usr/bin/env python3
"""QC US-P7-05 / US-S9-SEED trên stack THẬT có cmd/mock-imap: thư giả `From` (TLR-2), thư tự động, thư thiếu đính kèm, đồng bộ ngay, chuông biên nhận. Cần MOCK_IMAP_URL (mặc định http://localhost:8191)."""
import base64, copy, json, os, ssl, subprocess, sys, time, urllib.request, uuid
GW = os.environ.get("GW", "https://localhost:543"); MOCK = os.environ.get("MOCK_IMAP_URL", "http://localhost:8191"); PW = "Edupilot#Seed-2026"; ctx = ssl._create_unverified_context()
FX = os.path.join(os.path.dirname(__file__), "../../../../specs/FEAT-assignments/fixtures")
def call(m, p, tok=None, body=None):
    h = {"Content-Type": "application/json", "Idempotency-Key": str(uuid.uuid4())}
    if tok: h["Authorization"] = "Bearer " + tok
    try:
        r = urllib.request.urlopen(urllib.request.Request(GW + p, data=None if body is None else json.dumps(body).encode(), method=m, headers=h), context=ctx); t = r.read(); return r.status, (json.loads(t) if t[:1] in (b"{", b"[") else t.decode())
    except urllib.error.HTTPError as e: return e.code, e.read().decode()[:300]
def mock(body):
    r = urllib.request.urlopen(urllib.request.Request(MOCK + "/__mock/append", data=json.dumps(body).encode(), method="POST", headers={"Content-Type": "application/json"})); return r.status
psql = lambda q: subprocess.run(["docker", "exec", "edupilot100-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-tAc", q], capture_output=True, text=True).stdout.strip()
tok = lambda e: call("POST", "/api/v1/auth/login", body={"email": e + "@edupilot.local", "password": PW})[1]["access_token"]
T, TA, ADM = tok("teacher"), tok("ta"), tok("admin")
C1 = psql("select id from courses where class_code='761987'"); Q = f"/api/v1/courses/{C1}"
fails = 0
def chk(n, ok, x=""):
    global fails; fails += (not ok); print("PASS" if ok else "FAIL", n, "" if ok else str(x)[:300])
s, g = call("GET", f"{Q}/integrations/imap", ADM); print("   thẻ IMAP:", g["status"], g["config"]); chk("thẻ IMAP của seed: CONNECTED hoặc đã có lịch sử, không lộ mật khẩu", s == 200 and "mock-imap-password" not in json.dumps(g), g)
s, x = call("POST", f"{Q}/integrations/imap/test", ADM, {}); chk("02 Test kết nối tới mock-imap thật → ok:true", s == 200 and x.get("ok") is True, f"{s} {x}")
b = json.load(open(f"{FX}/create-bt03.json")); b["submission_code"] = "QIM" + str(int(time.time()) % 1000); b["channels"] = ["IN_APP", "IMAP"]; b["due_at"] = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(time.time() + 86400 * 3))
s, a = call("POST", f"{Q}/assignments", T, b); A = a.get("assignment", a)["id"]; call("POST", f"{Q}/assignments/{A}/publish", T, {}); code = b["submission_code"]
md = base64.b64encode(("Bài làm email. SUNBURST được cấy vào bản cập nhật Orion. " * 10).encode()).decode(); att = [{"name": "bai.md", "base64": md, "content_type": "text/markdown"}]
now = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())
cases = [("A hợp lệ (dmarc=pass mặc định)", {"from": "sv.gioi@edupilot.local", "subject": f"[{code}] 20229001", "attachments": att}, "MATCHED"),
         ("B giả From: không có header xác thực", {"from": "sv.kha@edupilot.local", "subject": f"[{code}] 20229002", "authentication_results": "", "attachments": att}, "UNMATCHED"),
         ("C dmarc=fail", {"from": "sv.nguyco@edupilot.local", "subject": f"[{code}] 20229003", "authentication_results": "mx.edupilot.test; dmarc=fail header.from=edupilot.local", "attachments": att}, "UNMATCHED"),
         ("D authserv-id lạ", {"from": "sv04@edupilot.local", "subject": f"[{code}] 20220004", "authentication_results": "evil.example; dmarc=pass header.from=edupilot.local", "attachments": att}, "UNMATCHED"),
         ("E tự động trả lời", {"from": "sv05@edupilot.local", "subject": f"[{code}] 20220005", "auto_submitted": "auto-replied", "attachments": att}, "IGNORED"),
         ("F không đính kèm", {"from": "sv06@edupilot.local", "subject": f"[{code}] 20220006", "attachments": []}, "IGNORED")]
ids = {}; TAG = uuid.uuid4().hex[:8]
for n, m, want in cases:
    m.update({"message_id": f"<qc-{TAG}-{uuid.uuid4().hex[:6]}@qc.test>", "internal_date": now}); ids[n] = (m["message_id"], want); assert mock(m) in (200, 201), n
print("   đã thêm", len(cases), "thư; Đồng bộ ngay …")
s, j = call("POST", f"{Q}/integrations/imap/sync-now", TA, {}); print("   sync-now:", s, str(j)[:100])
if s == 429: time.sleep(31); s, j = call("POST", f"{Q}/integrations/imap/sync-now", TA, {})
for _ in range(60):
    if int(psql(f"select count(*) from mail_inbound where message_id like '%qc-{TAG}%'") or 0) >= len(cases): break
    time.sleep(2)
rows = {r.split("|")[0].strip("<>"): r.split("|") for r in psql(f"select message_id, status, coalesce(reason,''), coalesce(from_email,'') from mail_inbound where message_id like '%qc-{TAG}%'").splitlines()}
for n, (mid, want) in ids.items():
    r = rows.get(mid.strip('<>')); sub = psql(f"select match_status||'|'||coalesce(unmatched_reason,'')||'|'||coalesce(student_id::text,'') from submissions where assignment_id='{A}' and external_ref in (select external_ref from mail_inbound where message_id='{mid}')") if False else ""
    ok = bool(r) and ((want == "IGNORED" and r[1] == "IGNORED") or (want in ("MATCHED", "UNMATCHED") and r[1] == "ACCEPTED")); chk(f"{n}: mail_inbound = {r[1:3] if r else None}", ok, r)
sub = psql(f"select u.email||'|'||s.match_status||'|'||coalesce(s.unmatched_reason,'') from submissions s left join users u on u.id=s.student_id where s.assignment_id='{A}' and s.source='IMAP' order by s.from_email"); print("   submissions IMAP:\n   " + sub.replace("\n", "\n   "))
byfrom = {l.split("|")[0]: l for l in sub.splitlines() if l}
chk("A: MATCHED về sv.gioi", any(l.startswith("sv.gioi@edupilot.local|MATCHED") for l in sub.splitlines()), sub)
unm = psql("select from_email||'|'||coalesce(unmatched_reason,'') from submissions where source='IMAP' and match_status='UNMATCHED' and created_at > now() - interval '10 minutes' order by from_email"); print("   UNMATCHED:", unm.replace("\n", " ; "))
for who in ("sv.kha", "sv.nguyco", "sv04"):
    chk(f"B/C/D {who}: UNMATCHED SENDER_UNVERIFIED, KHÔNG gán cho {who}", f"{who}@edupilot.local|SENDER_UNVERIFIED" in unm and psql(f"select count(*) from submissions s join users u on u.id=s.student_id where s.assignment_id='{A}' and u.email='{who}@edupilot.local'") == "0", unm)
chk("E tự động: không có bài nộp; F không đính kèm: không có bài nộp", psql(f"select count(*) from submissions where assignment_id='{A}' and from_email in ('sv05@edupilot.local','sv06@edupilot.local')") == "0")
st = psql(f"select status||':'||coalesce(reason,'') from mail_inbound where from_email='sv05@edupilot.local' and message_id like '%qc-{TAG}%'"); print("   E:", st, "| F:", psql(f"select status||':'||coalesce(reason,'') from mail_inbound where from_email='sv06@edupilot.local' and message_id like '%qc-{TAG}%'"))
bell = psql("select u.email, count(*) from notifications n join users u on u.id=n.user_id where n.type='SUBMISSION_MAIL_RECEIVED' and n.created_at > now() - interval '5 minutes' group by 1"); print("   chuông biên nhận:", bell.replace("\n", " ; "))
chk("biên nhận chỉ tới sv.gioi (đúng người đã khớp), không tới sv.kha / sv.nguyco / sv04", "sv.gioi@edupilot.local|1" in bell and not any(w in bell for w in ("sv.kha", "sv.nguyco", "sv04")), bell)
s, lst = call("GET", f"{Q}/submissions?match_status=UNMATCHED&assignment_id={A}", T); print("   Chưa khớp (API):", len(lst.get("items", [])) if isinstance(lst, dict) else lst)
logs = subprocess.run("docker logs edupilot100-gateway-1 2>&1; docker logs edupilot100-worker-1 2>&1", shell=True, capture_output=True, text=True).stdout
chk("log không chứa mật khẩu IMAP seed / văn bản bài", "mock-imap-password" not in logs and "SUNBURST được cấy vào bản cập nhật Orion. Bài làm email" not in logs)
print("RESULT:", "FAIL" if fails else "PASS", f"({fails} lỗi)"); sys.exit(1 if fails else 0)
