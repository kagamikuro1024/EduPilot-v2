import lib34 as L, m, time, json
L.memguard(); L.blank_poster()
L.home(); L.to_checkin(); time.sleep(2)
js = """(()=>{const e=document.activeElement; if(!e||e===document.body) return 'body'; const r=e.getBoundingClientRect(); const cs=getComputedStyle(e); return JSON.stringify({tag:e.tagName, txt:(e.innerText||e.getAttribute('aria-label')||e.placeholder||'').trim().slice(0,28), top:Math.round(r.top), left:Math.round(r.left), outline:cs.outlineStyle+' '+cs.outlineWidth+' '+cs.outlineColor, shadow:cs.boxShadow.slice(0,40), vis:r.width>0&&r.height>0})})()"""
L.adb("shell input keyevent KEYCODE_MOVE_HOME"); L.adb("shell input keyevent KEYCODE_TAB"); time.sleep(0.4)
seq = []
for i in range(14):
    d = m.cdp(js); seq.append(d); print(i, d, flush=True)
    L.adb("shell input keyevent KEYCODE_TAB"); time.sleep(0.4)
json.dump(seq, open(L.WORKD + "/a11_13.json", "w"), ensure_ascii=False)
L.shot("A11-13-focus")
