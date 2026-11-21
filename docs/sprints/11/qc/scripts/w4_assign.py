# Đợt R · làn 4: F9 Bài tập (tạo → nộp → chấm LLM thật → duyệt → công bố), nhánh muộn / chặn
from rlib import *
import hashlib, time, ssl, urllib.request, json, datetime as dt, uuid
c1 = C("teacher"); tag = str(int(time.time()))[-5:]
ctx = ssl.create_default_context(); ctx.check_hostname = False; ctx.verify_mode = ssl.CERT_NONE
def iso(days): return (dt.datetime.utcnow() + dt.timedelta(days=days)).replace(microsecond=0).isoformat() + "Z"
rub = [{"id": "c1", "name": "Nêu đúng khái niệm", "max_score": "5", "guidance": "Nêu đúng định nghĩa tường lửa và vị trí đặt: 4–5. Thiếu một ý: 2–3. Sai: 0–1."},
       {"id": "c2", "name": "Ví dụ minh hoạ", "max_score": "5", "guidance": "Có ví dụ cụ thể về quy tắc lọc gói: 4–5. Chung chung: 2–3. Không có: 0–1."}]
def mk(title, due, allow_late=True, publish=True):
    c, j, _ = api("teacher", "POST", f"/courses/{c1}/assignments", {"title": title, "description": "Trình bày tường lửa là gì, đặt ở đâu trong mạng và cho một ví dụ quy tắc lọc.", "type": "ESSAY", "max_score": "10", "due_at": due, "allow_late": allow_late, "late_penalty_per_day": "0.5", "late_max_days": 2, "appeal_days": 7, "channels": ["IN_APP"], "submission_code": "Q" + uuid.uuid4().hex[:6].upper(), "reference_answer": "Tường lửa là thiết bị/phần mềm lọc lưu lượng theo quy tắc, đặt ở biên mạng; ví dụ chặn cổng 23 từ ngoài vào.", "rubric": rub})
    aid = (j or {}).get("id") or ((j or {}).get("assignment") or {}).get("id")
    if publish and aid: api("teacher", "POST", f"/courses/{c1}/assignments/{aid}/publish", {})
    return c, j, aid
def submit(user, aid, lines):
    body = mkpdf(lines); sha = hashlib.sha256(body).hexdigest()
    c, pre, _ = api(user, "POST", f"/courses/{c1}/assignments/{aid}/submit/presign", {"filename": f"bai-{tag}.pdf", "size_bytes": len(body), "sha256": sha})
    if c not in (200, 201): return c, pre, None
    rq = urllib.request.Request(pre["url"], data=body, method=pre.get("method", "PUT"), headers={"Content-Type": "application/pdf", **(pre.get("headers") or {})})
    try: urllib.request.urlopen(rq, context=ctx, timeout=60)
    except urllib.error.HTTPError as e: return e.code, str(e), None
    c, j, _ = api(user, "POST", f"/courses/{c1}/assignments/{aid}/submit", {"upload_id": pre["upload_id"]})
    return c, j, ((j or {}).get("submission") or j or {}).get("id") if isinstance(j, dict) else None
# 1 tạo + công bố
c, j, aid = mk(f"QC-R BT {tag} tường lửa", iso(2))
ok("R-F9-01 GV tạo bài tập (rubric, muộn, phúc khảo)", c in (200, 201) and aid, f"{c} {str(j)[:120]}")
c, sg, _ = api("teacher", "POST", f"/courses/{c1}/assignments/{aid}/rubric/suggest", {})
ok("R-F9-02 AI gợi ý rubric (LLM)", c in (200, 202, 422), f"{c} {str(sg)[:140]}")
c, jl, _ = api("sv.kha", "GET", f"/courses/{c1}/assignments")
ok("R-F9-03 SV thấy bài đã công bố", c == 200 and aid in json.dumps(jl), f"{c}")
c, td, _ = api("sv.kha", "GET", "/me/today")
ok("R-F9-04 hạn nộp < 24 h → việc ở Hôm nay (bài hạn 48 h: chưa)", c == 200, f"{c}")
# 2 nộp trong app
c, sj, sid = submit("sv.kha", aid, ["Tuong lua la he thong loc luu luong mang theo quy tac.", "Dat o bien mang, giua mang noi bo va Internet.", "Vi du: chan cong 23 telnet tu ngoai vao."])
ok("R-F9-05 SV nộp bài trong app (PDF)", c in (200, 201, 202) and sid, f"{c} {str(sj)[:120]}")
c, mine, _ = api("sv.kha", "GET", f"/courses/{c1}/assignments/{aid}/mine")
ok("R-F9-06 SV thấy bài của mình, KHÔNG thấy điểm nháp", c == 200 and "ai_score" not in json.dumps(mine) and '"score"' not in json.dumps(mine), f"{c} {str(mine)[:200]}")
c, sj2, _ = submit("sv.nguyco", aid, ["Tuong lua loc goi tin. Vi du chan IP la."])
# 3 chấm (LLM 9router) — cả bài
t0 = time.time(); c, gj, _ = api("teacher", "POST", f"/courses/{c1}/assignments/{aid}/grade-all", {})
ok("R-F9-07 grade-all nhận việc (202 + job)", c in (200, 202), f"{c} {str(gj)[:100]}")
jid = (gj or {}).get("job_id")
res = None
for _ in range(180):
    c, sl, _ = api("teacher", "GET", f"/courses/{c1}/submissions?assignment_id={aid}")
    its = (sl or {}).get("items", []) if isinstance(sl, dict) else []
    done = [x for x in its if (x.get("result") or {}).get("status") in ("DRAFT", "REVIEWED") or x.get("grading_status") in ("DRAFT", "DONE")]
    if len(done) >= 2: res = its; break
    time.sleep(2)
ok("R-F9-08 chấm nháp xong (LLM thật) trong ≤ 6 phút", res is not None, f"{time.time()-t0:.0f}s; submissions={len(res) if res else 0}")
print("   mẫu:", json.dumps(res[0], ensure_ascii=False)[:400] if res else None)
print("w4a done", aid, sid)
open("/tmp/r_w4_state.json", "w").write(json.dumps({"aid": aid, "sid": sid, "tag": tag}))
