#!/usr/bin/env python3
"""E2 — ngưỡng escalation + hiệu chỉnh RAG_SIM_FLOOR / CEIL / trọng số độ tin cậy (US-P10-10 AC4, P-11-4, P-11-19, P-11-22). E4 dùng lại `run` cho từng model.

  e2.py run --tag P1 [--limit N] [--keep]   hỏi 120 câu vàng + 20 câu ngoài chủ đề qua Threads của lớp 1 (Sinh viên), đọc `ThreadPost.confidence` (Giảng viên),
                                            ghi raw/e2-<tag>.json (tiếp tục được); xong thì xoá mềm các thread đo (trừ --keep)
  e2.py report                              tính đường cong + hiệu chỉnh từ mọi raw/e2-*.json → e2.json / e2.md / e2-*.svg (không gọi mạng)

Chấm tất định (`norm` / `passes` của benchmarks/e2/check_golden.py), không LLM làm giám khảo. Độ tin cậy giả định (counterfactual) tính lại thuần từ
cos_top1 (đọc DB chỉ-đọc, API không trả) và groundedness suy ngược từ confidence đã lưu: conf = w·retr + (1−w)·ground, retr = clamp((cos−floor)/(ceil−floor)).
"""
import argparse
import json
import sys
import time
from pathlib import Path

import lib
from lib import Fail, call, lit, login

sys.path.insert(0, str(lib.ROOT / "benchmarks/e2"))
from check_golden import norm, passes  # noqa: E402 — định nghĩa chấm duy nhất (README E2)

GOLDEN = lib.ROOT / "benchmarks/e2/golden_e2.jsonl"
OFFTOPIC = lib.HERE / "data/e2_offtopic.jsonl"
STUDENT, TEACHER = "sv.kha", "teacher"
# Hằng số máy chủ lúc đo (internal/agent: RagSimFloor, RagSimCeil, Score; courses.escalation_threshold mặc định)
FLOOR0, CEIL0, W0, T0 = 0.25, 0.65, 0.6, 0.80
REFUSAL = norm("chưa tìm thấy nội dung này trong tài liệu của lớp")
THRESHOLDS = [round(0.30 + 0.05 * i, 2) for i in range(13)]  # 0,30 → 0,90 (AC4: ≥ 7 điểm)


def items():
    out = [json.loads(x) for x in GOLDEN.read_text(encoding="utf-8").splitlines() if x.strip()]
    return out + [json.loads(x) for x in OFFTOPIC.read_text(encoding="utf-8").splitlines() if x.strip()]


# ---- chạy -----------------------------------------------------------------------------------------------------------------
def wait_answer(cid, tid, timeout=240):
    t = login(TEACHER)
    end = time.time() + timeout
    while time.time() < end:
        st, v = call("GET", f"/courses/{cid}/threads/{tid}", t)
        if st == 200 and v["thread"].get("ai_state") in ("ANSWERED", "SKIPPED"):
            ai = next((p for p in v["posts"] if p["kind"] == "AI"), None)
            return {"state": v["thread"]["ai_state"], "skip_reason": v["thread"].get("ai_skip_reason") or None,
                    "answer": ai["body"] if ai else None, "confidence": ai.get("confidence") if ai else None, "citations": len(ai["citations"]) if ai else 0}
        time.sleep(1.5)
    return {"state": "TIMEOUT"}


