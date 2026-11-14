#!/usr/bin/env python3
"""Phiếu chấm MÙ cho `giaovien` (kiem-chung.md §1.4). Chỉ thư viện chuẩn.

  python3 benchmarks/guidance/kc/sheet.py                       sinh ~/kc-giaovien/phieu/<bt>/<kcNNN>.md cho 64 bài đo (BT-A kc021–kc060 + BT-B kc001–kc024)
  python3 benchmarks/guidance/kc/sheet.py --collect --report benchmarks/reports/guidance/kc-2026-11-15     đọc phiếu đã điền → labels.csv (`teacher_label`)
Phiếu CHỈ có đề, rubric, văn bản bài và khung điền; không có điểm AI, hồ sơ ẩn hay mức kỳ vọng. Điền «c1: 2.25» … mỗi tiêu chí bước 0,25 trong [0; 2,5].
Biến: KC_HOME (~/kc-giaovien), KC_DATA (kc/data).
"""
import argparse, csv, json, os, re, sys
from datetime import datetime, timezone
from decimal import Decimal, InvalidOperation
from pathlib import Path
from common import DATA, bt_students, die, measure_keys

HOME = Path(os.environ.get("KC_HOME", Path.home() / "kc-giaovien"))
TITLE = {"bt-a": "BT-A", "bt-b": "BT-B"}
LINE = re.compile(r"^\s*c([1-4])\s*[:=]\s*(.*?)\s*$", re.M)


def sheets_dir(a):
    return Path(a.phieu) if a.phieu else HOME / "phieu"


def build(a):
    rub = json.loads((DATA / "rubric.json").read_text(encoding="utf8"))
    n = 0
    for bt in ("bt-a", "bt-b"):
        d = sheets_dir(a) / bt
        d.mkdir(parents=True, exist_ok=True)
        by_key = {s["key"]: s for s in bt_students(bt)}
        for key in measure_keys(bt)[: a.limit or None]:
            txt = DATA / bt / f"{by_key[key]['code']}.txt"
            if not txt.exists():
                continue
            crit = "\n".join(f"- **c{i}** — {c['name']} (0–{c['max_score']}, bước 0,25): {c['guidance']}" for i, c in enumerate(rub["rubric"], 1))
            topic, body = rub["topic"][bt], txt.read_text(encoding="utf8").strip()
            form = "\n".join(f"c{i}: " for i in range(1, len(rub["rubric"]) + 1))
            (d / f"{key}.md").write_text(f"# Phiếu chấm {TITLE[bt]} — {key}\n\n## Đề bài\n{topic}\n\n## Rubric\n{crit}\n\n## Bài làm\n{body}\n\n## Điểm (điền số sau dấu hai chấm)\n{form}\n", encoding="utf8")
            n += 1
    return n


def collect(a):
    if not a.report:
        die("--collect cần --report <thư mục báo cáo>")
    rows, bad = [], []
    for bt in ("bt-a", "bt-b"):
        for key in measure_keys(bt)[: a.limit or None]:
            p = sheets_dir(a) / bt / f"{key}.md"
            if not p.exists():
                bad.append(f"{bt}/{key}: không có phiếu")
                continue
            got = {m.group(1): m.group(2) for m in LINE.finditer(p.read_text(encoding="utf8").split("## Điểm", 1)[-1])}
            try:
                sc = [Decimal(got[str(i)].replace(",", ".")) for i in range(1, 5)]
            except (KeyError, InvalidOperation):
                bad.append(f"{bt}/{key}: thiếu hoặc sai số ở c1–c4")
                continue
            if any(x < 0 or x > Decimal("2.5") or x % Decimal("0.25") != 0 for x in sc):
                bad.append(f"{bt}/{key}: điểm ngoài [0; 2,5] hoặc không theo bước 0,25")
                continue
            at = datetime.fromtimestamp(p.stat().st_mtime, timezone.utc).isoformat(timespec="seconds")
            rows.append([bt, key, *map(str, sc), str(sum(sc)), at])
    if bad:
        die("Phiếu chưa hợp lệ:\n  " + "\n  ".join(bad))
    out = Path(a.report)
    out.mkdir(parents=True, exist_ok=True)
    with (out / "labels.csv").open("w", newline="", encoding="utf8") as f:
        w = csv.writer(f)
        w.writerow(["bt", "code", "c1", "c2", "c3", "c4", "total", "labeled_at"])
        w.writerows(rows)
    print(f"labels.csv: {len(rows)} phiếu → {out}")


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("--collect", action="store_true")
    ap.add_argument("--report", help="thư mục báo cáo (benchmarks/reports/guidance/kc-<ngày>)")
    ap.add_argument("--phieu", help="thư mục phiếu (mặc định ~/kc-giaovien/phieu)")
    ap.add_argument("--limit", type=int, default=0, help="chỉ N phiếu đầu của mỗi bài tập (chạy thử; dùng cả khi --collect)")
    a = ap.parse_args()
    if a.collect:
        collect(a)
    else:
        print(f"Đã sinh {build(a)} phiếu ở {sheets_dir(a)}")


if __name__ == "__main__":
    main()
