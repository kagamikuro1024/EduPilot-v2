import lib34 as L, m, time, json, os
L.memguard()
pk = lambda: m.cdp("window.Capacitor.nativePromise('EduPilotAttest','sign',{message:'ins08'}).then(r=>r.public_key.slice(0,60)+'...'+r.public_key.slice(-12)).catch(e=>'ERR '+e)")
L.policy(L.CID["QCMOB2"], rot=60, geo=None)
L.to_checkin() if "checkin" not in (m.cdp("location.pathname") or "") else None
pk_truoc = pk(); print("khoa truoc go:", pk_truoc)
print("go:", L.adb("uninstall vn.edupilot.app")); print("cai:", L.adb("install -r " + os.path.expanduser("~/Documents/as-emu-evidence/app-debug-vps-HEAD.apk")).split("\n")[-1])
L.adb("shell am start -W -n vn.edupilot.app/.MainActivity"); time.sleep(6)
t = m.cdp("document.body.innerText.slice(0,200).replace(/\\n/g,' / ')"); print("sau cai, chua dang nhap:", "Đăng nhập" in t and "Tài khoản" not in t, "|", t[:90])
m.login("sv.qc2"); time.sleep(3); L.to_checkin(); time.sleep(2)
pk_sau = pk(); print("khoa sau cai:", pk_sau, "| khac khoa cu:", pk_sau != pk_truoc)
out = []
for k in (1, 2):
    w = L.newwin(L.CID["QCMOB2"])
    if k == 2: L.home(); L.to_checkin(); time.sleep(2)
    dt, r = L.submit_code(w); fl = [f["code"] for x in r for f in x.get("flags", [])]; out.append(fl); print("luot", k, ":", fl, "|", L.txt()[100:150], flush=True)
L.shot("INS-08-done")
L.policy(L.CID["QCMOB2"], rot=10, geo=None)
print("DEVICE_NEW dung 1 lan o luot dau, khong o luot sau:", "DEVICE_NEW" in out[0] and "DEVICE_NEW" not in out[1])
