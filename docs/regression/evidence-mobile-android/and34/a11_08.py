from a11_live import *
L.memguard(); L.home(); L.to_checkin(); time.sleep(2)
OBS2 = """(()=>{window.__lim=[];const t0=performance.now();new MutationObserver(()=>{const e=document.querySelector('[data-part=checkin]');if(!e)return;const p=e.querySelector('[data-part=limited]');if(p){const tx=p.innerText.trim();const l=window.__lim[window.__lim.length-1];if(!l||l.txt!==tx)window.__lim.push({ms:Math.round(performance.now()-t0),txt:tx})}}).observe(document.body,{subtree:true,childList:true,characterData:true});return 'ok'})()"""
tb(True); m.cdp(OBS2)
state = None
for i in range(8):
    L.click("Nhập mã") if m.cdp("document.querySelector('input')?'n':'y'") == 'y' else None
    time.sleep(0.8); L.type_code("AAA111"); time.sleep(0.5); L.click("Điểm danh"); time.sleep(2.2)
    state = m.cdp("document.querySelector('[data-part=checkin]').dataset.state"); print(i + 1, "state", state, flush=True)
    if state == "limited": break
if state == "limited":
    L.adb("logcat -c"); n0 = int(tts()); t0 = time.time(); last = 0; ev = []
    while time.time() - t0 < 50:
        time.sleep(1); n = int(tts()) - n0
        st = m.cdp("document.querySelector('[data-part=checkin]').dataset.state")
        if n != last: ev.append((round(time.time() - t0), "TTS", n)); last = n
        if st != "limited": ev.append((round(time.time() - t0), "state", st)); break
        if 9.5 < time.time() - t0 < 10.8: print("mốc 10 s: TTS +", n, flush=True)
    lim = json.loads(m.cdp("JSON.stringify(window.__lim)")); print("su kien (giay, loai, gia tri):", ev)
    print("lan doi noi dung vung thong bao:", len(lim), "| dau/cuoi:", lim[:1], lim[-1:]); L.shot("A11-08-limited")
tb(False)
