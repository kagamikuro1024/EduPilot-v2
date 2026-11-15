#!/usr/bin/env python3
"""E5 — câu hỏi AI sinh được giảng viên duyệt (US-P10-10 AC7). Chỉ HTTP.

  e5.py run      10 lô × 10 câu MCQ từ hai tài liệu seed lớp 1 (`POST …/questions/generate`, làn BATCH), tuần tự (≈ 1 lời gọi LLM / lô).
                 Trần 5 lô / lớp / giờ: POST bị 429 thì ghi (lô, giờ, mã), dừng; chạy lại `e5.py run` sau → tiếp lô dở, không sinh đôi.
                 Stack đã dựng lại (course_id khác raw) → raw cũ cất thành `e5-<commit cũ>.json`, bắt đầu lượt mới.
  e5.py export   lưu vết câu đã sinh → raw/e5-questions.json
  e5.py report   `GET …/questions/gen-stats?job_id` từng lô → e5.json / e5.md / e5.svg. Chưa có câu nào được duyệt → WAITING_HUMAN (không nội suy)
Không đếm câu đã duyệt sẵn trong seed (chỉ các lô do lệnh này tạo).
"""
import argparse
import json
import os
import subprocess
import time
from datetime import datetime

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


def gateway_started():
    """`State.StartedAt` của gateway stack đo: đổi giữa lô ⇒ gateway đã khởi động lại."""
    name = f"{os.environ.get('COMPOSE_PROJECT_NAME', 'ep-qc')}-gateway-1"
    r = subprocess.run(["docker", "inspect", "-f", "{{.State.StartedAt}}", name], capture_output=True, text=True)
    return r.stdout.strip() or None


