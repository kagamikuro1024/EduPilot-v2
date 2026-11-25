# Thư viện chạy bộ mobile Android (pha 2): adb + CDP vào WebView (qua cdp.mjs của dev) + API VPS. Mật khẩu chỉ đọc từ ~/.config/edupilot-team/secrets/vps-seed.txt, không in.
import os, sys, json, subprocess, time, re, datetime, urllib.parse
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import vps
AH = "/opt/homebrew/share/android-commandlinetools"; ADB = AH + "/platform-tools/adb"
WORK = os.path.expanduser("~/.config/edupilot-team/qc2-work/mob"); SHOTS = WORK + "/shots"; RES = WORK + "/results.jsonl"
EV = os.path.expanduser("~/Documents/as-emu-evidence/work")
PKG = "vn.edupilot.app"
def adb(c, check=False):
    r = subprocess.run(f"{ADB} {c}", shell=True, capture_output=True, text=True); return (r.stdout + r.stderr).strip()
def sh(c): return adb(f"shell {json.dumps(c)}")
def shot(name):
    os.makedirs(SHOTS, exist_ok=True); subprocess.run(f"{ADB} exec-out screencap -p > {SHOTS}/{name}.png", shell=True); return name
def uidump():
    adb("shell uiautomator dump /sdcard/ui.xml >/dev/null"); x = adb("exec-out cat /sdcard/ui.xml"); return x
def nodes(x=None):
    x = x or uidump(); out = []
    for m in re.finditer(r'<node [^>]*?text="([^"]*)"[^>]*?resource-id="([^"]*)"[^>]*?class="([^"]*)"[^>]*?content-desc="([^"]*)"[^>]*?clickable="(\w+)"[^>]*?bounds="\[(\d+),(\d+)\]\[(\d+),(\d+)\]"', x):
        t, rid, cl, cd, ck, x1, y1, x2, y2 = m.groups(); out.append(dict(text=t, id=rid, cls=cl, desc=cd, click=ck == "true", b=(int(x1), int(y1), int(x2), int(y2))))
    return out
def node_js(script):
    open(WORK + "/_t.mjs", "w").write('import { connect, sleep, shot } from "' + EV + '/cdp.mjs";\n' + script)
    r = subprocess.run(f"cd {EV} && SHOTS={SHOTS} node {WORK}/_t.mjs", shell=True, capture_output=True, text=True, timeout=120); return (r.stdout + r.stderr).strip()
def cdp(expr, timeout=120):
    return node_js('const p = await connect(); const v = await p.ev(' + json.dumps(expr) + '); console.log(typeof v === "string" ? v : JSON.stringify(v)); p.close();')
def result(tc, kq, note, shots=()):
    with open(RES, "a") as f: f.write(json.dumps({"tc": tc, "kq": kq, "note": note, "shots": list(shots)}, ensure_ascii=False) + "\n")
    print(tc, kq, note[:160])
def window(code="QCVPSA", minutes=30, session_no=None):
    k = vps.course(code); c, j = vps.call("teacher", "GET", f"/courses/{k['id']}/sessions?limit=100"); assert c == 200
    import datetime; now = datetime.datetime.utcnow().isoformat() + "Z"
    ss = j["items"]
    s = next((x for x in ss if x["session_no"] == session_no), None) if session_no else next((x for x in ss if x["starts_at"] <= now <= x["ends_at"]), ss[0])
    c, w = vps.call("teacher", "POST", f"/courses/{k['id']}/sessions/{s['id']}/checkin-windows", {"minutes": minutes})
    if c == 409: wid = w["details"]["window_id"]
    else: assert c == 201, (c, w); wid = w["id"]
    c, d = vps.call("teacher", "GET", f"/courses/{k['id']}/sessions/{s['id']}/checkin-windows/{wid}/display"); assert c == 200, (c, d)
    return dict(cid=k["id"], sid=s["id"], wid=wid, display=d, code=code)
def goto_checkin(w):
    u = w["display"]["qr_url"].replace("https://localhost", vps.HOST, 1)
    node_js(f'const p = await connect(); await p.goto({json.dumps(u)}); await sleep(2500); console.log(await p.url()); p.close();')
    return u
