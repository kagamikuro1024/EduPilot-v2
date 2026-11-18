# TC-09-16: đo thời gian học PRACTICE bật/tắt theo công tắc cá nhân (user_settings.track_study_time)
from pm import *
TAB = str(uuid.uuid4()); U = "sv.moi"; UID = sql(f"select id from users where email='{U}@edupilot.local'")[0][0]
def secs(): return int(sql(f"select coalesce(sum(active_seconds),0) from study_time_daily where student_id='{UID}' and area='PRACTICE' and course_id='{C1}'")[0][0])
def hb(): return api(U, "POST", "/activity/heartbeat", {"course_id": C1, "area": "PRACTICE", "tab_id": TAB})[0]
def settings(): return api(U, "GET", "/me/settings")[1]
print("ack", api(U, "POST", "/me/tracking-notice/ack", {})[0])
st = settings(); print("track hiện tại", st.get("track_study_time"), st.get("version"))
if not st.get("track_study_time"): api(U, "PUT", "/me/settings", {**{k: st[k] for k in ("notify_ticket_by_mail", "notify_answer_by_mail", "remind_deadline_by_mail", "reminders")}, "track_study_time": True, "version": st["version"]}); st = settings()
b0 = secs(); cs = []
for i in range(3): cs.append(hb()); time.sleep(32)
time.sleep(70); b1 = secs(); print("BẬT: giây PRACTICE", b0, "→", b1, cs)
r = api(U, "PUT", "/me/settings", {**{k: st[k] for k in ("notify_ticket_by_mail", "notify_answer_by_mail", "remind_deadline_by_mail", "reminders")}, "track_study_time": False, "version": st["version"]}); print("tắt", r[0], lastj(r[1], 120))
time.sleep(2); c0 = secs(); cs2 = []
for i in range(3): cs2.append(hb()); time.sleep(32)
time.sleep(70); c1 = secs(); print("TẮT: giây PRACTICE", c0, "→", c1, cs2)
st = settings(); api(U, "PUT", "/me/settings", {**{k: st[k] for k in ("notify_ticket_by_mail", "notify_answer_by_mail", "remind_deadline_by_mail", "reminders")}, "track_study_time": True, "version": st["version"]})
chk("TC-09-16", b1 > b0 and c1 == c0 and set(cs2) == {204}, f"sv.moi bật đo: 3 nhịp PRACTICE cách 32 s → study_time_daily PRACTICE {b0}→{b1} s; tắt `track_study_time` (PUT /me/settings {r[0]}): 3 nhịp (mã {sorted(set(cs2))}, vẫn 204) → {c0}→{c1} s (không ghi); đã bật lại công tắc")
