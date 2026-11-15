#!/usr/bin/env python3
"""Đo «học từ lần sửa» (FEAT-grading-guidance M1, US-GD-10 AC5, AC6, AC10): giao thức E3 trước / sau trên nhãn người (`teacher_label`). Chỉ thư viện chuẩn, chỉ HTTP (D46): chạy như một client ngoài.

  bash benchmarks/guidance/run.sh --labels <labels.csv> [--drafts <thư mục>]     báo cáo trước / sau → benchmarks/reports/guidance-before-after.{json,md,svg}; dòng kết luận «Bật được» | «Giữ tắt»
  bash benchmarks/guidance/run.sh --stats                                          chụp số đo theo phiên bản từ `…/guidance/stats` (n, MAE, lệch TB, tỉ lệ sửa), không cần nhãn
  bash benchmarks/guidance/run.sh --stats --cycle                                  một vòng Rút → Duyệt hết → Áp dụng → Chấm lại rồi chụp lại (cần provider sinh chữ: 9router)
Dữ liệu E3 (SRS §9): ≥ 60 bài một bài tập; 20 bài đầu «học», 40 bài sau «đo». `labels.csv` (`bt,code,c1..c4,total,labeled_at`) do người chấm MÙ; `--drafts` chứa `drafts-{b0r1,b0r2,b1r1,b1r2,bt-b-after}.json`
do `kc/measure.py snap` chụp (mặc định cùng thư mục với labels.csv). Quy trình đầy đủ và script nạp / chấm: `docs/sprints/10.6/kiem-chung.md`, `benchmarks/guidance/kc/`.
Phép MAE đọc điểm giảng viên từ `teacher_label` (cột c1..c4 của labels.csv); `expected` của ca hồi quy không dùng cho MAE. Số tính bằng decimal.
Thiếu nhãn người → ghi báo cáo `status: WAITING_HUMAN` kèm lý do, thoát 3, KHÔNG chạy giả và không dùng điểm LLM làm chuẩn. Gateway không chạy → thoát 2 «Gateway chưa chạy», không ghi báo cáo.
Kết luận (AC6): «Bật được» khi MAE tổng giảm ≥ 20 % VÀ |lệch TB| không xấu hơn B0 quá 0,20 VÀ số tiêu chí vốn đúng nay sai ≤ 1,5× nhiễu của B0; không thì «Giữ tắt» (không nới ngưỡng).
Biến: API_URL (https://localhost/api/v1), SEED_DEFAULT_PASSWORD (bắt buộc), BENCH_COURSE (761987) / BENCH_ASSIGNMENT (BT03) cho --stats; KC_CLASS (KC106) cho E3; JOB_TIMEOUT (600 s); BENCH_OUT (benchmarks/reports).
Không gọi nhà cung cấp thật ngoài 9router. So hồi quy theo ca: `go run ./cmd/regress run --course <id> --baseline benchmarks/guidance/baseline.json` ở stack đo; lần đầu `--write-baseline`.
"""
import json, os, re, ssl, sys, time, urllib.error, urllib.request, uuid
from datetime import datetime, timezone
from decimal import Decimal as D
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
API = os.environ.get("API_URL", "https://localhost/api/v1").rstrip("/")
PASSWORD = os.environ.get("SEED_DEFAULT_PASSWORD", "")
CLASS = os.environ.get("BENCH_COURSE", "761987")
CODE = os.environ.get("BENCH_ASSIGNMENT", "BT03")
JOB_TIMEOUT = int(os.environ.get("JOB_TIMEOUT", "600"))
OUT = Path(os.environ.get("BENCH_OUT", ROOT / "benchmarks" / "reports"))
CTX = ssl._create_unverified_context() if re.match(r"https://(localhost|127\.0\.0\.1)", API) else None
ORIGIN = re.match(r"(https?://[^/]+)", API).group(1)


