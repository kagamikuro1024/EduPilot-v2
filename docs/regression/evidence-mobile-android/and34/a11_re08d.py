# A11-08 đo lại, loại nhiễu: TalkBack TẮT khi nhập sai; chỉ bật SAU khi đã vào «limited», chờ 6 s ổn định rồi đếm TTS 12 s.
from a11_live import *
L.memguard(); m.login("sv.qc2"); time.sleep(3); L.home(); L.to_checkin(); time.sleep(2)
for i in range(8):
    if m.cdp("document.querySelector('input')?'y':'n'") == 'n': L.click("Nhập mã")
    time.sleep(0.8); L.type_code("AAA111"); time.sleep(0.5); L.click("Điểm danh"); time.sleep(2.2)
    st = m.cdp("document.querySelector('[data-part=checkin]').dataset.state"); print(i + 1, st, flush=True)
    if st == "limited": break
left0 = m.cdp("document.querySelector('[data-part=limited-live]')?document.querySelector('[data-part=limited-live]').innerText:'?'")
tb(True); time.sleep(6)
L.adb("logcat -c"); n0 = int(tts()); t0 = time.time(); ev = []; last = 0
while time.time() - t0 < 12:
    time.sleep(1); n = int(tts()) - n0
    if n != last: ev.append((round(time.time() - t0), n)); last = n
print("bat TalkBack khi con:", left0, "| sau 6 s on dinh, dem 12 s: TTS +", int(tts()) - n0, ev)
tb(False)
