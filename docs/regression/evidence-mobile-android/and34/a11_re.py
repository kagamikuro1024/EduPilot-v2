# DEF-MOB-15 nghiệm lại (VPS 96680658): A11-06/07/08 bằng DOM/ARIA (MutationObserver) + đếm lượt TTS TalkBack (âm thanh emulator tắt).
from a11_live import *
OBS = """(()=>{window.__r=[];const t0=performance.now();const reg=(n)=>{let e=n.nodeType===3?n.parentElement:n;while(e&&e!==document.body){const r=e.getAttribute&&e.getAttribute('role');const l=e.getAttribute&&e.getAttribute('aria-live');if(r==='alert'||r==='status'||l){return {k:(r==='alert'||r==='status')?'role='+r:'aria-live='+l,nestedLive:!!(e.parentElement&&e.parentElement.closest('[aria-live]')),txt:(e.innerText||'').trim().replace(/\\n/g,' ').slice(0,80)}}e=e.parentElement}return null};
const push=(h)=>{if(!h||!h.txt)return;const l=window.__r[window.__r.length-1];if(!l||l.k!==h.k||l.txt!==h.txt)window.__r.push(Object.assign({ms:Math.round(performance.now()-t0)},h))};
new MutationObserver(ms=>{for(const mu of ms){push(reg(mu.target));for(const a of mu.addedNodes)push(reg(a))}}).observe(document.body,{subtree:true,childList:true,characterData:true});return 'obs'})()"""
def snap(tag, n0):
    r = json.loads(m.cdp("JSON.stringify(window.__r)") or "[]")
    print(tag, "| TTS +", int(tts()) - n0, "| thong bao:", json.dumps(r, ensure_ascii=False)[:700], flush=True); return r
L.memguard(); L.blank_poster()
m.login("sv.qc2"); time.sleep(3)
tb(True)
w = L.newwin(L.CID["QCMOB2"]); L.home(); L.to_checkin(); time.sleep(2); L.adb("logcat -c")
# 07: mã sai
m.cdp(OBS); L.click("Nhập mã"); time.sleep(1); L.type_code("AAA111"); time.sleep(0.6); n0 = int(tts()); L.click("Điểm danh"); time.sleep(5)
r7 = snap("A11-07 ma sai", n0)
dom7 = m.cdp("JSON.stringify({alerts:[...document.querySelectorAll('[role=alert]')].map(e=>({txt:(e.innerText||'').trim().slice(0,60),nestedLive:!!(e.parentElement&&e.parentElement.closest('[aria-live]'))})),status:[...document.querySelectorAll('[role=status]')].length,live:[...document.querySelectorAll('[aria-live]')].map(e=>e.getAttribute('aria-live')+':'+(e.dataset.part||e.className.slice(0,20)))})"); print("DOM sau loi:", dom7, flush=True); L.shot("A11-07-re")
# 06: mã đúng
d = L.display(w); L.type_code(d["short_code"]); time.sleep(0.6); m.cdp("window.__r.length=0"); n0 = int(tts()); L.click("Điểm danh"); t0 = time.time()
while time.time() - t0 < 12 and not L.rows(w): time.sleep(0.2)
time.sleep(2); r6 = snap("A11-06 ma dung", n0); L.shot("A11-06-re")
# 08: 429
m.cdp("window.__r.length=0"); L.home(); L.to_checkin(); time.sleep(2)
for i in range(8):
    if m.cdp("document.querySelector('input')?'y':'n'") == 'n': L.click("Nhập mã")
    time.sleep(0.8); L.type_code("AAA111"); time.sleep(0.5); L.click("Điểm danh"); time.sleep(2.2)
    st = m.cdp("document.querySelector('[data-part=checkin]').dataset.state"); print(i + 1, "state", st, flush=True)
    if st == "limited": break
if st == "limited":
    m.cdp("window.__r.length=0"); L.adb("logcat -c"); n0 = int(tts()); t0 = time.time(); last = 0; ev = []
    while time.time() - t0 < 60:
        time.sleep(1); n = int(tts()) - n0
        if n != last: ev.append((round(time.time() - t0), n)); last = n
        if m.cdp("document.querySelector('[data-part=checkin]').dataset.state") != "limited": break
    time.sleep(2); r8 = snap("A11-08 429", n0); print("TTS theo giay:", ev); print("so lan doi noi dung vung thong bao:", len(r8)); L.shot("A11-08-re")
    json.dump(dict(r6=r6, r7=r7, r8=r8, dom7=dom7), open(L.WORKD + "/a11_re.json", "w"), ensure_ascii=False)
tb(False)
