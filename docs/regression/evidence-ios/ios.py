#!/usr/bin/env python3
"""Helper chạy pha 3 (iOS Simulator, Safari + PWA) cho bộ TC mobile 10.8.

Chỉ dùng công cụ có sẵn trên máy: xcrun simctl, osascript (System Events), tesseract.
Không in bí mật: mật khẩu đọc từ ~/.config/edupilot-team/secrets/seed-password.txt
(dùng qua biến môi trường trong lệnh gọi, không echo).

Ví dụ:
  python3 ios.py create qc3-se3 "iPhone SE (3rd generation)"
  python3 ios.py boot qc3-iphone17
  python3 ios.py open "https://<vps>/checkin?course=<uuid>"
  python3 ios.py shot 01-safari-checkin
  python3 ios.py ocr docs/.../shots/01-safari-checkin.png
  python3 ios.py ui dark
  python3 ios.py size accessibility-extra-extra-extra-large
  python3 ios.py perm grant location
  python3 ios.py tap 0.5 0.72
  python3 ios.py type "ABC 123"
  python3 ios.py batch1 <vps> <course-uuid>
"""
import json
import os
import re
import subprocess
import sys
import time
from pathlib import Path

HERE = Path(__file__).resolve().parent
SHOTS = HERE / "shots"
SHOTS.mkdir(exist_ok=True)
SIM = os.environ.get("SIM_NAME", "qc2-iphone17")


def run(cmd, check=True):
    p = subprocess.run(cmd, capture_output=True, text=True)
    if check and p.returncode != 0:
        raise SystemExit(f"FAIL {cmd[0]}… rc={p.returncode}\n{p.stderr[-800:]}")
    return p.stdout


def udid(name=SIM):
    for line in run(["xcrun", "simctl", "list", "devices", "available"]).splitlines():
        m = re.search(r"^\s+(.+?) \(([0-9A-F-]{36})\) \((Booted|Shutdown)\)", line)
        if m and m.group(1).strip() == name:
            return m.group(2)
    raise SystemExit(f"không thấy thiết bị {name}")


def booted():
    out = run(["xcrun", "simctl", "list", "devices", "booted"], check=False)
    m = re.search(r"\(([0-9A-F-]{36})\) \(Booted\)", out)
    return m.group(1) if m else None


def win_bounds():
    """Khung cửa sổ Simulator (điểm màn hình) để quy đổi toạ độ tap theo tỉ lệ."""
    scr = f'''
tell application "System Events" to tell process "Simulator"
  set w to window 1
  set p to position of w
  set s to size of w
  return (item 1 of p) & "," & (item 2 of p) & "," & (item 1 of s) & "," & (item 2 of s)
end tell'''
    out = run(["osascript", "-e", scr]).strip()
    x, y, w, h = (int(float(v)) for v in out.split(","))
    return x, y, w, h


def tap(fx, fy):
    x, y, w, h = win_bounds()
    # bỏ viền tiêu đề ~ 28 px và thanh công cụ của Simulator
    top = y + 28
    px, py = int(x + fx * w), int(top + fy * (h - 28))
    run(["osascript", "-e", f'tell application "System Events" to click at {{{px}, {py}}}'])
    return px, py


def shot(name):
    f = SHOTS / (name if name.endswith(".png") else name + ".png")
    run(["xcrun", "simctl", "io", "booted", "screenshot", str(f)])
    return f


def ocr(path):
    """OCR tiếng Việt: Vision (ocr.swift, dựng sẵn); tesseract trên máy KHÔNG có gói `vie`."""
    bin_path = HERE / "ocr"
    if not bin_path.exists():
        run(["swiftc", "-O", str(HERE / "ocr.swift"), "-o", str(bin_path)])
    p = subprocess.run([str(bin_path), str(path)], capture_output=True, text=True)
    if p.returncode == 0:
        return re.sub(r"\n{2,}", "\n", p.stdout).strip()
    p = subprocess.run(["tesseract", str(path), "-", "-l", "eng"], capture_output=True, text=True)
    return re.sub(r"\n{2,}", "\n", p.stdout).strip()


def main(argv):
    if not argv:
        print(__doc__)
        return 0
    cmd, *a = argv
    if cmd == "create":
        name, dtype = a[0], a[1]
        out = run(["xcrun", "simctl", "create", name, dtype], check=False)
        print(out.strip() or "đã có")
    elif cmd in ("boot", "up"):
        name = a[0] if a else SIM
        run(["xcrun", "simctl", "boot", name], check=False)
        run(["open", "-a", "Simulator"])
        run(["xcrun", "simctl", "bootstatus", name, "-b"], check=False)
        run(["xcrun", "simctl", "status_bar", "booted", "override",
             "--time", "9:41", "--batteryLevel", "100", "--batteryState", "charged"], check=False)
        print(name, udid(name))
    elif cmd == "shutdown":
        run(["xcrun", "simctl", "shutdown", a[0] if a else "all"], check=False)
    elif cmd == "open":
        run(["xcrun", "simctl", "openurl", "booted", a[0]])
    elif cmd == "shot":
        print(shot(a[0]))
    elif cmd == "ocr":
        print(ocr(a[0]))
    elif cmd == "ui":
        run(["xcrun", "simctl", "ui", "booted", "appearance", a[0]])
    elif cmd == "size":
        run(["xcrun", "simctl", "ui", "booted", "content_size", a[0]])
    elif cmd == "loc":
        run(["xcrun", "simctl", "location", "booted", "set", a[0], a[1]])
    elif cmd == "perm":
        # perm grant|revoke|reset <service>
        run(["xcrun", "simctl", "privacy", "booted", a[0], a[1]])
    elif cmd == "tap":
        print(tap(float(a[0]), float(a[1])))
    elif cmd == "type":
        run(["osascript", "-e", 'tell application "System Events" to keystroke "%s"' % a[0].replace('"', '\\"')])
    elif cmd == "batch1":
        # ca không cần tap: chỉ openurl + screenshot + OCR
        base, course = a[0], a[1]
        urls = [("01-root", f"{base}/"), ("02-checkin-safari", f"{base}/checkin?course={course}"),
                ("03-today", f"{base}/"), ("04-manifest", f"{base}/manifest.webmanifest")]
        res = []
        for name, u in urls:
            run(["xcrun", "simctl", "openurl", "booted", u])
            time.sleep(3)
            f = shot(name)
            res.append({"name": name, "url": u, "file": str(f.relative_to(HERE))})
        print(json.dumps(res, ensure_ascii=False, indent=1))
    else:
        raise SystemExit("lệnh lạ: " + cmd)
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
