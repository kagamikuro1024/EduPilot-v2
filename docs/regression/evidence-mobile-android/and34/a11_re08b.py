from a11_live import *
L.memguard(); L.home(); L.to_checkin(); time.sleep(2); tb(True)
for i in range(8):
    if m.cdp("document.querySelector('input')?'y':'n'") == 'n': L.click("Nhập mã")
    time.sleep(0.8); L.type_code("AAA111"); time.sleep(0.5); L.click("Điểm danh"); time.sleep(2.2)
    st = m.cdp("document.querySelector('[data-part=checkin]').dataset.state"); print(i + 1, "state", st, flush=True)
    if st == "limited": break
L.adb("logcat -c"); n0 = int(tts()); t0 = time.time(); ev = []; last = 0
while time.time() - t0 < 36:          # KHÔNG gọi CDP trong lúc đếm
    time.sleep(1); n = int(tts()) - n0
    if n != last: ev.append((round(time.time() - t0), n)); last = n
print("khong CDP trong luc dem: TTS +", int(tts()) - n0, "| 10 s dau:", sum(1 for e in ev if e[0] <= 10 and True) and [e for e in ev if e[0] <= 10][-1][1], "|", ev)
tb(False)
