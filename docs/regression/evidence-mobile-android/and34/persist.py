# DEF-MOB-16: phiên đăng nhập phải còn sau khi app xuống nền rồi bị tắt. Dùng: python3 persist.py <nhan> <so_lan> <home|swipe>
import lib34 as L, m, time, sys, re, subprocess, json
tag, n, mode = sys.argv[1], int(sys.argv[2]), sys.argv[3]
def refresh_log(since_s):
    out = subprocess.run(["ssh", "edupilot-vps", f'docker logs edupilot-gateway-1 --since {since_s}s 2>&1 | grep "auth/refresh" | grep -oE "\\"time\\":\\"[^\\"]{{11}}[0-9:]{{8}}|\\"status\\":[0-9]+" | paste - -'], capture_output=True, text=True).stdout.strip().replace('"time":"', "").replace('"status":', "").split("\n")
    return [x for x in out if x]
res = []
for i in range(1, n + 1):
    if not L.memguard(): print("DUNG: guard bo nho"); break
    L.adb("shell am force-stop vn.edupilot.app"); L.adb("shell pm clear vn.edupilot.app")
    L.adb("shell am start -W -n vn.edupilot.app/.MainActivity"); time.sleep(5)
    m.login("sv.qc2"); time.sleep(1 if mode == "swipe" else 3)
    ok0 = "Tài khoản" in (m.cdp("document.body.innerText") or "")
    t_login = time.time()
    if mode == "home":
        L.adb("shell input keyevent KEYCODE_HOME"); time.sleep(1.5)
        L.adb("shell am force-stop vn.edupilot.app")
    else:
        L.adb("shell input keyevent KEYCODE_APP_SWITCH"); time.sleep(2.0)
        L.adb("shell input swipe 540 1300 540 250 250"); time.sleep(2.5)
    alive = bool(L.adb("shell pidof vn.edupilot.app"))
    time.sleep(1)
    t0 = time.time(); L.adb("shell am start -W -n vn.edupilot.app/.MainActivity")
    got = None
    for _ in range(60):
        try: s = m.cdp("document.body ? document.body.innerText.slice(0,400) : ''") or ""
        except Exception: s = ""
        if "Tài khoản" in s and "Hôm nay" in s: got = round(time.time() - t0, 1); break
        time.sleep(0.25)
    path = m.cdp("location.pathname") if got else m.cdp("location.pathname")
    rl = refresh_log(int(time.time() - t_login) + 5)
    res.append(dict(rep=i, dang_nhap_truoc=ok0, tien_trinh_con_sau_tat=alive, vao_lai_dang_nhap_sau_s=got, path=path, refresh=rl)); print(res[-1], flush=True)
    if got: L.shot(f"DEF16-{tag}-{mode}-{i}")
    time.sleep(8)  # tránh dồn đăng nhập (10/phút)
json.dump(res, open(f"{L.WORKD}/persist_{tag}_{mode}.json", "w"), ensure_ascii=False)
