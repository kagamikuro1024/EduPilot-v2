# TC-04-11 (P-11-47): embedding THẬT gemini-embedding-001 trên ep-qc, trần 200 lời gọi. Khoá đọc từ ~/Documents/TA_Agent_v2-s95/.env.local, đẩy qua admin API, KHÔNG in / ghi env / commit. Xong gỡ provider + trả tuyến EMBEDDING cũ.
import os, re, subprocess, sys, time, uuid
from p10lib import *
env = {k: v for k, _, v in (l.strip().partition("=") for l in open(os.path.expanduser("~/Documents/TA_Agent_v2-s95/.env.local")) if "=" in l and not l.startswith("#"))}
KEY = env["GEMINI_API_KEY"].strip().strip('"').strip("'")
NAME = "QC Gemini embed (P-11-47)"
c1 = course("teacher", "761987")
c, rt, _, _ = api("admin", "GET", "/admin/llm/routes")
old = [r for r in rt["items"] if r["task"] == "EMBEDDING"][0]; OLD_IDS = [e["model_id"] for e in old["chain"]]
print("tuyến cũ:", [(e["provider_name"], e["model"]) for e in old["chain"]])
c, j, _, _ = api("admin", "POST", "/admin/llm/providers", {"name": NAME, "type": "gemini", "api_key": KEY, "enabled": True,
                 "models": [{"model": "gemini-embedding-001", "kind": "embedding", "dims": 1536, "price_in": "0", "price_out": "0", "enabled": True}]})
print("tạo provider:", c, "" if c < 300 else str(j)[:200]); assert c < 300
del KEY
ps = {p["name"]: p for p in api("admin", "GET", "/admin/llm/providers")[1]["items"]}; prov = ps[NAME]; mid = prov["models"][0]["id"]
t0 = sql("select clock_timestamp()")[0][0]
def restore():
    c, rt, _, _ = api("admin", "GET", "/admin/llm/routes"); cur = [r for r in rt["items"] if r["task"] == "EMBEDDING"][0]
    print("trả tuyến về trạng thái trước lượt đo:", api("admin", "PUT", "/admin/llm/routes", {"task": "EMBEDDING", "chain": OLD_IDS, "params": cur["params"], "version": cur["version"]})[0])
    print("xoá provider Gemini:", api("admin", "DELETE", f"/admin/llm/providers/{prov['id']}")[0])
try:
    c, cur, _, _ = api("admin", "GET", "/admin/llm/routes"); cur = [r for r in cur["items"] if r["task"] == "EMBEDDING"][0]
    print("đặt tuyến EMBEDDING:", api("admin", "PUT", "/admin/llm/routes", {"task": "EMBEDDING", "chain": [mid], "params": cur["params"], "version": cur["version"]})[0])
    subprocess.run(["docker", "exec", "ep-qc-redis-1", "sh", "-c", "redis-cli --scan --pattern 'ep:rl:insight-generate:*' | xargs -r redis-cli del"], capture_output=True)
    sql(f"update insight_reports set data_watermark = data_watermark - interval '1 day' where course_id='{c1}'")
    c, j, _, _ = api("teacher", "POST", f"/courses/{c1}/insights/knowledge-gap/generate", {"period_days": int(os.environ.get("PERIOD", "120"))}, headers={"Idempotency-Key": "qc-0411-" + uuid.uuid4().hex}); print("generate", c)
    st = None
    for _ in range(100):
        cc, jj, _, _ = api("teacher", "GET", f"/jobs/{j['job_id']}"); st = jj.get("status")
        if st in ("SUCCEEDED", "FAILED", "DEAD"): break
        time.sleep(3)
    c, lat, _, _ = api("teacher", "GET", f"/courses/{c1}/insights/knowledge-gap/latest"); pl = lat["report"]["payload"]
    calls = sql(f"select count(*), count(*) filter (where status<>'ok') from llm_audit where task='EMBEDDING' and created_at >= '{t0}'")[0]
    A = re.compile(r"phishing|ransomware|ddos|dos\b|man-in-the-middle|sql injection|zero-day|botnet|social engineering|mã độc|tấn công|an ninh|lừa đảo|virus|malware|bảo mật", re.I)
    B = re.compile(r"dự thi|học vụ|vắng|chuyên cần|phúc khảo|buộc thôi học|quy chế|quy định|thi cuối|điểm thi|tài liệu vào phòng", re.I)
    top = pl["topics"][:2]
    for t in pl["topics"][:5]: print(t["rank"], t["name"], t["named_by"], t["signals"], "|", " / ".join(t["examples"])[:120])
    txt = lambda t: t["name"] + " " + " ".join(t["examples"]) + " " + " ".join(t.get("keywords", []))
    okA = any(A.search(txt(t)) for t in top); okB = any(B.search(txt(t)) for t in top)
    named_ok = all(t["named_by"] == "llm" and len(t["name"]) > 5 and not re.search(r"^[\w, ]+$", "") for t in top)
    ok("TC-04-11 embedding thật + 9router: A và B đứng đầu, tên chủ đề có nghĩa, không 5xx", st == "SUCCEEDED" and okA and okB and all(t["named_by"] == "llm" for t in top), f"job={st}; top2={[t['name'] for t in top]}; A={okA} B={okB}; lời gọi EMBEDDING={calls[0]} (lỗi {calls[1]}), trần 200")
finally:
    restore()
