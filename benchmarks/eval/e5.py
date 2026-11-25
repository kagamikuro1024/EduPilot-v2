#!/usr/bin/env python3
"""E5 — câu hỏi AI sinh được giảng viên duyệt (US-P10-10 AC7). Chỉ HTTP.

  e5.py run      10 lô × 10 câu MCQ từ hai tài liệu seed lớp 1 (`POST …/questions/generate`, làn BATCH), tuần tự (≈ 1 lời gọi LLM / lô).
                 Trần 5 lô / lớp / giờ: POST bị 429 thì ghi (lô, giờ, mã), dừng; chạy lại `e5.py run` sau → tiếp lô dở, không sinh đôi.
                 Stack đã dựng lại (course_id khác raw) → raw cũ cất thành `e5-<commit cũ>.json`, bắt đầu lượt mới.
  e5.py export   lưu vết câu đã sinh → raw/e5-questions.json
  e5.py pack     gói duyệt mù → benchmarks/eval/human/e5_duyet_goi.csv (đề, phương án, đáp án đánh dấu, giải thích, nguồn + trích đoạn;
                 KHÔNG có điểm / nhận định của máy). Trích đoạn đọc `content_chunks` qua EVAL_PSQL (READ ONLY).
  e5.py apply    áp bản đã điền benchmarks/eval/human/e5_duyet.csv qua API (vai TEACHER): DUYỆT → review APPROVE; SỬA → PUT câu
                 (cột sua_*) rồi APPROVE; BỎ → review REJECT. Câu không còn PENDING thì bỏ qua (chạy lại an toàn) → raw/e5-apply.json
  e5.py report   `GET …/questions/gen-stats?job_id` từng lô → e5.json / e5.md / e5.svg. Chưa có câu nào được duyệt → WAITING_HUMAN (không nội suy)
Không đếm câu đã duyệt sẵn trong seed (chỉ các lô do lệnh này tạo).
"""
import argparse
import csv
import json
import os
import re
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
    return gateway_info()["started_at"]


def gateway_info():
    """Bản gateway đang chạy: image id + giờ tạo container. Image không mang sha git (không label, không /version) ⇒ sha của bản dựng
    do người dựng xác nhận, ghi tay vào `raw.builds[<image>]`; KHÔNG lấy git HEAD của worktree bộ đo."""
    name = f"{os.environ.get('COMPOSE_PROJECT_NAME', 'ep-qc')}-gateway-1"
    r = subprocess.run(["docker", "inspect", "-f", "{{.Image}}\t{{.Created}}\t{{.State.StartedAt}}", name], capture_output=True, text=True)
    img, created, started = (r.stdout.strip().split("\t") + [None, None, None])[:3]
    return {"image": (img or "").removeprefix("sha256:")[:12] or None, "created": created, "started_at": started}


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
            g = gateway_info()
            raw["jobs"][k] = {"topic": topic, "document": doc, "job_id": j["job_id"], "t0": t0, "gateway_started_at": g["started_at"], "gateway_image": g["image"], "gateway_created": g["created"]}
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


PACK, FILLED = lib.HERE / "human/e5_duyet_goi.csv", lib.HERE / "human/e5_duyet.csv"
LETTERS = "ABCDEFGH"
REJECT = {"": "UNSPECIFIED", "MO_HO": "MO_HO", "SAI_DAP_AN": "SAI_DAP_AN", "TRUNG_LAP": "TRUNG_LAP", "LAC_DE": "LAC_DE",
          "KHO_DE_BAT_THUONG": "KHO_DE_BAT_THUONG", "SAI_DINH_DANG": "SAI_DINH_DANG"}


def _questions():
    """100 câu theo thứ tự lô → [(mã E5-001…, câu)], từ raw/e5-questions.json (đã kiểm: cả 100 câu trong DB còn version 1, PENDING)."""
    items = [it for it in lib.load_raw("e5-questions.json")["items"] if it["http"] == 200]
    return [(f"E5-{i:03d}", it["question"]) for i, it in enumerate(items, 1)]


