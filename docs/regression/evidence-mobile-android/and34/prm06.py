import lib34 as L, m, time
L.memguard(); L.blank_poster()
L.adb("shell pm revoke vn.edupilot.app android.permission.CAMERA"); time.sleep(2)   # thu hoi quyen giet tien trinh app
L.adb("shell am start -W -n vn.edupilot.app/.MainActivity"); time.sleep(5); m.login("sv.qc2"); time.sleep(3)
L.to_checkin(); time.sleep(2)
L.click("Quét mã điểm danh")
for _ in range(15):
    if "GrantPermissionsActivity" in L.adb("shell dumpsys window | grep -m1 GrantPermissionsActivity"): break
    time.sleep(0.5)
print("hop quyen camera hien:", "GrantPermissionsActivity" in L.adb("shell dumpsys window | grep -m1 GrantPermissionsActivity")); L.shot("PRM-06-dialog")
print("chon Only this time:", L.tap_text("Only this time")); time.sleep(3)
v = m.cdp("(()=>{const v=document.querySelector('video'); return v? JSON.stringify({paused:v.paused,ready:v.readyState,w:v.videoWidth}):'novideo'})()"); print("quet lan 1, video:", v)
print("CAMERA:", L.adb("shell dumpsys package vn.edupilot.app | grep -E 'CAMERA: granted'")[:120])
L.click("Thoát"); time.sleep(1)
L.adb("shell input keyevent KEYCODE_HOME"); print("de nen 80 s...", flush=True); time.sleep(80)
print("CAMERA sau nen:", L.adb("shell dumpsys package vn.edupilot.app | grep -E 'CAMERA: granted'")[:120])
L.adb("shell am start -n vn.edupilot.app/.MainActivity"); time.sleep(3)
print("quay lai: url", m.cdp("location.pathname"))
t0 = time.time(); r = L.click("Quét mã điểm danh"); print("quet lan 2:", r)
state = None
for _ in range(20):
    if "GrantPermissionsActivity" in L.adb("shell dumpsys window | grep -m1 GrantPermissionsActivity"): state = "hop quyen hien lai"; break
    v = m.cdp("(()=>{const v=document.querySelector('video'); return !!(v&&v.readyState>=2&&v.videoWidth>0)})()")
    if v: state = "camera mo (khong hoi lai)"; break
    if m.cdp("!!document.querySelector('input')"): state = "o nhap ma"; break
    time.sleep(0.5)
print("ket qua lan 2:", state, round(time.time()-t0, 1), "s |", L.txt()[100:200]); L.shot("PRM-06-lan2")
print("crash/ANR:", L.adb("logcat -d | grep -ciE 'FATAL EXCEPTION|ANR in'"))
