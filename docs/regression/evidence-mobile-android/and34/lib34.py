# Thư viện lô Android-only (26/11). Chạy từ and34/. KHÔNG ghi /tmp: ảnh QR + log vào WORKD.
import os, sys, json, time, subprocess, datetime
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
import m
WORKD = os.path.expanduser("~/Documents/as-emu-evidence/work")
AVD = os.path.expanduser("~/.android/avd/ep-api36-ps.avd")
CID = {"QCMOB2": "01a21371-7998-75d8-bc50-58a8e2a3ff0b", "QCVPSC": "01a21311-dbf7-7e12-b5b1-b6875f936f03", "QCVPSD": "01a21312-117f-730b-8427-635d54124fea", "QCVPSB": "01a21311-a725-7b4d-a952-285569315a42"}
def sql(q): return subprocess.run(["ssh", "edupilot-vps", f'docker exec edupilot-postgres-1 psql -U edupilot -d edupilot -At -F"|" -c "{q}"'], capture_output=True, text=True).stdout.strip()
def policy(cid, rot=10, geo=None, cidrs=None, self_checkin=True):
    c, det = m.vps.call("teacher", "GET", f"/courses/{cid}")
    att = {"self_checkin": self_checkin, "require": "WEB", "rotate_seconds": rot, "late_after_min": 15}
    if geo: att["geofence"] = geo
    if cidrs: att["campus_cidrs"] = cidrs
    return m.vps.call("teacher", "PUT", f"/courses/{cid}/settings", {"version": det["settings"]["version"], "attendance": att})
def newwin(cid, minutes=30):
    sid = sql(f"select s.id from class_sessions s where s.course_id='{cid}' and (select count(*) from checkin_windows w where w.session_id=s.id)<2 order by s.starts_at desc limit 1")
    c, w = m.vps.call("teacher", "POST", f"/courses/{cid}/sessions/{sid}/checkin-windows", {"minutes": minutes})
    if c == 409 and w.get("code") == "WINDOW_ALREADY_OPEN":  # đóng cửa sổ đang mở (của lượt trước) rồi mở lại
        ow = w["details"]["window_id"]
        m.vps.call("teacher", "POST", f"/courses/{cid}/checkin-windows/{ow}/close", None)
        c, w = m.vps.call("teacher", "POST", f"/courses/{cid}/sessions/{sid}/checkin-windows", {"minutes": minutes})
    if c != 201: raise RuntimeError(("newwin", c, w))
    return dict(cid=cid, sid=sid, wid=w["id"])
def display(w):
    c, d = m.vps.call("teacher", "GET", f"/courses/{w['cid']}/sessions/{w['sid']}/checkin-windows/{w['wid']}/display"); return d
def rem(d):
    a = datetime.datetime.fromisoformat(d["step_expires_at"].replace("Z", "+00:00")); b = datetime.datetime.fromisoformat(d["server_time"].replace("Z", "+00:00"))
    return (a - b).total_seconds()
def rows(w): return [x for x in (m.vps.checkins(w["cid"], w["sid"]) or {}).get("items", []) if x.get("window_id") == w["wid"]]
def adb(c, t=60): return subprocess.run(f"adb {c}", shell=True, capture_output=True, text=True, timeout=t).stdout.strip()
def shot(name, d=None):
    d = d or os.path.expanduser("~/.config/edupilot-team/qc2-work/mob/shots"); p = f"{d}/{name}.png"
    subprocess.run(f"adb exec-out screencap -p > '{p}'", shell=True); return p
def poster(url, tag):
    import segno
    from PIL import Image
    qa = f"{WORKD}/_q.png"; segno.make(url, error="l").save(qa, scale=14, border=4)
    im = Image.open(qa).convert("RGB"); c = Image.new("RGB", (1024, 1024), "white"); c.paste(im.resize((720, 720), Image.NEAREST), (152, 228))
    p = f"{AVD}/qr-{tag}-{int(time.time()*1000)}.png"; c.save(p); adb(f"emu virtualscene-image wall {p}"); return p
def txt(): return m.cdp("document.body.innerText.replace(/\\n/g,' / ')")
def click(label): return m.cdp("(()=>{const b=[...document.querySelectorAll('button')].find(x=>x.innerText.trim()===" + json.dumps(label) + "&&!x.disabled&&x.offsetParent); if(!b) return 'nobtn'; b.click(); return 'ok'})()")
def type_code(code):
    m.cdp("(()=>{const i=document.querySelector('input'); const s=Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value').set; s.call(i,''); i.dispatchEvent(new Event('input',{bubbles:true})); i.focus(); return 1;})()")
    adb(f"shell input text {code.replace(' ', '')}")
