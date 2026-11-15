#!/usr/bin/env python3
"""E5 — câu hỏi AI sinh được giảng viên duyệt (US-P10-10 AC7). Chỉ HTTP.

  e5.py run      10 lô × 10 câu MCQ từ hai tài liệu seed lớp 1 (`POST …/questions/generate`, làn BATCH). Trần 5 lô / lớp / giờ:
                 gặp 429 thì dừng, ghi raw, in giờ chạy lại (tiếp tục được, không sinh đôi)
  e5.py report   `GET …/questions/gen-stats?job_id` từng lô → e5.json / e5.md / e5.svg. Chưa có câu nào được duyệt → WAITING_HUMAN (không nội suy)
Không đếm câu đã duyệt sẵn trong seed (chỉ các lô do lệnh này tạo).
"""
import argparse
import json

import lib
from lib import Fail, call, login

TEACHER = "teacher"
PLAN = [  # (tài liệu, chủ đề) — chủ đề khác nhau để lô không trùng nhau
    ("Network Security Threats", "Phân loại mối đe doạ an ninh mạng"),
    ("Network Security Threats", "Sâu máy tính, virus và trojan"),
    ("Network Security Threats", "Tấn công trinh sát"),
    ("Network Security Threats", "Tấn công truy cập và leo thang đặc quyền"),
    ("Network Security Threats", "Tấn công từ chối dịch vụ"),
    ("Network Security Threats", "Biện pháp giảm thiểu tấn công mạng"),
    ("Forecasting (QMB ch. 6b)", "Mô hình dự báo chuỗi thời gian"),
    ("Forecasting (QMB ch. 6b)", "Trung bình trượt và trung bình trượt có trọng số"),
    ("Forecasting (QMB ch. 6b)", "Làm trơn hàm mũ"),
    ("Forecasting (QMB ch. 6b)", "Đo sai số dự báo"),
]


def run():
    raw = lib.load_raw("e5.json") or {"jobs": {}}
    tok = login(TEACHER)
    cid = lib.course_by_code(tok, lib.CLASS1)
    docs = {d["title"]: d["id"] for d in lib.list_all(f"/courses/{cid}/documents", tok)}
    raw["course_id"] = cid
    for i, (doc, topic) in enumerate(PLAN):
        k = f"lo{i + 1:02d}"
        if raw["jobs"].get(k, {}).get("status") == "SUCCEEDED":
            continue
        if doc not in docs:
            raise Fail(f"lớp 1 không có tài liệu «{doc}»")
        if not raw["jobs"].get(k, {}).get("job_id"):
            t0 = lib.now_iso()
            try:
                st, j = call("POST", f"/courses/{cid}/questions/generate", login(TEACHER),
                             {"document_ids": [docs[doc]], "topic": topic, "difficulty": "MEDIUM", "count": 10, "types": ["MCQ_SINGLE"]}, idem=f"e5-{k}")
            except lib.Stop as e:
                raw["stopped"] = str(e)
                lib.save_raw("e5.json", raw)
                print(f"DỪNG ở {k}: {e} — chạy lại `e5.py run` sau khi hết giờ trần")
                return
            if st != 202:
                raise Fail(f"generate {k} → {st} {j}")
            raw["jobs"][k] = {"topic": topic, "document": doc, "job_id": j["job_id"], "t0": t0}
            lib.save_raw("e5.json", raw)
        r = lib.wait_job(login(TEACHER), raw["jobs"][k]["job_id"], timeout=900)
        raw["jobs"][k].update(status=r["status"], result=r.get("result"), error=r.get("error"), t1=lib.now_iso())
        lib.save_raw("e5.json", raw)
        print(f"  {k}: {r['status']} {json.dumps(r.get('result'), ensure_ascii=False)[:160]}")
    raw.pop("stopped", None)
    lib.save_raw("e5.json", raw)


def export():
    """Lưu vết 100 câu đã sinh (nội dung, đáp án, trích dẫn, trạng thái duyệt) trước khi dựng lại stack → raw/e5-questions.json. Không nhập lại được: E5 duyệt trên stack mới thì chạy lại `run`."""
    raw = lib.load_raw("e5.json")
    cid, out = raw["course_id"], []
    for k, j in sorted(raw["jobs"].items()):
        for qid in (j.get("result") or {}).get("created") or []:
            st, q = call("GET", f"/courses/{cid}/questions/{qid}", login(TEACHER))
            out.append({"batch": k, "topic": j["topic"], "http": st, "question": q if st == 200 else None})
    lib.save_raw("e5-questions.json", {"course_id": cid, "n": len(out), "items": out})
    print(f"→ raw/e5-questions.json ({len(out)} câu)")


