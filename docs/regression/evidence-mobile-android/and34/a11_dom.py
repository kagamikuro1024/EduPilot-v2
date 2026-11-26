import lib34 as L, m, time, json
OBS = """(()=>{window.__html=[];const c=()=>document.querySelector('[data-part=checkin]');new MutationObserver(()=>{const e=c();if(!e)return;const st=e.dataset.state;const last=window.__html[window.__html.length-1];if(!last||last.st!==st)window.__html.push({st,html:e.outerHTML.slice(0,700)})}).observe(document.body,{subtree:true,childList:true,attributes:true});return 'ok'})()"""
L.memguard(); w = L.newwin(L.CID["QCMOB2"]); L.home(); L.to_checkin(); time.sleep(2); m.cdp(OBS)
L.click("Nhập mã"); time.sleep(1); L.type_code("AAA111"); time.sleep(0.6); L.click("Điểm danh"); time.sleep(4)
err = m.cdp("JSON.stringify({alerts:document.querySelectorAll('[role=alert]').length, status:document.querySelectorAll('[role=status]').length, liveRegions:[...document.querySelectorAll('[aria-live]')].map(e=>e.getAttribute('aria-live')+':'+e.dataset.part), errHtml:(()=>{const e=[...document.querySelectorAll('[data-part=checkin] *')].find(x=>(x.innerText||'').startsWith('Mã không đúng')&&x.children.length===0);return e?e.outerHTML.slice(0,200):null})()})")
print("LOI:", err)
d = L.display(w); L.type_code(d["short_code"]); time.sleep(0.6); L.click("Điểm danh"); t0 = time.time()
while time.time() - t0 < 10 and not L.rows(w): time.sleep(0.2)
time.sleep(1.5)
h = json.loads(m.cdp("JSON.stringify(window.__html)"))
for x in h: print("state", x["st"], "|", x["html"][:420].replace("\n", " "))
json.dump(h, open(L.WORKD + "/a11_dom.json", "w"), ensure_ascii=False)
