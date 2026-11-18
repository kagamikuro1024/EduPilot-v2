# Smoke S10 lô B: today/chủ đề yếu, DTO allowlist, rời lớp (API/DB) — lớp QD10A
from pm import *
import threading, subprocess
TA = "QC-r10-W"   # chủ đề riêng cho kiểm Hôm nay (12 câu SHORT)
if len([q for q in bank() if q["topic"] == TA]) == 0: mk_short(TA, 12, "QCr10W")
def wrong_attempt(u, topic=TA, n=10):
    finish_open(u); c, a, _ = start(u, {"mode": "TOPIC", "topic": topic, "count": n}); assert c in (200, 201), (c, a)
    for it in a["items"]: put(u, a["id"], it["id"], {"text": "zzz-sai"})
    c, r, _ = submit(u, a["id"]); return a["id"], c
def flush(u, C=C1):
    uid = sql(f"select id from users where email='{u}@edupilot.local'")[0][0]
    subprocess.run(["docker", "exec", "ep-qc-redis-1", "redis-cli", "del", f"ep:today:{uid}:{C}"], capture_output=True)
def today(u, C=C1):
    c, j, _ = api(u, "GET", f"/courses/{C}/today"); items = []
    def walk(x):
        if isinstance(x, dict):
            if x.get("kind"): items.append(x)
            for v in x.values(): walk(v)
        elif isinstance(x, list):
            for v in x: walk(v)
    walk(j); return c, items
def weak_in(u, topic=None):
    c, it = today(u); return [i for i in it if i["kind"] == "PRACTICE_WEAK_TOPIC" and (topic is None or f"topic={topic}" in i.get("href", ""))], it
u = "sv.nguyco"
sql(f"update practice_attempts set started_at=now()-interval '9 days', submitted_at=now()-interval '9 days' where course_id='{C1}' and topic='{TA}' and student_id=(select id from users where email='{u}@edupilot.local')")
aid, c = wrong_attempt(u)
print("wrong attempt", c, "weakness", api(u, "GET", f"{P}/practice/weakness")[1])
sql(f"update practice_attempts set started_at=now()-interval '4 days', submitted_at=now()-interval '4 days' where course_id='{C1}' and topic='{TA}' and student_id=(select id from users where email='{u}@edupilot.local')")  # started_at cũng lùi: code đo mốc bằng LastStartedAt
flush(u); w, items = weak_in(u, TA)
print("today kinds", [i["kind"] for i in items], str(w)[:300])
chk("TC-09-06 (nền) việc Ôn chủ đề khi chưa luyện > 3 ngày", len(w) >= 1 and "topic=" in json.dumps(w), f"{len(w)} việc: {str(w)[:160]}")
if w: print("urgency", w[0].get("urgency"), "href", w[0].get("href"), "title", w[0].get("title"))
# TC-09-07: 2 ngày → không có; 4 ngày → có lại
sql(f"update practice_attempts set started_at=now()-interval '2 days', submitted_at=now()-interval '2 days' where course_id='{C1}' and topic='{TA}' and student_id=(select id from users where email='{u}@edupilot.local')"); flush(u); w2, _ = weak_in(u, TA)
sql(f"update practice_attempts set started_at=now()-interval '4 days', submitted_at=now()-interval '4 days' where course_id='{C1}' and topic='{TA}' and student_id=(select id from users where email='{u}@edupilot.local')"); flush(u); w4, _ = weak_in(u, TA)
chk("TC-09-07 luyện chủ đề W cách 2 ngày: không có việc W; ép 4 ngày: có lại", len(w2) == 0 and len(w4) >= 1, f"2 ngày: {len(w2)} việc; 4 ngày: {len(w4)} việc")
# TC-09-08: luyện xong → biến mất ≤ 5 s
assert len(w4) >= 1
finish_open(u); c, a, _ = start(u, {"mode": "TOPIC", "topic": TA, "count": 10})
for it in a["items"]: put(u, a["id"], it["id"], {"text": "zzz-sai"})
t0 = time.time(); submit(u, a["id"]); gone = None
while time.time() - t0 < 12:
    ww, _ = weak_in(u, TA)
    if not ww: gone = time.time() - t0; break
    time.sleep(1)
chk("TC-09-08 submit lượt TOPIC của chủ đề xong: việc Ôn chủ đề biến mất ≤ 5 s", gone is not None and gone <= 5, f"biến mất sau {gone if gone is None else round(gone,1)} s")
# TC-09-09 chỉ STUDENT
chk("TC-09-09 today của TEACHER/TA/ADMIN: không có PRACTICE_WEAK_TOPIC", all(not [i for i in today(x)[1] if i["kind"] == "PRACTICE_WEAK_TOPIC"] for x in ("teacher", "ta", "admin")), "3 vai kiểm")
# TC-09-14 staff: 403 
sid = sql("select id from users where email='sv.kha@edupilot.local'")[0][0]
aids = sql(f"select id from practice_attempts where course_id='{C1}' and student_id='{sid}' limit 1")[0][0]
codes = []
for x in ("teacher", "ta", "admin"):
    for pth in ("/practice/history", "/practice/weakness", "/practice/overview", f"/practice/attempts/{aids}"):
        codes.append((x, pth[:22], api(x, "GET", P + pth)[0]))
print("TC-09-14", codes)
# TC-09-18 SV rời lớp / lớp lưu trữ
v = "sv34"; before = api(v, "GET", f"{P}/practice/history")[1]; nb = len(before.get("items", before) if isinstance(before, dict) else before)
sql(f"update enrollments set status='REMOVED', removed_at=now() where course_id='{C1}' and user_id=(select id from users where email='{v}@edupilot.local')")
c, j, _ = api(v, "GET", f"{P}/practice/history"); body = json.dumps(j)[:160]
sql(f"update enrollments set status='ACTIVE', removed_at=null where course_id='{C1}' and user_id=(select id from users where email='{v}@edupilot.local')")
chk("TC-09-18 SV rời lớp (enrollment REMOVED): history lớp 1 → 403 hoặc rỗng (hành vi thật ghi lại)", c == 403 or (c == 200 and nb >= 0 and '"items": []' in json.dumps(j)), f"trước {nb} lượt; sau khi REMOVED: {c} {body}")
print("done s10b")
