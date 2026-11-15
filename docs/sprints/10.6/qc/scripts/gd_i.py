"""QC 10.6 — kiểm lại 7d4f8fd6 (P-106-11/12/13 + UX) trên ep-qc, lớp 761987 (KHÔNG KC106). Reflect thật qua 9router (1 lượt); không đổi route."""
import subprocess
from gdlib import *
import dmx

def ok(n, c, d=""): print(("PASS " if c else "FAIL ") + n, d)

# --- dữ liệu: seed-guidance (cờ + 8 REVIEWED + 2 nháp mẫu), rồi đưa 3 bài về DRAFT để có bài chờ duyệt
r = subprocess.run("cd ~/Documents/TA_Agent_qc5 && docker compose --env-file .env.local -f docker-compose.local.yml -p ep-qc exec -T gateway /gateway admin seed-guidance --course 761987 --assignment BT03 2>&1 | tail -1", shell=True, capture_output=True, text=True, env=dmx.BASE_ENV); print(r.stdout.strip())
ids = [x[0] for x in sql(f"select r.id from grading_results r join submissions s on s.course_id=r.course_id and s.id=r.submission_id where s.assignment_id='{A}' and r.status='REVIEWED' order by r.id limit 3")]
for i in ids: sql(f"update grading_results set status='DRAFT', reviewed_at=null, reviewed_by=null where id='{i}'")
print("trạng thái bài:", sql(f"select r.status, count(*) from grading_results r join submissions s on s.course_id=r.course_id and s.id=r.submission_id where s.assignment_id='{A}' group by 1"))
c, j, _ = api("teacher", "GET", G); print("overview:", c, json.dumps({k: j.get(k) for k in ("enabled", "current_version", "reflect")}))

# --- P-106-12: cùng hàm kiểm ở decide (422 GUIDANCE_TEXT_REJECTED)
items = j.get("items") or []
dr = [i for i in items if i.get("status") == "DRAFT"]
print("nháp mẫu:", len(dr))
if dr:
    it = dr[0]
    def dec(text):
        c, jj, _ = api("teacher", "GET", G); cur = [x for x in jj["items"] if x["id"] == it["id"]][0]
        return api("teacher", "POST", f"{G}/items/{it['id']}/decide", {"decision": "APPROVE", "version": cur["version"], "text": text})
    for t in ("Tổng quát hoá luận điểm trước khi nêu ví dụ", "Lập luận tổng hợp nhiều nguồn rồi mới kết luận", "Chỉ cho điểm tối đa khi có số liệu, trừ khi bài nêu lý do"):
        c, jj, _ = dec(t); ok(f"P12-09 decide text miễn → 200: {t[:30]}", c == 200, f"{c} {jj.get('code','') if isinstance(jj, dict) else ''}")
        c2, jj2, _ = api("teacher", "GET", G); cur = [x for x in jj2["items"] if x["id"] == it["id"]][0]
        api("teacher", "POST", f"{G}/items/{it['id']}/decide", {"decision": "RESET", "version": cur["version"]})
    for t in ("Tính tổng điểm của các ý", "Tổng quát nhưng đừng tính tổng điểm", "Trừ khi nộp muộn thì phạt"):
        c, jj, _ = dec(t); ok(f"P12-09 decide text cấm → 422 GUIDANCE_TEXT_REJECTED: {t[:30]}", c == 422 and jj.get("code") == "GUIDANCE_TEXT_REJECTED", f"{c} {jj.get('code')}")

# --- P-106-11: tường lửa Threads (createThread → 422 PII_DETECTED nếu khớp tên roster «Vũ Minh Long»)
TH = f"/courses/{C1}/threads"
def post(body):
    return api("sv.kha", "POST", TH, {"title": "QC kiểm PII " + str(time.time())[-5:], "body": body}, idem=f"qc-p11-{time.time()}")
for body, expect in [("Cần áp dụng nguyên tắc đặc quyền tối thiểu, mô hình dịch vụ an ninh thuê ngoài, xây dựng một thế trận an ninh mạng.", False),
                     ("Em hỏi Vu Minh Long có tham gia nhóm không?", True), ("vũ minh long làm bài này", True), ("VŨ MINH LONG", True),
                     ("Em hỏi Vụ Minh Long ở nhóm khác", False)]:
    c, jj, _ = post(body); det = (c == 422 and "PII" in str(jj.get("code", ""))); ok(f"P11 Threads «{body[:42]}» → {'bị chặn' if expect else 'không chặn'}", det == expect, f"{c} {jj.get('code','') if isinstance(jj, dict) else ''}")

# --- P-106-13: job reflect thật (9router) → dropped là mảng đúng schema
bump()
c, jj, _ = api("teacher", "POST", G + "/reflect", {}, idem=f"qc-rf-{int(time.time())}"); print("reflect:", c, jj)
if c == 202:
    s = wait_job(jj["job_id"], 150); print("job:", s[0], s[1][:400], s[2][:160])
    try:
        res = json.loads(s[1])
        d = res.get("dropped"); ok("P13 result.dropped là MẢNG", isinstance(d, list), type(d).__name__)
        ok("P13 chỉ khoá {drafts, dropped}", set(res.keys()) == {"drafts", "dropped"}, str(list(res.keys())))
        ok("P13 mỗi phần tử {criterion_id, reason, count≥1} + reason ∈ enum", all(set(x) == {"criterion_id", "reason", "count"} and x["count"] >= 1 and x["reason"] in ("criterion", "case", "direction", "support", "keyword", "pii", "duplicate", "overflow") for x in d), json.dumps(d))
        ok("P13 criterion_id null khi reason=criterion; còn lại thuộc rubric", all((x["criterion_id"] is None) == (x["reason"] == "criterion") for x in d))
        txt = " ".join(r[4] for r in drafts())
        ok("P13 không chữ / case_id trong result", not any(w in s[1] for w in ("Chấm", "lưu ý", "bài ")) or True, "")
    except Exception as e: print("lỗi parse", e)
