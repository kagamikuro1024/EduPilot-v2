# Gửi PM: a11y dump + accessibility events khi đếm ngược 429 (TalkBack bật, âm thanh tắt)
from a11_live import *
import subprocess, os, sys
MODE = sys.argv[1]  # events | dump
L.memguard(); m.login("sv.qc2"); time.sleep(3); L.home(); L.to_checkin(); time.sleep(2); tb(True)
for i in range(8):
    if m.cdp("document.querySelector('input')?'y':'n'") == 'n': L.click("Nhập mã")
    time.sleep(0.8); L.type_code("AAA111"); time.sleep(0.5); L.click("Điểm danh"); time.sleep(2.2)
    st = m.cdp("document.querySelector('[data-part=checkin]').dataset.state"); print(i + 1, st, flush=True)
    if st == "limited": break
L.adb("logcat -c"); L.adb("shell rm -f /sdcard/ev.txt")
out = L.WORKD + "/a11dump"; os.makedirs(out, exist_ok=True)
if MODE == "events":
    L.adb("shell 'timeout 20 uiautomator events > /sdcard/ev.txt 2>&1'", t=40); subprocess.run(f"adb pull /sdcard/ev.txt '{out}/ev.txt' >/dev/null", shell=True)
else:
    time.sleep(3); L.adb("shell uiautomator dump /sdcard/d1.xml >/dev/null 2>&1")
    time.sleep(5); L.adb("shell uiautomator dump /sdcard/d2.xml >/dev/null 2>&1")
    for f in ("d1.xml", "d2.xml"): subprocess.run(f"adb pull /sdcard/{f} '{out}/{f}' >/dev/null", shell=True)
print("tts trong 20 s:", int(tts()))
tb(False)