def memguard():
    r = subprocess.run(os.path.join(WORKD, "memguard.sh"), shell=True, capture_output=True, text=True); print(r.stdout.strip()); return r.returncode == 0
def tap_text(text, retries=3):
    """Chạm nút theo chữ qua uiautomator dump (hộp quyền hệ thống nằm ngoài WebView)."""
    import re, xml.etree.ElementTree as ET
    for _ in range(retries):
        adb("shell uiautomator dump /sdcard/ui.xml >/dev/null 2>&1"); x = adb("exec-out cat /sdcard/ui.xml")
        try: root = ET.fromstring(x)
        except Exception: time.sleep(1); continue
        for n in root.iter("node"):
            if (n.get("text") or "").strip() == text or (n.get("content-desc") or "").strip() == text:
                b = list(map(int, re.findall(r"\d+", n.get("bounds")))); adb(f"shell input tap {(b[0]+b[2])//2} {(b[1]+b[3])//2}"); return True
        time.sleep(1)
    return False
def fresh_app(login_user="sv.qc2", perm_reset=True):
    """pm clear + thu hồi quyền vị trí + mở app + đăng nhập một lần + chuyển trong app tới /checkin (chờ hộp quyền nếu có)."""
    adb("shell pm clear vn.edupilot.app")
    if perm_reset:
        for p in ("ACCESS_FINE_LOCATION", "ACCESS_COARSE_LOCATION"): adb(f"shell pm revoke vn.edupilot.app android.permission.{p}")
    adb("shell am start -W -n vn.edupilot.app/.MainActivity"); time.sleep(5)
    m.login(login_user); time.sleep(3)
def to_checkin():
    m.cdp("(()=>{const a=[...document.querySelectorAll('a')].find(x=>/\\/checkin/.test(x.getAttribute('href')||'')); if(a){a.click(); return 'ok'} return 'nolink'})()"); time.sleep(3)
def submit_code(w, wait=15):
    """Bấm Nhập mã → gõ mã hiện tại → Điểm danh. Trả (giây từ bấm tới hàng DB, các hàng)."""
    click("Nhập mã"); time.sleep(1.2)
    d = display(w); type_code(d["short_code"]); time.sleep(0.8)
    t0 = time.time(); click("Điểm danh"); r = []
    while time.time() - t0 < wait:
        r = rows(w)
        if r: break
        time.sleep(0.2)
    return round(time.time() - t0, 1), r
def qr_url(w):
    d = display(w); return d["qr_url"].replace("https://localhost", m.vps.HOST, 1), d
def home():
    m.cdp("(()=>{const a=[...document.querySelectorAll('a')].find(x=>x.getAttribute('href')==='/'||/Hôm nay/.test(x.textContent)&&x.offsetParent); if(a){a.click()}})()"); time.sleep(2.0)
def open_scan(grant=True):
    """Bấm «Quét mã điểm danh»; nếu hộp quyền camera hiện thì cấp «While using the app». Trả True khi <video> chạy."""
    r = click("Quét mã điểm danh")
    if r != "ok": return False
    for _ in range(25):
        if "GrantPermissionsActivity" in adb("shell dumpsys window | grep -m1 GrantPermissionsActivity"):
            if grant: tap_text("While using the app")
            else: return "dialog"
        if m.cdp("(()=>{const v=document.querySelector('video'); return !!(v&&v.readyState>=2&&v.videoWidth>0&&!v.paused)})()"): return True
        time.sleep(0.4)
    return False
def blank_poster():
    from PIL import Image
    p = f"{AVD}/blank-{int(time.time()*1000)}.png"; Image.new("RGB", (1024, 1024), "white").save(p); adb(f"emu virtualscene-image wall {p}")
def analyze(mp4, tag):
    """Từ video: A = khung QR hiện trong xem trước (độ lệch chuẩn vùng giữa tăng), C = khung xem trước biến mất (đang gửi). Trả (A_s, C_s, fps, n)."""
    import glob, shutil
    from PIL import Image, ImageStat
    d = f"{WORKD}/an_{tag}"; shutil.rmtree(d, ignore_errors=True); os.makedirs(d)
    subprocess.run(f"ffmpeg -loglevel error -i '{mp4}' -vf scale=108:-1 {d}/f%04d.png", shell=True)
    fr = sorted(glob.glob(d + "/f*.png")); st = []
    for f in fr:
        im = Image.open(f).convert("L"); W, H = im.size; box = im.crop((int(W*.1), int(H*.15), int(W*.9), int(H*.65)))
        s = ImageStat.Stat(box); st.append((s.mean[0], s.stddev[0]))
    return fr, st
