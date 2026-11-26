# A11-07 (mã sai) + A11-06 (mã đúng) cùng phiên, VPS 7b6b5ab3: TalkBack VERBOSE, lấy các lượt NÓI THẬT (SpeechControllerImpl "Speaking fragment").
from a11_live import *
import subprocess
OBS = open("a11_re.py", encoding="utf-8").read().split('OBS = """')[1].split('"""')[0]
def cycle(): tb(False); time.sleep(1); tb(True); time.sleep(10)
def dump(name): open(L.WORKD + f"/{name}-raw.log", "w").write(subprocess.run("adb logcat -d -v threadtime", shell=True, capture_output=True, text=True).stdout)
L.memguard(); L.blank_poster(); m.login("sv.qc2"); time.sleep(75)   # chờ hết cửa sổ giới hạn 429 của lần đo trước
w = L.newwin(L.CID["QCMOB2"]); L.home(); L.to_checkin(); time.sleep(2)
# 07: mã sai
L.click("Nhập mã"); time.sleep(1.2); L.type_code("AAA111"); time.sleep(0.8)
cycle(); m.cdp(OBS); L.adb("logcat -c"); L.adb("shell log -t QCMARK wrong"); L.click("Điểm danh"); time.sleep(9)
r7 = json.loads(m.cdp("JSON.stringify(window.__r)") or "[]"); dom7 = m.cdp("JSON.stringify({alerts:[...document.querySelectorAll('[role=alert]')].map(e=>(e.innerText||'').trim().slice(0,80)),status:[...document.querySelectorAll('[role=status]')].map(e=>(e.innerText||'').trim().slice(0,60))})")
dump("tb-a1107"); print("DOM 07:", dom7, "| thong bao:", json.dumps(r7, ensure_ascii=False)[:500], flush=True)
# 06: mã đúng (gõ lại khi TalkBack tắt)
tb(False); d = L.display(w); L.type_code(d["short_code"]); time.sleep(0.8)
cycle(); m.cdp("window.__r.length=0"); L.adb("logcat -c"); L.adb("shell log -t QCMARK right"); L.click("Điểm danh"); t0 = time.time()
while time.time() - t0 < 12 and not L.rows(w): time.sleep(0.2)
time.sleep(5); r6 = json.loads(m.cdp("JSON.stringify(window.__r)") or "[]"); dump("tb-a1106"); print("DOM 06 thong bao:", json.dumps(r6, ensure_ascii=False)[:500], "| rows:", len(L.rows(w)), flush=True)
L.shot("A11-06-re2"); json.dump(dict(r6=r6, r7=r7, dom7=dom7), open(L.WORKD + "/a11_0607v.json", "w"), ensure_ascii=False); tb(False)
