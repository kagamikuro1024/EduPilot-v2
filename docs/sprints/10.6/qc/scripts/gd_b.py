# 10.6 pha 2 — B: reflect → stub: ma trận chế độ. python gd_b.py
from gdlib import *
def clear(): sql("delete from grading_guidance_items")
def run(mode, key, expect_drafts=None, drop_key=None, tc="", keep=False):
    if not keep: clear()
    c, j, jb = reflect(mode, key)
    if c != 202: print(f"[{mode}] HTTP {c} {json.dumps(j)[:150]}"); return None
    res = json.loads(jb[1]) if jb[1].startswith("{") else {}
    n = len(drafts()); dr = res.get("dropped", {})
    cond = (jb[0] == "SUCCEEDED") and (expect_drafts is None or n == expect_drafts) and (drop_key is None or dr.get(drop_key, 0) >= 1)
    ok(f"{tc} [{mode}] job={jb[0]} drafts={n} (kỳ vọng {expect_drafts}) dropped={ {k: v for k, v in dr.items() if v} }", cond, jb[2][:100])
    return jb
run("rf_ok", "qc-rok-0002", 2, tc="G03-04/11/19")
d = drafts()
ok("G03-19 DRAFT có support = số id kết quả, source_result_ids không trùng users.id", all(int(x[3]) == int(x[5]) for x in d) and sql("select count(*) from grading_guidance_items i join users u on u.id = any(i.source_result_ids)")[0][0] == "0", str(d[:1]))
time.sleep(40)  # cầu dao GRADING có thể còn mở từ lượt trước
run("rf_bad_case", "qc-rbs-0001", 0, "case", "G03-13a")
run("rf_wrong_direction", "qc-rwd-0001", 0, None, "G03-13b")
# rf_bad_criterion / rf_fake_support vi phạm schema strict → cổng gateway chặn trước (BAD_RESPONSE) và mở cầu dao: chạy cuối
run("rf_low_support", "qc-rls-0001", 0, "support", "G03-14b")
run("rf_total_keyword", "qc-rtk-0001", 0, "keyword", "G03-15")
run("rf_pii_text", "qc-rpt-0001", 0, "pii", "G03-16")
run("rf_many", "qc-rmn-0001", 8, "overflow", "G03-18")
run("rf_empty", "qc-rem-0001", 0, None, "G03-21")
# trùng: dựng DRAFT trước rồi cho stub lặp lại đúng chữ
clear(); reflect("rf_ok", "qc-rdp-0001"); n0 = len(drafts())
run("rf_dup_text", "qc-rdp-0002", n0, "duplicate", "G03-17", keep=True)
for mode in ("error", "invalid"):
    clear(); c, j, jb = reflect(mode, f"qc-r{mode}-0001")
    a = sql(f"select count(*) from llm_audit where task='GRADING' and created_at > now() - interval '3 minutes' and status <> 'ok'")[0][0]
    ok(f"G03-22 [{mode}] job FAILED, 0 DRAFT, thử lại", jb[0] == "FAILED" and len(drafts()) == 0, f"{jb[0]} err={jb[2][:80]} llm_audit lỗi 3 phút={a}")
stub("allow")
time.sleep(40)
clear(); c, j, jb = reflect("rf_bad_criterion", "qc-rbc-0002"); ok("G03-12 criterion lạ: schema strict chặn trước (job FAILED, 0 DRAFT, không 500)", jb[0] == "FAILED" and len(drafts()) == 0, jb[2][:80])
time.sleep(40)
stub("allow")
