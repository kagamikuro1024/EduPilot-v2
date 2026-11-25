# Bộ dò cho phần còn lại pha 2 (Android emulator). Mỗi hàm trả (kq, note, shots) và tự chụp ảnh.
import m, json, time, os, subprocess
JS = lambda body: m.js(body, timeout=90)
GEO = m.GEO

def probe_sizes():
    out = {}
    for name, px, dpi in [("360","1080x2400","480"),("375","1125x2436","480"),("430","1290x2796","480")]:
        m.sh(f"wm size {px}"); m.sh(f"wm density {dpi}"); time.sleep(2); m.nav("/checkin",2000)
        out[name] = JS('const p=await connect(); console.log(await p.ev(' + json.dumps(GEO) + ')); p.close();')
        m.shot(f"CL-1-{name}-v2")
    m.sh("wm size reset"); m.sh("wm density reset")
    return out

def probe_font(scale, tag):
    m.sh(f"settings put system font_scale {scale}"); time.sleep(2); m.nav("/checkin",2200)
    r = JS('const p=await connect(); console.log(await p.ev(' + json.dumps(GEO) + ')); console.log(await p.ev("JSON.stringify({fs:getComputedStyle(document.body).fontSize, sw:document.documentElement.scrollWidth, cw:document.documentElement.clientWidth})")); p.close();')
    m.shot(f"CL-4-font-{tag}"); return r

def probe_state_dom():
    return JS('const p=await connect(); console.log(await p.ev("JSON.stringify({lang:document.documentElement.lang,title:document.title,h1:(document.querySelector(\'h1\')||{}).innerText,aria:[...document.querySelectorAll(\'[aria-live],[role=alert],[aria-describedby]\')].map(e=>e.tagName+\':\'+(e.getAttribute(\'aria-live\')||e.getAttribute(\'role\')||e.getAttribute(\'aria-describedby\'))).join(\',\')})")); p.close();')

def probe_error_dark():
    m.sh("cmd uimode night yes"); m.sh("cmd connectivity airplane-mode enable"); m.sh("am force-stop vn.edupilot.app")
    m.sh("am start -n vn.edupilot.app/.MainActivity >/dev/null"); time.sleep(6)
    m.shot("THM-05-error-dark")
    r = JS('const p=await connect(); console.log(await p.ev("JSON.stringify({bg:getComputedStyle(document.body).backgroundColor,color:getComputedStyle(document.body).color,text:document.body.innerText.slice(0,60)})")); p.close();')
    m.sh("cmd connectivity airplane-mode disable"); m.sh("cmd uimode night no"); time.sleep(4); return r