def die(msg, code=1):
    print(msg, file=sys.stderr)
    sys.exit(code)


def call(method, path, token=None, body=None, ok=(200, 201, 202)):
    h = {"Accept": "application/json", "Origin": ORIGIN}
    if token:
        h["Authorization"] = f"Bearer {token}"
    if method == "POST":
        h["Idempotency-Key"] = f"bench-{uuid.uuid4()}"
    data = None
    if body is not None:
        data = json.dumps(body).encode()
        h["Content-Type"] = "application/json"
    try:
        with urllib.request.urlopen(urllib.request.Request(API + path, data=data, headers=h, method=method), context=CTX, timeout=60) as r:
            raw = r.read()
            return r.status, (json.loads(raw) if raw else None)
    except urllib.error.HTTPError as e:
        raw = e.read()
        st, body = e.code, (json.loads(raw) if raw else None)
        if st not in ok:
            die(f"{method} {path} → {st} {body}")
        return st, body
    except (urllib.error.URLError, ConnectionError, TimeoutError) as e:
        die(f"Gateway chưa chạy: {e}", 2)


def login():
    if not PASSWORD:
        die("Thiếu SEED_DEFAULT_PASSWORD")
    _, b = call("POST", "/auth/login", body={"email": "teacher@edupilot.local", "password": PASSWORD})
    return b["access_token"]


def stats(token, course, aid):
    return call("GET", f"/courses/{course}/assignments/{aid}/guidance/stats", token)[1]["items"]


def wait_job(token, job_id):
    end = time.time() + JOB_TIMEOUT
    while time.time() < end:
        _, j = call("GET", f"/jobs/{job_id}", token)
        if j["status"] in ("SUCCEEDED", "FAILED"):
            return j
        time.sleep(2)
    die(f"Việc {job_id} quá {JOB_TIMEOUT} s")


def table(rows):
    print("phiên bản | n | MAE tiêu chí | MAE tổng | lệch TB (AI − GV) | tiêu chí sửa | tỉ lệ sửa")
    for r in rows:
        print(f"v{r['version']:<8} | {r['n']} | {r['mae_criterion']} | {r['mae_total']} | {r['bias_total']} | {r['edited_criteria']} | {r['edited_rate']}")


