from a11_live import *
L.memguard(); m.login("sv.qc2"); time.sleep(3)
tb(True)
# A11-07 mã sai
w = L.newwin(L.CID["QCMOB2"]); L.home(); L.to_checkin(); time.sleep(2); L.adb("logcat -c")
m.cdp(OBS); L.click("Nhập mã"); time.sleep(1); L.type_code("AAA111"); time.sleep(0.6); n0 = int(tts()); L.click("Điểm danh"); time.sleep(5)
dump("A11-07 mã sai", n0); time.sleep(6); print("sau 6 s cuộn/chờ thêm: TTS +", int(tts()) - n0)
# A11-06 mã đúng
d = L.display(w); L.adb("logcat -c"); m.cdp(OBS); L.type_code(d["short_code"]); time.sleep(0.6); n0 = int(tts()); L.click("Điểm danh"); time.sleep(6)
dump("A11-06 mã đúng", n0)
L.shot("A11-06-done")
tb(False)