def run(tag, limit, keep):
    raw = lib.load_raw(f"e2-{tag}.json") or {"tag": tag, "items": {}, "windows": []}
    stu = login(STUDENT)
    cid = lib.course_by_code(stu, lib.CLASS1)
    t0, todo = lib.now_iso(), [it for it in items() if raw["items"].get(it["id"], {}).get("state") not in ("ANSWERED", "SKIPPED", "BLOCKED")]
    print(f"E2 {tag}: {len(todo)} câu cần hỏi")
    try:
        for n, it in enumerate(todo[: limit or None], 1):
            q = it["question"]
            t = time.time()
            st, b = call("POST", f"/courses/{cid}/threads", login(STUDENT), {"title": q[:200], "body": q}, idem=f"e2-{tag}-{it['id']}")
            rec = {"set": it["set"], "answerable": it["answerable"], "http": st}
            if st == 422:
                rec.update(state="BLOCKED", code=(b or {}).get("code"))
            elif st != 201:
                rec.update(state="ERROR", code=(b or {}).get("code"))
            else:
                rec["tid"] = b["thread"]["id"]
                rec.update(wait_answer(cid, rec["tid"]))
                rec["client_ms"] = round((time.time() - t) * 1000)
            raw["items"][it["id"]] = rec
            lib.save_raw(f"e2-{tag}.json", raw)
            print(f"  {n}/{len(todo)} {it['id']} {rec['state']} conf={rec.get('confidence')} {rec.get('client_ms', '')}ms")
    except lib.Stop as e:
        raw["stopped"] = str(e)
        print(f"DỪNG: {e}")
    raw["windows"].append([t0, lib.now_iso()])
    tids = [r["tid"] for r in raw["items"].values() if r.get("tid")]
    raw["cos"] = {r["tid"]: r["cos"] for r in cos_top1(tids)}
    raw["course_id"] = cid
    raw["audit"] = [a for w in raw["windows"] for a in lib.audit(w[0], w[1], course=cid) if a["task"] in ("CHAT", "EMBEDDING")]
    lib.save_raw(f"e2-{tag}.json", raw)
    if not keep:
        for tid in tids:
            call("DELETE", f"/courses/{cid}/threads/{tid}", login(STUDENT))
    return raw


def cos_top1(tids):
    """cos_top1 = max cosine giữa vector thread (chính là vector truy vấn của Threads, answer.go) và đoạn sinh viên được thấy — cùng bộ lọc với RagSearch."""
    if not tids:
        return []
    arr = "array[" + ",".join(lit(t) for t in tids) + "]::uuid[]"
    return lib.sql(f"""select t.id::text as tid, round(max(1 - (c.embedding <=> t.embedding))::numeric, 4)::float as cos
        from forum_threads t join content_chunks c on c.course_ids @> array[t.course_id] join documents d on d.id = c.document_id
        where t.id = any({arr}) and t.embedding is not null and c.embedding is not null and d.status = 'READY' and d.use_for_rag
          and d.type <> 'ANSWER_KEY' and d.visible_to_students and c.audience = 'ALL'
        group by t.id""")


# ---- chấm + hiệu chỉnh (thuần) ---------------------------------------------------------------------------------------------
def clamp(x):
    return max(0.0, min(1.0, x))


def retr(cos, floor, ceil):
    return round(clamp((cos - floor) / (ceil - floor)), 3)


def label(it, rec, cos):
    """Một dòng phân tích: good = câu trả lời hiện ra sẽ ĐÚNG; mọi thứ khác là bad (sai / bịa / từ chối / không trả lời)."""
    ans = rec.get("answer") or ""
    refusal = REFUSAL in norm(ans)
    good = bool(it["answerable"] and rec["state"] == "ANSWERED" and not refusal and passes(ans, it))
    row = {"id": it["id"], "set": it["set"], "answerable": it["answerable"], "state": rec["state"], "refusal": refusal, "good": good, "cos": cos,
           "confidence": float(rec["confidence"]) if rec.get("confidence") not in (None, "") else None,
           "fabricated": bool(not it["answerable"] and rec["state"] == "ANSWERED" and not refusal)}
    if row["confidence"] is not None and cos is not None:
        r0 = retr(cos, FLOOR0, CEIL0)
        row["ground"] = round((row["confidence"] - W0 * r0) / (1 - W0), 3)  # suy ngược; kiểm nhất quán ở report
    return row


