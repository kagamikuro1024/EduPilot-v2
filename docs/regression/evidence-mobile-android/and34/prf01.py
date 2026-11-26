import lib34 as L, m, time, re, statistics, json
L.memguard()
m.login("sv.qc2"); time.sleep(4)
L.adb("shell input keyevent KEYCODE_HOME"); time.sleep(2)   # như người dùng thật: ẩn app (onPause → WebView ghi cookie) rồi vuốt tắt
res = []
for i in range(5):
    L.adb("shell am force-stop vn.edupilot.app"); time.sleep(2)
    t0 = time.time(); out = L.adb("shell am start -W -n vn.edupilot.app/.MainActivity"); tt = int(re.search(r"TotalTime: (\d+)", out).group(1))
    got = None
    for _ in range(60):
        s = m.cdp("document.body ? document.body.innerText.slice(0,400) : ''") or ""
        if "Tài khoản" in s and "Hôm nay" in s: got = round(time.time()-t0, 1); break
        time.sleep(0.25)
    res.append((tt, got)); print(i+1, "TotalTime", tt, "ms | «Hôm nay» da dang nhap sau", got, "s", flush=True)
    if got: L.shot("PRF-01-hom-nay")
    time.sleep(3); L.adb("shell input keyevent KEYCODE_HOME"); time.sleep(2)
print("trung vi TotalTime", statistics.median([r[0] for r in res]), "| lon nhat", max(r[0] for r in res), "| den Hom nay:", [r[1] for r in res])
json.dump(res, open(L.WORKD + "/prf01.json", "w"))
