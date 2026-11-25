import m, scn, scn2, time, json, subprocess, os

def log(*a): print(*a, flush=True)

def prep(user='sv.qc2'):
    scn2.fresh_login(user)
    m.sh("pm grant vn.edupilot.app android.permission.CAMERA"); time.sleep(0.5)
    e = scn2.entry()
    s = scn2.scan() if e else False
    return e, s

def case(name, fn):
    t0 = time.time()
    try:
        r = fn()
    except Exception as ex:
        r = ("ERR", str(ex)[:100])
    log(name, "->", r, "| %.0fs" % (time.time() - t0))

def sanity():
    w = scn2.fresh()
    e, s = prep()
    if not s: return ("nocamera", e)
    scn2.poster_url(w['d']['qr_url'], "sanity2")
    r, t, dt = scn2.wait_result(w, 20)
    return (r, round(dt, 1), len(scn.rows(w)))

def c06():
    w = scn.newwin(); u = w['d']['qr_url']
    m.vps.call('teacher', 'POST', f"/courses/{w['cid']}/sessions/{w['sid']}/checkin-windows/{w['wid']}/close", None)
    e, s = prep()
    if not s: return ("nocamera",)
    scn2.poster_url(u, "closed4")
    r, t, dt = scn2.wait_result(w, 20)
    return (r, "msg:", "Đã đóng" in t, len(scn.rows(w)))

def c14():
    w = scn2.fresh()
    e, s = prep()
    if not s: return ("nocamera",)
    scn2.poster_two("https://example.com/hello", w['d']['qr_url'], "mix2")
    r, t, dt = scn2.wait_result(w, 25)
    return (r, round(dt, 1), len(scn.rows(w)))

def c08():
    w = scn2.fresh()
    e, s = prep()
    if not s: return ("nocamera",)
    m.sh("input keyevent 3"); time.sleep(10)
    m.sh("am start -W -n vn.edupilot.app/.MainActivity >/dev/null"); time.sleep(2.5)
    t = scn.txt()
    out = dict(path=m.cdp("location.pathname"), video=m.cdp("(()=>{const v=document.querySelector('video'); return !!v})()"),
               scanbtn="Quét mã điểm danh" in t, input=m.cdp("!!document.querySelector('input')"))
    s2 = scn2.scan()
    if s2:
        scn2.poster_url(w['d']['qr_url'], "after8b")
        r, t2, dt = scn2.wait_result(w, 20)
        out["rescan"] = (r, round(dt, 1), len(scn.rows(w)))
    return out

def c09():
    w = scn2.fresh()
    e, s = prep()
    if not s: return ("nocamera",)
    before = m.sh("dumpsys activity activities | grep -c MainActivity")
    for i in range(4):
        m.sh("settings put system accelerometer_rotation 0")
        m.sh(f"settings put system user_rotation {i % 2}")
        time.sleep(1.5)
    m.sh("settings put system user_rotation 0")
    v = m.cdp("(()=>{const v=document.querySelector('video'); return v? v.videoWidth+'x'+v.videoHeight+' rs='+v.readyState:'none'})()")
    scn2.poster_url(w['d']['qr_url'], "after9")
    r, t, dt = scn2.wait_result(w, 20)
    return ("video", v, "res:", r, round(dt, 1), len(scn.rows(w)))

def c10():
    w = scn2.fresh()
    e, s = prep()
    if not s: return ("nocamera",)
    m.adb("emu network delay 2000")
    scn2.poster_url(w['d']['qr_url'], "slow10")
    r, t, dt = scn2.wait_result(w, 25)
    m.adb("emu network delay none")
    return (r, round(dt, 1), len(scn.rows(w)))

def c13():
    w = scn2.fresh()
    e, s = prep()
    if not s: return ("nocamera",)
    m.adb("emu network delay 3000")
    scn2.poster_url(w['d']['qr_url'], "off13")
    time.sleep(1.2)
    m.sh("svc wifi disable"); m.sh("svc data disable")
    r, t, dt = scn2.wait_result(w, 25)
    msg = "Chưa có mạng" in t; retry = "Thử lại" in t
    m.sh("svc wifi enable"); m.sh("svc data enable"); m.adb("emu network delay none")
    time.sleep(4)
    if retry:
        scn.click("Thử lại")
        time.sleep(6)
    return (r, round(dt, 1), "msg:", msg, "retry:", retry, len(scn.rows(w)))

scn.policy(60)
for name, fn in [("SCN-08", c08), ("SCN-09", c09), ("SCN-10", c10), ("SCN-13", c13)]:
    case(name, fn)