def stats_main(cycle):
    token = login()
    _, mine = call("GET", "/me/courses?limit=100", token)
    course = next((m["course"]["id"] for m in mine["items"] if m["course"]["class_code"] == CLASS), None)
    if not course:
        die(f"Không thấy lớp {CLASS}")
    _, lst = call("GET", f"/courses/{course}/assignments?limit=100", token)
    asg = next((a["id"] for a in lst["items"] if a["submission_code"] == CODE), None)
    if not asg:
        die(f"Không thấy bài tập {CODE}")
    _, ov = call("GET", f"/courses/{course}/assignments/{asg}/guidance", token)
    if not ov["enabled"]:
        die("Cờ «Lưu ý chấm» của lớp đang TẮT: chạy `SEED_GUIDANCE_DEMO=1 node scripts/seed.mjs` trước")
    report = {"at": datetime.now(timezone.utc).isoformat(), "course": CLASS, "assignment": CODE, "before": {"version": ov["current_version"], "stats": stats(token, course, asg)}}
    print("== Trước =="); table(report["before"]["stats"])
    if cycle:
        if not ov["reflect"]["eligible"]:
            die(f"Chưa đủ dữ liệu để rút lưu ý: {ov['reflect']}")
        _, r = call("POST", f"/courses/{course}/assignments/{asg}/guidance/reflect", token, {})
        j = wait_job(token, r["job_id"])
        report["reflect"] = {"status": j["status"], "result": j.get("result")}
        if j["status"] != "SUCCEEDED":
            die(f"Rút lưu ý lỗi: {j.get('error')}")
        _, ov = call("GET", f"/courses/{course}/assignments/{asg}/guidance?limit=100", token)
        drafts = [i for i in ov["items"] if i["status"] == "DRAFT"]
        for it in drafts:
            call("POST", f"/courses/{course}/assignments/{asg}/guidance/items/{it['id']}/decide", token, {"decision": "APPROVE", "version": it["version"]})
        if drafts:
            _, ap = call("POST", f"/courses/{course}/assignments/{asg}/guidance/apply", token, {"expected_version": ov["current_version"]})
            report["applied"] = {"version": ap["version"], "items": len(drafts), "stale": ap["stale"]}
            st, rg = call("POST", f"/courses/{course}/assignments/{asg}/regrade-stale", token, {})
            if st == 202:
                report["regrade"] = {"queued": rg["queued"], "status": wait_job(token, rg["job_id"])["status"]}
        _, ov = call("GET", f"/courses/{course}/assignments/{asg}/guidance", token)
        report["after"] = {"version": ov["current_version"], "stats": stats(token, course, asg)}
        print("== Sau =="); table(report["after"]["stats"])
        by = {r["version"]: r for r in report["after"]["stats"] if r["bias_total"] is not None}
        old, new = by.get(report["before"]["version"]), by.get(report["after"]["version"])
        if old and new and abs(float(new["bias_total"])) > abs(float(old["bias_total"])):
            report["verdict"] = "LÙI"
            print("LÙI: |lệch TB| tăng so với phiên bản trước", file=sys.stderr)
    OUT.mkdir(parents=True, exist_ok=True)
    path = OUT / f"guidance-{datetime.now().strftime('%Y%m%d-%H%M%S')}.json"
    path.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n")
    print(f"Báo cáo: {path}")
    sys.exit(1 if report.get("verdict") == "LÙI" else 0)


