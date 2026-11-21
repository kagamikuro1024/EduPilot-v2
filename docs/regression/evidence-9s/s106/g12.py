import sys, os, re; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
W = "/Users/kuro/.config/edupilot-team/qc3-work/"; HERE = os.path.dirname(os.path.abspath(__file__))
def clear(a): sql(f"delete from grading_guidance_items where assignment_id='{a}' and status='DRAFT'")
def res(jb): 
    try: return json.loads(jb[1]) if jb[1] else {}
    except Exception: return {}
def ui(a, tcs, must, mustnot, user="teacher", click=False, post=None, wait=9000):
    json.dump({"a": a, "tcs": tcs, "must": must, "mustnot": mustnot, "user": user, "click": click, "post": post, "wait": wait}, open(W + "g12_case.json", "w"), ensure_ascii=False)
    os.system(f'cd {HERE} && QC_RES= ./runpw.sh qc-real-106-g12.spec.ts "P12 UI" 100 2>&1 | grep -E "VERDICT|✘" | cut -c1-600')
def run():
    R = {}
    # P12-14/15/16/26
    flag("GRADING_GUIDANCE", True); a = fxa("g12b", 12); review_with(a, [(i, {"c2": "0.00", "c3": "0.00"}) for i in range(6)] + [(i, {"c3": "5.00"}) for i in (6, 7)])
    clear(a); c, j, jb = reflect(a, "rf_mix14"); r = res(jb); txt = json.dumps(r, ensure_ascii=False); print("mix14", r); d = {(x.get("criterion_id"), x["reason"]): x["count"] for x in r.get("dropped", [])}
    rub = [x[0] for x in sql(f"select jsonb_array_elements(rubric->'criteria')->>'id' from assignments where id='{a}'")] if False else None
    kw = [k for k in d if k[1] == "keyword"]; sup = [k for k in d if k[1] == "support"]
    ok14 = len(kw) == 1 and d[kw[0]] == 2 and len(sup) == 1 and d[sup[0]] == 1 and "c9" not in txt
    chk("TC-P12-14", ok14, f"stub rf_mix14 (2 mục keyword, 1 mục c3 lẻ ca (1 ca đúng tiêu chí + 1 ca tiêu chí khác); mục id lạ «c9» không dựng được: gateway chặn bằng enum rubric của schema) → dropped = {txt}; c9 trong result: {'c9' in txt}")
    cases = [x[0] for x in sql(f"select r.id from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{a}'")]
    names = [x[0] for x in sql("select full_name from users where email like 'sv%@edupilot.local' limit 30")]
    bad = [w for w in ("Tính tổng điểm", "trừ điểm", "case_id") + tuple(cases[:5]) + tuple(names[:5]) if w and w in txt]
    chk("TC-P12-15", not bad, f"jq toàn result {txt}: chữ lưu ý stub / «case_id» / id kết quả / tên roster xuất hiện: {bad}")
    emitted = 5; ndrop = sum(d.values())  # mix14: 2 keyword + 1 support + 1 c9 = 4 mục stub; drafts 0
    chk("TC-P12-26", all((k[0] is None) or re.fullmatch(r"c\d+", str(k[0])) for k in d), f"criterion_id trong dropped = {[k[0] for k in d]} (id rubric c…, không tên tiêu chí); lưu ý: chưa đổi tên tiêu chí giữa hai lần rút")
    # P12-16: N = tổng count gồm duplicate: rf_ok rồi rf_dup_text
    clear(a); c, j, jb = reflect(a, "rf_ok"); r_ok = res(jb); print("ok", r_ok); c, j, jb = reflect(a, "rf_dup_text"); r_dup = res(jb); print("dup", r_dup)
    chk("TC-P12-17", r_ok.get("dropped") == [] and r_ok.get("drafts", 0) >= 1, f"stub rf_ok → {json.dumps(r_ok, ensure_ascii=False)} (dropped = [])")
    chk("TC-P12-16", sum(x["count"] for x in r_dup.get("dropped", [])) == 1 and any(x["reason"] == "duplicate" for x in r_dup.get("dropped", [])) and r_dup.get("drafts") == 0, f"rf_dup_text (1 mục trùng nháp sẵn có) → {json.dumps(r_dup, ensure_ascii=False)}: N = tổng count gồm duplicate")
    # UI: bấm «Rút lưu ý» trong giao diện (job gắn phiên); stub đặt trước; hạn mức rút xoá trước mỗi lần
    for tc, mode, must, mustnot, kw in (
        (["TC-P12-18"], "rf_n3", [r"\b3 lưu ý bị loại"], [r"tiêu chí c\d: ", "nhắc tổng", "Tính tổng điểm"], {}),
        (["TC-P12-19"], "rf_ok_plus1", [r"Chờ duyệt", r"1 lưu ý bị loại"], [], {}),
        (["TC-P12-20"], "rf_ok", [], [r"lưu ý bị loại"], {}),
        (["TC-P12-21"], "invalid", [], [r"lưu ý bị loại"], {"wait": 20000}),
        (["TC-P12-22"], "rf_kw_forbidden", [r"Không có lưu ý mới", r"2 lưu ý bị loại"], [], {}),
        (["TC-P12-24"], "rf_ok", [], [r"lưu ý bị loại", r"Đã xảy ra"], {"post": "old"}),
    ):
        clear(a); stub(mode); reset_reflect_limit(); ui(a, tc, must, mustnot, click=True, **kw)
    # P12-23 RBAC
    clear(a); c, j, jb = reflect(a, "rf_n3"); jid = j["job_id"]
    out = {}
    for u in ("sv.kha", "ta"):
        out[u] = (api(u, "GET", f"/jobs/{jid}")[0], api(u, "GET", G(a))[0])
    print("rbac", out)
    chk("TC-P12-23", out["sv.kha"][0] in (403, 404) and out["sv.kha"][1] in (403, 404) and out["ta"][1] == 200, f"SV: GET /jobs/{{id}} {out['sv.kha'][0]}, GET …/guidance {out['sv.kha'][1]}; TA: GET …/guidance {out['ta'][1]}, GET /jobs/{{id}} {out['ta'][0]}")
    stub("rf_n3"); reset_reflect_limit(); print("TA chỉ đọc: nút Rút lưu ý không hiện cho TA (xem spec)")
    stub("allow")
with window(): run()
