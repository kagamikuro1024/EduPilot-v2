import m, json, time
out={}
m.nav("/checkin",2200)
out["no_geo_line"]=m.cdp("!document.body.innerText.includes('kiểm tra vị trí')")
m.cdp("(()=>{const i=document.querySelector('input'); const s=Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value').set; s.call(i,'ZZZ999'); i.dispatchEvent(new Event('input',{bubbles:true})); const b=[...document.querySelectorAll('button')].find(x=>x.innerText.trim()==='Điểm danh'&&!x.disabled); b&&b.click(); return 1})()")
time.sleep(4)
out["err"]=m.cdp("document.body.innerText.slice(0,240).replace(/\\n/g,' / ')")
m.shot("CL-14-error-qcmob1")
# LIF-03 lại
m.cdp("(()=>{const i=document.querySelector('input'); if(!i) return 'noinput'; i.focus(); const s=Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value').set; s.call(i,'ABC12'); i.dispatchEvent(new Event('input',{bubbles:true})); return i.value})()")
time.sleep(0.8); out["before_lock"]=m.cdp("(document.querySelector('input')||{}).value")
m.sh("input keyevent 26"); time.sleep(3); m.sh("input keyevent 26"); time.sleep(1); m.sh("input swipe 540 1800 540 600 250"); time.sleep(2)
out["after_unlock"]=m.cdp("(document.querySelector('input')||{}).value")
m.shot("LIF-03-unlock")
print(json.dumps(out,ensure_ascii=False))