def run():
    tok = login(TEACHER)
    cid = lib.course_by_code(tok, lib.CLASS1)
    raw = lib.load_raw("e5.json") or {"jobs": {}}
    if raw.get("course_id") and raw["course_id"] != cid:  # stack dựng lại: câu cũ không còn → cất lượt cũ, không ghi đè
        lib.save_raw(f"e5-{raw['meta']['commit']}.json", raw)
        print(f"lượt cũ (course {raw['course_id']}) → raw/e5-{raw['meta']['commit']}.json")
        raw = {"jobs": {}}
    raw.pop("meta", None)
    docs = {d["title"]: d["id"] for d in lib.list_all(f"/courses/{cid}/documents", tok)}
    raw["course_id"] = cid
    for i, (doc, topic) in enumerate(PLAN):
        k = f"lo{i + 1:02d}"
        if raw["jobs"].get(k, {}).get("t1"):  # lô đã kết thúc (kể cả FAILED / lỗi) giữ nguyên, không chạy lại để lấy số đẹp
            continue
        if doc not in docs:
            raise Fail(f"lớp 1 không có tài liệu «{doc}»")
        if not raw["jobs"].get(k, {}).get("job_id"):
            t0 = lib.now_iso()
            st, j = call("POST", f"/courses/{cid}/questions/generate", login(TEACHER),
                         {"document_ids": [docs[doc]], "topic": topic, "difficulty": "MEDIUM", "count": 10, "types": ["MCQ_SINGLE"]}, idem=f"e5-{cid}-{k}", tries=1)
            if st == 429:  # trần 5 lô / lớp / giờ: ghi đúng, không chờ trong lệnh
                raw.setdefault("rate_limited", []).append({"batch": k, "at": t0, "code": (j or {}).get("code"), "message": (j or {}).get("message")})
                lib.save_raw("e5.json", raw)
                print(f"DỪNG ở {k}: 429 {(j or {}).get('code')} lúc {t0} — chạy lại `e5.py run` sau khi hết giờ trần")
                return
            if st != 202:
                raise Fail(f"generate {k} → {st} {j}")
            raw["jobs"][k] = {"topic": topic, "document": doc, "job_id": j["job_id"], "t0": t0, "gateway_started_at": gateway_started()}
            lib.save_raw("e5.json", raw)
        try:
            r = lib.wait_job(login(TEACHER), raw["jobs"][k]["job_id"], timeout=900)
        except Fail as e:  # gateway mất giữa lô: ghi lô lỗi, dừng
            raw["jobs"][k].update(status="LOST", error={"code": "HARNESS", "message": str(e)[:200]}, t1=lib.now_iso())
            lib.save_raw("e5.json", raw)
            raise
        raw["jobs"][k].update(status=r["status"], result=r.get("result"), error=r.get("error"), t1=lib.now_iso())
        if gateway_started() != raw["jobs"][k]["gateway_started_at"]:  # gateway khởi động lại giữa lô ⇒ lô lỗi (PM), không chạy lại
            raw["jobs"][k].update(status=f"{r['status']}_GATEWAY_RESTARTED", error={"code": "GATEWAY_RESTARTED", "message": "gateway khởi động lại giữa lô"})
        lib.save_raw("e5.json", raw)
        print(f"  {k}: {raw['jobs'][k]['status']} {json.dumps(r.get('result'), ensure_ascii=False)[:160]}")
        time.sleep(8)  # 9router 20 rpm dùng chung với QC: lô tuần tự + nghỉ ⇒ ≪ 8 rpm
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
        if j.get("t1") and live:
            if j.get("status") == "SUCCEEDED":
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
    models = sorted({f"{a['provider']}/{a['model']}" for a in calls})
    obj = {"meta": lib.meta({"stack": "ep-qc", "model": ", ".join(models) or prov.get("P1", {}).get("model"), "via": "9router", "plan": PLAN}),
           "status": status, "reason": None if status == "OK" else f"Giảng viên (chủ dự án) mới duyệt {reviewed}/{created} câu; AC7: ghi cỡ mẫu thật, không nội suy.",
           "generated": created, "skipped_by_reason": skipped, "review": tot, "n_reviewed": reviewed, "rates": rate,
           "llm_calls": len(calls), "latency_ms": lib.pct([a["latency_ms"] for a in calls]), "tokens_in": sum(a["tokens_in"] for a in calls), "tokens_out": sum(a["tokens_out"] for a in calls),
           "jobs": raw["jobs"], "rate_limited": raw.get("rate_limited", [])}
    lib.svg_bars("e5.svg", [("duyệt nguyên", [tot["unchanged"]]), ("sửa nhẹ", [tot["edited"]]), ("loại", [tot["rejected"]]), ("chờ duyệt", [tot["pending"]])], ["số câu"],
                 f"E5 — {created} câu AI sinh: kết quả duyệt (n duyệt = {reviewed})", "câu", fmt="{:.0f}")
    pct = lambda x: "–" if rate is None else f"{rate[x] * 100:.1f} %".replace(".", ",")  # noqa: E731
    L = ["# E5 — câu hỏi AI sinh", "", f"Trạng thái: **{status}**. Stack `ep-qc` @ `{obj['meta']['commit']}`, {obj['meta']['timestamp'][:10]}. Model sinh: `{obj['meta']['config']['model']}` qua 9router.", "",
         f"- Sinh: **{created}** câu từ {len(raw['jobs'])} lô (10 câu / lô, MCQ một đáp án, mức MEDIUM, nguồn: «Network Security Threats», «Forecasting (QMB ch. 6b)»). Bỏ: " + (", ".join(f"{k} {v}" for k, v in skipped.items()) or "không") + ".",
         f"- Duyệt: **n = {reviewed}** — duyệt nguyên {tot['unchanged']} ({pct('unchanged')}), sửa nhẹ {tot['edited']} ({pct('edited')}), loại {tot['rejected']} ({pct('rejected')}); chờ duyệt {tot['pending']}.",
         f"- Lời gọi QUESTION_GEN (llm_audit): {len(calls)}, p50 {obj['latency_ms'].get('p50')} ms, p95 {obj['latency_ms'].get('p95')} ms; token vào {obj['tokens_in']}, ra {obj['tokens_out']}.", "",
         "## Từng lô", "", "| lô | chủ đề | POST (UTC) | xong (UTC) | giây | trạng thái | câu tạo | bỏ | lỗi | POST bị 429 trước lô | lời gọi LLM (trạng thái) |",
         "| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |"]
    rl = {}
    for x in raw.get("rate_limited", []):
        rl.setdefault(x["batch"], []).append(x["at"][11:19])
    iso = lambda s: datetime.fromisoformat(s.replace("Z", "+00:00"))  # noqa: E731
    secs = lambda j: round((iso(j["t1"]) - iso(j["t0"])).total_seconds()) if j.get("t1") else "–"  # noqa: E731
    for k, j in sorted(raw["jobs"].items()):
        res, au = j.get("result") or {}, j.get("audit") or []
        st_count = {}
        for a in au:
            st_count[a["status"]] = st_count.get(a["status"], 0) + 1
        n_created = res.get("created") if isinstance(res.get("created"), int) else len(res.get("created") or [])
        L.append(f"| {k} | {j['topic']} | {j['t0'][11:19]} | {(j.get('t1') or '')[11:19] or '–'} | {secs(j)} | {j.get('status') or 'đang chạy'} | {n_created} | {len(res.get('skipped') or [])} | "
                 f"{(j.get('error') or {}).get('code') if isinstance(j.get('error'), dict) else (j.get('error') or '–')} | {', '.join(rl.get(k, [])) or '–'} | "
                 + (", ".join(f"{s} {n}" for s, n in sorted(st_count.items())) or "–") + " |")
    L += ["", "![](e5.svg)", "", "## Cách duyệt (chủ dự án)", "",
          f"1. Mở `{os.environ.get('APP_PUBLIC_URL', 'http://localhost:580')}`, đăng nhập `teacher@edupilot.local` (mật khẩu seed), chọn lớp 761987 → **Ngân hàng câu hỏi** (`/questions`).",
          "2. Lọc **Nguồn = Sinh từ tài liệu**, **Trạng thái = Chờ duyệt**, rồi lần lượt **Chủ đề** = một trong 10 chủ đề ở bảng trên (lớp còn câu chờ duyệt khác do QC tạo, không thuộc E5; gen-stats chỉ đếm câu của 10 lô này).",
          "3. Mở từng câu: đúng như AI viết → `Duyệt` (tính «duyệt nguyên»); phải sửa → `Chỉnh sửa`, lưu, rồi `Duyệt` (tính «sửa nhẹ»); sai / lạc đề / trùng → menu `⋯` → `Loại`.",
          "4. Xong (hoặc dừng giữa chừng) chạy `python3 benchmarks/eval/e5.py report` (hay `make eval-report`): đọc gen-stats, không gọi LLM; tỉ lệ chỉ tính trên câu đã duyệt.", "",
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
