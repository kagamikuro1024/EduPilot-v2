# A11-08 kịch bản sát người dùng TalkBack: gõ mã + ẩn bàn phím khi TalkBack TẮT; bật TalkBack; chạm 1 lần để focus nút «Điểm danh», chờ yên; chạm đúp để kích hoạt (429 đã gây bằng API).
from a11_live import *
import api_sec as A, re, subprocess
def center(text):
    L.adb("shell uiautomator dump /sdcard/ui.xml >/dev/null 2>&1"); x = L.adb("exec-out cat /sdcard/ui.xml")
    for mm in re.finditer(r'<node[^>]*?(?:text|content-desc)="%s"[^>]*?bounds="\[(\d+),(\d+)\]\[(\d+),(\d+)\]"' % re.escape(text), x):
        a, b, c, d = map(int, mm.groups()); return (a + c) // 2, (b + d) // 2
L.memguard(); m.login("sv.qc2"); time.sleep(3); L.home(); L.to_checkin(); time.sleep(2)
sv = A.login("sv.qc2@edupilot.local", m.acct("sv.qc2")); cid = L.CID["QCMOB2"]
print("429 qua API:", [A.req("POST", f"/courses/{cid}/checkins", {"code": "AAA111"}, sv)[0] for _ in range(6)], flush=True)
L.click("Nhập mã"); time.sleep(1.2); L.type_code("AAA111"); time.sleep(0.8)
L.adb("shell input keyevent KEYCODE_BACK"); time.sleep(1.5)                       # ẩn bàn phím (TalkBack tắt)
xy = center("Điểm danh"); print("nut Điểm danh o", xy, flush=True)
tb(False); time.sleep(1); tb(True); time.sleep(10)
L.adb("shell input tap %d %d" % xy); time.sleep(8)                                   # chạm 1 lần: focus nút (không kích hoạt)
L.adb("logcat -c"); L.adb("shell log -t QCMARK click")
L.adb("shell input tap %d %d" % xy); time.sleep(0.12); L.adb("shell input tap %d %d" % xy)   # chạm đúp = kích hoạt
t0 = time.time(); lim = None
while time.time() - t0 < 70:
    time.sleep(1)
    st = m.cdp("document.querySelector('[data-part=checkin]')?document.querySelector('[data-part=checkin]').dataset.state:'?'")
    if st == "limited" and lim is None: lim = round(time.time() - t0)
    if lim is not None and st != "limited": break
time.sleep(5); print("limited tu giay", lim, "| het gio o giay", round(time.time() - t0))
open(L.WORKD + "/tb-real-raw.log", "w").write(subprocess.run("adb logcat -d -v threadtime", shell=True, capture_output=True, text=True).stdout)
tb(False)
