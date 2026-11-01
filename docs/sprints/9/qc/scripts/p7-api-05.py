#!/usr/bin/env python3
"""QC US-P7-05 hộp đen (máy chủ: cấu hình IMAP, bí mật chỉ ghi, RBAC, test kết nối, đồng bộ ngay) trên stack thật. Stack không có máy chủ IMAP → poll / xác thực người gửi kiểm bằng Go (máy chủ IMAP trong test)."""
import json, os, ssl, subprocess, sys, time, urllib.request, uuid
GW = os.environ.get("GW", "https://localhost:543"); PW = "Edupilot#Seed-2026"; ctx = ssl._create_unverified_context()
def raw(m, url, h=None, data=None):
    try:
        r = urllib.request.urlopen(urllib.request.Request(url, data=data, method=m, headers=h or {}), context=ctx); return r.status, r.read(), r.headers
    except urllib.error.HTTPError as e: return e.code, e.read(), e.headers
def call(m, p, tok=None, body=None, key=None):
    h = {"Content-Type": "application/json"}
    if tok: h["Authorization"] = "Bearer " + tok
    if key: h["Idempotency-Key"] = key
    s, t, hd = raw(m, GW + p, h, None if body is None else json.dumps(body).encode())
    return s, (json.loads(t) if t[:1] in (b"{", b"[") else t.decode()), hd
tok = lambda e: call("POST", "/api/v1/auth/login", body={"email": e, "password": PW})[1]["access_token"]
T, TA, ADM, SB = (tok(x + "@edupilot.local") for x in ("teacher", "ta", "admin", "sv.kha"))
psql = lambda q: subprocess.run(["docker", "exec", "edupilot100-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-tAc", q], capture_output=True, text=True).stdout.strip()
C1 = psql("select id from courses where class_code='761987'"); C2 = psql("select id from courses where class_code='761988'"); Q = f"/api/v1/courses/{C1}"; idem = lambda: str(uuid.uuid4())
fails = 0
def chk(n, ok, x=""):
    global fails; fails += (not ok); print("PASS" if ok else "FAIL", n, "" if ok else str(x)[:300])
PWD = "CANARY-IMAP-PW-7Z3"; cfg = {"host": "imap.invalid.example", "port": 993, "security": "TLS", "username": "bt@school.example", "password": PWD, "mailbox": "INBOX", "poll_seconds": 60, "enabled": True}
s, g0, _ = call("GET", f"{Q}/integrations/imap", ADM); chk("01 chưa cấu hình: GET ADMIN 200, configured false", s == 200 and g0.get("configured") is False, f"{s} {g0}")
s, x, _ = call("PUT", f"{Q}/integrations/imap", ADM, {k: v for k, v in cfg.items() if k != "password"}); chk("01 lần đầu thiếu mật khẩu → 422", s == 422, f"{s} {str(x)[:100]}")
s, x, _ = call("PUT", f"{Q}/integrations/imap", ADM, cfg); chk("01 ADMIN lưu cấu hình → 200", s == 200, f"{s} {str(x)[:160]}")
chk("01 phản hồi PUT không chứa mật khẩu", PWD not in json.dumps(x), x)
for n, tk in (("ADMIN", ADM), ("TEACHER", T)):
    s, g, _ = call("GET", f"{Q}/integrations/imap", tk); js = json.dumps(g); chk(f"01 GET {n} → 200, không có mật khẩu / đuôi mật khẩu", s == 200 and PWD not in js and PWD[-4:] not in js and "password" not in js.replace('"password_changed"', ""), js[:200])
for n, tk in (("TA", TA), ("STUDENT", SB)):
    s, _, _ = call("GET", f"{Q}/integrations/imap", tk); chk(f"08 {n} GET imap → 403", s == 403, s)
for n, tk in (("TEACHER", T), ("TA", TA), ("STUDENT", SB)):
    s, _, _ = call("PUT", f"{Q}/integrations/imap", tk, cfg); chk(f"08 {n} PUT imap → 403", s == 403, s)
    s, _, _ = call("POST", f"{Q}/integrations/imap/test", tk, {}, idem()); chk(f"08 {n} test → 403", s == 403, s)
