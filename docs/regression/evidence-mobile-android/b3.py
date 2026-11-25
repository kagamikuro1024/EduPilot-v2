import m, json, time
out={}
# CL-14: mã sai → chữ lỗi
m.cdp("(()=>{const i=document.querySelector('input'); const s=Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value').set; s.call(i,'ZZZ999'); i.dispatchEvent(new Event('input',{bubbles:true})); const b=[...document.querySelectorAll('button')].find(x=>x.innerText.trim()==='Điểm danh'&&!x.disabled); b&&b.click(); return 1})()")
time.sleep(4)
out["err"]=m.cdp("document.body.innerText.slice(0,260).replace(/\\n/g,' / ')")
m.shot("CL-14-error")
# LIF-03: khoá màn khi đang gõ dở
m.cdp("(()=>{const i=document.querySelector('input'); if(i){i.focus(); const s=Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value').set; s.call(i,'ABC12'); i.dispatchEvent(new Event('input',{bubbles:true}));} return 1})()")
time.sleep(0.6); m.sh("input keyevent 26"); time.sleep(3); m.sh("input keyevent 26"); time.sleep(2); m.sh("input swipe 540 1800 540 600 250"); time.sleep(2)
out["lif03_val"]=m.cdp("(()=>{const i=document.querySelector('input'); return i?i.value:'noinput'})()")
m.shot("LIF-03-lock-unlock")
# LIF-05: am kill khi ở nền
m.sh("input keyevent KEYCODE_HOME"); time.sleep(1); m.sh("am kill vn.edupilot.app"); time.sleep(1)
m.sh("am start -n vn.edupilot.app/.MainActivity >/dev/null"); time.sleep(5)
out["lif05"]=m.cdp("location.pathname+' | '+document.body.innerText.slice(0,60).replace(/\\n/g,' / ')")
m.shot("LIF-05-after-kill")
# SAF-04: ngang + cutout tall
m.sh("cmd overlay enable com.android.internal.display.cutout.emulation.tall"); m.sh("settings put system accelerometer_rotation 0"); m.sh("settings put system user_rotation 1"); time.sleep(2)
m.nav("/checkin",2200)
out["saf04"]=m.js('const p=await connect(); console.log(await p.ev(' + json.dumps(m.GEO) + ')); p.close();', timeout=60)
m.shot("SAF-04-landscape-cutout")
m.sh("settings put system user_rotation 0"); m.sh("cmd overlay disable com.android.internal.display.cutout.emulation.tall")
print(json.dumps(out,ensure_ascii=False))
