from a11_live import *
L.home(); L.to_checkin(); time.sleep(3); tb(True); time.sleep(3); L.adb("logcat -c"); n0 = int(tts()); ev = []; last = 0; t0 = time.time()
while time.time() - t0 < 25:
    time.sleep(1); n = int(tts()) - n0
    if n != last: ev.append((round(time.time() - t0), n)); last = n
print("NEN (khong lam gi, 25 s): TTS +", int(tts()) - n0, ev)
tb(False)
