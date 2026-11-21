#!/usr/bin/env python3
"""`make eval` / `make eval-report` (US-P10-10, FR-20). Mỗi bước bỏ qua nếu kết quả thô đã có (chạy lại = tiếp tục).

  python3 benchmarks/eval/run.py            đo E1–E6 trên stack ep-qc rồi viết báo cáo
  python3 benchmarks/eval/run.py --report   chỉ tính báo cáo từ raw/ (không cần stack, không gọi LLM — trừ gen-stats E5 nếu stack còn)
Biến: API_URL (https://localhost:943/api/v1), SEED_DEFAULT_PASSWORD, EVAL_P1 (ag/gemini-3.8-flash), EVAL_P2 (ag/claude-sonnet-4-6),
EVAL_REAL_EMBED=1 (nhúng thật cho E1/E2/E4 — chỉ khi PM cho phép; xong tự trả về nhúng giả), EVAL_DATE (thư mục báo cáo eval-<ngày>).
"""
import os
import subprocess
import sys

import lib

P1 = os.environ.get("EVAL_P1", "ag/gemini-3.8-flash")
P2 = os.environ.get("EVAL_P2", "ag/claude-sonnet-4-6")


def step(done_marker, *args):
    if done_marker and (lib.RAW / done_marker).exists():
        print(f"· bỏ qua {' '.join(args)} (đã có raw/{done_marker})")
        return
    print(f"▶ {' '.join(args)}", flush=True)
    r = subprocess.run([sys.executable, str(lib.HERE / args[0]), *args[1:]])
    if r.returncode:
        lib.die(f"bước {' '.join(args)} thoát {r.returncode}", r.returncode)


def measure():
    step("e6-r1.json", "e6.py", "run", "--tag", "r1")
    step("e1-fake-a/e1.json", "e1.py", "run", "--tag", "fake-a")
    step("e1-fake-b/e1.json", "e1.py", "run", "--tag", "fake-b")
    step(None, "stack.py", "route", "--tag", "P1", "--model", P1)
    step("e3-P1r1.json", "e3.py", "setup")
    step("e3-P1r1.json", "e3.py", "grade", "--tag", "P1r1", "--first")
    step("e3-P1r2.json", "e3.py", "grade", "--tag", "P1r2")
    step(None, "e5.py", "run")
    real = os.environ.get("EVAL_REAL_EMBED") == "1"
    if real:
        step("e2-P2.json", "stack.py", "embed-real")
        step("e1-real/e1.json", "e1.py", "run", "--tag", "real")
    step("e2-P1.json", "e2.py", "run", "--tag", "P1")
    step(None, "stack.py", "route", "--tag", "P2", "--model", P2)
    step("e2-P2.json", "e2.py", "run", "--tag", "P2")
    if real:
        step(None, "stack.py", "embed-fake")
    step("e3-P2r1.json", "e3.py", "grade", "--tag", "P2r1")
    step(None, "e3.py", "grade", "--tag", "P2r1", "--retry-failed")  # nhà cung cấp quá tải giữa lượt: chấm lại tuần tự phần FAILED
    step(None, "stack.py", "route", "--tag", "P1", "--model", P1)
    step(None, "e1.py", "leak")
    step("e6-r2.json", "e6.py", "run", "--tag", "r2")


def report():
    for s in ("e1.py", "e2.py", "e3.py", "e4.py", "e5.py", "e6.py"):
        step(None, s, "report")
    print(f"Báo cáo: {lib.OUT}")


if __name__ == "__main__":
    if "--report" not in sys.argv:
        if not lib.ready():
            lib.die("Gateway chưa chạy")  # AC12: thoát ≠ 0, không ghi báo cáo
        lib.refuse_demo()
        measure()
    report()