def report():
    raw = lib.load_raw("e5.json")
    if not raw or not raw.get("jobs"):
        raise Fail("chưa có raw/e5.json (chạy `e5.py run`)")
    prov = json.loads((lib.RAW / "providers.json").read_text()) if (lib.RAW / "providers.json").exists() else {}
    live, cid = lib.ready(), raw["course_id"]  # stack tắt → dùng gen-stats / llm_audit đã lưu lần trước (make eval-report không cần stack)
    tot = {"unchanged": 0, "edited": 0, "rejected": 0, "pending": 0}
    skipped, created, calls = {}, 0, []
    for k, j in sorted(raw["jobs"].items()):
        res = j.get("result") or {}
        created += res.get("created", 0) if isinstance(res.get("created"), int) else len(res.get("created") or [])
        for s in res.get("skipped") or []:
            skipped[s["reason"]] = skipped.get(s["reason"], 0) + 1
        if j.get("status") == "SUCCEEDED" and live:
            st, g = call("GET", f"/courses/{cid}/questions/gen-stats?job_id={j['job_id']}", login(TEACHER))
            j["gen_stats"] = g if st == 200 else {"http": st}
            j["audit"] = lib.audit(j["t0"], j["t1"], task="QUESTION_GEN", course=cid)
        for key in tot:
            tot[key] += (j.get("gen_stats") or {}).get(key, 0)
        calls += j.get("audit") or []
    lib.save_raw("e5.json", raw)
    reviewed = tot["unchanged"] + tot["edited"] + tot["rejected"]
    rate = {k: round(tot[k] / reviewed, 4) for k in ("unchanged", "edited", "rejected")} if reviewed else None
    status = "OK" if reviewed >= 100 else "WAITING_HUMAN"
    obj = {"meta": lib.meta({"stack": "ep-qc", "model": prov.get("P1", {}).get("model"), "via": "9router", "plan": PLAN}),
           "status": status, "reason": None if status == "OK" else f"Giảng viên (chủ dự án) mới duyệt {reviewed}/{created} câu; AC7: ghi cỡ mẫu thật, không nội suy.",
           "generated": created, "skipped_by_reason": skipped, "review": tot, "n_reviewed": reviewed, "rates": rate,
           "llm_calls": len(calls), "latency_ms": lib.pct([a["latency_ms"] for a in calls]), "tokens_in": sum(a["tokens_in"] for a in calls), "tokens_out": sum(a["tokens_out"] for a in calls),
           "jobs": raw["jobs"]}
    lib.svg_bars("e5.svg", [("duyệt nguyên", [tot["unchanged"]]), ("sửa nhẹ", [tot["edited"]]), ("loại", [tot["rejected"]]), ("chờ duyệt", [tot["pending"]])], ["số câu"],
                 f"E5 — {created} câu AI sinh: kết quả duyệt (n duyệt = {reviewed})", "câu", fmt="{:.0f}")
    pct = lambda x: "–" if rate is None else f"{rate[x] * 100:.1f} %".replace(".", ",")  # noqa: E731
    L = ["# E5 — câu hỏi AI sinh", "", f"Trạng thái: **{status}**. Stack `ep-qc` @ `{obj['meta']['commit']}`, {obj['meta']['timestamp'][:10]}. Model sinh: `{obj['meta']['config']['model']}` qua 9router.", "",
         f"- Sinh: **{created}** câu từ {len(raw['jobs'])} lô (10 câu / lô, MCQ một đáp án, mức MEDIUM, nguồn: «Network Security Threats», «Forecasting (QMB ch. 6b)»). Bỏ: " + (", ".join(f"{k} {v}" for k, v in skipped.items()) or "không") + ".",
         f"- Duyệt: **n = {reviewed}** — duyệt nguyên {tot['unchanged']} ({pct('unchanged')}), sửa nhẹ {tot['edited']} ({pct('edited')}), loại {tot['rejected']} ({pct('rejected')}); chờ duyệt {tot['pending']}.",
         f"- Lời gọi QUESTION_GEN (llm_audit): {len(calls)}, p50 {obj['latency_ms'].get('p50')} ms, p95 {obj['latency_ms'].get('p95')} ms; token vào {obj['tokens_in']}, ra {obj['tokens_out']}.", "",
         "![](e5.svg)", "", "## Cách duyệt (chủ dự án)", "", "Đăng nhập `teacher@edupilot.local` → Ngân hàng câu hỏi lớp 761987 → lọc «Chờ duyệt» → với từng câu: `Duyệt` (nguyên), sửa rồi `Duyệt` (sửa nhẹ), hoặc `Loại`. Xong chạy `make eval-report`: gen-stats đọc lại, không gọi LLM.", "",
         "## Giới hạn", "", "- Dữ liệu mô phỏng (D44): tài liệu seed lớp 1; chưa kiểm chứng với câu hỏi giảng viên thật dùng trong lớp.",
         "- Tỉ lệ chỉ tính trên câu đã duyệt (AC7); chưa duyệt đủ 100 câu thì ghi cỡ mẫu thật, không nội suy.", "- Người duyệt: chủ dự án (một người, không có người duyệt thứ hai)."]
    lib.write_report("e5", obj, L)


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("cmd", choices=["run", "report", "export"])
    a = ap.parse_args()
    try:
        if a.cmd == "run":
            lib.refuse_demo()
            if not lib.ready():
                lib.die("Gateway chưa chạy")
            run()
        elif a.cmd == "export":
            export()
        else:
            report()
    except Fail as e:
        lib.die(f"LỖI: {e}", 2)


if __name__ == "__main__":
    main()
