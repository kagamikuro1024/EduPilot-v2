# scan.py <thư-mục-ảnh-tên> : lấy mã QR hiện tại của cửa sổ trong win.json, mở bằng Safari của simulator (như quét QR), chờ, chụp, in dòng điểm danh của sv.kha.
import sys, json, subprocess, time, os
import vps
W = json.load(open(os.path.expanduser("~/.config/edupilot-team/qc2-work/win.json")))
U = open(os.path.join(os.path.dirname(os.path.abspath(__file__)), "udid.txt")).read().strip()
subprocess.run(["xcrun", "simctl", "openurl", U, vps.HOST + "/"], check=True); time.sleep(6)  # tải lại trang khác để mã QR (phần #) được đọc lúc tải
c, d = vps.call("teacher", "GET", f"/courses/{W['cid']}/sessions/{W['sid']}/checkin-windows/{W['wid']}/display"); assert c == 200, (c, d)
url = d["qr_url"].replace("https://localhost", vps.HOST, 1)
subprocess.run(["xcrun", "simctl", "openurl", U, url], check=True)
time.sleep(float(sys.argv[2]) if len(sys.argv) > 2 else 12)
subprocess.run(["xcrun", "simctl", "io", U, "screenshot", sys.argv[1]], check=True, capture_output=True)
j = vps.checkins(W["cid"], W["sid"]); print(json.dumps(j, ensure_ascii=False)[:1500])
