#!/usr/bin/env python3
"""Đăng nhập EduPilot bằng tài khoản SEED (dữ liệu giả) và ghi «phiên đã lưu» cho eval-insights.

Lý do: agent jev bỏ ô mật khẩu khỏi action space (snapshot.js: type password không bao giờ lộ ra), nên persona
không tự đăng nhập được. eval-insights dựng phiên bằng một container Chrome có VNC (cần Docker image riêng);
script này thay bước đó: lấy cookie làm mới `ep_rt` qua API rồi ghi đúng dạng tệp của auth_sessions.save_login
vào data/auth/<workspace>/<host>.json của bản clone (thư mục dữ liệu chạy, git bỏ qua). Không in cookie.

  python3 seed_session.py <email> <workspace_id> [site=http://ep.localhost:5800] [clone=~/Documents/eval-insights-ro]
Mật khẩu: SEED_DEFAULT_PASSWORD (mặc định mật khẩu seed). Đăng nhập gọi thẳng LOGIN_BASE (mặc định http://localhost:580);
cookie ghi cho host của `site` (trang persona mở, qua host_proxy.py).
"""
import json, os, sys, time, urllib.request
from datetime import datetime, timezone
from http.cookies import SimpleCookie
from pathlib import Path
from urllib.parse import urlparse

email, ws = sys.argv[1], sys.argv[2]
site = (sys.argv[3] if len(sys.argv) > 3 else "http://ep.localhost:5800").rstrip("/")
login = os.environ.get("LOGIN_BASE", "http://localhost:580").rstrip("/")
clone = Path(sys.argv[4] if len(sys.argv) > 4 else "~/Documents/eval-insights-ro").expanduser()
pw = os.environ.get("SEED_DEFAULT_PASSWORD", "Edupilot#Seed-2026")
host = urlparse(site).hostname

req = urllib.request.Request(f"{login}/api/v1/auth/login", json.dumps({"email": email, "password": pw}).encode(),
                             {"Content-Type": "application/json", "Origin": login})
with urllib.request.urlopen(req, timeout=20) as r:
    raw = r.headers.get_all("Set-Cookie") or []
cookies = []
for line in raw:
    c = SimpleCookie()
    c.load(line)
    for name, m in c.items():
        cookies.append({"name": name, "value": m.value, "domain": host, "path": m["path"] or "/",
                        "expires": time.time() + int(m["max-age"] or 3600), "httpOnly": True,
                        "secure": True, "sameSite": "Lax", "session": False})
if not cookies:
    sys.exit("không nhận được cookie")
d = clone / "backend/data/auth" / ws
d.mkdir(parents=True, exist_ok=True, mode=0o700)
p = d / f"{host}.json"
fd = os.open(p, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600)
with os.fdopen(fd, "w") as f:
    json.dump({"host": host, "url": site + "/", "saved_at": datetime.now(timezone.utc).isoformat(),
               "cookies": cookies, "origins": []}, f)
print(f"ok: {email} → {p.name} ({len(cookies)} cookie)")
