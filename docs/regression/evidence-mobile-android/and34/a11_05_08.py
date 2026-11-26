import lib34 as L, m, time, json, api_sec as A
L.memguard(); L.blank_poster(); L.home(); L.to_checkin(); time.sleep(2)
print("quet:", L.open_scan())
print("A11-05 khung quet DOM:", m.cdp("(()=>{const v=document.querySelector('video'); const box=v&&(v.closest('[aria-label],[role]')||v.parentElement); return JSON.stringify({video:{aria:v&&v.getAttribute('aria-label'),role:v&&v.getAttribute('role'),tabindex:v&&v.getAttribute('tabindex')}, wrapper:box&&{tag:box.tagName,aria:box.getAttribute('aria-label'),role:box.getAttribute('role')}, thoat:[...document.querySelectorAll('button')].filter(b=>b.innerText.trim()==='Thoát').map(b=>({aria:b.getAttribute('aria-label'),vis:!!b.offsetParent}))})})()"))
L.shot("A11-05-scan"); L.click("Thoát"); time.sleep(1)
# A11-08: gây 429 bằng API (mã sai liên tiếp), rồi nhập sai ở giao diện để vào trạng thái «limited»
sv = A.login("sv.qc2@edupilot.local", m.acct("sv.qc2")); cid = L.CID["QCMOB2"]; n429 = None
for i in range(1, 25):
    c, j = A.req("POST", f"/courses/{cid}/checkins", {"code": "AAA111"}, sv)
    if c == 429: n429 = i; print("429 o lan", i, "| Retry-After/details:", str(j)[:160]); break
print("429?", n429)
