# CHẨN ĐOÁN (không phải verdict): khi đếm ngược 429, thử aria-hidden lên <p data-part=limited> rồi đếm TTS 10 s.
from a11_live import *
L.memguard(); m.login("sv.qc2"); time.sleep(3); L.home(); L.to_checkin(); time.sleep(2); tb(True)
for i in range(8):
    if m.cdp("document.querySelector('input')?'y':'n'") == 'n': L.click("Nhập mã")
    time.sleep(0.8); L.type_code("AAA111"); time.sleep(0.5); L.click("Điểm danh"); time.sleep(2.2)
    st = m.cdp("document.querySelector('[data-part=checkin]').dataset.state"); print(i + 1, st, flush=True)
    if st == "limited": break
print("HTML:", m.cdp("document.querySelector('[data-part=checkin]').outerHTML.slice(0,900)"))
print("aria-hidden dat tren p.limited:", m.cdp("(()=>{const p=document.querySelector('[data-part=limited]'); if(!p) return 'khong co'; p.setAttribute('aria-hidden','true'); return p.outerHTML.slice(0,200)})()"))
L.adb("logcat -c"); n0 = int(tts()); ev = []; last = 0; t0 = time.time()
while time.time() - t0 < 14:
    time.sleep(1); n = int(tts()) - n0
    if n != last: ev.append((round(time.time() - t0), n)); last = n
print("SAU aria-hidden: TTS +", int(tts()) - n0, "trong 14 s:", ev)
tb(False)
