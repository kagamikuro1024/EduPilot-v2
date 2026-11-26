from a11_live import *
L.memguard(); L.blank_poster(); L.home(); L.to_checkin(); time.sleep(2)
tb(True); time.sleep(2)
print("quet:", L.open_scan(), "| video:", m.cdp("document.querySelector('video')?'co':'khong'"))
# Thoát ở (540, 1639) trên màn 1080x2400 (đo từ ảnh trước: 353,1072 trên ảnh 706 rộng ×1.53)
L.adb("logcat -c"); n0 = int(tts())
L.adb("shell input tap 540 1000"); time.sleep(2); L.shot("A11-05-focus-khung"); print("chạm khung camera -> TTS +", int(tts()) - n0)
L.adb("shell input tap 540 1640"); time.sleep(2); L.shot("A11-05-focus-thoat"); print("chạm Thoát (đưa focus) -> TTS +", int(tts()) - n0)
L.adb("shell input tap 540 1640"); time.sleep(0.15); L.adb("shell input tap 540 1640"); time.sleep(2)   # hai lần chạm nhanh = kích hoạt của TalkBack
print("sau hai lần chạm: video =", m.cdp("document.querySelector('video')?'con':'het'"), "| TTS +", int(tts()) - n0)
print("camera: ", L.adb("shell dumpsys media.camera | grep -E '(CONNECT|DISCONNECT) device .* vn.edupilot.app' | head -1"))
tb(False)
