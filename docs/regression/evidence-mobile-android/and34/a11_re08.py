from a11_live import *
OBS = """(()=>{window.__r=[];window.__all={};const t0=performance.now();const reg=(n)=>{let e=n.nodeType===3?n.parentElement:n;while(e&&e!==document.body){const r=e.getAttribute&&e.getAttribute('role');const l=e.getAttribute&&e.getAttribute('aria-live');if(r==='alert'||r==='status'||l)return {k:(r==='alert'||r==='status')?'role='+r:'aria-live='+l,txt:(e.innerText||'').trim().replace(/\\n/g,' ').slice(0,80)};e=e.parentElement}return null};
new MutationObserver(ms=>{for(const mu of ms){const e=mu.target.nodeType===3?mu.target.parentElement:mu.target;const sig=e?(e.tagName+'.'+(e.dataset&&e.dataset.part||'')+'.'+String(e.className).slice(0,24)+' | '+(e.innerText||'').trim().slice(0,28)):'?';if(/giây|Thử lại/.test((e&&e.innerText)||'')){window.__all[sig]=(window.__all[sig]||0)+1}
const h=reg(mu.target);if(h&&h.txt){const l=window.__r[window.__r.length-1];if(!l||l.k!==h.k||l.txt!==h.txt)window.__r.push(Object.assign({ms:Math.round(performance.now()-t0)},h))}}}).observe(document.body,{subtree:true,childList:true,characterData:true,attributes:true,attributeFilter:['aria-label','aria-describedby','title','disabled']});return 'obs'})()"""
L.memguard(); L.home(); L.to_checkin(); time.sleep(2); tb(True)
m.cdp(OBS); st = None
for i in range(8):
    if m.cdp("document.querySelector('input')?'y':'n'") == 'n': L.click("Nhập mã")
    time.sleep(0.8); L.type_code("AAA111"); time.sleep(0.5); L.click("Điểm danh"); time.sleep(2.2)
    st = m.cdp("document.querySelector('[data-part=checkin]').dataset.state"); print(i + 1, "state", st, flush=True)
    if st == "limited": break
L.adb("logcat -c"); n0 = int(tts()); t0 = time.time(); ev = []; last = 0
while time.time() - t0 < 60:
    time.sleep(1); n = int(tts()) - n0
    if n != last: ev.append((round(time.time() - t0), n)); last = n
    if m.cdp("document.querySelector('[data-part=checkin]').dataset.state") != "limited": break
time.sleep(2)
r = json.loads(m.cdp("JSON.stringify(window.__r)")); allm = json.loads(m.cdp("JSON.stringify(window.__all)"))
print("vung thong bao (toan bo, tu luc bat dau nhap sai):", json.dumps(r, ensure_ascii=False)[:600]); print("so lan doi vung thong bao:", len(r))
print("TTS +", int(tts()) - n0, "theo giay:", ev)
print("phan tu co chu «giay/Thu lai» doi noi dung (so lan):", json.dumps(allm, ensure_ascii=False)[:600])
print("primary:", m.cdp("JSON.stringify([...document.querySelectorAll('button')].filter(b=>b.offsetParent&&/Điểm danh/.test(b.innerText||'')).map(b=>({txt:b.innerText.trim().slice(0,40),aria:b.getAttribute('aria-label'),desc:b.getAttribute('aria-describedby'),dis:b.disabled})))"))
json.dump(dict(r=r, all=allm, tts=ev), open(L.WORKD + "/a11_re08.json", "w"), ensure_ascii=False)
tb(False)