def escalated(row, floor, ceil, w, t):
    """Theo chính sách chat (SRS 4.8 + P-11-18): không trả lời / cos < sàn / lời từ chối → escalate; còn lại conf < ngưỡng."""
    if row["state"] != "ANSWERED" or row["refusal"] or row["cos"] is None or row["cos"] < floor:
        return True
    conf = row["confidence"] if (floor, ceil, w) == (FLOOR0, CEIL0, W0) else w * retr(row["cos"], floor, ceil) + (1 - w) * clamp(row.get("ground", 0.0))
    return round(conf, 3) < t


def point(rows, floor, ceil, w, t):
    esc = [escalated(r, floor, ceil, w, t) for r in rows]
    shown = [r for r, e in zip(rows, esc) if not e]
    good = [e for r, e in zip(rows, esc) if r["good"]]
    bad = [e for r, e in zip(rows, esc) if not r["good"]]
    return {"t": t, "n": len(rows), "escalate_rate": round(sum(esc) / len(rows), 4) if rows else None,
            "error_rate": round(sum(1 for r in shown if not r["good"]) / len(shown), 4) if shown else 0.0,
            "false_ticket_rate": round(sum(good) / len(good), 4) if good else None, "catch_rate": round(sum(bad) / len(bad), 4) if bad else None}


def grid():
    for f100 in range(25, 71, 5):
        for span in range(10, 41, 5):
            for w in (0.4, 0.5, 0.6, 0.7, 0.8):
                f, c = f100 / 100, (f100 + span) / 100
                if c <= 0.98:
                    yield f, c, w


def fit(rows, t=T0):
    """Chọn (floor, ceil, w) ở ngưỡng t cố định (P-11-22: không hạ 0,80): «ticket giả» thấp nhất mà «bắt câu sai» KHÔNG kém cấu hình hiện tại
    trên cùng tập; hoà → bắt nhiều hơn, rồi escalate ít hơn, rồi gần cấu hình cũ (ít đổi nhất)."""
    base = point(rows, FLOOR0, CEIL0, W0, t)["catch_rate"] or 0
    best = None
    for f, c, w in grid():
        p = point(rows, f, c, w, t)
        if p["false_ticket_rate"] is None or (p["catch_rate"] or 0) < base:
            continue
        key = (-p["false_ticket_rate"], p["catch_rate"] or 0, -p["escalate_rate"], -(abs(f - FLOOR0) + abs(c - CEIL0) + abs(w - W0)))
        if best is None or key > best[0]:
            best = (key, {"floor": f, "ceil": c, "w": w, **p})
    return best[1] if best else None


def analyse(tag, raw):
    it_by = {it["id"]: it for it in items()}
    rows, excluded = [], {}
    for iid, rec in sorted(raw["items"].items()):
        if rec["state"] in ("ANSWERED", "SKIPPED") and rec.get("skip_reason") != "LLM_UNAVAILABLE":
            rows.append(label(it_by[iid], rec, raw.get("cos", {}).get(rec.get("tid"))))
        else:
            excluded[iid] = rec["state"] + (f" {rec.get('code') or rec.get('skip_reason') or ''}").rstrip()
    golden = [r for r in rows if r["set"] != "offtopic"]
    incons = [r["id"] for r in rows if "ground" in r and not -0.01 <= r["ground"] <= 1.01]
    dist = {name: lib.pct([r["cos"] for r in rows if r["cos"] is not None and pred(r)])
            for name, pred in (("in_answerable", lambda r: r["answerable"]), ("out_golden_unanswerable", lambda r: not r["answerable"] and r["set"] != "offtopic"),
                               ("out_offtopic", lambda r: r["set"] == "offtopic"))}
    curve_now = [point(golden, FLOOR0, CEIL0, W0, t) for t in THRESHOLDS]
    rec_all = fit(rows)
    odd = [r for r in rows if int(r["id"][-2:]) % 2]
    even = [r for r in rows if not int(r["id"][-2:]) % 2]
    half = {}
    for name, a, b in (("fit_odd_eval_even", odd, even), ("fit_even_eval_odd", even, odd)):
        f = fit(a)
        half[name] = {"params": f and {k: f[k] for k in ("floor", "ceil", "w")}, "fit": f and {k: f[k] for k in ("false_ticket_rate", "catch_rate")},
                      "holdout": f and point(b, f["floor"], f["ceil"], f["w"], T0)}
    out = {"tag": tag, "n_rows": len(rows), "n_golden": len(golden), "excluded": excluded, "inconsistent_ground": incons,
           "accuracy_answerable": round(sum(r["good"] for r in golden if r["answerable"]) / max(1, sum(1 for r in golden if r["answerable"])), 4),
           "fabrication_unanswerable": round(sum(r["fabricated"] for r in rows if not r["answerable"]) / max(1, sum(1 for r in rows if not r["answerable"])), 4),
           "cos_distribution": dist, "curve_current": curve_now,
           "current_at_t0": point(rows, FLOOR0, CEIL0, W0, T0), "recommended": rec_all, "split_check": half,
           "curve_recommended": [point(golden, rec_all["floor"], rec_all["ceil"], rec_all["w"], t) for t in THRESHOLDS] if rec_all else None,
           "leaked_at_t0": [r["id"] for r in rows if not r["good"] and not escalated(r, FLOOR0, CEIL0, W0, T0)],
           "false_tickets_at_t0": [r["id"] for r in rows if r["good"] and escalated(r, FLOOR0, CEIL0, W0, T0)],
           "rows": rows}
    return out