def e3(a):
    """Giao thức E3 trước / sau trên nhãn người; xem docstring đầu tệp."""
    sys.path.insert(0, str(ROOT / "benchmarks" / "guidance" / "kc"))
    import measure
    from common import find_assignment, find_course, login
    token, _ = login("teacher")  # gateway không chạy → thoát 2 «Gateway chưa chạy», chưa ghi gì (AC10)
    labels = Path(a.labels)
    OUT.mkdir(parents=True, exist_ok=True)
    stem = OUT / "guidance-before-after"
    if not labels.exists():
        reason = f"Chưa có nhãn người (teacher_label): không thấy {labels}. Người chấm mù theo kc/sheet.py rồi `sheet.py --collect`; không dùng điểm LLM làm chuẩn."
        stem.with_suffix(".json").write_text(json.dumps({"status": "WAITING_HUMAN", "reason": reason, "at": datetime.now(timezone.utc).isoformat()}, ensure_ascii=False, indent=2) + "\n")
        stem.with_suffix(".md").write_text(f"# Học từ lần sửa — trước / sau\n\n**WAITING_HUMAN.** {reason}\n", encoding="utf8")
        print("WAITING_HUMAN: " + reason, file=sys.stderr)
        sys.exit(3)
    d = Path(a.drafts) if a.drafts else labels.parent
    snaps = {p.stem.removeprefix("drafts-"): json.loads(p.read_text())["items"] for p in d.glob("drafts-*.json")}
    order = json.loads((measure.DATA / "order.json").read_text()) if (measure.DATA / "order.json").exists() else None
    m, errs = measure.compute(snaps, measure.read_labels(labels), order)
    if errs:
        die("Số đo không dựng được (HARNESS_LỖI):\n  " + "\n  ".join(errs))
    b0, b1 = m["b0"], m["b1"]
    drop = (b0["mae_total"] - b1["mae_total"]) / b0["mae_total"] if b0["mae_total"] else None
    base = D("1.5") * m["noise_items"] if m["noise_items"] else 4
    checks = {"mae_total_giam_20pct": drop is not None and drop >= D("0.20"), "lech_tb_khong_xau_hon_0.20": abs(b1["bias_total"]) <= abs(b0["bias_total"]) + D("0.20"), "von_dung_nay_sai_le_1.5x_co_so": m["was_right_now_wrong"] <= base}
    verdict = "Bật được" if all(checks.values()) else "Giữ tắt"
    course = find_course(token)
    _, ov = call("GET", f"/courses/{course}/assignments/{find_assignment(token, course, 'bt-a')['id']}/guidance", token)
    out = {"status": "DONE", "verdict": verdict, "checks": checks, "mae_total_drop": drop, "label_source": a.label_source, "provider": a.provider, "date": datetime.now().strftime("%Y-%m-%d"),
           "n_measure": m["n_a"], "guidance_counts": ov.get("counts"), "metrics": m}
    stem.with_suffix(".json").write_text(json.dumps(measure.jsonable(out), ensure_ascii=False, indent=2) + "\n")
    stem.with_suffix(".svg").write_text(measure.svg(m), encoding="utf8")
    c = lambda b: "đạt" if b else "KHÔNG"
    stem.with_suffix(".md").write_text(f"""# Học từ lần sửa — trước / sau

**Kết luận: {verdict}.**

| Chỉ số (đo, n = {m['n_a']}) | B0 (v0) | B1 (v1) |
| --- | --- | --- |
| MAE tổng | {m['b0']['mae_total']:.3f} | {m['b1']['mae_total']:.3f} |
| MAE tiêu chí | {m['b0']['mae_criterion']:.3f} | {m['b1']['mae_criterion']:.3f} |
| Lệch TB có dấu (AI − giảng viên) | {m['b0']['bias_total']:.3f} | {m['b1']['bias_total']:.3f} |
| Tiêu chí phải sửa tay (Δ tuyệt đối ≥ 0,5) | {m['fix_b0']} | {m['fix_b1']} |

Bài tốt lên / xấu đi: {m['better']} / {m['worse']}. Lưu ý (theo trạng thái): {json.dumps(ov.get('counts'), ensure_ascii=False)}.

1. MAE tổng giảm ≥ 20 % ({'—' if drop is None else f'{drop:.1%}'}): {c(checks['mae_total_giam_20pct'])}
2. |lệch TB| không xấu hơn B0 quá 0,20: {c(checks['lech_tb_khong_xau_hon_0.20'])}
3. Tiêu chí vốn đúng nay sai ({m['was_right_now_wrong']}) ≤ 1,5× cơ sở (nhiễu {m['noise_items']}): {c(checks['von_dung_nay_sai_le_1.5x_co_so'])}

Nguồn nhãn: {a.label_source}. Nhà cung cấp / model: {a.provider}. Ngày: {out['date']}.

## Giới hạn
Bài mô phỏng; một model chấm; một bài tập học; n = {m['n_a']}; nhãn từ {a.label_source}, không thay nhãn người thật nếu không phải người.
""", encoding="utf8")
    print(f"{verdict} → {stem}.{{json,md,svg}}")
    sys.exit(0 if verdict == "Bật được" else 1)


def main():
    import argparse
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("--stats", action="store_true", help="chụp số đo từ API (không cần nhãn)")
    ap.add_argument("--cycle", action="store_true", help="với --stats: chạy một vòng Rút → Duyệt → Áp dụng → Chấm lại")
    ap.add_argument("--labels", default=str(OUT / "guidance" / "labels.csv"))
    ap.add_argument("--drafts")
    ap.add_argument("--label-source", default="người chấm mù (E3)")
    ap.add_argument("--provider", default=os.environ.get("BENCH_PROVIDER", "9router (ag/gemini-3.8-flash)"))
    a = ap.parse_args()
    if a.cycle and not a.stats:
        ap.error("--cycle đi kèm --stats")
    stats_main(a.cycle) if a.stats else e3(a)


if __name__ == "__main__":
    main()