def pack():
    qs = _questions()
    nopt = max(len(q["options"]) for _, q in qs)
    chunk_ids = sorted({c["chunk_id"] for _, q in qs for c in q.get("citations") or []})
    text = {r["id"]: r["text"] for r in lib.sql("select id::text, text from content_chunks where id in (" + ",".join(map(lib.lit, chunk_ids)) + ")")}
    letters = LETTERS[:nopt]
    PACK.parent.mkdir(exist_ok=True)
    with PACK.open("w", newline="", encoding="utf-8") as f:
        w = csv.writer(f)
        w.writerow(["stt", "ma", "chu_de", "de", *letters, "dap_an_danh_dau", "giai_thich", "nguon", "trich_doan",
                    "quyet_dinh (DUYỆT / SỬA / BỎ)", "ghi_chu", "sua_de", *[f"sua_{x}" for x in letters], "sua_dap_an", "sua_giai_thich",
                    "ly_do_bo (" + " / ".join(k for k in REJECT if k) + ")"])
        for i, (ma, q) in enumerate(qs, 1):
            opts = sorted(q["options"], key=lambda o: o["position"])
            key = [LETTERS[k] for k, o in enumerate(opts) if o["id"] in q["answer_key"]["option_ids"]]
            cites = q.get("citations") or []
            w.writerow([i, ma, q["topic"], q["stem"], *[o["body"] for o in opts], *[""] * (nopt - len(opts)), ",".join(key), q.get("explanation") or "",
                        "; ".join(f"{c['document_title']}, tr. {c['page']}" for c in cites),
                        "\n---\n".join(text.get(c["chunk_id"], "") for c in cites), "", "", "", *[""] * nopt, "", "", ""])
    lib.save_raw("e5-duyet-map.json", {"note": "mã E5-… của e5_duyet_goi.csv → id câu trong question_bank (lớp E5)", "map": {ma: q["id"] for ma, q in qs}})
    print(f"→ {PACK} ({len(qs)} câu, {len(chunk_ids)} trích đoạn)")


def _norm(s):
    return (s or "").strip().upper().replace("Ệ", "E").replace("Ử", "U").replace("Ỏ", "O").replace("Ư", "U").replace("Ơ", "O")