def app_text(): return cdp("document.body.innerText.slice(0,400).replace(/\\n/g,' / ')")
def close_window(w): return vps.call("teacher", "POST", f"/courses/{w['cid']}/checkin-windows/{w['wid']}/close", None) if False else vps.call("teacher", "POST", f"/courses/{w['cid']}/sessions/{w['sid']}/checkin-windows/{w['wid']}/close", None)
def start_app(): sh(f"am start -W -n {PKG}/.MainActivity >/dev/null"); time.sleep(3)
def login(user):
    os.environ["PW_"] = vps.PW
    return node_js(f'''const p = await connect(); await p.goto("{vps.HOST}/login"); await sleep(2500);
if (await p.has("Đăng nhập EduPilot")) {{ await p.type('input[autocomplete=username]', "{user}@edupilot.local"); await p.type('input[type=password]', process.env.PW_); await p.clickText("Đăng nhập"); await sleep(4000); }}
console.log((await p.url())); p.close();''')

def shim_scan(path="/checkin", wait=3000):
    W = json.load(open(WORK + "/win.json"))
    u = W.get("qr_url") or (vps.HOST + path)
    open(WORK + "/_s.mjs", "w").write(f'''import {{ connect, sleep, shot }} from "{EV}/cdp.mjs";
import {{ SHIM }} from "{EV}/shim.mjs";
const p = await connect(); await p.send("Page.addScriptToEvaluateOnNewDocument", {{ source: SHIM }});
await p.goto({json.dumps(u)}); await sleep({wait});
console.log("BTN:" + await p.ev("!!document.body.innerText.match(/Quét mã điểm danh/)"));
await p.clickText("Quét mã điểm danh").catch(e => console.log("clickerr")); await sleep(2000);
console.log("SCAN:" + await p.ev("document.body.innerText.slice(0,200).replace(/\\\\n/g,' | ')"));
console.log("HAS_THOAT:" + await p.ev("!!document.body.innerText.match(/^Thoát$|Thoát/m)"));
p.close();''')
    r = subprocess.run(f"cd {EV} && SHOTS={SHOTS} node {WORK}/_s.mjs", shell=True, capture_output=True, text=True, timeout=150); return (r.stdout + r.stderr).strip()

def js(body, timeout=90):
    open(WORK + "/_j.mjs", "w").write(f'import {{ connect, sleep, shot }} from "{EV}/cdp.mjs";\nimport {{ SHIM }} from "{EV}/shim.mjs";\n' + body)
    r = subprocess.run(f"cd {EV} && SHOTS={SHOTS} node {WORK}/_j.mjs", shell=True, capture_output=True, text=True, timeout=timeout)
    out = (r.stdout + r.stderr).strip()
    if r.returncode: out = "ERR " + out[-300:]
    return out

def nav(path="/checkin", wait=2500):
    return js(f'const p = await connect(); await p.goto("{vps.HOST}" + {json.dumps(path)}); await sleep({wait}); console.log("URL " + await p.url()); p.close();')

GEO = '''(()=>{const d={w:innerWidth,h:innerHeight,sw:document.documentElement.scrollWidth,cw:document.documentElement.clientWidth,
inp:(()=>{const i=document.querySelector("input");return i?Math.round(i.getBoundingClientRect().top)+".."+Math.round(i.getBoundingClientRect().bottom):"-"})(),
btn:(()=>{const b=[...document.querySelectorAll("button")].find(x=>x.innerText.trim()==="Điểm danh");return b?Math.round(b.getBoundingClientRect().top)+".."+Math.round(b.getBoundingClientRect().bottom)+" h"+Math.round(b.getBoundingClientRect().height):"-"})(),
nav:(()=>{const n=document.querySelector("nav,[data-ep-nav]");return n?Math.round(n.getBoundingClientRect().top)+".."+Math.round(n.getBoundingClientRect().bottom):"-"})(),
h1:(()=>{const h=document.querySelector("h1");return h?Math.round(h.getBoundingClientRect().top)+" "+h.innerText.trim():"-"})(),
sai:(()=>{const s=getComputedStyle(document.documentElement);return s.getPropertyValue("--sai-top")+"/"+s.getPropertyValue("--sai-bottom")})()};return JSON.stringify(d)})()'''
