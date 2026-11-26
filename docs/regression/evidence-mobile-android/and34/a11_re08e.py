# A11-08 đo chuẩn (gõ mã khi TalkBack tắt, bật TalkBack rồi mới bấm): 5-6 lần sai qua API (tới 429) (im lặng) -> bật TalkBack -> 1 lần nhập sai ở UI (429) -> đếm TTS theo giây suốt đếm ngược tới hết giờ.
from a11_live import *
import api_sec as A
OBS = """(()=>{window.__r=[];const t0=performance.now();const reg=(n)=>{let e=n.nodeType===3?n.parentElement:n;while(e&&e!==document.body){const r=e.getAttribute&&e.getAttribute('role');const l=e.getAttribute&&e.getAttribute('aria-live');if(r==='alert'||r==='status'||l)return {k:(r==='alert'||r==='status')?'role='+r:'aria-live='+l,txt:(e.innerText||'').trim().replace(/\\n/g,' ').slice(0,80)};e=e.parentElement}return null};
new MutationObserver(ms=>{for(const mu of ms){for(const n of [mu.target,...mu.addedNodes]){const h=reg(n);if(h&&h.txt){const l=window.__r[window.__r.length-1];if(!l||l.k!==h.k||l.txt!==h.txt)window.__r.push(Object.assign({s:Math.round((performance.now()-t0)/1000)},h))}}}}).observe(document.body,{subtree:true,childList:true,characterData:true});return 'obs'})()"""
L.memguard(); m.login("sv.qc2"); time.sleep(3); L.home(); L.to_checkin(); time.sleep(2)
sv = A.login("sv.qc2@edupilot.local", m.acct("sv.qc2")); cid = L.CID["QCMOB2"]
for i in range(6): print("API sai", i + 1, A.req("POST", f"/courses/{cid}/checkins", {"code": "AAA111"}, sv)[0], flush=True)
L.click("Nhập mã"); time.sleep(1.2); L.type_code("AAA111"); time.sleep(0.8)   # gõ khi TalkBack TẮT (tránh đọc lại từng ký tự gõ)
tb(True); time.sleep(8); m.cdp(OBS)
L.adb("logcat -c"); n0 = int(tts()); t0 = time.time(); L.click("Điểm danh"); ev = []; last = 0; lim_t = None
while time.time() - t0 < 60:
    time.sleep(1); n = int(tts()) - n0
    if n != last: ev.append((round(time.time() - t0), n)); last = n
    st = m.cdp("document.querySelector('[data-part=checkin]')?document.querySelector('[data-part=checkin]').dataset.state:'?'")
    if st == "limited" and lim_t is None: lim_t = round(time.time() - t0)
    if lim_t is not None and st != "limited": break
time.sleep(3); n = int(tts()) - n0
r = json.loads(m.cdp("JSON.stringify(window.__r)")); print("bat dau limited o giay", lim_t, "| het gio o giay", round(time.time() - t0))
print("vung thong bao (s so voi luc bam):", json.dumps(r, ensure_ascii=False)); print("so thong bao:", len(r))
print("TTS tong +", n, "| theo giay (giay, cong don):", ev)
print("10 s dau sau limited:", [e for e in ev if lim_t is not None and lim_t <= e[0] <= lim_t + 10])
json.dump(dict(r=r, tts=ev, lim=lim_t), open(L.WORKD + "/a11_re08e.json", "w"), ensure_ascii=False)
tb(False)