def report():
    tags = sorted(p.stem[3:] for p in lib.RAW.glob("e2-P*.json"))
    if not tags:
        raise Fail("chưa có raw/e2-*.json (chạy `e2.py run` trước)")
    res = {t: analyse(t, lib.load_raw(f"e2-{t}.json")) for t in tags}
    prov = json.loads((lib.RAW / "providers.json").read_text()) if (lib.RAW / "providers.json").exists() else {}
    embed = f"thật `gemini-embedding-001` (1536 chiều) trong lượt đo — {prov.get('embed_real_calls', '?')} lời gọi nhúng (llm_audit), xong đã trả về nhúng giả" if prov.get("embed_real_calls") else prov.get("embedding", "?")
    status = "OK" if all(r["n_golden"] >= 100 for r in res.values()) else "PARTIAL"
    allrows = [r for v in res.values() for r in v["rows"]]
    comb = fit(allrows)  # MỘT cấu hình cho cả hai model (FLOOR / CEIL thuộc model nhúng, w thuộc công thức) — đây là số đề xuất cho dev
    if comb:
        same = point(allrows, comb["floor"], comb["ceil"], comb["w"], T0)
        fl = [f / 1000 for f in range(250, 701, 5) if f / 1000 < comb["ceil"]]
        comb["floor_same_range"] = [min(x for x in fl if point(allrows, x, comb["ceil"], comb["w"], T0) == same), max(x for x in fl if point(allrows, x, comb["ceil"], comb["w"], T0) == same)]
        comb["per_provider"] = {t: point(v["rows"], comb["floor"], comb["ceil"], comb["w"], T0) for t, v in res.items()}
        comb["current_per_provider"] = {t: v["current_at_t0"] for t, v in res.items()}
    obj = {"meta": lib.meta({"stack": "ep-qc", "providers": prov, "embedding": embed, "constants_at_run": {"RAG_SIM_FLOOR": FLOOR0, "RAG_SIM_CEIL": CEIL0, "w_retr": W0, "escalation_threshold": T0}}),
           "status": status, "labels": (lib.human_review("e2") or {}).get("status", "WAITING_HUMAN"), "human_review": lib.human_review("e2"),
           "labels_note": "Nhãn do BA soạn; chủ dự án soát 20 % (24 mục): gói e2-soat-nhan.csv → benchmarks/eval/human/e2_soat.csv.",
           "thresholds": THRESHOLDS, "recommendation": comb, "by_provider": {t: {k: v for k, v in r.items() if k != "rows"} for t, r in res.items()}}
    lib.save_raw("e2-rows.json", {t: r["rows"] for t, r in res.items()})
    xs = THRESHOLDS
    for t, r in res.items():
        series = [("escalate-rate (hiện tại)", [(p["t"], p["escalate_rate"]) for p in r["curve_current"]]), ("error-rate (hiện tại)", [(p["t"], p["error_rate"]) for p in r["curve_current"]])]
        if r["curve_recommended"]:
            series += [("escalate-rate (đề xuất)", [(p["t"], p["escalate_rate"]) for p in r["curve_recommended"]]), ("error-rate (đề xuất)", [(p["t"], p["error_rate"]) for p in r["curve_recommended"]])]
        lib.svg_lines(f"e2-curve-{t}.svg", series, f"E2 — {prov.get(t, {}).get('model', t)}: tỉ lệ escalate / lỗi theo ngưỡng", "ngưỡng độ tin cậy", "tỉ lệ", 0.30, 0.90, xticks=xs, vline=(T0, "0,80 hiện hành"))
    first = res[tags[0]]
    lib.svg_bars("e2-cos.svg", [(k.replace("_", " "), [v.get("p50"), v.get("p95"), v.get("max")]) for k, v in first["cos_distribution"].items()], ["p50", "p95", "max"],
                 "E2 — phân bố cos_top1 (gemini-embedding-001)", "cos", ymax=1.0)
    lib.write_report("e2", obj, md(obj, res, prov))


