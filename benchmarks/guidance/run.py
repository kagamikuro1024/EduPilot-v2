#!/usr/bin/env python3
"""Đo «học từ lần sửa» (FEAT-grading-guidance M1, US-GD-10): số đo theo PHIÊN BẢN lưu ý của một bài tập. Chỉ thư viện chuẩn, chỉ HTTP (D46): chạy như một client ngoài.

  bash benchmarks/guidance/run.sh            chụp số đo hiện có (n, MAE tiêu chí / tổng, lệch TB có dấu, tỉ lệ sửa) theo từng phiên bản
  bash benchmarks/guidance/run.sh --cycle    chạy một vòng: Rút lưu ý → Duyệt hết → Áp dụng → Chấm lại bài nháp bản cũ, rồi chụp lại (cần provider sinh chữ: 9router ở môi trường đo)
Biến: API_URL (https://localhost/api/v1), SEED_DEFAULT_PASSWORD (bắt buộc), BENCH_COURSE (761987), BENCH_ASSIGNMENT (BT03), JOB_TIMEOUT (600 s), BENCH_OUT (benchmarks/reports).
Điều kiện: stack đã dựng + `SEED_GUIDANCE_DEMO=1 node scripts/seed.mjs` (cờ lớp BẬT, 8 kết quả đã duyệt «sở thích gắt»: tiêu chí đầu giảng viên thấp hơn AI 1,0).
Số đo là chuỗi decimal của máy chủ (`GET …/guidance/stats`); script không tính lại điểm. Mốc đạt (SRS §9): sau một vòng `|lệch TB|` ở phiên bản mới KHÔNG lớn hơn phiên bản cũ.
So hồi quy theo ca (chấm lại văn bản bài với lưu ý hiện hành): `go run ./cmd/regress run --course <id> --baseline benchmarks/guidance/baseline.json` ở stack đo; lần đầu `--write-baseline`.
Không gọi nhà cung cấp thật ngoài 9router; thoát mã 2 «Gateway chưa chạy» khi không nối được.
"""
import json, os, re, ssl, sys, time, urllib.error, urllib.request, uuid
from datetime import datetime, timezone
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


def main():
    cycle = "--cycle" in sys.argv[1:]
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


if __name__ == "__main__":
    main()
