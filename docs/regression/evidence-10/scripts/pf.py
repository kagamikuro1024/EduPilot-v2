from pb import *
import threading
load_keys = None
n0 = len(bank()); ex0 = sql("select count(*) from question_bank where origin in ('EXTRACTED','GENERATED')")[0][0]
de = docid("QC Đề mẫu")["id"]; dk = docid("QC Đáp án mẫu")["id"]
c, a, _ = start("sv.gioi", {"mode": "TOPIC", "topic": "Mật mã công khai", "count": 5}); items = a["items"]
for it in items[:3]:
    q = it["question"]; ans = {"option_ids": [q["options"][0]["id"]]} if q["type"] in ("MCQ_SINGLE", "MCQ_MULTI") else ({"value": True} if q["type"] == "TRUE_FALSE" else {"text": "x"})
    put("sv.gioi", a["id"], it["id"], ans)
res = {}
def do_exp(i): res[f"exp{i}"] = sse("sv.gioi", f"{P}/practice/attempts/{a['id']}/items/{items[i]['id']}/explain")
th = [threading.Thread(target=do_exp, args=(i,)) for i in range(3)]; [t.start() for t in th]
c, j, _ = api("teacher", "POST", f"{P}/questions/extract", {"document_id": de, "answer_key_document_id": dk}); jid = j.get("job_id"); print("extract", c, j)
[t.join() for t in th]
for k, (st, t1, tt, txt) in sorted(res.items()): print(k, st, f"{t1:.0f}ms", f"{tt:.0f}ms", txt[:110].replace("\n", " | "))
ok("TC-06-20 explain khi LLM quá tải: có luồng nhận lỗi tại chỗ (event error), không treo", any("error" in r[3] or r[0] != 200 for r in res.values()) , f"{[ (k, r[0], 'error' in r[3]) for k, r in sorted(res.items())]}")
# phản hồi đúng/sai không đổi, nộp bình thường
c2, got, _ = att("sv.gioi", a["id"]); ok("TC-06-20b phản hồi đúng/sai sẵn có không đổi, nộp bình thường", all(i["feedback"] for i in got["items"][:3]) and submit("sv.gioi", a["id"])[0] == 200, "")
t0 = time.time(); J = wait_job(jid, t=120); print("EXTRACT JOB", J["status"], json.dumps(J.get("error"), ensure_ascii=False), json.dumps(J.get("result"), ensure_ascii=False)[:200], f"{time.time()-t0:.0f}s")
n1 = sql("select count(*) from question_bank where origin in ('EXTRACTED','GENERATED')")[0][0]
ok("TC-02-14 trích khi quá tải: job FAILED OVERLOADED, không câu nửa vời", J["status"] in ("FAILED", "DEAD") and "OVERLOADED" in json.dumps(J.get("error")) and n1 == ex0, f"status={J['status']} error={json.dumps(J.get('error'), ensure_ascii=False)[:120]} câu trích/sinh {ex0}→{n1}")