def fmt(x, d=3):
    return "–" if x is None else f"{x:.{d}f}".replace(".", ",")


def md(obj, res, prov):
    L = ["# E2 — ngưỡng escalation và hiệu chỉnh độ tin cậy", "",
         f"Trạng thái: **{obj['status']}** · nhãn: **{obj['labels']}** (BA soạn; chủ dự án soát 24 mục, `HUONG-DAN-CHU-DU-AN.md`). Stack `ep-qc` (sha bản gateway không ghi lúc đo); báo cáo dựng bằng bộ đo @ `{obj['meta']['commit']}`, {obj['meta']['timestamp'][:10]}. "
         f"Nhúng: {obj['meta']['config']['embedding']}. Hằng số lúc đo: RAG_SIM_FLOOR 0,25 · RAG_SIM_CEIL 0,65 · conf = 0,6·retr + 0,4·ground · ngưỡng lớp 0,80. "
         "Bảng theo ngưỡng tính trên câu vàng; bảng trước–sau và đề xuất tính trên câu vàng + 20 câu ngoài chủ đề.", ""]
    rc = obj["recommendation"]
    if rc:
        L += ["## Đề xuất cấu hình (một bộ số cho cả hai model; dev áp qua env, không viết cứng — P-11-19)", "",
              f"`RAG_SIM_FLOOR` = **{fmt(rc['floor'], 2)}** (mọi giá trị trong [{fmt(rc['floor_same_range'][0], 3)}; {fmt(rc['floor_same_range'][1], 3)}] cho cùng kết quả trên bộ này), "
              f"`RAG_SIM_CEIL` = **{fmt(rc['ceil'], 2)}**, trọng số truy xuất w = **{fmt(rc['w'], 1)}** (conf = {fmt(rc['w'], 1)}·retr + {fmt(1 - rc['w'], 1)}·ground), ngưỡng lớp giữ **0,80**.", "",
              "| model | cấu hình | ticket giả | bắt câu sai | escalate | lỗi lọt |", "| --- | --- | --- | --- | --- | --- |"]
        for t in rc["per_provider"]:
            a, b = rc["current_per_provider"][t], rc["per_provider"][t]
            L += [f"| {t} | hiện tại | {fmt(a['false_ticket_rate'])} | {fmt(a['catch_rate'])} | {fmt(a['escalate_rate'])} | {fmt(a['error_rate'])} |",
                  f"| {t} | đề xuất | {fmt(b['false_ticket_rate'])} | {fmt(b['catch_rate'])} | {fmt(b['escalate_rate'])} | {fmt(b['error_rate'])} |"]
        L += ["", "Sàn không tách được câu ngoài chủ đề khỏi câu trong tài liệu với model nhúng này: cos của câu ngoài chủ đề lên tới 0,60 trong khi câu trả lời đúng thấp nhất ≈ 0,57 (bảng phân bố). "
              "Câu ngoài tài liệu được chặn nhờ câu từ chối (P-11-18) và độ tin cậy, không nhờ sàn; sàn cao hơn ≈ 0,57 bắt đầu bỏ câu đúng.", "",
              "**Quyết định PM (P-11-33, 2026-11-15):** giữ tham số hiện hành (FLOOR 0,25 · CEIL 0,65 · w 0,6 · ngưỡng 0,80); đóng P-11-19, P-11-22. Bảng «đề xuất» ở trên chỉ để tham khảo.", ""]
    for t, r in res.items():
        p = prov.get(t, {})
        L += [f"## {t}: `{p.get('model', '?')}` qua {p.get('via', '9router')}", "",
              f"- {r['n_golden']} câu vàng + {r['n_rows'] - r['n_golden']} câu ngoài chủ đề phân tích được; loại {len(r['excluded'])}: " + (", ".join(f"{k} ({v})" for k, v in r["excluded"].items()) or "không") + ".",
              f"- Đúng trên câu trả lời được: **{fmt(r['accuracy_answerable'])}**; bịa trên câu không trả lời được: **{fmt(r['fabrication_unanswerable'])}**.",
              f"- Groundedness suy ngược ngoài [0, 1] (cos đo lại lệch với cos lúc trả lời): {len(r['inconsistent_ground'])} mục" + (f" ({', '.join(r['inconsistent_ground'][:10])})" if r["inconsistent_ground"] else "") + ".",
              f"- Ở 0,80 hiện hành — câu sai lọt tới sinh viên: {', '.join(r['leaked_at_t0']) or 'không'}; câu đúng bị đẩy ticket: {', '.join(r['false_tickets_at_t0']) or 'không'}.", "",
              f"![](e2-curve-{t}.svg)", "", "| ngưỡng | escalate (hiện tại) | lỗi lọt (hiện tại) | ticket giả trong tài liệu | bắt được câu sai | escalate (đề xuất) | lỗi lọt (đề xuất) |", "| --- | --- | --- | --- | --- | --- | --- |"]
        for i, c in enumerate(r["curve_current"]):
            rr = (r["curve_recommended"] or [{}] * len(THRESHOLDS))[i]
            L.append(f"| {fmt(c['t'], 2)} | {fmt(c['escalate_rate'])} | {fmt(c['error_rate'])} | {fmt(c['false_ticket_rate'])} | {fmt(c['catch_rate'])} | {fmt(rr.get('escalate_rate'))} | {fmt(rr.get('error_rate'))} |")
        cd = r["cos_distribution"]
        L += ["", "Phân bố cos_top1 (P-11-19):", "", "| nhóm | n | p50 | p95 | max |", "| --- | --- | --- | --- | --- |"]
        L += [f"| {k} | {v.get('n')} | {fmt(v.get('p50'))} | {fmt(v.get('p95'))} | {fmt(v.get('max'))} |" for k, v in cd.items()]
        a, rc = r["current_at_t0"], r["recommended"]
        L += ["", "Trước–sau ở ngưỡng 0,80 (P-11-22: ngưỡng giữ nguyên; chọn floor / ceil / w cho «ticket giả» thấp nhất mà «bắt câu sai» không kém cấu hình hiện tại):", "",
              "| cấu hình | floor | ceil | w_retr | ticket giả | bắt câu sai | escalate | lỗi lọt |", "| --- | --- | --- | --- | --- | --- | --- | --- |",
              f"| hiện tại | 0,25 | 0,65 | 0,6 | {fmt(a['false_ticket_rate'])} | {fmt(a['catch_rate'])} | {fmt(a['escalate_rate'])} | {fmt(a['error_rate'])} |"]
        if rc:
            L.append(f"| đề xuất | {fmt(rc['floor'], 2)} | {fmt(rc['ceil'], 2)} | {fmt(rc['w'], 1)} | {fmt(rc['false_ticket_rate'])} | {fmt(rc['catch_rate'])} | {fmt(rc['escalate_rate'])} | {fmt(rc['error_rate'])} |")
        else:
            L.append("| đề xuất | – | – | – | không cấu hình nào giữ ticket giả ≤ 20 % | | | |")
        for name, h in r["split_check"].items():
            if h["params"]:
                L.append(f"| kiểm chéo {name} | {fmt(h['params']['floor'], 2)} | {fmt(h['params']['ceil'], 2)} | {fmt(h['params']['w'], 1)} | {fmt(h['holdout']['false_ticket_rate'])} | {fmt(h['holdout']['catch_rate'])} | {fmt(h['holdout']['escalate_rate'])} | {fmt(h['holdout']['error_rate'])} |")
        L.append("")
    L += ["![](e2-cos.svg)", "", "## Định nghĩa", "",
          "- **good**: câu trả lời được (`answerable`), AI trả lời, không phải câu từ chối, đủ mọi nhóm `must_contain`. Mọi trường hợp khác là **bad** (sai, thiếu ý, bịa, từ chối, không trả lời).",
          "- **escalate** ở ngưỡng t: Threads bỏ qua (SKIPPED) hoặc cos_top1 < sàn hoặc câu từ chối (P-11-18) hoặc độ tin cậy < t — đúng chính sách chat riêng (SRS 4.8).",
          "- **escalate-rate** = escalate / tổng; **lỗi lọt** (error-rate) = bad không escalate / số câu hiện cho sinh viên; **ticket giả trong tài liệu** = good bị escalate / good; **bắt câu sai** = bad bị escalate / bad.",
          "- Độ tin cậy giả định: retr = clamp((cos − floor) / (ceil − floor)); ground suy ngược = (conf − 0,6·retr₀) / 0,4 với retr₀ ở 0,25 / 0,65; conf′ = w·retr′ + (1 − w)·ground. Không gọi lại LLM.", "",
          "## Giới hạn", "",
          "- Dữ liệu mô phỏng (D44): câu hỏi do BA soạn từ tài liệu seed lớp 1, chưa kiểm chứng trên câu hỏi thật của sinh viên. Nhãn **chưa có người ngoài đội soát** (`WAITING_HUMAN`).",
          "- Chấm `must_contain` so chuỗi: câu đúng diễn đạt khác biến thể bị tính sai — sai số thuộc phép đo (README E2), không sửa nhãn cho vừa kết quả.",
          "- cos_top1 đọc từ DB chỉ-đọc (API không trả cos): max cos trên mọi đoạn sinh viên được thấy, có thể lệch nhẹ với `BestCosine(hits)` của top-K sau RRF; mục lệch được đếm ở trên.",
          "- Hiệu chỉnh khớp trên chính bộ 140 mục: kiểm chéo nửa–nửa ở bảng trên cho thấy độ ổn định; đề xuất là **số cấu hình** cho dev áp (env, không viết cứng — P-11-19), cần đo lại sau khi áp.",
          "- Số đo LLM qua 9router (thuê bao): độ trễ gồm cả proxy. Câu Threads chạy làn NEAR_REALTIME, một lời gọi sinh chữ mỗi câu (D47)."]
    return L


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    sp = ap.add_subparsers(dest="cmd", required=True)
    r = sp.add_parser("run")
    r.add_argument("--tag", required=True)
    r.add_argument("--limit", type=int, default=0)
    r.add_argument("--keep", action="store_true")
    sp.add_parser("report")
    a = ap.parse_args()
    try:
        if a.cmd == "run":
            lib.refuse_demo()
            if not lib.ready():
                lib.die("Gateway chưa chạy")
            run(a.tag, a.limit, a.keep)
        else:
            report()
    except Fail as e:
        lib.die(f"LỖI: {e}", 2)


if __name__ == "__main__":
    main()
