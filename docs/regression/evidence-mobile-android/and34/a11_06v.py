# A11-06 (mã đúng) VPS 7b6b5ab3, TalkBack VERBOSE: đặt rotate 60 s để mã không đổi trong lúc tắt/bật TalkBack; khôi phục rotate 10 sau khi xong.
from a11_live import *
import subprocess
OBS = open("a11_re.py", encoding="utf-8").read().split('OBS = """')[1].split('"""')[0]
L.memguard(); L.blank_poster(); print("policy 60:", L.policy(L.CID["QCMOB2"], rot=60)[0], flush=True)
try:
    m.login("sv.qc2"); time.sleep(3); w = L.newwin(L.CID["QCMOB2"]); L.home(); L.to_checkin(); time.sleep(2)
    L.click("Nhập mã"); time.sleep(1.2); d = L.display(w); L.type_code(d["short_code"]); time.sleep(0.8)
    tb(False); time.sleep(1); tb(True); time.sleep(10); m.cdp(OBS); L.adb("logcat -c"); L.adb("shell log -t QCMARK right")
    L.click("Điểm danh"); t0 = time.time()
    while time.time() - t0 < 12 and not L.rows(w): time.sleep(0.2)
    time.sleep(6); r6 = json.loads(m.cdp("JSON.stringify(window.__r)") or "[]")
    open(L.WORKD + "/tb-a1106-raw.log", "w").write(subprocess.run("adb logcat -d -v threadtime", shell=True, capture_output=True, text=True).stdout)
    print("DOM 06:", json.dumps(r6, ensure_ascii=False)[:600], "| rows:", len(L.rows(w)), flush=True); L.shot("A11-06-re2"); tb(False)
finally:
    print("policy 10:", L.policy(L.CID["QCMOB2"], rot=10)[0])