def apply():
    if not FILLED.exists():
        raise Fail(f"chưa có {FILLED}")
    raw = lib.load_raw("e5.json")
    cid, idmap = raw["course_id"], lib.load_raw("e5-duyet-map.json")["map"]
    with FILLED.open(encoding="utf-8-sig", newline="") as f:
        rows = list(csv.DictReader(f))
    col = lambda r, p: next((v for k, v in r.items() if k and k.startswith(p)), "")  # noqa: E731  — tiêu đề có phần chú thích trong ngoặc
    out = lib.load_raw("e5-apply.json") or {"items": {}}
    for r in rows:
        ma, dec = r["ma"].strip(), _norm(col(r, "quyet_dinh"))
        if not dec:
            continue
        if dec not in ("DUYET", "SUA", "BO"):
            raise Fail(f"{ma}: quyết định lạ «{col(r, 'quyet_dinh')}»")
        qid, tok = idmap[ma], login(TEACHER)
        st, q = call("GET", f"/courses/{cid}/questions/{qid}", tok)
        if st != 200:
            raise Fail(f"{ma}: GET → {st} {q}")
        if q["review_status"] != "PENDING":
            print(f"  {ma}: đã {q['review_status']}, bỏ qua")
            continue
        ver = q["version"]
        if dec == "SUA":
            opts = sorted(q["options"], key=lambda o: o["position"])
            bodies = [(r.get(f"sua_{LETTERS[k]}") or "").strip() or o["body"] for k, o in enumerate(opts)]
            key = (r.get("sua_dap_an") or "").strip().upper()
            correct = [LETTERS.index(x.strip()) for x in key.split(",") if x.strip()] if key else \
                [k for k, o in enumerate(opts) if o["id"] in q["answer_key"]["option_ids"]]
            body = {"type": q["type"], "title": q["title"], "topic": q["topic"], "difficulty": q["difficulty"],
                    "stem": (r.get("sua_de") or "").strip() or q["stem"], "options": [{"body": b, "pinned_last": o["pinned_last"]} for b, o in zip(bodies, opts)],
                    "correct": correct, "explanation": (r.get("sua_giai_thich") or "").strip() or q.get("explanation"), "version": ver}
            if body["stem"] == q["stem"] and bodies == [o["body"] for o in opts] and not key and body["explanation"] == q.get("explanation"):
                raise Fail(f"{ma}: SỬA nhưng không có cột sua_* nào khác bản gốc")
            st, q2 = call("PUT", f"/courses/{cid}/questions/{qid}", tok, body)
            if st != 200:
                raise Fail(f"{ma}: PUT → {st} {q2}")
            ver = q2["version"]
        review = {"decision": "REJECT" if dec == "BO" else "APPROVE", "version": ver}
        if dec == "BO":
            reason = col(r, "ly_do_bo").strip().upper()
            if reason not in REJECT:
                raise Fail(f"{ma}: lý do bỏ lạ «{reason}»")
            review["reject_reason"] = REJECT[reason]
        st, q3 = call("PUT", f"/courses/{cid}/questions/{qid}/review", tok, review)
        if st != 200:
            raise Fail(f"{ma}: review → {st} {q3}")
        out["items"][ma] = {"question_id": qid, "decision": dec, "reason": review.get("reject_reason"), "status": q3["review_status"], "version": q3["version"], "at": lib.now_iso()}
        lib.save_raw("e5-apply.json", out)
        print(f"  {ma}: {dec} → {q3['review_status']} v{q3['version']}")
    print(f"đã áp {len(out['items'])} câu → raw/e5-apply.json")


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
    obj = {"meta": lib.meta({"stack": "ep-qc (sinh) → ep-eval (duyệt)", "model": ", ".join(models) or prov.get("P1", {}).get("model"), "via": "9router", "plan": PLAN}),
           "status": status, "reason": None if status == "OK" else f"Người duyệt mới duyệt {reviewed}/{created} câu; AC7: ghi cỡ mẫu thật, không nội suy.",
           "generated": created, "skipped_by_reason": skipped, "review": tot, "n_reviewed": reviewed, "rates": rate,
           "llm_calls": len(calls), "latency_ms": lib.pct([a["latency_ms"] for a in calls]), "tokens_in": sum(a["tokens_in"] for a in calls), "tokens_out": sum(a["tokens_out"] for a in calls),
           "jobs": raw["jobs"], "rate_limited": raw.get("rate_limited", [])}
    # lý do loại theo lô: raw/e5-apply.json (quyết định người duyệt) ⨝ câu → lô (raw/e5.json)
    batch_of = {q: k for k, j in raw["jobs"].items() for q in (j.get("result") or {}).get("created") or []}
    reasons, by_batch_reason = {}, {}
    for it in ((lib.load_raw("e5-apply.json") or {}).get("items") or {}).values():
        if it["decision"] == "BO":
            reasons[it.get("reason") or "?"] = reasons.get(it.get("reason") or "?", 0) + 1
            by_batch_reason.setdefault(batch_of.get(it["question_id"]), {}).setdefault(it.get("reason") or "?", 0)
            by_batch_reason[batch_of.get(it["question_id"])][it.get("reason") or "?"] += 1
    obj["rejected_by_reason"] = reasons
    obj["usable_rate"] = round((tot["unchanged"] + tot["edited"]) / reviewed, 4) if reviewed else None
    qs = [it["question"] for it in (lib.load_raw("e5-questions.json") or {}).get("items") or [] if it.get("question")]
    key_a = sum(sorted(q["options"], key=lambda o: o["position"])[0]["id"] in q["answer_key"]["option_ids"] for q in qs)
    leak_cn = sum(bool(re.search(r"\[C\d+\]", (q.get("explanation") or "") + q["stem"])) for q in qs)
    pages = {}
    for it in (lib.load_raw("e5-questions.json") or {}).get("items") or []:
        pages.setdefault(it["batch"], []).extend(c["page"] for c in it["question"].get("citations") or [])
    obj["findings_data"] = {"answer_key_A": key_a, "citation_marker_in_text": leak_cn, "n": len(qs), "cited_pages": {k: [min(v), max(v)] for k, v in sorted(pages.items()) if v}}
    lib.svg_bars("e5.svg", [("duyệt nguyên", [tot["unchanged"]]), ("sửa nhẹ", [tot["edited"]]), ("loại", [tot["rejected"]]), ("chờ duyệt", [tot["pending"]])], ["số câu"],
                 f"E5 — {created} câu AI sinh: kết quả duyệt (n duyệt = {reviewed})", "câu", fmt="{:.0f}")
    pct = lambda x: "–" if rate is None else f"{rate[x] * 100:.1f} %".replace(".", ",")  # noqa: E731
    iso = lambda s: datetime.fromisoformat(s.replace("Z", "+00:00"))  # noqa: E731
    build_of = lambda j: lib.build_of(j.get("gateway_image"), j["t0"])  # noqa: E731
    L = ["# E5 — câu hỏi AI sinh", "", f"Trạng thái: **{status}**. Sinh trên stack `ep-qc` (2026-11-15; bản gateway của từng lô ở cột «bản dựng», sha do người dựng xác nhận, image không mang sha); "
         "duyệt trên stack `ep-eval` (VPS, 2026-11-25: DB `ep-qc` dump 2026-11-25 + gateway 708dc729, `benchmarks/eval/ep-eval.compose.yml`). "
         f"Bộ đo `benchmarks/eval/e5.py` @ `{obj['meta']['commit']}`, {obj['meta']['timestamp'][:10]}. Model sinh: `{obj['meta']['config']['model']}` qua 9router.", "",
         f"- Sinh: **{created}** câu từ {len(raw['jobs'])} lô (10 câu / lô, MCQ một đáp án, mức MEDIUM, nguồn: «Network Security Threats», «Forecasting (QMB ch. 6b)»). Bỏ: " + (", ".join(f"{k} {v}" for k, v in skipped.items()) or "không") + ".",
         f"- Duyệt: **n = {reviewed}** — duyệt nguyên {tot['unchanged']} ({pct('unchanged')}), sửa nhẹ {tot['edited']} ({pct('edited')}), loại {tot['rejected']} ({pct('rejected')}); chờ duyệt {tot['pending']}. "
         + (f"**Dùng được (duyệt nguyên + sửa nhẹ): {obj['usable_rate'] * 100:.1f} %**".replace(".", ",") if reviewed else "")
         + (". Loại theo lý do: " + ", ".join(f"{k} {v}" for k, v in sorted(reasons.items(), key=lambda x: -x[1])) if reasons else "") + ".",
         f"- Lời gọi QUESTION_GEN (llm_audit): {len(calls)}, p50 {obj['latency_ms'].get('p50')} ms, p95 {obj['latency_ms'].get('p95')} ms; token vào {obj['tokens_in']}, ra {obj['tokens_out']}.", "",
         "## Từng lô", "", "| lô | chủ đề | bản dựng | POST (UTC) | xong (UTC) | giây | trạng thái | câu tạo | bỏ | lỗi | POST bị 429 trước lô | lời gọi LLM (trạng thái) |",
         "| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |"]
    rl = {}
    for x in raw.get("rate_limited", []):
        rl.setdefault(x["batch"], []).append(x["at"][11:19])
    secs = lambda j: round((iso(j["t1"]) - iso(j["t0"])).total_seconds()) if j.get("t1") else "–"  # noqa: E731
    for k, j in sorted(raw["jobs"].items()):
        res, au = j.get("result") or {}, j.get("audit") or []
        st_count = {}
        for a in au:
            st_count[a["status"]] = st_count.get(a["status"], 0) + 1
        n_created = res.get("created") if isinstance(res.get("created"), int) else len(res.get("created") or [])
        L.append(f"| {k} | {j['topic']} | {build_of(j)} | {j['t0'][11:19]} | {(j.get('t1') or '')[11:19] or '–'} | {secs(j)} | {j.get('status') or 'đang chạy'} | {n_created} | {len(res.get('skipped') or [])} | "
                 f"{(j.get('error') or {}).get('code') if isinstance(j.get('error'), dict) else (j.get('error') or '–')} | {', '.join(rl.get(k, [])) or '–'} | "
                 + (", ".join(f"{s} {n}" for s, n in sorted(st_count.items())) or "–") + " |")
    for k in sorted(set(rl) - set(raw["jobs"])):
        L.append(f"| {k} | {dict((f'lo{i + 1:02d}', t) for i, (_, t) in enumerate(PLAN))[k]} | – | – | – | – | chưa nhận (429) | 0 | 0 | – | {', '.join(rl[k])} | – |")
    if reviewed:
        L += ["", "## Kết quả duyệt theo lô", "", "| lô | chủ đề | duyệt nguyên | sửa nhẹ | loại | lý do loại | trang trích dẫn |", "| --- | --- | --- | --- | --- | --- | --- |"]
        for k, j in sorted(raw["jobs"].items()):
            g, pg = j.get("gen_stats") or {}, obj["findings_data"]["cited_pages"].get(k)
            L.append(f"| {k} | {j['topic']} | {g.get('unchanged', '–')} | {g.get('edited', '–')} | {g.get('rejected', '–')} | "
                     + (", ".join(f"{r} {n}" for r, n in sorted(by_batch_reason.get(k, {}).items(), key=lambda x: -x[1])) or "–") + f" | {f'{pg[0]}–{pg[1]}' if pg else '–'} |")
    if lib.builds():
        L += ["", "Bản dựng (`raw/builds.json`): " + "; ".join(f"`{b['sha']}` = image `{b['image']}` tạo {b['created'][:19]}Z ({b['source']})" for b in lib.builds()) + "."]
    fd = obj["findings_data"]
    L += ["", "![](e5.svg)", "", "## Phát hiện của người duyệt", "",
          "Người duyệt ghi khi duyệt; số trong ngoặc do Tech Lead kiểm lại trên dữ liệu (2026-11-25).", "",
          "1. Lô 09 «Làm trơn hàm mũ» và lô 10 «Đo sai số dự báo»: tài liệu «Forecasting (QMB ch. 6b)» không có hai chủ đề này mà máy vẫn sinh đủ 10 câu / lô "
          "(kiểm: 19 đoạn, 23 trang của tài liệu trong `content_chunks` không có «smoothing», «forecast error», «MAD», «mean absolute deviation»). Hệ thống chưa từ chối khi chủ đề không có trong nguồn.",
          "2. Lô 03–05 (trinh sát, truy cập, từ chối dịch vụ) chỉ trích trang ≤ 50 của «Network Security Threats» "
          f"(kiểm: tài liệu 141 trang; trích dẫn lô 01–06 nằm trong trang {min(v[0] for k, v in fd['cited_pages'].items() if k <= 'lo06')}–{max(v[1] for k, v in fd['cited_pages'].items() if k <= 'lo06')}, "
          "trong khi «reconnaissance» có ở trang 46, 69, 72, 79, 91, 119; «denial of service» ở 17, 46, 60, 69, 79, 91; «access attack» ở 46, 69, 79, 91): phần sau trang 50 của tài liệu không được dùng.",
          f"3. Khoảng 45 ý khác nhau cho 100 câu (ước lượng của người duyệt): nhiều câu hỏi lại cùng một ý; {reasons.get('TRUNG_LAP', 0)} câu bị loại vì `TRUNG_LAP`.",
          f"4. Đáp án đúng nằm ở phương án A trong **{fd['answer_key_A']}/{fd['n']}** câu (thứ tự phương án như máy sinh).",
          f"5. **{fd['citation_marker_in_text']}/{fd['n']}** câu lộ mã trích dẫn nội bộ `[Cn]` trong chữ hiện cho người đọc (giải thích; 1 câu cả trong đề).", "",
          "## Cách duyệt", "",
          f"Người duyệt {lib.LABELER} duyệt mù trên gói `benchmarks/eval/human/e5_duyet_goi.csv` (đề, phương án, đáp án đánh dấu, giải thích, nguồn + trích đoạn; không có điểm / nhận định của máy), "
          "điền `benchmarks/eval/human/e5_duyet.csv` (DUYỆT / SỬA kèm bản sửa / BỎ kèm lý do). `e5.py apply` áp từng quyết định qua API của `ep-eval` bằng vai giảng viên "
          "(DUYỆT → duyệt; SỬA → lưu bản sửa rồi duyệt, gen-stats tính «sửa nhẹ»; BỎ → loại kèm lý do) → `raw/e5-apply.json`; `e5.py report` đọc gen-stats từng lô.", "",
         "## Giới hạn", "", "- Dữ liệu mô phỏng (D44): tài liệu seed lớp 1; chưa kiểm chứng với câu hỏi giảng viên thật dùng trong lớp.",
         "- Tỉ lệ chỉ tính trên câu đã duyệt (AC7); chưa duyệt đủ 100 câu thì ghi cỡ mẫu thật, không nội suy.", f"- Người duyệt: {lib.LABELER}; một người, không có người duyệt thứ hai."]
    lib.write_report("e5", obj, L)


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("cmd", choices=["run", "report", "export", "pack", "apply"])
    a = ap.parse_args()
    try:
        if a.cmd == "run":
            lib.refuse_demo()
            if not lib.ready():
                lib.die("Gateway chưa chạy")
            run()
        elif a.cmd == "export":
            export()
        elif a.cmd == "pack":
            pack()
        elif a.cmd == "apply":
            apply()
        else:
            report()
    except Fail as e:
        lib.die(f"LỖI: {e}", 2)


if __name__ == "__main__":
    main()
