# A11-08 (DEF-MOB-15c, VPS 1021a702): TalkBack log VERBOSE -> lấy NỘI DUNG từng lượt đọc (logcat tag `talkback`, dòng `EVENT_*: ttsOutput= ...`).
from a11_live import *
import api_sec as A, re
def utter():
    out = L.adb("logcat -d -v time -s talkback:V", t=90) if False else subprocess.run("adb logcat -d -v threadtime", shell=True, capture_output=True, text=True).stdout
    res = []
    for ln in out.split("\n"):
        mm = re.search(r"(\d\d:\d\d:\d\d\.\d+)\s+\d+\s+\d+\s+V talkback: TalkBackFeedbackProvider:\s+(EVENT_\w+):\s+ttsOutput=\s*(.*?)\s{2,}queueMode", ln)
        if mm and mm.group(3).strip(): res.append((mm.group(1), mm.group(2), mm.group(3).strip()))
    mk = [re.search(r"(\d\d:\d\d:\d\d\.\d+)", l).group(1) for l in out.split("\n") if "QCMARK" in l and re.search(r"(\d\d:\d\d:\d\d\.\d+)", l)]
    return res, mk
def secs(a, b):
    f = lambda s: sum(float(x) * k for x, k in zip(s.split(":"), (3600, 60, 1)))
    return round(f(a) - f(b), 1)
import subprocess
L.memguard(); m.login("sv.qc2"); time.sleep(3); L.home(); L.to_checkin(); time.sleep(2)
sv = A.login("sv.qc2@edupilot.local", m.acct("sv.qc2")); cid = L.CID["QCMOB2"]
print("429 qua API:", [A.req("POST", f"/courses/{cid}/checkins", {"code": "AAA111"}, sv)[0] for _ in range(6)], flush=True)
L.click("Nhập mã"); time.sleep(1.2); L.type_code("AAA111"); time.sleep(0.8)          # gõ khi TalkBack TẮT
tb(False); time.sleep(1); tb(True); time.sleep(10)                                   # tắt/bật: xoá hàng đợi TTS, chờ ổn định
L.adb("logcat -c"); L.adb("shell log -t QCMARK pre-click"); time.sleep(6); L.adb("shell log -t QCMARK click")
L.click("Điểm danh"); t0 = time.time(); lim = None
while time.time() - t0 < 70:
    time.sleep(1)
    st = m.cdp("document.querySelector('[data-part=checkin]')?document.querySelector('[data-part=checkin]').dataset.state:'?'")
    if st == "limited" and lim is None: lim = round(time.time() - t0)
    if lim is not None and st != "limited": break
time.sleep(4); res, mk = utter(); click_t = mk[-1] if mk else None
print("limited tu giay", lim, "| het gio o giay", round(time.time() - t0), "| moc click:", click_t, "| moc truoc click:", mk[:1])
pre = [r for r in res if click_t and secs(r[0], click_t) < 0]
print("TRUOC click (6 s im lang) so luot:", len(pre), [(r[0][-9:], r[2]) for r in pre])
post = [(secs(r[0], click_t), r[1], r[2]) for r in res if click_t and secs(r[0], click_t) >= 0]
print("SAU click: tong", len(post)); 
for p in post: print("  +%5.1fs %-28s %s" % p)
first10 = [p for p in post if p[0] <= 10]; print("10 s DAU:", len(first10), "luot")
json.dump(dict(pre=pre, post=post, lim=lim), open(L.WORKD + "/a11_final.json", "w"), ensure_ascii=False)
open(L.WORKD + "/a11_final_talkback.log", "w").write("\n".join(f"{a} {b} {c}" for a, b, c in res))
tb(False)