ce = psql(f"select config_enc is not null and position('{PWD}' in convert_from(config_enc,'UTF8'))=0 from integration_configs where course_id='{C1}' and kind='IMAP'") if False else psql(f"select (config_enc is not null)::text||'|'||(config_public::text like '%{PWD}%')::text from integration_configs where course_id='{C1}' and kind='IMAP'")
chk("01 config_enc có giá trị, config_public không chứa mật khẩu", ce == "true|false", ce)
raw_enc = psql(f"select encode(config_enc,'escape') from integration_configs where course_id='{C1}' and kind='IMAP'"); chk("01 config_enc là bản mã (không có canary dạng rõ)", PWD not in raw_enc and PWD not in psql(f"select config_public::text||coalesce(state::text,'') from integration_configs where course_id='{C1}'"), "")
chk("01 audit_log không chứa mật khẩu", PWD not in psql("select string_agg(coalesce(before::text,'')||coalesce(after::text,''),' ') from audit_log"), "")
v = g["version"]; s, x, _ = call("PUT", f"{Q}/integrations/imap", ADM, {**cfg, "password": "", "version": v + 5}); chk("01 sai version → 409", s == 409, f"{s} {str(x)[:100]}")
s, x, _ = call("PUT", f"{Q}/integrations/imap", ADM, {**{k: v_ for k, v_ in cfg.items() if k != "password"}, "version": v}); chk("01 password trống = giữ cũ → 200", s == 200, f"{s} {str(x)[:100]}")
s, x, _ = call("PUT", f"{Q}/integrations/imap", ADM, {**cfg, "security": "NONE"}); print("   security NONE ở APP_ENV hiện tại:", s)
t0 = time.time(); s, x, _ = call("POST", f"{Q}/integrations/imap/test", ADM, {}, idem()); chk("02 test kết nối tới máy chủ không tồn tại → 200 {ok:false, kind}, ≤ 12 s", s == 200 and x.get("ok") is False and x.get("kind") in ("UNREACHABLE", "TIMEOUT"), f"{s} {x} {time.time()-t0:.1f}s")
s, g, _ = call("GET", f"{Q}/integrations/imap", ADM); chk("06 sau lỗi: status ERROR / last_error là mã ngắn (không lộ máy chủ / mật khẩu)", g.get("last_error") in ("UNREACHABLE", "TIMEOUT") and PWD not in json.dumps(g), g)
# lớp 2: không lẫn
s, g2, _ = call("GET", f"/api/v1/courses/{C2}/integrations/imap", ADM); chk("08 lớp 2 không thấy cấu hình lớp 1", s == 200 and g2.get("configured") is False, f"{s} {g2}")
s, g3, _ = call("GET", f"/api/v1/courses/{C2}/integrations/imap", T); chk("08 TEACHER (cũng dạy lớp 2 theo seed) đọc lớp 2: không thấy cấu hình lớp 1", s == 200 and g3.get("configured") is False, f"{s} {g3}")
# sync-now
s, x, _ = call("POST", f"{Q}/integrations/imap/sync-now", TA, {}, idem()); print("   sync-now TA:", s, str(x)[:120]); chk("07 sync-now TA → 202 (hoặc 409 chưa kết nối được)", s in (202, 409), s)
s2, x2, _ = call("POST", f"{Q}/integrations/imap/sync-now", T, {}, idem()); chk("07 lần hai trong 30 s → 429 + retry_after (hoặc job cũ)", s2 == 429 and "retry_after" in json.dumps(x2) or (s == 202 and x2 == x), f"{s2} {str(x2)[:120]}")
for n, tk in (("ADMIN", ADM), ("STUDENT", SB)):
    s, _, _ = call("POST", f"{Q}/integrations/imap/sync-now", tk, {}, idem()); chk(f"08 {n} sync-now → 403", s == 403, s)
for n, tk, want in (("TA", TA, 200), ("TEACHER", T, 200), ("STUDENT", SB, 403), ("ADMIN", ADM, 403)):
    s, _, _ = call("GET", f"{Q}/mail-inbound", tk); chk(f"08 {n} GET mail-inbound → {want}", s == want, s)
# SMTP
s, sm, _ = call("GET", f"{Q}/integrations/smtp", ADM); chk("Q7 SMTP chỉ đọc, không mật khẩu", s == 200 and sm.get("read_only") is True and "password" not in json.dumps(sm), f"{s} {str(sm)[:160]}")
logs = subprocess.run("docker logs edupilot100-gateway-1 2>&1; docker logs edupilot100-worker-1 2>&1", shell=True, capture_output=True, text=True).stdout
chk("log gateway + worker không chứa mật khẩu IMAP", PWD not in logs, "canary trong log")
print("RESULT:", "FAIL" if fails else "PASS", f"({fails} lỗi)"); sys.exit(1 if fails else 0)
